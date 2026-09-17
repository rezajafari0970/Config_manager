import os,subprocess,time,concurrent.futures
from .provider_pool import active
from .health_settings import load
from .provider_score import rank,record
def run(port,url,upload=False):
 cfg=load();cmd=['/usr/bin/curl','-fsS','--connect-timeout',str(cfg['connect_timeout']),'--max-time',str(cfg['probe_timeout']),'--socks5-hostname',f'127.0.0.1:{port}'];payload=None
 if upload:payload=os.urandom(cfg['upload_bytes']);cmd+=['-X','POST','--data-binary','@-']
 else:url+=('?ckSize='+str(cfg['download_kb']) if '?' not in url else '&ckSize='+str(cfg['download_kb']))
 t=time.time();p=subprocess.run(cmd+[url],input=payload,stdout=subprocess.PIPE,stderr=subprocess.PIPE);ms=round((time.time()-t)*1000);return {'ok':p.returncode==0,'ms':ms,'bytes':len(payload) if upload else len(p.stdout),'error':p.stderr.decode(errors='ignore')[-120:]}
def race(port,items,key,upload=False,width=2):
 with concurrent.futures.ThreadPoolExecutor(max_workers=width) as e:
  fs={e.submit(run,port,x[key],upload):x for x in items[:width]}
  results=[]
  for f in concurrent.futures.as_completed(fs):
   x=fs[f];r=f.result();record(x['base'],r['ok'],r['ms']);results.append((x,r))
   if r['ok']:return x,r,results
 return (results[0][0],results[0][1],results) if results else (None,{'ok':False,'ms':0},[])
def pair(port):
 items=rank(active());cfg=load();bases={x.get('base') for x in items}
 if len(bases)<cfg['min_active_providers']:return {'ok':False,'defer':True,'reason':'insufficient-independent-providers','active':len(items),'independent':len(bases),'required':cfg['min_active_providers']}
 downs=items[:4];d,dr,_=race(port,downs,'download',False,2);ups=[x for x in items if not d or x['base']!=d['base']][:4];u,ur,_=race(port,ups,'upload',True,2)
 if not d or not u:return {'ok':False,'reason':'provider-race-empty','active':len(items)}
 return {'ok':bool(dr['ok'] and ur['ok']),'download':{**dr,'provider':d['name'],'base':d['base']},'upload':{**ur,'provider':u['name'],'base':u['base']},'active':len(items),'raced':True}