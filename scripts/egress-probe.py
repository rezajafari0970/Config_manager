#!/usr/bin/env python3
"""Trusted driver orchestrator. Configs enter exclusively through stdin.
Each job owns one isolated netns/cgroup. Core fallback never weakens TLS.
"""
import argparse,base64,hashlib,hmac,secrets,fcntl,ipaddress,contextlib,copy,importlib.util,json,os,pathlib,re,signal,subprocess,sys,threading,time
HERE=pathlib.Path(__file__).resolve().parent
def module(name,file):
 spec=importlib.util.spec_from_file_location(name,HERE/file);m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m
lab=module('egress_lab','live-probe.py');scope=module('egress_scope','egress-preflight.py');country=module('egress_country','egress-country.py')
CORES=[('xray-26.11.9','/opt/config-repair/core-26.11.9/xray','e87bce586acdb9136aa5b1f1d9c8db9f1ea1d145b6d86368c7f6f662f9f6a66d'),('xray-26.10.31','/opt/config-repair/core-26.10.31/xray','cc4b84d3069569f297a61f04b914f2bb37e9d202fd7b84cb50def1fc30fd40e1'),('xray-26.7.28','/opt/config-repair/core-26.7.28/xray','64d46afb80adea1bf97a0d467e83f4a9ac1ebd0995891e84bca3f1a1d1affb1d'),('xray-26.5.9','/opt/config-repair/core-26.5.9/xray','b2c525e082cf2fef460499c88838d355f0b9bfb5a00bdb3eaa99b6af63825006')]
NATIVE=('sing-box-1.14.4','/opt/config-repair/sing-box-1.14.4/sing-box','1264361b5f8868c8768f7555d3e77e20118bf503c41ac08bb290a0d99056aa3a')
def cleanup(prefix,slot=None,run=None):
 # Reuse only the shared sandbox's resource deletion mechanics, with a new
 # disjoint ownership regex. Never delete another pipeline's namespaces.
 parent=pathlib.Path('/sys/fs/cgroup')/(prefix+'-sandboxes')
 owned=re.compile('^'+prefix+r'[0-9a-f]{6}20[01][0-9a-f]$')
 children=[p for p in parent.glob(prefix+'*') if owned.fullmatch(p.name) and (slot is None or p.name.endswith(format(8192+slot,'x'))) and (run is None or p.name.startswith(prefix+run))]
 for p in children:
  with contextlib.suppress(OSError):(p/'cgroup.kill').write_text('1')
 adapter=module('egress_cleanup','probe-one.py');adapter.PREFIX=prefix;adapter.OWN=re.compile('^'+prefix+r'[0-9a-f]{6}20[01][0-9a-f]$');adapter.cleanup(slot,run)
 for p in children:
  with contextlib.suppress(OSError):p.rmdir()
class EgressBox(lab.Sandbox):
 def __init__(self,core,runid,index,prefix='cge'):
  super().__init__(core,runid,index,prefix)
  parent=pathlib.Path('/sys/fs/cgroup')/(prefix+'-sandboxes')
  with open('/run/'+prefix+'-sandbox-budget.lock','a') as lock:
   fcntl.flock(lock,fcntl.LOCK_EX);parent.mkdir(exist_ok=True)
   (parent/'memory.max').write_text(str(1024*1024*1024));(parent/'pids.max').write_text('4096');(parent/'cpu.max').write_text('200000 100000')
   (parent/'cgroup.subtree_control').write_text('+memory +pids +cpu')
  self.cg=parent/self.name;self.policy_key=None
 def network_command(self):return ['/usr/bin/nsenter','--net=/run/netns/'+self.name,'--']
 def launch_command(self):return ['/usr/bin/python3',str(HERE/'egress-launch.py'),str(self.cg)]
 def policy(self,resolved):
  key=json.dumps(resolved,sort_keys=True,separators=(',',':'))
  if key==getattr(self,'policy_key',None):return
  super().policy(resolved);self.policy_key=key
class NativeBox(EgressBox):
 def core_mounts(self):
  out=['--ro-bind','/lib64/ld-linux-x86-64.so.2','/lib64/ld-linux-x86-64.so.2']
  for name in ('libdl.so.2','libpthread.so.0','libc.so.6'):out+=['--ro-bind','/usr/lib/x86_64-linux-gnu/'+name,'/lib/x86_64-linux-gnu/'+name]
  return out
 def core_command(self,test):return ['/xray','check' if test else 'run','-c','/config.json']
class SessionBox(EgressBox):
 driver='xray'
 def core_mounts(self):return NativeBox.core_mounts(self) if self.driver!='xray' else []
 def core_command(self,test):return NativeBox.core_command(self,test) if self.driver!='xray' else lab.Sandbox.core_command(self,test)
