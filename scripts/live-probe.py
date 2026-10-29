#!/usr/bin/env python3
"""Temporary, isolated Linux test driver. No production DB writes or healthy export.

Root is needed only by the trusted supervisor for netns/nft/cgroups. Untrusted
configs execute as nobody in a read-only, capability-free mount/PID namespace.
Only pre-resolved public proxy endpoint IP:port pairs and DNS:53 can leave it.
Receipts contain hashes and bounded diagnosis codes, never credentials/logs.
"""
import argparse, base64, concurrent.futures, contextlib, copy, fcntl, hashlib, ipaddress
import json, os, pathlib, resource, signal, sqlite3, subprocess
import tempfile, threading, time, urllib.request, uuid

ROOT = pathlib.Path(__file__).resolve().parents[1]
IP = '/usr/sbin/ip' if pathlib.Path('/usr/sbin/ip').exists() else '/usr/bin/ip'
NFT = '/usr/sbin/nft'
TARGET = 'v2rayng-2.3.6-lib40-xray-26.7.28'
TESTS = [('https://www.gstatic.com/generate_204', 204), ('https://www.cloudflare.com/cdn-cgi/trace', 200)]
ALLOWED_PROTOCOLS = {'vless','vmess','trojan','shadowsocks','hysteria','wireguard','socks','http'}
lock = threading.Lock()
STOP = threading.Event()

def digest(b): return hashlib.sha256(b).hexdigest()
def cmd(args, **kw): return subprocess.run(args, check=True, capture_output=True, **kw)
def api(path):
    return urllib.request.urlopen(urllib.request.Request('http://127.0.0.1:18191'+path,
        headers={'Host':'85.192.63.181:18443'}), timeout=30)
def classify(log, fallback):
    text = log.decode(errors='replace').lower() if isinstance(log, bytes) else log.lower()
    rules = [('feature "allowinsecure" has been removed','removed_allow_insecure'), ('without tls is prohibited','public_endpoint_requires_tls'), ('without tls or other encryption is prohibited','public_endpoint_requires_tls'), ('invalid uuid','invalid_uuid'),
        ('unknown config id','unsupported_protocol_or_transport'), ('unknown transport protocol','unsupported_transport'),
        ('flow has been removed','removed_trojan_flow'), ('certificate signed by unknown authority','tls_untrusted_certificate'), ('certificate has expired','tls_expired_certificate'), ('certificate is valid for','tls_name_mismatch'), ('cannot validate certificate for','tls_name_mismatch'), ('certificate','tls_certificate_failure'),
        ('reality','reality_handshake_or_settings'), ('failed to parse','core_parse_failure'),
        ('cannot unmarshal','core_schema_mismatch'), ('no such file','missing_runtime_asset'),
        ('connection refused','endpoint_refused'), ('i/o timeout','connection_timeout'),
        ('network is unreachable','network_unreachable'), ('no such host','dns_failure'),
        ('authentication','authentication_failure'), ('eof','peer_closed_connection')]
    for needle, code in rules:
        if needle in text: return code
    return fallback

def endpoints(config):
    """Never infer dial targets by recursively trusting arbitrary address keys."""
    result=[]
    for ob in config.get('outbounds',[]):
        protocol=ob.get('protocol');s=ob.get('settings',{})
        if protocol in {'freedom','blackhole','dns','loopback'}: continue
        if protocol not in ALLOWED_PROTOCOLS: raise ValueError('outbound_driver_pending')
        if protocol=='wireguard':
            for peer in s.get('peers',[]):
                host,sep,p=peer.get('endpoint','').rpartition(':')
                if not sep: raise ValueError('wireguard_endpoint_invalid')
                result.append((host.strip('[]'),int(p)))
        elif s.get('address'):
            result.append((s['address'],int(s.get('port',0))))
        else:
            servers=s.get('vnext',s.get('servers',[]))
            for server in servers: result.append((server['address'],int(server['port'])))
    if not result: raise ValueError('no_proxy_endpoint')
    resolved={}
    for host,port in result:
        if not isinstance(host,str) or not 1<=port<=65535: raise ValueError('invalid_proxy_endpoint')
        # Resolver in a killable subprocess: a broken DNS lookup cannot stall the pool.
        helper='import socket,json,sys;print(json.dumps(sorted({x[4][0] for x in socket.getaddrinfo(sys.argv[1],None,socket.AF_UNSPEC,socket.SOCK_STREAM)})))'
        try: ips=json.loads(cmd(['/usr/bin/python3','-c',helper,host],timeout=8).stdout)
        except (subprocess.SubprocessError,json.JSONDecodeError): raise ValueError('endpoint_dns_failed')
        if not ips or any(not ipaddress.ip_address(ip).is_global or ipaddress.ip_address(ip).is_multicast or ipaddress.ip_address(ip).is_reserved for ip in ips): raise ValueError('nonpublic_endpoint_blocked')
        ipv4=[ip for ip in ips if ipaddress.ip_address(ip).version==4]
        if not ipv4: raise ValueError('ipv6_only_probe_pending')
        resolved[host]={'ips':ipv4,'port':port}
    return resolved

