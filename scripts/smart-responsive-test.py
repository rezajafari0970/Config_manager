#!/usr/bin/env python3
"""Pinned-core comparison using synthetic peers in a closed network namespace.
Tests requested leastLoad distribution, DNS, endpoint bootstrap and failed-peer exclusion.
leastPing is an explicit comparison fixture, not the generated production strategy.
No live subscription, Android, carrier, throughput or real QUIC claim.
"""
import copy,hashlib,importlib.util,json,math,os,pathlib,secrets,socket,struct
import subprocess,sys,threading,time
from http.server import BaseHTTPRequestHandler,ThreadingHTTPServer
HERE=pathlib.Path(__file__).resolve().parent
def module(name,path):
 spec=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m
traffic=module('traffic',HERE/'traffic-policy-test.py')
def services():
 def dns_answer(b,bootstrap=False):
  end=12
  while b[end]:end+=1+b[end]
  end+=1;kind=struct.unpack('!H',b[end:end+2])[0];question=b[12:end+4]
  data=socket.inet_aton('127.0.0.1' if bootstrap else '203.0.113.7') if kind==1 else b'\x02ok'
  return b[:2]+b'\x81\x80'+struct.pack('!4H',1,1,0,0)+question+b'\xc0\x0c'+struct.pack('!HHIH',kind,1,10,len(data))+data
 def dns_tcp(port,bootstrap):
  s=socket.socket();s.setsockopt(socket.SOL_SOCKET,socket.SO_REUSEADDR,1);s.bind(('127.0.0.1',port));s.listen()
  def serve(c):
   with c:
    c.settimeout(4)
    try:
     while True:
      n=struct.unpack('!H',traffic.exact(c,2))[0];r=dns_answer(traffic.exact(c,n),bootstrap);c.sendall(struct.pack('!H',len(r))+r)
    except OSError:pass
  while True:threading.Thread(target=serve,args=(s.accept()[0],),daemon=True).start()
 def dns_udp():
  s=socket.socket(socket.AF_INET,socket.SOCK_DGRAM);s.bind(('127.0.0.1',53))
  while True:
   b,peer=s.recvfrom(4096);s.sendto(dns_answer(b),peer)
 def relay():
  s=socket.socket();s.setsockopt(socket.SOL_SOCKET,socket.SO_REUSEADDR,1);s.bind(('127.0.0.1',24446));s.listen()
  def accept(c):
   try:d=socket.create_connection(('127.0.0.1',24449),timeout=4)
   except OSError:c.close();return
   def pump(a,b):
    try:
     while True:
      chunk=a.recv(65536)
      if not chunk:break
      time.sleep(.12);b.sendall(chunk)
    except OSError:pass
    finally:
     try:b.shutdown(socket.SHUT_WR)
     except OSError:pass
   t=threading.Thread(target=pump,args=(c,d),daemon=True);t.start();pump(d,c);t.join(timeout=5);c.close();d.close()
  while True:threading.Thread(target=accept,args=(s.accept()[0],),daemon=True).start()
 for fn,args in [(dns_tcp,(24453,False)),(dns_tcp,(24454,True)),(dns_udp,()),(relay,())]:
  threading.Thread(target=fn,args=args,daemon=True).start()
 class Handler(BaseHTTPRequestHandler):
  def do_GET(self):self.send_response(204);self.send_header('X-Fixture-Path','slow' if self.client_address[0]=='127.0.0.11' else 'fast');self.end_headers()
  def log_message(self,*args):pass
 ThreadingHTTPServer(('127.0.0.1',24447),Handler).serve_forever()
