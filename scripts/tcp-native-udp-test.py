#!/usr/bin/env python3
"""Closed-namespace TCP upload/native UDP fixture. Not Internet or TLS evidence."""
import concurrent.futures,select,hashlib,http.client,http.server,importlib.util,json,os,pathlib,secrets,socket,struct,subprocess,sys,threading,time
HERE=pathlib.Path(__file__).resolve().parent
def echo():
 state={'outer_connections':0};barrier=threading.Barrier(3)
 def relay(client):
  try:
   with client,socket.create_connection(('127.0.0.1',24443),3) as upstream:
    while True:
     ready=select.select([client,upstream],[],[],10)[0]
     if not ready:return
     for src in ready:
      data=src.recv(65535)
      if not data:return
      (upstream if src is client else client).sendall(data)
  except OSError:pass
 def tap():
  with socket.socket() as s:
   s.setsockopt(socket.SOL_SOCKET,socket.SO_REUSEADDR,1);s.bind(('127.0.0.1',24442));s.listen()
   while True:
    client,_=s.accept();state['outer_connections']+=1
    threading.Thread(target=relay,args=(client,),daemon=True).start()
 threading.Thread(target=tap,daemon=True).start()
 class Handler(http.server.BaseHTTPRequestHandler):
  protocol_version='HTTP/1.1'
  def log_message(self,*a):pass
  def do_GET(self):
   if self.path=='/count':body=json.dumps(state).encode()
   else:
    barrier.wait(timeout=10);body=b'x'*2097152
   self.send_response(200);self.send_header('Content-Length',str(len(body)));self.end_headers();self.wfile.write(body)
  def do_POST(self):
   size=int(self.headers.get('Content-Length','0'))
   if size!=1048576:self.send_error(400);return
   data=self.rfile.read(size);body=hashlib.sha256(data).hexdigest().encode()
   self.send_response(200);self.send_header('Content-Length',str(len(body)));self.end_headers();self.wfile.write(body)
 def udp(port):
  with socket.socket(socket.AF_INET,socket.SOCK_DGRAM) as s:
   s.bind(('127.0.0.1',port))
   while True:
    data,peer=s.recvfrom(65535);s.sendto(data,peer)
 for port in (443,24445):threading.Thread(target=udp,args=(port,),daemon=True).start()
 http.server.ThreadingHTTPServer(('127.0.0.1',24444),Handler).serve_forever()
def exercise():
 spec=importlib.util.spec_from_file_location('tc',HERE/'transport-checks.py');tc=importlib.util.module_from_spec(spec);spec.loader.exec_module(tc)
 def tcp(_):
  payload=secrets.token_bytes(1048576);results=[]
  for method,body,wanted in [('GET',b'',b'x'*2097152),('POST',payload,hashlib.sha256(payload).hexdigest().encode())]:
   s,_=tc.socks(1,'127.0.0.1',24444,8,10808)
   with s:
    s.sendall((method+' / HTTP/1.1\r\nHost: fixture\r\nContent-Length: '+str(len(body))+'\r\nConnection: close\r\n\r\n').encode()+body)
    r=http.client.HTTPResponse(s);r.begin();got=r.read(2097153);results.append(r.status==200 and got==wanted)
  return all(results)
 def udp():
  successes=0
  for dest in (443,24445):
   control,relay=tc.socks(3,'0.0.0.0',0,8,10808)
   with control,socket.socket(socket.AF_INET,socket.SOCK_DGRAM) as s:
    s.settimeout(3);prefix=b'\0\0\0\1'+socket.inet_aton('127.0.0.1')+struct.pack('!H',dest)
    for size in (32,1200,1450,1440)*3:
     payload=secrets.token_bytes(size);s.sendto(prefix+payload,relay);data,_=s.recvfrom(65535);successes+=data==prefix+payload
  return successes
 with concurrent.futures.ThreadPoolExecutor(max_workers=4) as pool:
  jobs=[pool.submit(tcp,i) for i in range(3)];results=[j.result() for j in jobs]
  c=http.client.HTTPConnection('127.0.0.1',24444,timeout=3);c.request('GET','/count');outer=json.loads(c.getresponse().read())['outer_connections'];c.close()
  jobs=[pool.submit(tcp,i) for i in range(3)];u=pool.submit(udp);results += [j.result() for j in jobs];count=u.result()
 assert all(results) and count==24
 print(json.dumps({'cold_concurrent_tcp_outer_connections':outer,'tcp_complete_pairs':sum(results),'download_bytes_each':2097152,'upload_echo_sha256_bytes_each':1048576,'udp_exact_echo':count,'udp_attempts':24,'packet_sizes':[32,1200,1450,1440],'scope':'closed loopback namespace; no Internet, TLS or carrier proof'}))
def main():
 core=sys.argv[1];spec=importlib.util.spec_from_file_location('lab',HERE/'live-probe.py');lab=importlib.util.module_from_spec(spec);spec.loader.exec_module(lab)
 assert hashlib.sha256(pathlib.Path(core).read_bytes()).hexdigest() in {'e87bce586acdb9136aa5b1f1d9c8db9f1ea1d145b6d86368c7f6f662f9f6a66d','cc4b84d3069569f297a61f04b914f2bb37e9d202fd7b84cb50def1fc30fd40e1'}
 stream={'network':'tcp','security':'none','tcpSettings':{'header':{'type':'http'}}}
 client={'tag':'proxy','protocol':'vless','settings':{'address':'127.0.0.1','port':24442,'id':'11111111-1111-4111-8111-111111111111','encryption':'none'},'streamSettings':stream,'mux':{'enabled':True,'concurrency':1,'xudpConcurrency':-1,'xudpProxyUDP443':'skip'}}
 config={'inbounds':[{'listen':'127.0.0.1','port':10808,'protocol':'socks','settings':{'udp':True},'tag':'socks'},{'listen':'127.0.0.1','port':24443,'protocol':'vless','settings':{'clients':[{'id':'11111111-1111-4111-8111-111111111111'}],'decryption':'none'},'streamSettings':stream,'tag':'fixture'}],'outbounds':[client,{'tag':'fixture-direct','protocol':'freedom','settings':{'finalRules':[{'action':'allow','network':'tcp,udp','ip':['127.0.0.1/32'],'port':'443,24444,24445'}]}}],'routing':{'rules':[{'type':'field','inboundTag':['fixture'],'outboundTag':'fixture-direct'},{'type':'field','inboundTag':['socks'],'outboundTag':'proxy'}]},'log':{'loglevel':'warning'}}
 with lab.Sandbox(core,secrets.token_hex(3),8199,prefix='ctf') as box:
  box.policy({})
  echoer=subprocess.Popen([lab.IP,'netns','exec',box.name,sys.executable,__file__,'--echo'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
  try:
   process=box.launch(json.dumps(config).encode())
   for _ in range(50):
    if process.poll() is not None:raise RuntimeError('core_exit')
    check=box.net([sys.executable,'-c','import socket; s=socket.socket(); print(s.connect_ex(("127.0.0.1",10808)))'])
    if check.stdout.strip()==b'0':break
    time.sleep(.05)
   report=box.net([sys.executable,__file__,'--exercise'],timeout=35)
   print(report.stdout.decode())
  finally:
   echoer.terminate()
   try:echoer.wait(timeout=3)
   except subprocess.TimeoutExpired:echoer.kill();echoer.wait()
if __name__=='__main__':
 if sys.argv[1]=='--echo':echo()
 elif sys.argv[1]=='--exercise':exercise()
 else:main()