def runtime_config(config, resolved):
    c=copy.deepcopy(config)
    tags=[x.get('tag') for x in c.get('inbounds',[]) if x.get('protocol')=='socks']
    tag=next((x for x in tags if x),'socks')
    c['inbounds']=[{'tag':tag,'listen':'127.0.0.1','port':10808,'protocol':'socks','settings':{'auth':'noauth','udp':True}}]
    c['log']={'loglevel':'warning','access':'/dev/stdout','error':'/dev/stderr'}
    # No direct-route success can be reported as a proxy connection.
    for ob in c.get('outbounds',[]):
        if ob.get('protocol')=='freedom': ob['protocol']='blackhole';ob['settings']={}
    # Pin DNS answers to the allowlist; no address rewriting or SNI changes.
    dns=c.setdefault('dns',{});hosts=dns.setdefault('hosts',{})
    for host, entry in resolved.items():
        try: ipaddress.ip_address(host)
        except ValueError:
            # Preserve an existing mapping rather than silently overriding it.
            if host in hosts and hosts[host] not in (entry['ips'],entry['ips'][0]): raise ValueError('custom_dns_mapping_requires_review')
            hosts[host]=entry['ips']
    c.pop('api',None);c.pop('metrics',None)
    return c

class Sandbox:
    def __init__(self, core, runid, index):
        self.core=pathlib.Path(core).resolve();self.name='crp'+runid[:6]+format(index,'x')
        self.table=self.name;self.hostif=self.name+'h';self.peer=self.name+'p'
        # /30 from the dedicated benchmark range; none of it is allowed as egress.
        base=int(ipaddress.IPv4Address('198.18.0.0'))+4*(index+1)
        if index>=32760: raise ValueError('namespace_address_capacity')
        self.gateway=str(ipaddress.IPv4Address(base+1));self.addr=str(ipaddress.IPv4Address(base+2))
        self.cg=pathlib.Path('/sys/fs/cgroup')/self.name;self.temp=None;self.process=None
    def __enter__(self):
        try:
            self.temp=tempfile.TemporaryDirectory(prefix=self.name+'-',dir='/dev/shm')
            self.work=pathlib.Path(self.temp.name);os.chmod(self.work,0o700)
            self.cg.mkdir();(self.cg/'memory.max').write_text(str(256*1024*1024));(self.cg/'pids.max').write_text('128');(self.cg/'cpu.max').write_text('100000 100000')
            cmd([IP,'netns','add',self.name]);cmd([IP,'link','add',self.hostif,'type','veth','peer','name',self.peer])
            cmd([IP,'link','set',self.peer,'netns',self.name]);cmd([IP,'addr','add',self.gateway+'/30','dev',self.hostif]);cmd([IP,'link','set',self.hostif,'up'])
            self.net([IP,'addr','add',self.addr+'/30','dev',self.peer]);self.net([IP,'link','set',self.peer,'up']);self.net([IP,'link','set','lo','up'])
            self.net([IP,'route','add','default','via',self.gateway])
            # Unique scoped NAT table. Never flush or modify pre-existing tables.
            cmd([NFT,'-f','-'],input=(f'table ip {self.table} {{ chain nat {{ type nat hook postrouting priority srcnat; policy accept; ip saddr {self.addr} oifname != "{self.hostif}" masquerade; }}; chain ingress {{ type filter hook input priority 0; policy accept; iifname "{self.hostif}" drop; }}; }}\n').encode())
            self.policy({})
            return self
        except BaseException:
            self.__exit__(None,None,None);raise
    def net(self,args,**kw): return cmd([IP,'netns','exec',self.name]+args,**kw)
    def policy(self,resolved):
        pairs=sorted({(ip,v['port']) for v in resolved.values() for ip in v['ips']})
        # Namespace-local rules apply equally to all transports and all DNS rebinds.
        allow='\n'.join(f'ip daddr {ip} tcp dport {p} accept\nip daddr {ip} udp dport {p} accept' for ip,p in pairs)
        rules='flush ruleset\ntable inet guard { chain out { type filter hook output priority 0; policy drop; oifname "lo" accept; ip daddr 1.1.1.1 udp dport 53 accept; ip daddr 1.1.1.1 tcp dport 53 accept; '+allow+'\n}; chain in { type filter hook input priority 0; policy drop; iifname "lo" accept; ct state established,related accept; }; }\n'
        self.net([NFT,'-f','-'],input=rules.encode())
    def launch(self,b,test=False):
        (self.work/'config.json').write_bytes(b);os.chmod(self.work/'config.json',0o444)
        (self.work/'resolv.conf').write_text('nameserver 1.1.1.1\noptions timeout:2 attempts:1\n')
        args=[IP,'netns','exec',self.name,'/usr/bin/bwrap','--unshare-user','--unshare-pid','--unshare-ipc','--unshare-uts','--unshare-cgroup','--die-with-parent','--new-session','--cap-drop','ALL','--uid','65534','--gid','65534','--clearenv','--setenv','SSL_CERT_FILE','/etc/ssl/certs/ca-certificates.crt','--setenv','GOMAXPROCS','1','--setenv','GOMEMLIMIT','128MiB','--setenv','XRAY_LOCATION_ASSET','/assets','--ro-bind','/etc/ssl/certs/ca-certificates.crt','/etc/ssl/certs/ca-certificates.crt','--ro-bind',str(self.core),'/xray','--ro-bind',str(self.work/'config.json'),'/config.json','--ro-bind',str(self.work/'resolv.conf'),'/etc/resolv.conf','--proc','/proc','--dev','/dev','--tmpfs','/tmp','--dir','/assets']
        for name in ['geoip.dat','geosite.dat']:
            asset=self.core.parent/name
            if asset.exists():args+=['--ro-bind',str(asset),'/assets/'+name]
        args+=['--','/xray','run']+(['-test'] if test else [])+['-c','/config.json']
        self.log=tempfile.TemporaryFile(dir='/dev/shm')
        self.process=subprocess.Popen(['/usr/bin/python3',str(pathlib.Path(__file__).resolve()),'--sandbox-launch',str(self.cg)]+args,stdout=self.log,stderr=subprocess.STDOUT,start_new_session=True)
        return self.process
    def logs(self):
        self.log.seek(0);return self.log.read(128*1024)
    def stop(self):
        if self.process:
            with contextlib.suppress(ProcessLookupError):os.killpg(self.process.pid,signal.SIGKILL)
            with contextlib.suppress(subprocess.TimeoutExpired):self.process.wait(timeout=3)
            if (self.cg/'cgroup.kill').exists(): (self.cg/'cgroup.kill').write_text('1')
            self.process=None
        if hasattr(self,'log'):self.log.close()
    def __exit__(self,*exc):
        self.stop()
        for args in [[NFT,'delete','table','ip',self.table],[IP,'link','del',self.hostif],[IP,'netns','del',self.name]]:
            subprocess.run(args,capture_output=True)
        for _ in range(20):
            try:self.cg.rmdir();break
            except FileNotFoundError:break
            except OSError:time.sleep(.05)
        if self.temp:self.temp.cleanup()

