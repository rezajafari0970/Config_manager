import subprocess,time,os
DOWNLOADS=[('cloudflare','https://speed.cloudflare.com/__down?bytes=65536'),('google','https://www.google.com/generate_204'),('microsoft','https://www.msftconnecttest.com/connecttest.txt')]
UPLOADS=[('cloudflare','https://speed.cloudflare.com/__up')]
def curl(port,args,timeout=8):
 cmd=['/usr/bin/curl','--silent','--show-error','--fail','--max-time',str(timeout),'--socks5-hostname',f'127.0.0.1:{port}']+args;t=time.time();p=subprocess.run(cmd,stdout=subprocess.PIPE,stderr=subprocess.PIPE);return p.returncode==0,len(p.stdout),round((time.time()-t)*1000),p.stderr.decode(errors='ignore')[-160:]
def download(port):
 out=[]
 for name,url in DOWNLOADS:
  ok,n,ms,e=curl(port,[url]);out.append({'provider':name,'ok':ok,'bytes':n,'ms':ms,'error':e})
  if ok and n>0:return True,out
 return False,out
def upload(port):
 payload=os.urandom(32768);out=[]
 for name,url in UPLOADS:
  ok,n,ms,e=curl(port,['-X','POST','--data-binary','@-',url],timeout=8) if False else (False,0,0,'')
  p=subprocess.run(['/usr/bin/curl','--silent','--show-error','--fail','--max-time','8','--socks5-hostname',f'127.0.0.1:{port}','-X','POST','--data-binary',f'@/dev/stdin',url],input=payload,stdout=subprocess.PIPE,stderr=subprocess.PIPE);ok=p.returncode==0;out.append({'provider':name,'ok':ok,'sent_bytes':len(payload),'response_bytes':len(p.stdout),'error':p.stderr.decode(errors='ignore')[-160:]})
  if ok:return True,out
 return False,out