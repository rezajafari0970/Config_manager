#!/usr/bin/env python3
"""Real tunneled TLS DoH and cold-start failure in a closed synthetic namespace.
The only substitutions are fixture endpoints/CA and accelerated observation timing.
No external DNS, production credentials, system trust changes or carrier claim.
"""
import hashlib,importlib.util,json,os,pathlib,secrets,socket,struct
import subprocess,sys,tempfile,threading,time
from http.server import BaseHTTPRequestHandler,ThreadingHTTPServer
HERE=pathlib.Path(__file__).resolve().parent

def module(name,path):
 spec=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m
traffic=module('bootstrap_traffic',HERE/'traffic-policy-test.py')

def services(work):
 lock=threading.Lock()
 def record(**item):
  with lock:
   with (work/'trace.jsonl').open('a') as f:f.write(json.dumps(dict(at=time.monotonic(),**item))+'\n')
 def answer(b,peer,service):
  end=12;labels=[]
  while b[end]:
   n=b[end];labels.append(b[end+1:end+1+n].decode());end+=1+n
  end+=1;name='.'.join(labels);kind=struct.unpack('!H',b[end:end+2])[0]
  allowed=peer=='127.0.0.10' if name=='peer.test' else peer in ('127.0.0.10','127.0.0.11')
  record(name=name,kind=kind,peer=peer,service=service,allowed=allowed)
  if not allowed:raise ValueError('direct or circular DNS')
  time.sleep(.08) # Keep both parallel resolver requests observable.
  data=socket.inet_aton('127.0.0.1' if name=='peer.test' else '203.0.113.7') if kind==1 else b'\x02ok'
  return b[:2]+b'\x81\x80'+struct.pack('!4H',1,1,0,0)+b[12:end+4]+b'\xc0\x0c'+struct.pack('!HHIH',kind,1,30,len(data))+data
 def txt():
  s=socket.socket();s.setsockopt(socket.SOL_SOCKET,socket.SO_REUSEADDR,1);s.bind(('127.0.0.1',24453));s.listen()
  def serve(c,peer):
   with c:
    c.settimeout(3)
    try:
     while True:
      n=struct.unpack('!H',traffic.exact(c,2))[0];r=answer(traffic.exact(c,n),peer[0],'txt');c.sendall(struct.pack('!H',len(r))+r)
    except (OSError,ValueError):pass
  while True:
   c,peer=s.accept();threading.Thread(target=serve,args=(c,peer),daemon=True).start()
 threading.Thread(target=txt,daemon=True).start()
 class HTTP(BaseHTTPRequestHandler):
  def do_GET(self):record(service='http',peer=self.client_address[0]);self.send_response(204);self.end_headers()
  def log_message(self,*args):pass
 server=ThreadingHTTPServer(('127.0.0.1',24447),HTTP)
 (work/'ready').touch();server.serve_forever()

def exercise():
 for kind in (1,16):
  query=b'\x12\x34\x01\x00'+struct.pack('!4H',1,0,0,0)+b'\x03app\x04test\x00'+struct.pack('!HH',kind,1)
  s,_=traffic.socks(1,53)
  with s:
   s.sendall(struct.pack('!H',len(query))+query);n=struct.unpack('!H',traffic.exact(s,2))[0];reply=traffic.exact(s,n)
   assert reply.endswith(socket.inet_aton('203.0.113.7') if kind==1 else b'\x02ok')
 print(json.dumps({'a':True,'txt':True}))

