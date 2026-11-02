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
  def do_GET(self):self.send_response(204);self.end_headers()
  def log_message(self,*args):pass
 ThreadingHTTPServer(('127.0.0.1',24447),Handler).serve_forever()
def exercise():
 durations=[]
 for _ in range(24):
  start=time.monotonic();s,_=traffic.socks(1,24447)
  with s:
   s.sendall(b'GET /generate_204 HTTP/1.1\r\nHost: fixture.test\r\nConnection: close\r\n\r\n')
   head=s.recv(4096);assert b' 204 ' in head
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
 print(json.dumps({'requests':len(durations),'slow_requests':sum(x>180 for x in durations),'p50_ms':ordered[len(ordered)//2],'p95_ms':ordered[math.ceil(.95*len(ordered))-1],'dns':dns}))
def main():
 request=json.load(sys.stdin);core=pathlib.Path(request['core']).resolve()
 assert os.geteuid()==0 and hashlib.sha256(core.read_bytes()).hexdigest()=='64d46afb80adea1bf97a0d467e83f4a9ac1ebd0995891e84bca3f1a1d1affb1d'
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
   for ob in proxies:
    settings=ob['settings']
    if 'address' in settings:settings['address']='peer.test'
    else:settings['vnext'][0]['address']='peer.test'
   config['dns']['servers'].insert(0,{'address':'tcp+local://127.0.0.1:24454','domains':['full:peer.test'],'skipFallback':True})
  config['inbounds'].append({'tag':'fixture-server','listen':'127.0.0.1','port':24443,'protocol':'vless','settings':{'decryption':'none','clients':[{'id':'11111111-1111-4111-8111-111111111111'}]},'streamSettings':{'network':'tcp','security':'none'}})
  spare=copy.deepcopy(config['inbounds'][-1]);spare.update(tag='fixture-spare',port=24449);config['inbounds'].append(spare)
  config['outbounds'].append({'tag':'fixture-direct','protocol':'freedom','settings':{'finalRules':[{'action':'allow','network':'tcp,udp','ip':['127.0.0.1/32'],'port':'53,24447,24453'}]}})
  config['routing']['rules'].insert(0,{'type':'field','inboundTag':['fixture-server','fixture-spare'],'outboundTag':'fixture-direct'})
  # Probe destination and test DNS services are local; shipped timing is retained.
  config['burstObservatory']['pingConfig']['destination']='http://127.0.0.1:24447/generate_204'
  with lab.Sandbox(str(core),secrets.token_hex(3),8199,prefix='csrf') as box:
   box.policy({})
   child=subprocess.Popen([lab.IP,'netns','exec',box.name,sys.executable,__file__,'--services'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
   try:
    time.sleep(.2);process=box.launch(json.dumps(config).encode());time.sleep(3)
    assert process.poll() is None,'synthetic core exited'
    result=box.net([sys.executable,__file__,'--exercise'],timeout=40)
    report[phase]=json.loads(result.stdout)
    if phase=='leastload':
     # Fail the fast peer after both peers carried requests; verify subsequent exclusion.
     box.net(['/usr/sbin/nft','add','table','inet','fixture_failure'])
     box.net(['/usr/sbin/nft','add','chain','inet','fixture_failure','out','{ type filter hook output priority 200; policy accept; }'])
     box.net(['/usr/sbin/nft','add','rule','inet','fixture_failure','out','tcp','dport','24443','drop'])
     started=time.monotonic();recovered=False;failures=0;successes=0
     while time.monotonic()-started<28:
      try:
       r=box.net(['/usr/bin/curl','--noproxy','','--proxy','socks5h://127.0.0.1:10808','--max-time','1','--silent','--output','/dev/null','--write-out','%{http_code}','http://127.0.0.1:24447/generate_204'],timeout=2)
       if r.stdout==b'204':successes+=1
       else:failures+=1;successes=0
      except (subprocess.SubprocessError,OSError):failures+=1;successes=0
      if time.monotonic()-started>=14 and successes>=3:
       recovered=True;break
      time.sleep(.3)
     assert recovered,'failed-peer exclusion did not settle within test deadline'
     result=box.net([sys.executable,__file__,'--exercise'],timeout=40)
     post=json.loads(result.stdout)
     assert post['slow_requests']==post['requests'] and all(post['dns'].values()),post
     report[phase]['post_failure']=post
     report[phase]['verified_recovery_seconds']=round(time.monotonic()-started,2)
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
