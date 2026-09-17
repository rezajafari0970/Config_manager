import os,random,subprocess,time,concurrent.futures
from .provider_pool import active
from .health_settings import load
from .provider_score import rank,record
def run(port,url,upload=False):
 cfg=load();cmd=['/usr/bin/curl','-fsS','--connect-timeout',str(cfg['connect_timeout']),'--max-time',str(cfg['probe_timeout']),'--socks5-hostname',f'127.0.0.1:{port}'];payload=None
 if upload:payload=os.urandom(load()['upload_bytes']);cmd+=['-X','POST','--data-binary','@-']
 else:
  kb=load()['download_kb'];url+=(('?ckSize='+str(kb)) if '?' not in url else ('&ckSize='+str(kb)))
 t=time.time();p=subprocess.run(cmd+[url],input=payload,stdout=subprocess.DEVNULL if upload else subprocess.PIPE,stderr=subprocess.PIPE);ms=round((time.time()-t)*1000);return {'ok':p.returncode==0,'ms':ms,'bytes':len(payload) if upload else len(p.stdout),'error':p.stderr.decode(errors='ignore')[-120:]}
def pair(port):
 items=active();cfg=load();bases={x.get('base') for x in items};
 if len(bases)<cfg['min_active_providers']:return {'ok':False,'defer':True,'reason':'insufficient-independent-providers','active':len(items),'independent':len(bases),'required':cfg['min_active_providers']}
 items=rank(items);down=items[0] if items else None;up=next((x for x in items[1:] if x['base']!=down['base']),None) if down else None
 if not down or not up:return {'ok':False,'reason':'insufficient-independent-providers','active':len(items)}
 with concurrent.futures.ThreadPoolExecutor(max_workers=2) as e:
  fd=e.submit(run,port,down['download'],False);fu=e.submit(run,port,up['upload'],True);d=fd.result();u=fu.result()
 record(down['base'],d['ok'],d['ms']);record(up['base'],u['ok'],u['ms']);return {'ok':bool(d['ok'] and u['ok']),'download':{**d,'provider':down['name'],'base':down['base']},'upload':{**u,'provider':up['name'],'base':up['base']},'active':len(items)}