import os,tempfile,subprocess
BIN='/usr/local/libexec/config-manager-sing-box'
def check(raw):
 fn=None
 try:
  with tempfile.NamedTemporaryFile('w',suffix='.json',delete=False) as f:f.write(raw);fn=f.name
  p=subprocess.run(['/usr/bin/timeout','2s',BIN,'check','-c',fn],stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True,timeout=2.5)
  return {'ok':p.returncode==0,'rc':p.returncode,'error':' '.join(p.stdout.split())[-600:]}
 except subprocess.TimeoutExpired:return {'ok':False,'rc':124,'error':'TIMEOUT'}
 finally:
  if fn:
   try:os.unlink(fn)
   except:pass