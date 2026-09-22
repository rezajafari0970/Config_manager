import json,pathlib,os
P=pathlib.Path(os.environ.get('CONFIG_MANAGER_HEALTH_SETTINGS','/root/Config_manager/data/health_settings.json'))
DEFAULT={'enabled':True,'retry_seconds':30,'recheck_seconds':300,'upload_bytes':32768,'download_kb':64,'max_concurrency':8,'min_active_providers':2,'probe_timeout':4,'connect_timeout':2,'startup_timeout_ms':1200,'batch_multiplier':3,'config_lifetime_seconds':7200}
def load():
 try:return {**DEFAULT,**json.loads(P.read_text())}
 except:return DEFAULT.copy()
def save(x):
 d=load()
 for k in DEFAULT:
  if k in x:d[k]=bool(x[k]) if k=='enabled' else int(x[k])
 d['config_lifetime_seconds']=max(300,min(d['config_lifetime_seconds'],604800));d['retry_seconds']=max(10,min(d['retry_seconds'],600));d['recheck_seconds']=max(60,min(d['recheck_seconds'],86400));d['upload_bytes']=max(1024,min(d['upload_bytes'],1048576));d['download_kb']=max(1,min(d['download_kb'],10240));d['max_concurrency']=max(1,min(d['max_concurrency'],16));d['min_active_providers']=max(2,min(d['min_active_providers'],10));d['probe_timeout']=max(2,min(d['probe_timeout'],10));d['connect_timeout']=max(1,min(d['connect_timeout'],5));d['startup_timeout_ms']=max(500,min(d['startup_timeout_ms'],3000));d['batch_multiplier']=max(1,min(d['batch_multiplier'],6));P.write_text(json.dumps(d));return d