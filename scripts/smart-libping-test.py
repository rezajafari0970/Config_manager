#!/usr/bin/env python3
"""Exact Lib v40 app-filter reproduction, synthetic VLESS, closed namespace."""
import copy,importlib.util,json,os,pathlib,secrets,subprocess,sys,time
HERE=pathlib.Path(__file__).resolve().parent
def serve():
 import socket,socketserver,threading,uuid
 from http.server import BaseHTTPRequestHandler,ThreadingHTTPServer
 class HTTP(BaseHTTPRequestHandler):
  def do_GET(self):self.send_response(204);self.end_headers()
  def log_message(self,*args):pass
 class Proxy(socketserver.BaseRequestHandler):
  def handle(self):
   c=self.request
   def read(n):
    b=b''
    while len(b)<n:
     v=c.recv(n-len(b))
     if not v:raise EOFError()
     b+=v
    return b
   try:
    assert read(1)==b'\x00' and read(16)==uuid.UUID('11111111-1111-4111-8111-111111111111').bytes
    read(read(1)[0]);assert read(1)==b'\x01'
    port=int.from_bytes(read(2),'big');kind=read(1)[0]
    host=socket.inet_ntoa(read(4)) if kind==1 else read(read(1)[0]).decode() if kind==2 else socket.inet_ntop(socket.AF_INET6,read(16))
    assert host=='127.0.0.1' and port==24447
    with socket.create_connection((host,port),2) as peer:
     c.sendall(b'\x00\x00')
     request=b''
     while b'\r\n\r\n' not in request:request+=c.recv(4096)
     peer.sendall(request)
     while True:
      b=peer.recv(4096)
      if not b:break
      c.sendall(b)
   except (OSError,EOFError,AssertionError):pass
 class Server(socketserver.ThreadingTCPServer):allow_reuse_address=True;daemon_threads=True
 threading.Thread(target=ThreadingHTTPServer(('127.0.0.1',24447),HTTP).serve_forever,daemon=True).start()
 Server(('0.0.0.0',24443),Proxy).serve_forever()
def main():
 assert os.geteuid()==0
 request=json.load(sys.stdin);helper=pathlib.Path(request['helper']).resolve()
 spec=importlib.util.spec_from_file_location('libping_sandbox',HERE/'live-probe.py');lab=importlib.util.module_from_spec(spec);spec.loader.exec_module(lab)
 after=request['config'];before=copy.deepcopy(after)
 before['outbounds'].sort(key=lambda x:x['tag']!='block')
 relay=copy.deepcopy(after)
 first=relay['outbounds'][0];settings=first['settings']
 node=settings if 'address' in settings else settings['vnext'][0]
 node['address']='127.0.0.1';node['port']=39081
 relay['inbounds'].append({'tag':'fixture-relay','listen':'127.0.0.1','port':39081,'protocol':'dokodemo-door','settings':{'address':'127.0.0.1','port':24443,'network':'tcp','followRedirect':False}})
 relay['outbounds'].append({'tag':'fixture-transport','protocol':'freedom','settings':{}})
 relay['routing']['rules'].insert(0,{'type':'field','inboundTag':['fixture-relay'],'ip':['127.0.0.1'],'port':'24443','network':'tcp','outboundTag':'fixture-transport'})
 redirect=copy.deepcopy(after)
 first=redirect['outbounds'][0];settings=first['settings'];node=settings if 'address' in settings else settings['vnext'][0]
 node['address']='127.0.0.1';node['port']=24443
 first.setdefault('streamSettings',{})['sockopt']={'dialerProxy':'fixture-fixed-transport'}
 redirect['outbounds'].append({'tag':'fixture-fixed-transport','protocol':'freedom','settings':{'redirect':'203.0.113.81:24443'}})
 negative=copy.deepcopy(redirect);negative['outbounds'][-1]['settings']['redirect']='203.0.113.81:1'
 results={}
 for name,config in [('block_first',before),('independent_ip_first',after),('local_relay_default',relay),('outbound_redirect',redirect),('dead_redirect',negative)]:
  with lab.Sandbox(str(helper),secrets.token_hex(3),8198,prefix='cstp') as box:
   box.net([lab.IP,'addr','add','203.0.113.81/32','dev','lo'])
   # No external DNS or Internet is permitted in this regression.
   box.net([lab.NFT,'-f','-'],input=b'flush ruleset\ntable inet guard { chain out { type filter hook output priority 0; policy drop; oifname "lo" accept; }; chain in { type filter hook input priority 0; policy drop; iifname "lo" accept; }; }\n')
   child=subprocess.Popen([lab.IP,'netns','exec',box.name,sys.executable,__file__,'--serve'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
   try:
    time.sleep(.25)
    process=box.launch(json.dumps(config).encode());process.wait(timeout=12)
    result=json.loads(box.logs());assert result['core']=='26.10.31',result
    results[name]=result
   finally:
    child.terminate();child.wait(timeout=3)
 assert not results['block_first']['passed'] and results['independent_ip_first']['passed'] and not results['local_relay_default']['passed'] and results['outbound_redirect']['passed'] and not results['dead_redirect']['passed'],results
 print(json.dumps({'passed':True,'lib':'v40','android_apk':'not_run','results':results}))
if __name__=='__main__':
 if '--serve' in sys.argv:serve()
 else:main()
