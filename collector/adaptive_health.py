import concurrent.futures,json,time
from .health_worker import step
from .health_settings import load
from .resource_scheduler import health_concurrency,save_metrics
from .db import connect
def capacity():return health_concurrency(load()['max_concurrency'])
def backlog():
 c=connect();n=c.execute("select count(*) from test_candidates where stage in ('queued','retry_wait')").fetchone()[0];c.close();return n
def run_batch():
 if not load()['enabled']:return {'disabled':True}
 n=capacity();q=backlog();workers=min(12,max(4,n+4 if q>100 else n+2));jobs=min(max(workers,workers*load()['batch_multiplier']),max(workers,q))
 if q==0:return {'concurrency':n,'pipeline_workers':0,'results':[],'idle':True}
 with concurrent.futures.ThreadPoolExecutor(max_workers=workers) as e:r=list(e.map(lambda _:step(),range(jobs)))
 save_metrics({'health_concurrency':n,'pipeline_workers':workers,'jobs':len(r),'backlog':q});return {'concurrency':n,'pipeline_workers':workers,'results':r}
if __name__=='__main__':print(json.dumps(run_batch()))