import os,concurrent.futures,json
from .health_worker import step
from .health_settings import load
def capacity():
 cfg=load();cpus=os.cpu_count() or 1;loadavg=os.getloadavg()[0];mem={}
 for l in open('/proc/meminfo'):
  k,v=l.split(':',1);mem[k]=int(v.split()[0])
 free=mem.get('MemAvailable',0)/max(mem.get('MemTotal',1),1)
 if free<.15 or loadavg>cpus*.85:return 1
 if free<.30 or loadavg>cpus*.60:return 2
 return min(cfg['max_concurrency'],max(4,cpus*2))
def run_batch():
 if not load()['enabled']:return {'disabled':True}
 n=capacity()
 with concurrent.futures.ThreadPoolExecutor(max_workers=n) as e:r=list(e.map(lambda _:step(),range(n)))
 return {'concurrency':n,'results':r}
if __name__=='__main__':print(json.dumps(run_batch()))