import json,time,pathlib,subprocess
STATE=pathlib.Path('/root/Config_manager/data/provider_health.json')
PROVIDERS={'download':[('cloudflare','https://speed.cloudflare.com/__down?bytes=65536'),('google','https://www.google.com/generate_204'),('microsoft','https://www.msftconnecttest.com/connecttest.txt')],'upload':[('cloudflare','https://speed.cloudflare.com/__up')]}
def direct(url,method='GET'):
 cmd=['/usr/bin/curl','-fsS','--max-time','6','-o','/dev/null','-w','%{http_code}',url]
 if method=='POST':cmd[1:1]=['-X','POST','--data-binary','probe=1']
 p=subprocess.run(cmd,stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,text=True);return p.returncode==0 and p.stdout.strip().startswith(('2','3'))
def refresh():
 d={'updated':time.time(),'download':{},'upload':{}}
 for typ,items in PROVIDERS.items():
  for name,url in items:d[typ][name]={'healthy':direct(url,'POST' if typ=='upload' else 'GET'),'checked':time.time()}
 STATE.write_text(json.dumps(d));return d
def load(max_age=120):
 try:
  d=json.loads(STATE.read_text());return d if time.time()-d['updated']<max_age else refresh()
 except:return refresh()