import os,random,subprocess,time
from .provider_pool import active
def run(port,url,upload=False):
 cmd=['/usr/bin/curl','-fsS','--max-time','8','--socks5-hostname',f'127.0.0.1:{port}'];payload=None
 if upload:payload=os.urandom(32768);cmd+=['-X','POST','--data-binary','@-']
 else:url+=('?ckSize=64' if '?' not in url else '&ckSize=64')
 t=time.time();p=subprocess.run(cmd+[url],input=payload,stdout=subprocess.PIPE,stderr=subprocess.PIPE);return {'ok':p.returncode==0,'ms':round((time.time()-t)*1000),'bytes':len(payload) if upload else len(p.stdout),'error':p.stderr.decode(errors='ignore')[-120:]}
def pair(port):
 items=active();random.shuffle(items);down=items[0] if items else None;up=next((x for x in items[1:] if x['base']!=down['base']),None) if down else None
 if not down or not up:return {'ok':False,'reason':'insufficient-independent-providers','active':len(items)}
 d=run(port,down['download'],False);u=run(port,up['upload'],True);return {'ok':bool(d['ok'] and u['ok']),'download':{**d,'provider':down['name'],'base':down['base']},'upload':{**u,'provider':up['name'],'base':up['base']},'active':len(items)}