def test_candidate(box,candidate,timeout):
    b=base64.b64decode(candidate['config'],validate=True)
    if digest(b)!=candidate['sha256']: raise ValueError('candidate_hash_mismatch')
    result={'core':'not_run','connection':'not_run','code':'not_run','runtime_sha256':'','endpoints_passed':0}
    box.policy({});p=box.launch(b,True)
    try:p.wait(timeout=10)
    except subprocess.TimeoutExpired:result['code']='core_validation_timeout';box.stop();return result
    log=box.logs();rc=p.returncode;box.stop()
    if rc!=0:
        code=classify(log,'core_rejected')
        if not log.startswith(b'Xray '):code='probe_infrastructure_error'
        if rc<0:code='probe_resource_or_signal_failure'
        blocked=code in {'missing_runtime_asset','probe_infrastructure_error','probe_resource_or_signal_failure'}
        result.update(core='blocked' if blocked else 'failed',code=code);return result
    result['core']='passed'
    try:
        config=json.loads(b);resolved=endpoints(config);runtime=runtime_config(config,resolved)
    except (ValueError,TypeError,KeyError) as e:
        code=str(e);result['code']=code if code in {'outbound_driver_pending','wireguard_endpoint_invalid','no_proxy_endpoint','invalid_proxy_endpoint','endpoint_dns_failed','nonpublic_endpoint_blocked','ipv6_only_probe_pending','custom_dns_mapping_requires_review'} else 'probe_preparation_failed';return result
    box.policy(resolved);runtime_bytes=json.dumps(runtime,separators=(',',':')).encode();result['runtime_sha256']=digest(runtime_bytes)
    p=box.launch(runtime_bytes);codes=[]
    # Readiness waits are local and do not reject slow remote proxies with TCP ping.
    ready=False
    for _ in range(30):
        if p.poll() is not None:break
        check=box.net(['/usr/bin/python3','-c','import socket; s=socket.socket();s.settimeout(.1); print(s.connect_ex(("127.0.0.1",10808)))'])
        if check.stdout.strip()==b'0':ready=True;break
        time.sleep(.1)
    if ready:
        for url,expected in TESTS:
            args=[IP,'netns','exec',box.name,'/usr/bin/curl','--noproxy','','--proxy','socks5h://127.0.0.1:10808','--max-time',str(timeout),'--connect-timeout',str(timeout),'--max-filesize','65536','--silent','--show-error','--output','/dev/null','--write-out','%{http_code}',url]
            try:r=subprocess.run(args,capture_output=True,timeout=timeout+2,env={'PATH':'/usr/bin:/usr/sbin'});passed=r.returncode==0 and r.stdout.strip()==str(expected).encode()
            except subprocess.TimeoutExpired:passed=False
            codes.append(passed)
            if not passed and len(codes)==1:
                # A second independent destination distinguishes endpoint-specific failures.
                continue
    log=box.logs();box.stop();result['endpoints_passed']=sum(codes)
    if codes and all(codes):result.update(connection='passed',code='https_proxy_verified')
    elif any(codes):result.update(connection='partial',code='test_destination_partial')
    else:result.update(connection='failed',code=classify(log,'proxy_request_failed' if ready else 'runtime_start_failed'))
    return result

