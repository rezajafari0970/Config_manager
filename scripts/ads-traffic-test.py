#!/usr/bin/env python3
"""Synthetic chain proof in a closed namespace; no production traffic or credentials."""
import copy,hashlib,importlib.util,json,os,pathlib,secrets,socket,struct,subprocess,sys,tempfile,threading,time
HERE=pathlib.Path(__file__).resolve().parent

def echo():
 def udp(port):
  s=socket.socket(socket.AF_INET,socket.SOCK_DGRAM);s.bind(('127.0.0.1',port))
  while True:
   b,peer=s.recvfrom(8192);s.sendto(peer[0].encode()+b':'+b,peer)
 for port in (443,24445):threading.Thread(target=udp,args=(port,),daemon=True).start()
 s=socket.socket();s.setsockopt(socket.SOL_SOCKET,socket.SO_REUSEADDR,1);s.bind(('127.0.0.1',24444));s.listen()
 def client(c,peer):
  with c:
   c.settimeout(3)
   try:c.sendall(peer[0].encode()+b':'+c.recv(8192))
   except OSError:pass
 while True:
  c,peer=s.accept();threading.Thread(target=client,args=(c,peer),daemon=True).start()

def payload_http():
 from http.server import BaseHTTPRequestHandler,ThreadingHTTPServer
 class Handler(BaseHTTPRequestHandler):
  def log_message(self,*args):pass
  def do_GET(self):
   self.send_response(200);self.send_header('Content-Length','65536');self.end_headers();self.wfile.write(b'x'*65536)
 ThreadingHTTPServer(('127.0.0.1',24447),Handler).serve_forever()

def exact(s,n):
 out=b''
 while len(out)<n:
  b=s.recv(n-len(out))
  if not b:raise OSError('closed')
  out+=b
 return out

def address(domain,port):
 name=b'browserleaks.com'
 return (b'\x03'+bytes([len(name)])+name if domain else b'\x01'+socket.inet_aton('127.0.0.1'))+struct.pack('!H',port)

def socks(cmd,port,domain=False):
 s=socket.create_connection(('127.0.0.1',10808),timeout=2);s.settimeout(2)
 try:
  s.sendall(b'\x05\x01\x00');assert exact(s,2)==b'\x05\x00'
  s.sendall(bytes([5,cmd,0])+address(domain,port));h=exact(s,4);assert h[:2]==b'\x05\x00'
  if h[3]==1:host=socket.inet_ntoa(exact(s,4))
  elif h[3]==4:host=socket.inet_ntop(socket.AF_INET6,exact(s,16))
  else:host=exact(s,exact(s,1)[0]).decode()
  return s,(host,struct.unpack('!H',exact(s,2))[0])
 except BaseException:s.close();raise

def exercise(domain):
 token=secrets.token_bytes(24);want=(b'127.0.0.3:' if domain else b'127.0.0.2:')+token;result={}
 try:
  s,_=socks(1,24444,domain)
  with s:s.sendall(token);result['tcp']=exact(s,len(want))==want
 except (OSError,AssertionError):result['tcp']=False
 for port,key in [(24445,'udp'),(443,'udp443')]:
  udp=socket.socket(socket.AF_INET,socket.SOCK_DGRAM);udp.bind(('127.0.0.1',0));udp.settimeout(2)
  try:
   ctl,relay=socks(3,udp.getsockname()[1])
   with ctl,udp:
    udp.sendto(b'\0\0\0'+address(domain,port)+token,relay);reply,_=udp.recvfrom(8192);result[key]=reply.endswith(want)
  except (OSError,AssertionError):result[key]=False;udp.close()
 print(json.dumps(result))

