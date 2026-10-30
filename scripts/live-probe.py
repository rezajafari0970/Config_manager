#!/usr/bin/env python3
"""Isolated Linux validation driver for the transient JSON conversion pipeline.

Root is needed only by the trusted supervisor for netns/nft/cgroups. Untrusted
configs execute as nobody in a read-only, capability-free mount/PID namespace.
Only pre-resolved public proxy endpoint IP:port pairs and DNS:53 can leave it.
Receipts contain hashes and bounded diagnosis codes, never credentials/logs.
"""
import argparse, base64, concurrent.futures, contextlib, copy, fcntl, hashlib, ipaddress
import json, os, pathlib, resource, signal, subprocess
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
    result=[];hopping={}
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
            hop=ob.get('streamSettings',{}).get('finalmask',{}).get('quicParams',{}).get('udpHop',{}).get('ports','')
            if hop:
                for entry in hop.split(','):
                    bounds=entry.strip().split('-')
                    if len(bounds)>2 or any(not x.strip().isdigit() or not 1<=int(x)<=65535 for x in bounds) or len(bounds)==2 and int(bounds[0])>int(bounds[1]):raise ValueError('port_hopping_invalid')
                    hopping.setdefault(s['address'],set()).add('-'.join(str(int(x)) for x in bounds))
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
        # IPv6 remains blocked by the namespace inet policy. A mixed pool can
        # still be tested through its actual IPv4 members without removing or
        # rewriting IPv6 outbounds. This proves the config on this network,
        # not connectivity of every individual pool member.
        if not ipv4: continue
        previous=resolved.get(host,{})
        ports=set(previous.get('ports',[]))|{str(port)}|hopping.get(host,set())
        resolved[host]={'ips':ipv4,'port':port,'ports':sorted(ports)}
    if not resolved: raise ValueError('ipv6_only_probe_pending')
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
    def __init__(self, core, runid, index, prefix="crp"):
        self.core=pathlib.Path(core).resolve();self.name=prefix+runid[:6]+format(index,'x')
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
        pairs=sorted({(ip,str(p)) for v in resolved.values() for ip in v['ips'] for p in v.get('ports',[v['port']])})
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

if __name__=='__main__':
    import sys
    if len(sys.argv)>2 and sys.argv[1]=='--sandbox-launch':
        cg=pathlib.Path(sys.argv[2]);(cg/'cgroup.procs').write_text(str(os.getpid()))
        resource.setrlimit(resource.RLIMIT_NOFILE,(256,256));resource.setrlimit(resource.RLIMIT_FSIZE,(1024*1024,1024*1024));resource.setrlimit(resource.RLIMIT_CORE,(0,0))
        os.execv(sys.argv[3],sys.argv[3:])
    raise SystemExit("use probe-one.py")