class ResourcePool:
 def __init__(self,args):self.args=args;self.box=None;self.pins={};self.geo=country.Registry()
 def __enter__(self):return self
 def __exit__(self,*exc):self.close();self.geo.close()
 def close(self):
  if self.box is not None:
   box=self.box;self.box=None;box.__exit__(None,None,None)
 def verify(self,path,pin):
  p=pathlib.Path(path);st=p.stat();key=(st.st_dev,st.st_ino,st.st_size,st.st_mtime_ns,pin)
  if self.pins.get(path)!=key:
   if lab.digest(p.read_bytes())!=pin:raise ValueError('core_pin_mismatch')
   self.pins[path]=key
 @contextlib.contextmanager
 def lease(self,path,driver):
  if self.box is None:
   box=SessionBox(path,self.args.run_id,8192+self.args.slot,prefix=self.args.prefix)
   box.__enter__();self.box=box
  box=self.box;box.stop();box.policy({});box.core=pathlib.Path(path).resolve();box.driver=driver
  try:yield box
  finally:
   box.stop();box.policy({})
   for _ in range(20):
    if (box.cg/'pids.current').read_text().strip()=='0':break
    time.sleep(.05)
   else:self.close();raise ValueError('sandbox_not_empty')
   for name in ('config.json','hosts','resolv.conf'):(box.work/name).unlink(missing_ok=True)
def runtime(config,driver):
 if driver=='xray':
  return scope.runtime(config,lab)
 out=config['outbounds']
 if len(out)!=1 or out[0].get('detour') or out[0].get('server_ports') or out[0].get('domain_resolver'):raise ValueError('native_path_unsupported')
 ob=out[0];host=ob['server'];port=ob['server_port']
 resolved=lab.endpoints({'outbounds':[{'protocol':'socks','settings':{'servers':[{'address':host,'port':port}]}}]})
 obj=copy.deepcopy(config);previous=next((x for x in obj.get('inbounds',[]) if x.get('type')=='socks'),{})
 obj['inbounds']=[{'type':'socks','tag':previous.get('tag','socks'),'listen':'127.0.0.1','listen_port':10808}];obj['log']={'level':'warn'}
 obj['outbounds'][0]['server']=resolved[host]['ips'][0]
 obj['outbounds'][0].setdefault('tls',{}).setdefault('server_name',host)
 return resolved,obj
def core_order(plan):
 # Known capability dispatch, not a network-failure guess or a TLS downgrade.
 # These exact public plaintext VLESS inputs are prohibited on the newer cores.
 try:
  cfg=json.loads(base64.b64decode(plan['variants'][0]['config'],validate=True))
  for ob in cfg.get('outbounds',[]):
   if ob.get('protocol')!='vless' or ob.get('streamSettings',{}).get('security','none') not in ('','none'):continue
   settings=ob.get('settings',{})
   servers=settings.get('vnext',[])
   if settings.get('address'):servers=[dict(address=settings['address'],users=[settings])]
   for server in servers:
    try:public=ipaddress.ip_address(server.get('address','')).is_global
    except ValueError:public=bool(server.get('address')) and server['address'].lower() not in ('localhost',) and not server['address'].lower().endswith(('.localhost','.local'))
    if public and all(u.get('encryption','none') in ('','none') for u in server.get('users',[])):
     return [CORES[-1]]+CORES[:-1]
 except (ValueError,KeyError,TypeError):pass
 return CORES
_control_cache=(0,'')
def control_ipv4():
 # A dual-stack control name may otherwise use IPv6 while Ads uses IPv4.
 # Resolve the fixed public name outside the tunnel; NEVER add it to the
 # namespace firewall. The SOCKS connection and verified TLS still use proxy.
 global _control_cache
 now=time.monotonic()
 if _control_cache[0]>now:return _control_cache[1]
 value=''
 try:
  helper='import socket,json;print(json.dumps(sorted({x[4][0] for x in socket.getaddrinfo("www.cloudflare.com",443,socket.AF_INET,socket.SOCK_STREAM)})))'
  r=subprocess.run(['/usr/bin/python3','-c',helper],capture_output=True,check=True,timeout=3)
  ips=json.loads(r.stdout)
  valid=[str(ipaddress.ip_address(ip)) for ip in ips if ipaddress.ip_address(ip).version==4 and ipaddress.ip_address(ip).is_global and not ipaddress.ip_address(ip).is_multicast and not ipaddress.ip_address(ip).is_reserved]
  if valid:value=valid[0]
 except Exception:pass
 _control_cache=(now+(300 if value else 5),value)
 return value