def main():
 request=json.load(sys.stdin);core=pathlib.Path(request['core']).resolve()
 assert os.geteuid()==0 and hashlib.sha256(core.read_bytes()).hexdigest() in {'64d46afb80adea1bf97a0d467e83f4a9ac1ebd0995891e84bca3f1a1d1affb1d','cc4b84d3069569f297a61f04b914f2bb37e9d202fd7b84cb50def1fc30fd40e1','e87bce586acdb9136aa5b1f1d9c8db9f1ea1d145b6d86368c7f6f662f9f6a66d'}
 spec=importlib.util.spec_from_file_location('ads_lab',HERE/'live-probe.py');lab=importlib.util.module_from_spec(spec);spec.loader.exec_module(lab)
 with tempfile.TemporaryDirectory(prefix='synthetic-smart-ads-') as directory:
  work=pathlib.Path(directory);key=work/'key.pem';cert=work/'cert.pem'
  subprocess.run(['openssl','req','-x509','-newkey','rsa:2048','-nodes','-days','1','-subj','/CN=localhost','-keyout',str(key),'-out',str(cert)],check=True,capture_output=True,timeout=10)
  pin=hashlib.sha256(subprocess.check_output(['openssl','x509','-in',str(cert),'-outform','DER'])).hexdigest()
  base=request['config']
  for ob in base['outbounds']:
   if ob['tag'].startswith('proxy-') or ob['tag']=='ads-01':
    assert ob['protocol']=='vless';ob['streamSettings']['tlsSettings']={'serverName':'localhost','pinnedPeerCertSha256':pin}
  def server(tag,port):return {'tag':tag,'listen':'127.0.0.1','port':port,'protocol':'vless','settings':{'decryption':'none','clients':[{'id':'11111111-1111-4111-8111-111111111111','flow':'xtls-rprx-vision'}]},'streamSettings':{'network':'tcp','security':'tls','tlsSettings':{'certificates':[{'certificate':cert.read_text().strip().splitlines(),'key':key.read_text().strip().splitlines()}]}}}
  base['inbounds'] += [server('fixture-main',24443),server('fixture-ads',24448)]
  for name,ip in [('main','127.0.0.2'),('ads','127.0.0.3')]:base['outbounds'].append({'tag':'fixture-'+name+'-direct','protocol':'freedom','sendThrough':ip,'settings':{'domainStrategy':'UseIPv4','finalRules':[{'action':'allow','network':'tcp,udp','ip':['127.0.0.1/32'],'port':'443,24444,24445,24447,24448'}]}})
  base['routing']['rules'][:0]=[{'type':'field','inboundTag':['fixture-main'],'outboundTag':'fixture-main-direct'},{'type':'field','inboundTag':['fixture-ads'],'source':['127.0.0.2/32'],'outboundTag':'fixture-ads-direct'},{'type':'field','inboundTag':['fixture-ads'],'outboundTag':'block'}]
  base['dns']['hosts']={'browserleaks.com':'127.0.0.1'}
  base['burstObservatory']['pingConfig']['destination']='http://127.0.0.1:24447/payload'
  reports={}
  for phase in ('chained','ads-down','bypass-negative'):
   cfg=copy.deepcopy(base)
   if phase=='ads-down':cfg['inbounds']=[x for x in cfg['inbounds'] if x['tag']!='fixture-ads']
   if phase=='bypass-negative':next(x for x in cfg['outbounds'] if x['tag']=='ads-01')['streamSettings']['sockopt'].pop('dialerProxy')
   with lab.Sandbox(str(core),secrets.token_hex(3),8199,prefix='csta') as box:
    box.policy({});children=[]
    try:
     for args in [[str(HERE/'ads-traffic-test.py'),'--echo'],[str(HERE/'ads-traffic-test.py'),'--http64']]:children.append(subprocess.Popen([lab.IP,'netns','exec',box.name,sys.executable]+args,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL))
     process=box.launch(json.dumps(cfg).encode())
     for _ in range(60):
      if process.poll() is not None:raise RuntimeError('core exited '+box.logs().decode(errors='replace')[-1200:])
      r=box.net([sys.executable,'-c','import socket;s=socket.socket();print(s.connect_ex(("127.0.0.1",10808)))'])
      if r.stdout.strip()==b'0':break
      time.sleep(.1)
     deadline=time.monotonic()+60
     while time.monotonic()<deadline:
      state=box.net(['/usr/bin/curl','--noproxy','*','-fsS','--max-time','2','http://127.0.0.1:11111/debug/vars'])
      ns=json.loads(state.stdout).get('observatory',{})
      def good(tag):
       n=ns.get(tag,{});h=n.get('health_ping',{});return n.get('alive') and h.get('all',0)>0 and h.get('fail',0)==0
      if good('proxy-01') and ((phase=='chained' and good('ads-01')) or (phase!='chained' and ns.get('ads-01',{}).get('health_ping',{}).get('fail',0)>0)):break
      time.sleep(.25)
     results={mode:json.loads(box.net([sys.executable,str(HERE/'ads-traffic-test.py'),'--exercise',mode],timeout=20).stdout) for mode in ['main','ads']}
     assert all(results['main'].values()),(phase,results)
     assert all(v==(phase=='chained') for v in results['ads'].values()),(phase,results,box.logs().decode(errors='replace')[-1000:])
     reports[phase]=results
    finally:
     for child in children:
      child.terminate()
      try:child.wait(timeout=3)
      except subprocess.TimeoutExpired:child.kill();child.wait()
  print(json.dumps({'passed':True,'phases':reports,'scope':'synthetic pinned-core chained TCP/UDP/UDP443; no live QUIC or carrier claim'}))
if __name__=='__main__':
 if len(sys.argv)>1 and sys.argv[1]=='--http64':payload_http()
 elif len(sys.argv)>1 and sys.argv[1]=='--echo':echo()
 elif len(sys.argv)>1 and sys.argv[1]=='--exercise':exercise(sys.argv[2]=='ads')
 else:main()
