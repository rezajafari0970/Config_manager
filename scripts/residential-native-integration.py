#!/usr/bin/env python3
"""Synthetic TUIC/Hysteria protocol round trips in a deny-all namespace.
No public Ads request or live Residential assertion. A fixture-only direct
outbound is the internal server's echo egress, never a probe fallback.
"""
import base64,copy,importlib.util,json,os,pathlib,subprocess,tempfile,time
HERE=pathlib.Path(__file__).resolve().parent;ROOT=HERE.parent
os.environ['SHEN_RESIDENTIAL_TEST_PREFIX']='crt'
def load(name,file):
 spec=importlib.util.spec_from_file_location(name,HERE/file);m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m
res=load('res','residential-probe.py');native=load('native','residential-native.py');lab=res.lab
# Dedicated fixture range; Residential worker cleanup must not touch this test.
lab.SANDBOX_ADDRESS_BASES['crn']=8704
CORE=pathlib.Path('/opt/config-repair/sing-box-1.14.4/sing-box')
CLIENT=r'''
import socket,threading,struct,json
payload=b'native-isolated-synthetic-nonce'
def exact(s,n):
 b=b''
 while len(b)<n:
  q=s.recv(n-len(b))
  if not q:raise RuntimeError('closed')
  b+=q
 return b
def connect(cmd,target):
 s=socket.create_connection(('127.0.0.1',10808),timeout=3);s.sendall(b'\x05\x01\x00');assert exact(s,2)==b'\x05\x00'
 s.sendall(bytes([5,cmd,0,1])+socket.inet_aton('127.0.0.1')+struct.pack('!H',target));h=exact(s,4);assert h[1]==0
 if h[3]==1:host=socket.inet_ntoa(exact(s,4))
 elif h[3]==4:host=socket.inet_ntop(socket.AF_INET6,exact(s,16))
 else:host=exact(s,exact(s,1)[0]).decode()
 port=struct.unpack('!H',exact(s,2))[0];return s,(host,port)
tcp=socket.socket();tcp.bind(('127.0.0.1',0));tcp.listen();tp=tcp.getsockname()[1]
def echo():
 c,_=tcp.accept();data=c.recv(1024);c.sendall(data);c.close();tcp.close()
threading.Thread(target=echo,daemon=True).start()
s,_=connect(1,tp);s.sendall(payload);assert exact(s,len(payload))==payload;s.close()
u=socket.socket(socket.AF_INET,socket.SOCK_DGRAM);u.bind(('127.0.0.1',0));up=u.getsockname()[1]
def echo_udp():
 b,a=u.recvfrom(4096);u.sendto(b,a);u.close()
threading.Thread(target=echo_udp,daemon=True).start()
a,relay=connect(3,0);client=socket.socket(socket.AF_INET,socket.SOCK_DGRAM);client.settimeout(3)
client.sendto(b'\x00\x00\x00\x01'+socket.inet_aton('127.0.0.1')+struct.pack('!H',up)+payload,relay)
reply,_=client.recvfrom(4096);assert reply.endswith(payload);a.close();client.close();print(json.dumps({'tcp':True,'udp':True}))
'''
def main():
 assert os.geteuid()==0
 pin=next(c for c in json.loads((HERE/'residential-cores.json').read_text())['cores'] if c['id']=='sing-box-1.14.4')
 assert lab.digest(CORE.read_bytes())==pin['sha256']
 receipt={'core':pin['id'],'sha256':pin['sha256'],'evidence':'synthetic isolated protocol fixture only','live_residential_proof':False,'results':[]}
 with tempfile.TemporaryDirectory(prefix='p31-native-',dir=ROOT/'.local') as directory:
  work=pathlib.Path(directory)
  subprocess.run(['go','test','./internal/residential','-run','^TestNativeCorePlanFixture$'],cwd=ROOT,env=os.environ|{'NATIVE_TEST_PLAN_DIR':directory},check=True,stdout=subprocess.DEVNULL)
  subprocess.run(['openssl','req','-x509','-newkey','rsa:2048','-nodes','-keyout',str(work/'key'),'-out',str(work/'cert'),'-days','1','-subj','/CN=example.com','-addext','subjectAltName=DNS:example.com'],check=True,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
  cert=(work/'cert').read_text().splitlines();key=(work/'key').read_text().splitlines()
  for kind in ['tuic','hysteria','hysteria2']:
   plan=json.loads((work/(kind+'.json')).read_text())
   for variant in plan['variants']:
    config=json.loads(base64.b64decode(variant['config']))
    with native.sandbox_type(lab)(CORE,'b31abc',8199,prefix='crn') as box:
     res.strict_policy(box,{})
     p=box.launch(json.dumps(config).encode(),True);p.wait(timeout=10)
     if p.returncode:raise RuntimeError(kind+' core validation: '+box.logs().decode()[-1500:])
     box.stop();resolved,runtime=native.prepare(config)
     assert list(resolved)==['8.8.8.8']
     # Assign this fixture address locally. No packet can leave this namespace.
     box.net([lab.IP,'addr','add','8.8.8.8/32','dev','lo'])
     runtime['outbounds'][0]['tls']['certificate']=cert
     server={'type':kind,'tag':'fixture-server','listen':'8.8.8.8','listen_port':24443,'tls':{'enabled':True,'certificate':cert,'key':key}}
     if kind=='tuic':server['users']=[{'uuid':'00000000-0000-4000-8000-000000000001','password':'synthetic'}];server['tls']['alpn']=['h3']
     elif kind=='hysteria':server.update(users=[{'auth_str':'synthetic'}],up_mbps=20,down_mbps=10)
     else:server['users']=[{'password':'synthetic'}]
     runtime['inbounds'].append(server);runtime['outbounds'].append({'type':'direct','tag':'fixture-echo'})
     runtime['route']['rules']=[{'inbound':['fixture-server'],'action':'route','outbound':'fixture-echo'}]
     p=box.launch(json.dumps(runtime).encode())
     for _ in range(30):
      if p.poll() is not None:raise RuntimeError(kind+' runtime: '+box.logs().decode()[-1500:])
      ready=box.net(['/usr/bin/python3','-c','import socket;s=socket.socket();print(s.connect_ex(("127.0.0.1",10808)))'],timeout=2)
      if ready.stdout.strip()==b'0':break
      time.sleep(.1)
     result=box.net(['/usr/bin/python3','-c',CLIENT],timeout=15)
     if result.returncode:raise RuntimeError(kind+' traffic: '+result.stderr.decode()[-1000:]+' '+box.logs().decode()[-1500:])
     passed=json.loads(result.stdout);box.stop()
     _,client_only=native.prepare(config);client_only['outbounds'][0]['tls']['certificate']=cert
     process=box.launch(json.dumps(client_only).encode())
     for _ in range(30):
      if process.poll() is not None:raise RuntimeError('negative fixture failed to start')
      ready=box.net(['/usr/bin/python3','-c','import socket;s=socket.socket();print(s.connect_ex(("127.0.0.1",10808)))'],timeout=2)
      if ready.stdout.strip()==b'0':break
      time.sleep(.1)
     negative=subprocess.run([lab.IP,'netns','exec',box.name,'/usr/bin/python3','-c',CLIENT],capture_output=True,timeout=15)
     assert negative.returncode!=0,'dead upstream leaked to local echo'
     receipt['results'].append({'kind':kind,'variant':variant['name'],'core_valid':True,'failed_upstream_blocks':True,**passed})
 receipt['passed']=True
 (ROOT/'.local/p31-native-integration.json').write_text(json.dumps(receipt,indent=2));print(json.dumps(receipt))
if __name__=='__main__':main()