def main():
 request=json.load(sys.stdin);core=pathlib.Path(request['core']).resolve()
 assert os.geteuid()==0 and hashlib.sha256(core.read_bytes()).hexdigest()=='64d46afb80adea1bf97a0d467e83f4a9ac1ebd0995891e84bca3f1a1d1affb1d'
 config=request['config'];resolvers=config['dns']['servers'];assert len(resolvers)==4
 assert [x['address'] for x in resolvers]==['https://8.8.8.8/dns-query','https://1.1.1.1/dns-query','https://1.1.1.1/dns-query','https://8.8.8.8/dns-query']
 for n,server in enumerate(resolvers):
  assert server['timeoutMs']==2500 and server['skipFallback']==(n<2)
  server['address']='https://127.0.0.'+('2' if n in (0,3) else '3')+':24454/dns-query'
 assert config['routing']['balancers'][1]['selector']==['proxy-01']
 assert resolvers[0]['domains']==resolvers[1]['domains']==['full:peer.test']
 for i,port in enumerate((24443,24449)):
  config['inbounds'].append({'tag':'fixture-'+str(i),'listen':'127.0.0.1','port':port,'protocol':'vless','settings':{'decryption':'none','clients':[{'id':'11111111-1111-4111-8111-111111111111'}]},'streamSettings':{'network':'tcp','security':'none'}})
  config['outbounds'].append({'tag':'fixture-exit-'+str(i),'sendThrough':'127.0.0.'+str(10+i),'protocol':'freedom','settings':{'finalRules':[{'action':'allow','network':'tcp','ip':['127.0.0.0/8'],'port':'24447,24453,24454'}]}})
  config['routing']['rules'].insert(0,{'type':'field','inboundTag':['fixture-'+str(i)],'outboundTag':'fixture-exit-'+str(i)})
 next(x for x in config['outbounds'] if x['tag']=='smart-dns-out')['settings'].update(rewriteAddress='127.0.0.1',rewritePort=24453)
 config['burstObservatory']['pingConfig'].update(destination='http://127.0.0.1:24447/generate_204',interval='1s',sampling=1,timeout='500ms')
 with tempfile.TemporaryDirectory(prefix='synthetic-bootstrap-') as directory:
  work=pathlib.Path(directory)
  subprocess.run(['openssl','req','-x509','-newkey','rsa:2048','-nodes','-days','1','-subj','/CN=bootstrap-fixture','-addext','subjectAltName=IP:127.0.0.2,IP:127.0.0.3','-keyout',str(work/'key.pem'),'-out',str(work/'cert.pem')],check=True,capture_output=True,timeout=10)
  # Bind only the ephemeral fixture CA inside this test sandbox. Neither the
  # original sandbox driver nor host trust store is modified.
  driver=(HERE/'live-probe.py').read_text();needle="'--ro-bind','/etc/ssl/certs/ca-certificates.crt','/etc/ssl/certs/ca-certificates.crt'"
  assert driver.count(needle)==1
  (work/'driver.py').write_text(driver.replace(needle,"'--ro-bind',"+repr(str(work/'cert.pem'))+",'/etc/ssl/certs/ca-certificates.crt'"))
  subprocess.run(['go','build','-trimpath','-buildvcs=false','-o',str(work/'doh'),str(HERE/'testdata/doh-fixture.go')],check=True,capture_output=True,timeout=60)
  lab=module('bootstrap_sandbox',work/'driver.py')
  with lab.Sandbox(str(core),secrets.token_hex(3),8200,prefix='cstb') as box:
   # Close even the driver's normal DNS exception. The fixture has no successful egress. UseIPv4 system-fallback attempts are counted.
   box.net([lab.NFT,'-f','-'],input=b'flush ruleset\ntable inet guard { counter dns_fallback { packets 0 bytes 0; }; chain out { type filter hook output priority 0; policy drop; oifname "lo" accept; ip daddr 1.1.1.1 udp dport 53 counter name dns_fallback drop; ip daddr 1.1.1.1 tcp dport 53 counter name dns_fallback drop; counter; }; chain in { type filter hook input priority 0; policy drop; iifname "lo" accept; }; }\n')
   child=subprocess.Popen([lab.IP,'netns','exec',box.name,sys.executable,__file__,'--services',str(work)],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
   doh=subprocess.Popen([lab.IP,'netns','exec',box.name,str(work/'doh'),str(work)],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
   try:
    for _ in range(50):
     if (work/'ready').exists() and (work/'doh-ready').exists():break
     time.sleep(.1)
    assert (work/'ready').exists(),'fixture not ready'
    process=box.launch(json.dumps(config).encode());time.sleep(5)
    assert process.poll() is None,'synthetic core exited'
    try:result=box.net([sys.executable,__file__,'--exercise'],timeout=15)
    except subprocess.CalledProcessError as e:raise RuntimeError(e.stderr.decode()+str((work/'trace.jsonl').read_text() if (work/'trace.jsonl').exists() else 'no trace')+box.logs().decode(errors='replace')[-2000:]) from None
    assert all(json.loads(result.stdout).values())
    events=[json.loads(x) for x in (work/'trace.jsonl').read_text().splitlines()]
    bootstrap=[x for x in events if x.get('name')=='peer.test']
    assert {x['service'] for x in bootstrap}=={'127.0.0.2','127.0.0.3'},events
    assert all(x['peer']=='127.0.0.10' and x['allowed'] and x['http']==2 for x in bootstrap),events
    assert any(x.get('service')=='http' and x['peer']=='127.0.0.11' for x in events),'domain member never connected'
    assert all(x.get('allowed',True) for x in events),'DNS bypassed proxy'
    assert {x.get('name') for x in events if x.get('name')}=={'peer.test','app.test'},events
    def dns_attempts():
     state=json.loads(box.net([lab.NFT,'-j','list','counter','inet','guard','dns_fallback']).stdout)
     return next(x['counter']['packets'] for x in state['nftables'] if 'counter' in x)
    startup_attempts=dns_attempts()
    # Cold restart with the independent IP peer unavailable: block fallback,
    # never recursively bootstrap via domain or silently fall back to direct.
    box.stop();box.net([lab.NFT,'add','table','inet','fixture_failure'])
    box.net([lab.NFT,'add','chain','inet','fixture_failure','out','{ type filter hook output priority 200; policy accept; }'])
    box.net([lab.NFT,'add','rule','inet','fixture_failure','out','tcp','dport','24443','drop'])
    count=len(events);box.launch(json.dumps(config).encode());time.sleep(3)
    result=subprocess.run([lab.IP,'netns','exec',box.name,'/usr/bin/curl','--noproxy','','--proxy','socks5h://127.0.0.1:10808','--max-time','2','--silent','--output','/dev/null','--write-out','%{http_code}','http://127.0.0.1:24447/generate_204'],capture_output=True,timeout=4)
    assert result.returncode!=0 and result.stdout!=b'204','cold bootstrap failed open'
    later=[json.loads(x) for x in (work/'trace.jsonl').read_text().splitlines()][count:]
    assert not any(x.get('name')=='peer.test' for x in later),'bootstrap bypassed dead IP pool'
    blocked=dns_attempts()
    print(json.dumps({'passed':True,'tls_doh_resolvers':2,'bootstrap_ip_only':True,'domain_member_connected':True,'dns_a_and_txt_tunneled':True,'cold_proxy_failure_closed':True,'blocked_startup_dns_attempts':startup_attempts,'blocked_cold_failure_dns_attempts':blocked-startup_attempts,'useipv4_system_fallback':'possible; prevented by fixture firewall, not by this JSON','carrier_tests':'not_run'}))
   finally:
    for process in (child,doh):
     process.terminate()
     try:process.wait(timeout=3)
     except subprocess.TimeoutExpired:process.kill();process.wait()
if __name__=='__main__':
 if len(sys.argv)>1 and sys.argv[1]=='--services':services(pathlib.Path(sys.argv[2]))
 elif len(sys.argv)>1 and sys.argv[1]=='--exercise':exercise()
 else:main()
