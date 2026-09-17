import os,concurrent.futures,json
from .health_worker import step
from .health_settings import load
from .resource_scheduler import health_concurrency,save_metrics
def capacity():return health_concurrency(load()['max_concurrency'])
def run_batch():
 if not load()['enabled']:return {'disabled':True}
 n=capacity();workers=min(8,max(4,n+2))
 jobs=workers*load()['batch_multiplier']
 with concurrent.futures.ThreadPoolExecutor(max_workers=workers) as e:r=list(e.map(lambda _:step(),range(jobs)))
 save_metrics({'health_concurrency':n,'pipeline_workers':workers,'jobs':len(r)});return {'concurrency':n,'pipeline_workers':workers,'results':r}
if __name__=='__main__':print(json.dumps(run_batch()))