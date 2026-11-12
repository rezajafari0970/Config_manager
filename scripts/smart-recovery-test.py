#!/usr/bin/env python3
"""Reproduce slow-but-working leastLoad exclusion using real pinned cores, closed netns."""
import base64,copy,hashlib,importlib.util,json,pathlib,secrets,socket,struct,subprocess,sys,threading,time
HERE=pathlib.Path(__file__).resolve().parent
def mod(name,path):
 s=importlib.util.spec_from_file_location(name,path);m=importlib.util.module_from_spec(s);s.loader.exec_module(m);return m
lab=mod('lab',HERE/'live-probe.py')
fixture=mod('fixture',HERE/'smart-responsive-test.py')
def services(delay):
 # Reuse the fixture DNS and HTTP server; our relay has a separate port.
 threading.Thread(target=fixture.services,daemon=True).start()
 listener=socket.socket();listener.setsockopt(socket.SOL_SOCKET,socket.SO_REUSEADDR,1);listener.bind(('127.0.0.1',24456));listener.listen()
 def serve(c):
  try:d=socket.create_connection(('127.0.0.1',24449),timeout=5)
  except OSError:c.close();return
  def pump(a,b):
   try:
    while True:
     data=a.recv(65536)
     if not data:break
     time.sleep(delay);b.sendall(data)
   except OSError:pass
   finally:
    try:b.shutdown(socket.SHUT_WR)
    except OSError:pass
  t=threading.Thread(target=pump,args=(c,d),daemon=True);t.start();pump(d,c);t.join(5);c.close();d.close()
 while True:threading.Thread(target=serve,args=(listener.accept()[0],),daemon=True).start()
def recovery(c):
 # Candidate-only until the real test proves it. No direct or unobserved fallback.
 r=c['routing'];pool=r['balancers'][0]
 pool['fallbackTag']='smart-recovery'
 r['balancers'].append({'tag':'smart-recovery-balancer','selector':pool['selector'][:],'strategy':{'type':'leastLoad','settings':{'expected':1,'maxRTT':'5s','tolerance':0.5}},'fallbackTag':'block'})
 r['rules'].insert(0,{'type':'field','inboundTag':['smart-recovery-egress'],'network':'tcp,udp','balancerTag':'smart-recovery-balancer'})
 c['outbounds'].append({'tag':'smart-recovery','protocol':'loopback','settings':{'inboundTag':'smart-recovery-egress'}})
 return c
def config(candidate):
 c={'log':{'loglevel':'warning'},'inbounds':[{'tag':'socks','listen':'127.0.0.1','port':10808,'protocol':'socks','settings':{'udp':True}},{'tag':'fixture-server','listen':'127.0.0.1','port':24449,'protocol':'vless','settings':{'decryption':'none','clients':[{'id':'11111111-1111-4111-8111-111111111111'}]}}],
 'outbounds':[{'tag':'proxy-01','protocol':'vless','settings':{'address':'127.0.0.1','port':24456,'id':'11111111-1111-4111-8111-111111111111','encryption':'none'}},{'tag':'block','protocol':'blackhole'},{'tag':'fixture-exit','protocol':'freedom','settings':{'finalRules':[{'action':'allow','network':'tcp,udp','ip':['127.0.0.1/32'],'port':'53,24447,24453'}]}}],
 'routing':{'balancers':[{'tag':'smart-parallel-balancer','selector':['proxy-'],'strategy':{'type':'leastLoad','settings':{'expected':3,'maxRTT':'2s','tolerance':0.15}},'fallbackTag':'block'}],'rules':[{'type':'field','inboundTag':['fixture-server'],'outboundTag':'fixture-exit'},{'type':'field','network':'tcp,udp','balancerTag':'smart-parallel-balancer'}]},
 'burstObservatory':{'subjectSelector':['proxy-'],'pingConfig':{'destination':'http://127.0.0.1:24447/generate_204','interval':'10s','sampling':1,'timeout':'5s','httpMethod':'GET'}},
 'metrics':{'tag':'metrics','listen':'127.0.0.1:11111'}}
 if candidate and len(sys.argv)>2:
  generated=json.loads(pathlib.Path(sys.argv[2]).read_text())
  generated['inbounds']+=c['inbounds'][1:]
  generated['outbounds'].append(c['outbounds'][-1])
  generated['routing']['rules'].insert(0,c['routing']['rules'][0])
  generated['burstObservatory']['pingConfig']['destination']=c['burstObservatory']['pingConfig']['destination']
  generated['dns']['servers']=[{'address':'tcp://127.0.0.1:24453','timeoutMs':2500}]
  next(x for x in generated['outbounds'] if x['tag']=='smart-dns-out')['settings'].update(rewriteAddress='127.0.0.1',rewritePort=24453)
  return generated
 return recovery(c) if candidate else c
