import subprocess,concurrent.futures,time,json,pathlib
from .provider_catalog import load
CACHE=pathlib.Path('/root/Config_manager/data/provider_pool.json')
def check(x):
 def c(args):return subprocess.run(['/usr/bin/curl','-fsS','--max-time','3','-o','/dev/null']+args,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL).returncode==0
 return x if c([x['download']+'?ckSize=64']) and c(['-X','POST','--data-binary','probe=1',x['upload']]) else None
def refresh():
 with concurrent.futures.ThreadPoolExecutor(max_workers=8) as e:items=[x for x in e.map(check,load()) if x]
 CACHE.write_text(json.dumps({'updated':time.time(),'items':items}));return items
def active(max_age=120):
 try:
  d=json.loads(CACHE.read_text());return d['items'] if time.time()-d['updated']<max_age else d['items']
 except:return []