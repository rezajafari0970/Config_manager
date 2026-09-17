import os,concurrent.futures,json
from .health_worker import step
from .health_settings import load
from .resource_scheduler import health_concurrency,save_metrics
def capacity():return health_concurrency(load()['max_concurrency'])
def run_batch():
 if not load()['enabled']:return {'disabled':True}
 n=capacity()
 with concurrent.futures.ThreadPoolExecutor(max_workers=n) as e:r=list(e.map(lambda _:step(),range(n)))
 save_metrics({'health_concurrency':n,'jobs':len(r)});return {'concurrency':n,'results':r}
if __name__=='__main__':print(json.dumps(run_batch()))