def raw_shared_egress(rows,waves):
 # Rejection only; Go independently verifies every required observation.
 stable={}
 for wave in range(1,waves+1):
  paired={}
  for row in rows:
   if row.get('wave')!=wave or not all(row.get(k) for k in ('echo','tls','same_stream')):continue
   domain=row.get('domain');family=row.get('family');source=row.get('ip')
   group='cf' if domain in ('www.cloudflare.com','one.one.one.one') else 'ipify' if domain in ('api.ipify.org','api6.ipify.org') else ''
   if group:
    previous=paired.setdefault(family,{}).get(group)
    if previous and previous!=source:return False
    paired[family][group]=source;stable.setdefault(family,set()).add(source)
  for domain in ['googleads.g.doubleclick.net','pubads.g.doubleclick.net','securepubads.g.doubleclick.net']:
   matching=[x for x in rows if x.get('wave')==wave and x.get('domain')==domain and all(x.get(k) for k in ('echo','tls','same_stream','actual_ads'))]
   if len(matching)!=1:return False
   row=matching[0];group=paired.get(row.get('family'),{})
   if len(group)!=2 or any(x!=row.get('ip') for x in group.values()):return False
 return bool(stable) and all(len(sources)==1 for sources in stable.values())
def private_sources(samples,key):
 for row in samples:
  value=row.pop('ip','')
  row['source']=hmac.new(key,value.encode(),hashlib.sha256).hexdigest() if value and row.get('echo') else ''
 return samples
def error_code(error):
 return 'driver_subprocess_timeout' if isinstance(error,subprocess.TimeoutExpired) else 'driver_subprocess_failed' if isinstance(error,subprocess.CalledProcessError) else str(error) if isinstance(error,ValueError) and re.fullmatch('[a-z0-9_]{1,60}',str(error)) else 'driver_infrastructure_error'