def status(box):
 return json.loads(box.net(['/usr/bin/curl','--noproxy','*','--silent','--max-time','2','http://127.0.0.1:11111/debug/vars']).stdout)['observatory'].get('proxy-01',{})
def request(box):
 start=time.monotonic()
 r=subprocess.run([lab.IP,'netns','exec',box.name,'/usr/bin/curl','--noproxy','','--proxy','socks5h://127.0.0.1:10808','--silent','--max-time','6','--output','/dev/null','--write-out','%{http_code}','http://127.0.0.1:24447/generate_204'],capture_output=True,timeout=8)
 return {'passed':r.returncode==0 and r.stdout==b'204','elapsed_ms':round(1000*(time.monotonic()-start),2),'curl_exit':r.returncode}
def dns(box):
 program=r"""import importlib.util,pathlib,socket,struct,json
spec=importlib.util.spec_from_file_location('traffic',pathlib.Path('scripts/traffic-policy-test.py').resolve());m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
out={}
for typ in (1,16):
 q=b'\x12\x34\x01\x00'+struct.pack('!4H',1,0,0,0)+b'\x03app\x04test\x00'+struct.pack('!HH',typ,1)
 expected=socket.inet_aton('203.0.113.7') if typ==1 else b'\x02ok'
 s,_=m.socks(1,53);s.settimeout(7)
 with s:
  s.sendall(struct.pack('!H',len(q))+q);n=struct.unpack('!H',m.exact(s,2))[0];out['tcp_'+str(typ)]=m.exact(s,n).endswith(expected)
 u=socket.socket(socket.AF_INET,socket.SOCK_DGRAM);u.bind(('127.0.0.1',0));u.settimeout(7);control,relay=m.socks(3,u.getsockname()[1])
 with control,u:
  u.sendto(b'\x00\x00\x00\x01'+socket.inet_aton('127.0.0.1')+struct.pack('!H',53)+q,relay);out['udp_'+str(typ)]=u.recvfrom(8192)[0].endswith(expected)
print(json.dumps(out))
"""
 r=box.net([sys.executable,'-c',program],timeout=35)
 return json.loads(r.stdout)
def main():
 core=sys.argv[1];out={}
 assert hashlib.sha256(pathlib.Path(core).read_bytes()).hexdigest() in {'e87bce586acdb9136aa5b1f1d9c8db9f1ea1d145b6d86368c7f6f662f9f6a66d','cc4b84d3069569f297a61f04b914f2bb37e9d202fd7b84cb50def1fc30fd40e1'},'unpinned core'
 for name,delay,candidate in [('baseline_slow',1.15,False),('candidate_slow',1.15,True),('candidate_fast',0,True)]:
  with lab.Sandbox(core,secrets.token_hex(3),8199,prefix='csrf') as box:
   box.policy({});svc=subprocess.Popen([lab.IP,'netns','exec',box.name,sys.executable,__file__,'--services',str(delay)],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
   try:
    time.sleep(.3);p=box.launch(json.dumps(config(candidate)).encode())
    for _ in range(180):
     time.sleep(.5)
     assert p.poll() is None,box.logs().decode(errors='replace')
     try:s=status(box)
     except (subprocess.SubprocessError,KeyError,ValueError):continue
     if s.get('alive') and (s.get('delay',0)>2000 if delay else s.get('delay',9999)<2000):break
    else:raise AssertionError(('no observation',name,s))
    rows=[request(box) for _ in range(3)]
    assert all(r['passed']==candidate for r in rows),(name,s,rows)
    out[name]={'observer':s,'requests':rows}
    if name=='candidate_slow':
     if len(sys.argv)>2:
      out[name]['dns']=dns(box);assert all(out[name]['dns'].values()),out[name]['dns']
     box.net(['/usr/sbin/nft','add','table','inet','failure'])
     box.net(['/usr/sbin/nft','add','chain','inet','failure','out','{ type filter hook output priority 200; policy accept; }'])
     box.net(['/usr/sbin/nft','add','rule','inet','failure','out','tcp','dport','24456','reject'])
     for _ in range(180):
      time.sleep(.5);s=status(box)
      if s and not s.get('alive'):break
     else:raise AssertionError('dead peer not observed')
     row=request(box);assert not row['passed'];out['all_dead']={'observer':s,'request':row}
   finally:
    svc.terminate()
    try:svc.wait(3)
    except subprocess.TimeoutExpired:svc.kill();svc.wait()
 print(json.dumps({'passed':True,'core_sha256':hashlib.sha256(pathlib.Path(core).read_bytes()).hexdigest(),'scope':'closed namespace; TCP relay adds 1150ms each direction; baseline observation 10s/1, generated candidate retains supplied observation timing', 'generated_envelope':len(sys.argv)>2,'results':out}))
if __name__=='__main__':
 if sys.argv[1]=='--services':services(float(sys.argv[2]))
 else:main()
