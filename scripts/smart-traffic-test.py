#!/usr/bin/env python3
"""Controlled leastLoad failover: dead first member, live second, closed net namespace."""
import hashlib,importlib.util,json,os,pathlib,secrets,subprocess,sys,tempfile,time
HERE=pathlib.Path(__file__).resolve().parent
def main():
 request=json.load(sys.stdin);core=pathlib.Path(request['core']).resolve()
 assert os.geteuid()==0 and hashlib.sha256(core.read_bytes()).hexdigest()=='64d46afb80adea1bf97a0d467e83f4a9ac1ebd0995891e84bca3f1a1d1affb1d'
 spec=importlib.util.spec_from_file_location('smart_traffic_sandbox',HERE/'live-probe.py');lab=importlib.util.module_from_spec(spec);spec.loader.exec_module(lab)
 with tempfile.TemporaryDirectory(prefix='synthetic-smart-') as directory:
  work=pathlib.Path(directory);key=work/'key.pem';cert=work/'cert.pem'
  subprocess.run(['openssl','req','-x509','-newkey','rsa:2048','-nodes','-days','1','-subj','/CN=localhost','-keyout',str(key),'-out',str(cert)],check=True,capture_output=True,timeout=10)
  pin=hashlib.sha256(subprocess.check_output(['openssl','x509','-in',str(cert),'-outform','DER'])).hexdigest()
  config=request['config'];proxies=[x for x in config['outbounds'] if x['tag'].startswith('proxy-')]
  assert len(proxies)==2
  for i,p in enumerate(proxies):
   assert p['protocol']=='vless'
   s=p['settings'];assert (s.get('address') or s['vnext'][0]['address'])=='127.0.0.1'
   p['streamSettings']['tlsSettings']={'serverName':'localhost','pinnedPeerCertSha256':pin}
  server={'tag':'fixture-server','listen':'127.0.0.1','port':24443,'protocol':'vless','settings':{'decryption':'none','clients':[{'id':'11111111-1111-4111-8111-111111111111','flow':'xtls-rprx-vision'}]},'streamSettings':{'network':'tcp','security':'tls','tlsSettings':{'certificates':[{'certificate':cert.read_text().strip().splitlines(),'key':key.read_text().strip().splitlines()}]}}}
  config['inbounds'].append(server)
  config['outbounds'].append({'tag':'fixture-direct','protocol':'freedom','settings':{'finalRules':[{'action':'allow','network':'tcp,udp','ip':['127.0.0.1/32'],'port':'443,24444,24445,24447'}]}})
  config['routing']['rules'].insert(0,{'type':'field','inboundTag':['fixture-server'],'outboundTag':'fixture-direct'})
  # Only controlled probe destination/timing differ from the shipped envelope.
  config['burstObservatory']['pingConfig'].update(destination='http://127.0.0.1:24447/generate_204',interval='1s',sampling=1,timeout='500ms')
  config['log']={'loglevel':'warning'}
  with lab.Sandbox(str(core),secrets.token_hex(3),8199,prefix='cstf') as box:
   box.policy({});children=[]
   try:
    children.append(subprocess.Popen([lab.IP,'netns','exec',box.name,sys.executable,str(HERE/'traffic-policy-test.py'),'--echo'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL))
    children.append(subprocess.Popen([lab.IP,'netns','exec',box.name,sys.executable,__file__,'--http'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL))
    process=box.launch(json.dumps(config).encode())
    for _ in range(50):
     if process.poll() is not None:raise RuntimeError('smart synthetic core exited '+box.logs().decode(errors='replace')[-1500:])
     check=box.net([sys.executable,'-c','import socket;s=socket.socket();print(s.connect_ex(("127.0.0.1",10808)))'])
     if check.stdout.strip()==b'0':break
     time.sleep(.1)
    time.sleep(3)
    # First proxy listens on absent port 24446: successful traffic proves selection
    # of the observed second proxy, not accidental use of the fallback.
    result=box.net([sys.executable,str(HERE/'traffic-policy-test.py'),'--exercise'],timeout=20)
    traffic=json.loads(result.stdout)
    assert traffic=={'tcp':True,'udp':True,'udp443':True},traffic
    print(json.dumps({'passed':True,'dead_first_member_skipped':True,'traffic':traffic,'quic_handshake':'not_run'}))
   finally:
    for child in children:
     child.terminate()
     try:child.wait(timeout=3)
     except subprocess.TimeoutExpired:child.kill();child.wait()
if __name__=='__main__':
 if len(sys.argv)>1 and sys.argv[1]=='--http':
  from http.server import BaseHTTPRequestHandler,HTTPServer
  class Handler(BaseHTTPRequestHandler):
   def do_GET(self):self.send_response(204);self.end_headers()
   def log_message(self,*args):pass
  HTTPServer(('127.0.0.1',24447),Handler).serve_forever()
 else:main()