def exercise():
 durations=[];paths=[]
 for _ in range(24):
  start=time.monotonic();s,_=traffic.socks(1,24447)
  with s:
   s.sendall(b'GET /generate_204 HTTP/1.1\r\nHost: fixture.test\r\nConnection: close\r\n\r\n')
   head=b''
   while b'\r\n\r\n' not in head and len(head)<8192:
    chunk=s.recv(4096)
    if not chunk:break
    head+=chunk
   assert b' 204 ' in head
   path='slow' if b'X-Fixture-Path: slow' in head else 'fast' if b'X-Fixture-Path: fast' in head else None
   assert path is not None;paths.append(path)
  durations.append(round((time.monotonic()-start)*1000,2))
 dns={}
 for kind,dest,key in [(1,'203.0.113.99','intercept_a'),(16,'127.0.0.1','relay_txt')]:
  query=b'\x12\x34\x01\x00'+struct.pack('!4H',1,0,0,0)+b'\x03app\x04test\x00'+struct.pack('!HH',kind,1)
  udp=socket.socket(socket.AF_INET,socket.SOCK_DGRAM);udp.bind(('127.0.0.1',0));udp.settimeout(.7)
  control,relay=traffic.socks(3,udp.getsockname()[1])
  with control,udp:
   udp.sendto(b'\x00\x00\x00\x01'+socket.inet_aton(dest)+struct.pack('!H',53)+query,relay)
   try:
    b,_=udp.recvfrom(8192)
    dns[key]=b.endswith(socket.inet_aton('203.0.113.7') if kind==1 else b'\x02ok')
   except socket.timeout:dns[key]=False
 ordered=sorted(durations)
 print(json.dumps({'requests':len(durations),'slow_requests':paths.count('slow'),'p50_ms':ordered[len(ordered)//2],'p95_ms':ordered[math.ceil(.95*len(ordered))-1],'dns':dns}))
def main():
 request=json.load(sys.stdin);core=pathlib.Path(request['core']).resolve()
 assert os.geteuid()==0 and hashlib.sha256(core.read_bytes()).hexdigest() in {'64d46afb80adea1bf97a0d467e83f4a9ac1ebd0995891e84bca3f1a1d1affb1d','cc4b84d3069569f297a61f04b914f2bb37e9d202fd7b84cb50def1fc30fd40e1','e87bce586acdb9136aa5b1f1d9c8db9f1ea1d145b6d86368c7f6f662f9f6a66d'}
 lab=module('responsive_sandbox',HERE/'live-probe.py');report={}
 assert request['config']['routing']['balancers'][0]['strategy']=={'type':'leastLoad','settings':{'expected':3,'maxRTT':'2s','tolerance':.15}}
 for phase in ('reference_leastping','leastload','domain'):
  config=copy.deepcopy(request['config'])
  if phase=='reference_leastping':
   config['routing']['balancers'][0]['strategy']={'type':'leastPing'}
  # Non-IP forwarding remains tunneled; redirect only the test resolver here.
  next(x for x in config['outbounds'] if x['tag']=='smart-dns-out')['settings'].update(rewriteAddress='127.0.0.1',rewritePort=24453)
  proxies=[x for x in config['outbounds'] if x['tag'].startswith('proxy-')]
  assert len(proxies)==2 and all(x['protocol']=='vless' for x in proxies)
  config['dns']['servers']=[{'address':'tcp://127.0.0.1:24453','skipFallback':False}]
  if phase=='domain':
   # Only endpoint-bootstrap phase uses accelerated observation; strategy and DNS route stay unchanged.
   config['burstObservatory']['pingConfig'].update(interval='1s',sampling=1,timeout='2s')
   for ob in proxies[:1]:
    settings=ob['settings']
    if 'address' in settings:settings['address']='peer.test'
    else:settings['vnext'][0]['address']='peer.test'
   config['routing']['balancers'][1]['selector']=[proxies[1]['tag']]
   config['dns']['servers'].insert(0,{'address':'tcp://127.0.0.1:24454','domains':['full:peer.test'],'skipFallback':True,'tag':'dns-bootstrap'})
  config['inbounds'].append({'tag':'fixture-server','listen':'127.0.0.1','port':24443,'protocol':'vless','settings':{'decryption':'none','clients':[{'id':'11111111-1111-4111-8111-111111111111'}]},'streamSettings':{'network':'tcp','security':'none'}})
  spare=copy.deepcopy(config['inbounds'][-1]);spare.update(tag='fixture-spare',port=24449);config['inbounds'].append(spare)
  for tag,source in [('fixture-server','127.0.0.10'),('fixture-spare','127.0.0.11')]:
   config['outbounds'].append({'tag':tag+'-exit','protocol':'freedom','sendThrough':source,'settings':{'finalRules':[{'action':'allow','network':'tcp,udp','ip':['127.0.0.1/32'],'port':'53,24447,24453,24454'}]}})
   config['routing']['rules'].insert(0,{'type':'field','inboundTag':[tag],'outboundTag':tag+'-exit'})
  # Local probe destination; shipped timing remains in the IP comparison/recovery phase.
  config['burstObservatory']['pingConfig']['destination']='http://127.0.0.1:24447/generate_204'
  with lab.Sandbox(str(core),secrets.token_hex(3),8199,prefix='csrf') as box:
   box.policy({})
   child=subprocess.Popen([lab.IP,'netns','exec',box.name,sys.executable,__file__,'--services'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
   try:
    time.sleep(.2);process=box.launch(json.dumps(config).encode());time.sleep(3)
    assert process.poll() is None,'synthetic core exited'
    # Wait for actual peer observations, not a fixed assumed bootstrap delay.
    for _ in range(60):
     status=json.loads(box.net(['/usr/bin/curl','--noproxy','*','--max-time','2','--silent','http://127.0.0.1:11111/debug/vars']).stdout)['observatory']
     if all(status.get(p['tag'],{}).get('alive') and status[p['tag']].get('delay',999999)<2000 for p in proxies):break
     time.sleep(.5)
    else:raise RuntimeError('peers not observed: '+json.dumps(status))
    try:result=box.net([sys.executable,__file__,'--exercise'],timeout=40)
    except subprocess.CalledProcessError as e:raise RuntimeError('exercise '+phase+': '+e.stderr.decode(errors='replace')[-2000:]) from e
    report[phase]=json.loads(result.stdout)
    if phase=='leastload':
     # Fail the fast peer after both peers carried requests; verify subsequent exclusion.
     box.net(['/usr/sbin/nft','add','table','inet','fixture_failure'])
     box.net(['/usr/sbin/nft','add','chain','inet','fixture_failure','out','{ type filter hook output priority 200; policy accept; }'])
     box.net(['/usr/sbin/nft','add','rule','inet','fixture_failure','out','tcp','dport','24443','drop'])
     started=time.monotonic();recovered=False;failures=0;successes=0;observed_unavailable=None
     # Three random successful requests do not prove leastLoad has excluded a
     # failed peer. Require the pinned core's actual exclusion predicate.
     ping=config['burstObservatory']['pingConfig']
     observation_window=max(10,float(ping['interval'].removesuffix('s')))*ping['sampling']
     deadline=2*observation_window+float(ping['timeout'].removesuffix('s'))+5
     while time.monotonic()-started<deadline:
      try:
       r=box.net(['/usr/bin/curl','--noproxy','','--proxy','socks5h://127.0.0.1:10808','--max-time','1','--silent','--output','/dev/null','--write-out','%{http_code}','http://127.0.0.1:24447/generate_204'],timeout=2)
       if r.stdout==b'204':successes+=1
       else:failures+=1;successes=0
      except (subprocess.SubprocessError,OSError):failures+=1;successes=0
      status=json.loads(box.net(['/usr/bin/curl','--noproxy','*','--max-time','2','--silent','http://127.0.0.1:11111/debug/vars']).stdout)['observatory']
      peer=status.get(proxies[1]['tag'],{});health=peer.get('health_ping',{})
      # Pinned shouldSelectNode rejects fail/all > tolerance even while alive
      # remains true (alive means at least one retained sample succeeded).
      unavailable=bool(peer) and (not peer.get('alive',False) or peer.get('delay',0)>=2000 or (health.get('all',0)>0 and health.get('fail',0)/health['all']>.15))
      if unavailable and observed_unavailable is None:observed_unavailable=round(time.monotonic()-started,2);successes=0
      if unavailable and observed_unavailable is not None and successes>=3:
       recovered=True;break
      time.sleep(.3)
     assert recovered,'failed-peer exclusion did not settle within test deadline: '+json.dumps({'status':status,'successes':successes,'failures':failures,'observed':observed_unavailable})
     verified_recovery=round(time.monotonic()-started,2)
     try:result=box.net([sys.executable,__file__,'--exercise'],timeout=40)
     except subprocess.CalledProcessError as e:raise RuntimeError('post-failure exercise: '+e.stderr.decode(errors='replace')[-2000:]+'; observer='+json.dumps(status)) from e
     post=json.loads(result.stdout)
     assert post['slow_requests']==post['requests'] and all(post['dns'].values()),post
     report[phase]['post_failure']=post
     report[phase]['verified_recovery_seconds']=verified_recovery
     report[phase]['observed_unavailable_seconds']=observed_unavailable
     report[phase]['requests_failed_during_detection']=failures

   finally:
    child.terminate()
    try:child.wait(timeout=3)
    except subprocess.TimeoutExpired:child.kill();child.wait()
 assert report['reference_leastping']['slow_requests']==0,report
 for phase in ('leastload','domain'):
  assert 0<report[phase]['slow_requests']<report[phase]['requests'],report
 for phase in report:
  assert all(report[phase]['dns'].values()),report
 print(json.dumps({'passed':True,'comparison':report,'scope':'closed synthetic namespace; 120ms per-direction delay on one peer','carrier_tests':'not_run','quic_handshake':'not_run'}))
if __name__=='__main__':
 if len(sys.argv)>1 and sys.argv[1]=='--services':services()
 elif len(sys.argv)>1 and sys.argv[1]=='--exercise':exercise()
 else:main()
