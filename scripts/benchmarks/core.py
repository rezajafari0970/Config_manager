"""Owned-child core adapter. Never stops or edits another VPN instance."""
import copy,ctypes,hashlib,json,os,pathlib,signal,socket,subprocess,threading,time,sys

def digest(data): return hashlib.sha256(data).hexdigest()
def encoded(config): return json.dumps(config,sort_keys=True,separators=(',',':')).encode()
def free_port():
    with socket.socket() as s:
        s.bind(('127.0.0.1',0));return s.getsockname()[1]
def parent_guard():
    libc=ctypes.CDLL(None,use_errno=True)
    if libc.prctl(1,signal.SIGTERM,0,0,0)!=0: raise OSError('parent_death_signal')
    if os.getppid()==1: os.kill(os.getpid(),signal.SIGTERM)
def validate(core,path,timeout=10):
    try:
        p=subprocess.run([str(core),'run','-test','-c',str(path)],capture_output=True,timeout=timeout)
        pathlib.Path(path).with_suffix('.validation.log').write_bytes((p.stdout+p.stderr)[-65536:])
        return {'status':'passed' if p.returncode==0 else 'rejected','exit_code':p.returncode,'diagnostic_sha256':digest(p.stdout+p.stderr)}
    except subprocess.TimeoutExpired:return {'status':'infrastructure_timeout'}

class Session:
    def __init__(self,core,config,work):
        self.core=pathlib.Path(core);self.original=config;self.work=pathlib.Path(work)
        self.process=None;self.log=None;self.stop_sampling=threading.Event();self.max_rss=0;self.cpu_ticks=0
    def __enter__(self):
        self.work.mkdir(parents=True,exist_ok=True,mode=0o700)
        self.original_bytes=encoded(self.original);(self.work/'original.json').write_bytes(self.original_bytes)
        self.build=validate(self.core,self.work/'original.json')
        if self.build['status']!='passed':raise ValueError('core_validation_'+self.build['status'])
        c=copy.deepcopy(self.original);ins=c.get('inbounds',[])
        socks_ins=[i for i in ins if i.get('protocol')=='socks']
        if len(socks_ins)!=1 or any(i.get('listen')!='127.0.0.1' or not isinstance(i.get('port'),int) for i in ins):raise ValueError('unsupported_benchmark_inbounds')
        mapping={};used=set()
        def allocate():
            for _ in range(100):
                port=free_port()
                if port not in used:used.add(port);return port
            raise RuntimeError('local_port_allocation')
        for i in ins:
            old=i['port']
            if old in mapping:raise ValueError('unsupported_duplicate_inbound_port')
            mapping[old]=allocate();i['port']=mapping[old]
        self.port=socks_ins[0]['port']
        def rewrite(obj):
            if isinstance(obj,dict):
                if obj.get('address') in ('127.0.0.1','localhost') and isinstance(obj.get('port'),int) and obj['port'] in mapping:obj['port']=mapping[obj['port']]
                for v in obj.values():rewrite(v)
            elif isinstance(obj,list):
                for v in obj:rewrite(v)
        rewrite(c.get('outbounds',[]));self.port_map=mapping
        self.metrics_port=None
        if c.get('metrics'):
            self.metrics_port=allocate();c['metrics']['listen']='127.0.0.1:'+str(self.metrics_port)
        self.runtime_bytes=encoded(c);(self.work/'runtime.json').write_bytes(self.runtime_bytes)
        self.log=(self.work/'core.log').open('wb');self.started=time.monotonic()
        env=dict(os.environ,GOMAXPROCS='2');env.pop('XRAY_LOCATION_ASSET',None)
        self.process=subprocess.Popen([sys.executable,str(pathlib.Path(__file__).resolve()),'--exec',str(self.core),str(self.work/'runtime.json')],stdout=self.log,stderr=self.log,env=env,start_new_session=True)
        try:
            self.sampler=threading.Thread(target=self.sample,daemon=True);self.sampler.start()
            while time.monotonic()-self.started<4:
                if self.process.poll() is not None:raise RuntimeError('core_startup_failed')
                try:
                    with socket.create_connection(('127.0.0.1',self.port),.05):break
                except OSError:time.sleep(.02)
            else:raise RuntimeError('local_listener_timeout')
            self.startup_ms=round((time.monotonic()-self.started)*1000,2);return self
        except BaseException:self.__exit__(None,None,None);raise
    def sample(self):
        while not self.stop_sampling.is_set():
            try:
                p=pathlib.Path('/proc')/str(self.process.pid)
                stat=(p/'stat').read_text().rsplit(')',1)[1].split();self.cpu_ticks=int(stat[11])+int(stat[12])
                for line in (p/'status').read_text().splitlines():
                    if line.startswith('VmRSS:'):self.max_rss=max(self.max_rss,int(line.split()[1])*1024)
            except (OSError,ValueError,IndexError):pass
            self.stop_sampling.wait(.2)
    def resources(self):
        elapsed=max(.001,time.monotonic()-self.started);cpu=self.cpu_ticks/os.sysconf('SC_CLK_TCK')
        return {'core_cpu_seconds':round(cpu,3),'core_cpu_one_cpu_percent':round(100*cpu/elapsed,2),'core_peak_rss_bytes':self.max_rss,'observed_seconds':round(elapsed,3),'scope':'owned Xray process only'}
    def __exit__(self,*args):
        self.stop_sampling.set()
        if self.process and self.process.poll() is None:
            self.process.terminate()
            try:self.process.wait(timeout=3)
            except subprocess.TimeoutExpired:self.process.kill();self.process.wait()
        if hasattr(self,'sampler'):self.sampler.join(timeout=1)
        if self.log:self.log.close()

if __name__=='__main__':
    if len(sys.argv)!=4 or sys.argv[1]!='--exec':raise SystemExit(2)
    parent_guard();os.execv(sys.argv[2],[sys.argv[2],'run','-c',sys.argv[3]])
