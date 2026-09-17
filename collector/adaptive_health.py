import os,concurrent.futures,json
from .health_worker import step
def capacity():
 cpus=os.cpu_count() or 1;load=os.getloadavg()[0];mem={}
 for l in open('/proc/meminfo'):
  k,v=l.split(':',1);mem[k]=int(v.split()[0])
 free=mem.get('MemAvailable',0)/max(mem.get('MemTotal',1),1)
 if free<.15 or load>cpus*.85:return 1
 if free<.30 or load>cpus*.60:return 2
 return min(4,max(2,cpus//2))
def run_batch():
 n=capacity()
 with concurrent.futures.ThreadPoolExecutor(max_workers=n) as e:r=list(e.map(lambda _:step(),range(n)))
 return {'concurrency':n,'results':r}
if __name__=='__main__':print(json.dumps(run_batch()))