def database(path):
    path=pathlib.Path(path);path.parent.mkdir(parents=True,exist_ok=True,mode=0o700)
    if path.parent.stat().st_mode & 0o077:raise ValueError('bank_directory_must_be_private')
    db=sqlite3.connect(path);os.chmod(path,0o600);db.execute('PRAGMA journal_mode=WAL')
    db.executescript('''CREATE TABLE IF NOT EXISTS runs(id TEXT PRIMARY KEY,target TEXT,core_sha256 TEXT,planner_sha256 TEXT,started INTEGER,finished INTEGER,summary TEXT);
    CREATE TABLE IF NOT EXISTS attempts(run_id TEXT,artifact_id TEXT,original_sha256 TEXT,candidate_sha256 TEXT,rule TEXT,kind TEXT,core TEXT,connection TEXT,code TEXT,runtime_sha256 TEXT,android_import TEXT,tested_at INTEGER,expires_at INTEGER,duration_ms INTEGER,endpoints_passed INTEGER,recommended INTEGER,PRIMARY KEY(run_id,original_sha256,candidate_sha256));
    CREATE TABLE IF NOT EXISTS knowledge(code TEXT PRIMARY KEY,occurrences INTEGER,first_seen INTEGER,last_seen INTEGER,verified_successes INTEGER);''');db.execute('UPDATE attempts SET expires_at=expires_at/1000 WHERE expires_at>1000000000000');db.commit();return db

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--core',default=str(ROOT/'.local/xray-26.7.28/xray'));parser.add_argument('--planner',default=str(ROOT/'bin/config-probe-plan'));parser.add_argument('--bank',default=str(ROOT/'.local/probe-bank/evidence.db'));parser.add_argument('--workers',type=int,default=4);parser.add_argument('--timeout',type=int,default=8);parser.add_argument('--limit',type=int,default=0);parser.add_argument('--self-test',action='store_true');a=parser.parse_args()
    if os.geteuid()!=0:raise SystemExit('root supervisor required')
    signal.signal(signal.SIGTERM,lambda *_:STOP.set());signal.signal(signal.SIGINT,lambda *_:STOP.set())
    if not 1<=a.workers<=8 or not 1<=a.timeout<=60:raise SystemExit('invalid worker/deadline settings')
    corehash=digest(pathlib.Path(a.core).read_bytes());expected=json.loads((ROOT/'compatibility/target.lock.json').read_text())['core']['linux_amd64_binary_sha256']
    if corehash!=expected:raise SystemExit('pinned core hash mismatch')
    runid=uuid.uuid4().hex
    global_lock=open('/run/config-repair-probe.lock','w');fcntl.flock(global_lock,fcntl.LOCK_EX|fcntl.LOCK_NB)
    if a.self_test:
        with Sandbox(a.core,runid,0) as box:
            # Fail closed if the kernel cannot enforce namespace-local egress.
            for host in ['127.0.0.1','169.254.169.254',box.gateway,'1.1.1.1']:
                r=box.net(['/usr/bin/python3','-c','import socket,sys; s=socket.socket();s.settimeout(.3);print(s.connect_ex((sys.argv[1],443)))',host])
                if r.stdout.strip()==b'0':raise SystemExit('egress isolation failed')
            box.policy({'cloudflare':{'ips':['1.1.1.1','1.0.0.1'],'port':443}})
            positive_ok=False
            for address in ['1.1.1.1','1.0.0.1']:
                try:
                    positive=box.net(['/usr/bin/curl','--noproxy','*','--max-time','10','--silent','--output','/dev/null','--write-out','%{http_code}','https://'+address+'/cdn-cgi/trace'],timeout=12)
                    if positive.stdout.strip()==b'200':positive_ok=True;break
                except subprocess.SubprocessError:pass
            if not positive_ok:raise SystemExit('allowed egress positive control failed')
            box.policy({})
            conf={'inbounds':[],'outbounds':[{'protocol':'blackhole','settings':{}}],'routing':{}}
            p=box.launch(json.dumps(conf).encode(),True);p.wait(timeout=10);ok=p.returncode==0;log=box.logs();box.stop()
            if not ok:raise SystemExit('sandbox core launch failed: '+classify(log,'sandbox_error'))
            marker=box.work/'host-only.pem';marker.write_text('synthetic host filesystem canary')
            denied={'inbounds':[],'outbounds':[{'protocol':'trojan','settings':{'address':'1.1.1.1','port':443,'password':'synthetic'},'streamSettings':{'security':'tls','tlsSettings':{'certificates':[{'certificateFile':str(marker),'usage':'verify'}]}}}],'routing':{}}
            p=box.launch(json.dumps(denied).encode(),True);p.wait(timeout=10);isolated=p.returncode!=0 and b'no such file' in box.logs();box.stop()
            if not isolated:raise SystemExit('host filesystem isolation control failed')
        print(json.dumps({'sandbox':'passed','public_private_metadata_host_egress':'blocked','host_filesystem':'unavailable_to_core','core_hash':corehash}));return
    # Snapshot metadata; originals are read through the panel's existing access gates.
    records=[];seen=set();page=0
    while True:
        with api('/api/state?page='+str(page)) as f:state=json.load(f)
        if not state['outputs']:break
        for item in state['outputs']:
            if item['disabled'] or item['hash'] in seen:continue
            seen.add(item['hash']);records.append(item)
        page+=1
    # Interleave kinds, so an optional canary is representative.
    groups={}
    for item in records:groups.setdefault(item['kind'],[]).append(item)
    records=[]
    while any(groups.values()):
        for values in groups.values():
            if values:records.append(values.pop())
    if a.limit:records=records[:a.limit]
    metadata={'driver_sha256':digest(pathlib.Path(__file__).read_bytes()),'ca_sha256':digest(pathlib.Path('/etc/ssl/certs/ca-certificates.crt').read_bytes()),'timeout_seconds':a.timeout,'workers':a.workers}
    db=database(a.bank);db.execute('INSERT INTO runs VALUES(?,?,?,?,?,?,?)',(runid,TARGET,corehash,digest(pathlib.Path(a.planner).read_bytes()),int(time.time()),None,json.dumps(metadata)));db.commit()
    counts={};completed=0
    def process(index,item):
        start=time.monotonic();base={'artifact_id':item['id'],'original_sha256':item['hash'],'kind':item['kind'],'expires_at':item['expires']//1000,'android_import':'not_run'}
        def failure(code):return [base|{'candidate_sha256':'','rule':'','core':'not_run','connection':'not_run','code':code,'runtime_sha256':'','endpoints_passed':0,'recommended':False,'duration_ms':int((time.monotonic()-start)*1000)}]
        try:
            if STOP.is_set():return failure('cancelled')
            if int(time.time())>=base['expires_at']:return failure('source_expired_or_unavailable')
            with api('/output/raw/'+item['id']) as f:raw=f.read(2*1024*1024+1)
            if digest(raw)!=item['hash']:return failure('source_changed_before_test')
            p=cmd([a.planner],input=raw,timeout=10);plan=json.loads(p.stdout)
            if not plan['candidates']:return failure(plan['code'])
            rows=[]
            with Sandbox(a.core,runid,index) as box:
                for candidate in plan['candidates']:
                    if STOP.is_set():break
                    result=test_candidate(box,candidate,a.timeout)
                    rows.append(base|result|{'candidate_sha256':candidate['sha256'],'rule':candidate['rule'],'duration_ms':int((time.monotonic()-start)*1000),'recommended':result['connection']=='passed' and candidate['requires_connection'] and int(time.time())<base['expires_at']})
                    if result['connection']=='passed':break
            return rows
        except urllib.error.HTTPError:return failure('source_expired_or_unavailable')
        except Exception as e:return failure('probe_infrastructure_'+type(e).__name__)
    def record(rows):
        nonlocal completed
        now=int(time.time())
        for r in rows:
            db.execute('INSERT OR REPLACE INTO attempts VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)',(runid,r['artifact_id'],r['original_sha256'],r['candidate_sha256'],r['rule'],r['kind'],r['core'],r['connection'],r['code'],r['runtime_sha256'],r['android_import'],now,r['expires_at'],r['duration_ms'],r['endpoints_passed'],int(r['recommended'])))
            db.execute('INSERT INTO knowledge VALUES(?,1,?,?,?) ON CONFLICT(code) DO UPDATE SET occurrences=occurrences+1,last_seen=excluded.last_seen,verified_successes=verified_successes+excluded.verified_successes',(r['code'],now,now,int(r['connection']=='passed')))
            counts[r['code']]=counts.get(r['code'],0)+1
        db.commit();completed+=1
        if completed%20==0:print(json.dumps({'completed':completed,'total':len(records),'counts':counts}),flush=True)
    # Allocate a unique /30 per concurrent worker; indexes are monotonically unique
    # and never share a namespace. Use at most 60 in-flight slots.
    with concurrent.futures.ThreadPoolExecutor(max_workers=a.workers) as pool:
        futures=[pool.submit(process,i,item) for i,item in enumerate(records)]
        for future in concurrent.futures.as_completed(futures):record(future.result())
    summary=metadata|{'run_id':runid,'configs':len(records),'completed':completed,'counts':counts,'android_import':'not_run','cancelled':STOP.is_set(),'scope':'temporary_socks_inbound_direct_blocked_https_probe'}
    db.execute('UPDATE runs SET finished=?,summary=? WHERE id=?',(int(time.time()),json.dumps(summary),runid));db.commit()
    # Invalidated calibration runs never contribute learned outcomes.
    db.execute('DELETE FROM knowledge')
    db.execute("INSERT INTO knowledge SELECT a.code,count(*),min(a.tested_at),max(a.tested_at),sum(a.connection='passed') FROM attempts a JOIN runs r ON r.id=a.run_id WHERE r.finished IS NOT NULL AND json_extract(r.summary,'$.invalidated') IS NULL GROUP BY a.code")
    db.commit();db.close();print(json.dumps(summary),flush=True)

if __name__=='__main__':
    import sys
    if len(sys.argv)>2 and sys.argv[1]=='--sandbox-launch':
        cg=pathlib.Path(sys.argv[2]);(cg/'cgroup.procs').write_text(str(os.getpid()))
        resource.setrlimit(resource.RLIMIT_NOFILE,(256,256));resource.setrlimit(resource.RLIMIT_FSIZE,(1024*1024,1024*1024));resource.setrlimit(resource.RLIMIT_CORE,(0,0))
        os.execv(sys.argv[3],sys.argv[3:])
    main()
