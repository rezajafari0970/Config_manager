#!/usr/bin/env python3
"""Pinned-core regression for small-probe success with a truncated payload.
Closed Linux namespace, synthetic VLESS peers, loopback-only HTTP origins.
The fixture URL is substituted; production HTTPS is checked in the real lane.
"""
import argparse,base64,copy,hashlib,http.server,importlib.util,json,pathlib,secrets,socket,subprocess,sys,threading,time
HERE=pathlib.Path(__file__).resolve().parent
s=importlib.util.spec_from_file_location('lab',HERE/'live-probe.py');lab=importlib.util.module_from_spec(s);s.loader.exec_module(lab)
UUID='11111111-1111-4111-8111-111111111111'
SIZE=65536
def services(state_path):
 class Handler(http.server.BaseHTTPRequestHandler):
  protocol_version='HTTP/1.1'
  def log_message(self,*args):pass
  def do_GET(self):
   try:state=json.loads(pathlib.Path(state_path).read_text())
   except Exception:state={'bad':'partial','good':'full'}
   mode=state[self.server.role]
   if self.path.startswith('/app'):
    with pathlib.Path(state_path).with_name('app-hits.log').open('a') as hit:hit.write(self.server.role+' '+self.path+'\n')
   if self.path=='/generate_204':
    self.send_response(204);self.send_header('Content-Length','0');self.end_headers();return
   if self.path in ('/short-error','/short-ok','/short-redirect'):
    self.send_response({'/short-error':503,'/short-ok':200,'/short-redirect':302}[self.path]);self.send_header('Content-Length','3')
    if self.path=='/short-redirect':self.send_header('Location','/payload')
    self.end_headers();self.wfile.write(b'bad');return
   if mode=='slow':time.sleep(3)
   self.send_response(200);self.send_header('Content-Length',str(SIZE));self.send_header('Connection','close');self.end_headers()
   body=(self.server.role.encode()+b'.')*(SIZE//4+1);body=body[:SIZE]
   try:
    if mode=='partial':
     self.wfile.write(body[:32768]);self.wfile.flush();time.sleep(12)
    else:self.wfile.write(body);self.wfile.flush()
   except (BrokenPipeError,ConnectionResetError):pass
   self.close_connection=True
 for port,role in [(24447,'bad'),(24448,'good')]:
  server=http.server.ThreadingHTTPServer(('127.0.0.1',port),Handler);server.daemon_threads=True;server.role=role
  threading.Thread(target=server.serve_forever,daemon=True).start()
 while True:time.sleep(1)
def fixture(profile,baseline=False):
 c=copy.deepcopy(profile)
 c['burstObservatory']['pingConfig']['destination']='http://127.0.0.1:24447/'+('generate_204' if baseline else 'payload')
 c['metrics']={'tag':'metrics','listen':'127.0.0.1:11111'}
 peers=[];exits=[];rules=[]
 for role,port,dest in [('bad',24449,24447),('good',24450,24448)]:
  tag='fixture-'+role
  peers.append({'tag':tag,'listen':'127.0.0.1','port':port,'protocol':'vless','settings':{'decryption':'none','clients':[{'id':UUID}]}})
  exits.append({'tag':tag+'-exit','protocol':'freedom','settings':{'redirect':'127.0.0.1:'+str(dest),'finalRules':[{'action':'allow','network':'tcp','ip':['127.0.0.1/32'],'port':str(dest)}]}})
  rules.append({'type':'field','inboundTag':[tag],'outboundTag':tag+'-exit'})
 c['inbounds']+=peers;c['outbounds']+=exits;c['routing']['rules']=rules+c['routing']['rules']
 return c
def nodes(box):
 r=box.net(['/usr/bin/curl','--noproxy','*','-fsS','--max-time','2','http://127.0.0.1:11111/debug/vars'])
 return json.loads(r.stdout).get('observatory',{})
def eligible(v):
 h=v.get('health_ping',{});return bool(v.get('alive')) and v.get('delay',99999)<10000 and h.get('all',0)>0 and h.get('fail',0)/h['all']<=.15
def wait(box,predicate,limit=85):
 start=time.monotonic();last={}
 while time.monotonic()-start<limit:
  time.sleep(.25)
  try:last=nodes(box)
  except Exception:continue
  if predicate(last):return {'ms':round((time.monotonic()-start)*1000),'nodes':last}
 raise AssertionError(('observation_timeout',last))
def request(box,timeout=2,path='/app'):
 start=time.monotonic()
 r=subprocess.run([lab.IP,'netns','exec',box.name,'/usr/bin/curl','--noproxy','','--proxy','socks5h://127.0.0.1:10808','-sS','--max-time',str(timeout),'--max-filesize','100000','--output','/dev/null','--write-out','%{json}','http://127.0.0.1:24447'+path],capture_output=True,timeout=timeout+3)
 try:v=json.loads(r.stdout)
 except ValueError:v={}
 return {'passed':r.returncode==0 and v.get('http_code')==200 and v.get('size_download')==SIZE,'http_code':v.get('http_code'),'exit':r.returncode,'bytes':v.get('size_download'),'ms':round((time.monotonic()-start)*1000)}
def run(args,baseline):
 profile=json.loads(pathlib.Path(args.baseline if baseline else args.candidate).read_text());out={}
 with lab.Sandbox(args.core,secrets.token_hex(3),args.slot,prefix='cstf') as box:
  box.policy({});state=box.work/'state.json'
  def mode(bad,good):
   tmp=state.with_suffix('.tmp');tmp.write_text(json.dumps({'bad':bad,'good':good}));tmp.replace(state)
  mode('partial','full')
  service=subprocess.Popen([lab.IP,'netns','exec',box.name,sys.executable,__file__,'--services',str(state)],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
  try:
   time.sleep(.2);c=fixture(profile,baseline);proc=box.launch(json.dumps(c).encode())
   if baseline:
    out['initial']=wait(box,lambda ns:len(ns)>=2 and all(x.get('alive') for x in ns.values()),20)
    out['baseline_http']=request(box,path='/generate_204');assert out['baseline_http']['http_code']==204,out
    out['requests']=[request(box) for _ in range(8)]
    assert out['initial']['nodes']['proxy-01']['alive'],'small probe did not reproduce false health'
   else:
    out['initial']=wait(box,lambda ns:eligible(ns.get('proxy-02',{})) and ns.get('proxy-01',{}).get('health_ping',{}).get('fail',0)>0,25)
    out['requests']=[request(box) for _ in range(8)];assert all(x['passed'] for x in out['requests']),out
    mode('partial','partial')
    out['degraded']=wait(box,lambda ns:len(ns)>=2 and not any(eligible(x) for x in ns.values()))
    out['all_failed_request']=request(box,path='/app-block-proof');assert not out['all_failed_request']['passed'],out
    hit=state.with_name('app-hits.log');assert '/app-block-proof' not in hit.read_text(),'all-down traffic reached a fixture origin'
    out['all_down_origin_requests']=0
    mode('full','partial')
    out['recovered']=wait(box,lambda ns:eligible(ns.get('proxy-01',{})) and not eligible(ns.get('proxy-02',{})))
    out['recovered_requests']=[request(box) for _ in range(3)];assert all(x['passed'] for x in out['recovered_requests']),out
    # A slow but fully transferring path belongs to recovery, never to a
    # permissive failed-body fallback. Restart only this owned test instance.
    box.stop();mode('partial','slow');proc=box.launch(json.dumps(c).encode())
    out['slow']=wait(box,lambda ns:eligible(ns.get('proxy-02',{})) and ns['proxy-02'].get('delay',0)>2000 and ns.get('proxy-01',{}).get('health_ping',{}).get('fail',0)>0,25)
    out['slow_request']=request(box,5);assert out['slow_request']['passed'],out
    out['known_false_acceptance']={}
    for path in ['/short-error','/short-ok','/short-redirect']:
     box.stop();short=copy.deepcopy(c);short['burstObservatory']['pingConfig']['destination']='http://127.0.0.1:24447'+path
     proc=box.launch(json.dumps(short).encode())
     out['known_false_acceptance'][path]=wait(box,lambda ns:len(ns)>=2 and all(eligible(x) for x in ns.values()),15)
   assert proc.poll() is None,box.logs().decode(errors='replace')
  finally:
   service.terminate()
   try:service.wait(3)
   except subprocess.TimeoutExpired:service.kill();service.wait()
 return out
def main():
 p=argparse.ArgumentParser();p.add_argument('--core',required=True);p.add_argument('--baseline',required=True);p.add_argument('--candidate',required=True);p.add_argument('--slot',type=int,default=8210);p.add_argument('--output',required=True);a=p.parse_args()
 sha=hashlib.sha256(pathlib.Path(a.core).read_bytes()).hexdigest()
 assert sha in {'e87bce586acdb9136aa5b1f1d9c8db9f1ea1d145b6d86368c7f6f662f9f6a66d','cc4b84d3069569f297a61f04b914f2bb37e9d202fd7b84cb50def1fc30fd40e1'},'unpinned core'
 report={'scope':'closed netns; generated Smart envelopes; synthetic partial/full/recovery network phases; not a carrier test','core_sha256':sha,'results':{}}
 for name,baseline in [('baseline',True),('candidate',False)]:
  report['results'][name]=run(a,baseline);pathlib.Path(a.output).write_text(json.dumps(report,indent=2));print(json.dumps({'phase':name,'done':True}),flush=True)
 report['passed']=True;pathlib.Path(a.output).write_text(json.dumps(report,indent=2));print(json.dumps({'passed':True,'core_sha256':sha}),flush=True)
if __name__=='__main__':
 if len(sys.argv)>1 and sys.argv[1]=='--services':services(sys.argv[2])
 else:main()