def investigate(req,args,pool=None):
 if pool is None:
  with ResourcePool(args) as own:return investigate(req,args,own)
 plan=req['plan'];r=dict(version='ads-egress.v5',hash=req['hash'],started=req['started'],finished=0,core='',core_sha='',infrastructure=False,code='driver_unavailable',checks=[],samples=[],health=[],attempts=[],wall_ms=0,waves=3 if req.get('deep') else 2)
 begin=time.monotonic();best=-1
 if len(plan.get('variants',[]))!=2:return r
 control_ip=control_ipv4()
 for core,path,pin in (core_order(plan) if plan['driver']=='xray' else [NATIVE]):
  at=time.monotonic();checks=[];samples=[];health=[];code='core_unavailable';stages={};checkpoint=at
  def mark(name):
   nonlocal checkpoint
   current=time.monotonic();stages[name]=(current-checkpoint)*1000;checkpoint=current
  try:
   pool.verify(path,pin);mark('pin')
   with pool.lease(path,plan['driver']) as box:
    mark('namespace')
    for v in plan['variants']:
     data=base64.b64decode(v['config'],validate=True)
     if len(data)>8*1024*1024 or lab.digest(data)!=v['hash']:raise ValueError('variant_hash_mismatch')
     config=json.loads(data);box.policy({})
     # Successful isolated runtime startup validates the exact bytes too. Avoid
     # spawning an identical -test core before every actual launch.
     check=dict(name=v['name'],hash=v['hash'],runtime_hash='',core=False,code='runtime_pending');checks.append(check)
     resolved,obj=runtime(config,plan['driver']);body=data if plan['driver']=='xray' else json.dumps(obj,separators=(',',':')).encode();box.resolved=resolved;box.policy(resolved);check['runtime_hash']=lab.digest(body);mark(v['name']+'_resolve')
     p=box.launch(body);ready=False
     for _ in range(30):
      if p.poll() is not None:break
      ok=box.net(['/usr/bin/python3','-c','import socket;s=socket.socket();s.settimeout(.1);print(s.connect_ex(("127.0.0.1",10808)))'],timeout=2)
      if ok.stdout.strip()==b'0':ready=True;break
      time.sleep(.1)
     mark(v['name']+'_ready')
     if not ready:
      code=lab.classify(box.logs(),'runtime_start_failed');check.update(core=False,code=code);break
     check.update(core=True,code='exact_runtime_started')
     def observe(start,waves,timeout):
      result=box.net(['/usr/bin/python3',str(HERE/'egress-session.py')],input=json.dumps({'variant':v['name'],'timeout':timeout,'start_wave':start,'waves':waves,'sequential':sequential,'adaptive':bool(req.get('deep')),'control_ipv4':control_ip}).encode(),timeout=max(25,timeout*(waves if sequential else 2)+10))
      if len(result.stdout)>131072:raise ValueError('session_output_limit')
      rows=json.loads(result.stdout)
      if not isinstance(rows,list) or not min(waves,2)*5<=len(rows)<=waves*7:raise ValueError('session_contract')
      return rows
     # XHTTP and deep retries avoid overloading one remote path with 21 connections.
     sequential=bool(req.get('deep')) or any(o.get('streamSettings',{}).get('network') in ('xhttp','splithttp') for o in config.get('outbounds',[]))
     rows=observe(1,r['waves'],8 if req.get('deep') else 3)
     samples+=rows
     # Existing members always receive a real payload check. New candidates
     # only pay this cost when a dedicated Ads path remains possible.
     if req.get('accepted') or any(x.get('actual_ads') and x.get('echo') for x in rows) and not raw_shared_egress(rows,max(x['wave'] for x in rows)):
      result=box.net(['/usr/bin/python3',str(HERE/'egress-health.py')],timeout=12)
      h=json.loads(result.stdout);h['variant']=v['name'];health.append(h)
     box.stop();code='measured';mark(v['name']+'_probes')
     if not req.get('accepted') and v['name']=='raw' and raw_shared_egress(rows,max(x['wave'] for x in rows)):code='raw_shared_egress';break
     if not req.get('accepted') and not req.get('deep') and not any(x.get('echo') for x in rows):
      code='network_retry_pending';break
   mark('reset')
   score=sum(bool(x.get('echo')) for x in samples)
   if len(checks)==2 and all(x['core'] for x in checks):score+=100
   if score>best:r.update(core=core,core_sha=pin,checks=checks,samples=samples,health=health,code=code);best=score
   r['attempts'].append(dict(core=core,code=code,ms=(time.monotonic()-at)*1000,stages=stages))
   # A validated running core does not need four repetitions of a network timeout.
   # Schema/start failures still use the full pinned compatibility matrix.
   if len(checks)==2 and all(x.get('core') for x in checks) or code in ('network_retry_pending','raw_shared_egress'):break
  except Exception as error:
   code=error_code(error)
   r['attempts'].append(dict(core=core,code=code,ms=(time.monotonic()-at)*1000,stages=stages))
   if best<0:r['code']=code
   if code=='ipv6_only_probe_pending':break
 # A separate host control is only an outage check, never Ads IP evidence.
 try:
  if any(x.get('echo') for x in r['samples']):raise ValueError('outage_check_unnecessary')
  result=subprocess.run(['/usr/bin/curl','--noproxy','*','--silent','--fail','--max-time','5','https://1.1.1.1/cdn-cgi/trace'],capture_output=True,timeout=6)
  r['infrastructure']=result.returncode==0 and b'ip=' in result.stdout
 except Exception:pass
 if len(r['health'])==2 and all(not any(x.get('passed') for x in h.get('providers',[])) for h in r['health']):
  try:
   q=subprocess.run(['/usr/bin/python3',str(HERE/'egress-health.py'),'--host-control'],capture_output=True,timeout=12)
   h=json.loads(q.stdout);h['variant']='host';r['health'].append(h)
  except Exception:pass
 r['waves']=max([2]+[x.get('wave',0) for x in r['samples']])
 r['samples']=pool.geo.attach(r['samples'])
 r['samples']=private_sources(r['samples'],secrets.token_bytes(32))
 r['finished']=int(time.time()*1000);r['wall_ms']=(time.monotonic()-begin)*1000
 return r
def main():
 a=argparse.ArgumentParser();a.add_argument('--prefix',default='cge',choices=['cge','cgt']);a.add_argument('--slot',type=int,default=1);a.add_argument('--run-id');a.add_argument('--cleanup',action='store_true');a.add_argument('--serve',action='store_true');args=a.parse_args()
 if os.geteuid()!=0 or not 0<=args.slot<=31 or args.run_id and not re.fullmatch('[0-9a-f]{6}',args.run_id):raise SystemExit('invalid supervisor request')
 if args.cleanup:cleanup(args.prefix,args.slot if args.slot else None,args.run_id);return
 if not args.run_id:raise SystemExit('run id required')
 parent=os.getppid();signal.signal(signal.SIGTERM,lambda *_:sys.exit(143))
 def watch():
  while True:
   time.sleep(.25)
   if os.getppid()!=parent or parent==1:os.kill(os.getpid(),signal.SIGTERM);return
 threading.Thread(target=watch,daemon=True).start()
 try:
  with ResourcePool(args) as pool:
   if args.serve:
    while True:
     line=sys.stdin.buffer.readline(24*1024*1024)
     if not line:break
     if not line.endswith(b'\n'):raise ValueError('request_limit')
     print(json.dumps(investigate(json.loads(line),args,pool)),flush=True)
   else:print(json.dumps(investigate(json.load(sys.stdin),args,pool)))
 finally:cleanup(args.prefix,args.slot,args.run_id)
if __name__=='__main__':main()
