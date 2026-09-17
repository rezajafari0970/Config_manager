import os,time,json,pathlib
P=pathlib.Path('/root/Config_manager/data/resource_scheduler.json')
def snapshot():
 cpus=os.cpu_count() or 1;load=os.getloadavg()[0];m={}
 for l in open('/proc/meminfo'):
  k,v=l.split(':',1);m[k]=int(v.split()[0])
 free=m.get('MemAvailable',0)/max(1,m.get('MemTotal',1));return {'cpus':cpus,'load':load,'load_ratio':load/cpus,'mem_free_ratio':free}
def health_concurrency(maxc):
 s=snapshot()
 if s['mem_free_ratio']<.15 or s['load_ratio']>.9:return 1
 if s['mem_free_ratio']<.25 or s['load_ratio']>.75:return min(2,maxc)
 if s['load_ratio']>.55:return min(4,maxc)
 return min(maxc,max(4,s['cpus']*2))
def save_metrics(extra={}):
 x={**snapshot(),**extra,'updated':time.time()};P.write_text(json.dumps(x));return x