import json,tempfile,subprocess,time,os,socket
from .network_sandbox import build,free_port
def build_multi(rows):
 out=[];ins=[];ports=[]
 for i,r in enumerate(rows):
  p=free_port();o=build(r['kind'],r['raw'],p);proxy=o['outbounds'][0];tag=f'p{i}';proxy['tag']=tag;out.append(proxy);ins.append({'listen':'127.0.0.1','port':p,'protocol':'socks','tag':f'i{i}','settings':{'udp':True}});ports.append(p)
 routing={'rules':[{'type':'field','inboundTag':[f'i{i}'],'outboundTag':f'p{i}'} for i in range(len(rows))]};return {'log':{'loglevel':'none'},'inbounds':ins,'outbounds':out,'routing':routing},ports
def start(rows):
 try:o,ports=build_multi(rows);f=tempfile.NamedTemporaryFile('w',suffix='.json',delete=False);json.dump(o,f);f.close()
 except Exception as e:return {'ok':False,'error':'build:'+type(e).__name__}
 p=subprocess.Popen(['/usr/local/bin/xray','run','-c',f.name],stdout=subprocess.DEVNULL,stderr=subprocess.PIPE,text=True,start_new_session=True);deadline=time.time()+2
 while time.time()<deadline:
  if p.poll() is not None:break
  ok=0
  for port in ports:
   s=socket.socket();s.settimeout(.03)
   try:s.connect(('127.0.0.1',port));ok+=1
   except:pass
   s.close()
  if ok==len(ports):return {'ok':True,'process':p,'file':f.name,'ports':ports}
  time.sleep(.03)
 err=p.stderr.read()[-500:] if p.poll() is not None else 'timeout';p.kill();os.unlink(f.name);return {'ok':False,'error':err}
def stop(h):
 p=h.get('process');
 if p and p.poll() is None:p.terminate()
 try:p.wait(timeout=1)
 except:
  if p:p.kill()
 try:os.unlink(h.get('file',''))
 except:pass