import json,pathlib
P=pathlib.Path('/root/Config_manager/data/health_settings.json')
DEFAULT={'enabled':True,'retry_seconds':30,'recheck_seconds':300,'upload_bytes':32768,'download_kb':64,'max_concurrency':8,'min_active_providers':2}
def load():
 try:return {**DEFAULT,**json.loads(P.read_text())}
 except:return DEFAULT.copy()
def save(x):
 d=load()
 for k in DEFAULT:
  if k in x:d[k]=bool(x[k]) if k=='enabled' else int(x[k])
 d['retry_seconds']=max(10,min(d['retry_seconds'],600));d['recheck_seconds']=max(60,min(d['recheck_seconds'],86400));d['upload_bytes']=max(1024,min(d['upload_bytes'],1048576));d['download_kb']=max(1,min(d['download_kb'],10240));d['max_concurrency']=max(1,min(d['max_concurrency'],16));d['min_active_providers']=max(2,min(d['min_active_providers'],10));P.write_text(json.dumps(d));return d