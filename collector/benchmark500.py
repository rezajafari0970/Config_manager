import time,json,pathlib
from .db import connect
from .resource_scheduler import snapshot
P=pathlib.Path('/root/Config_manager/data/benchmark500.json')
def counts():
 c=connect();a=c.execute('SELECT COUNT(*) FROM health_attempts').fetchone()[0];q=c.execute("SELECT COUNT(*) FROM test_candidates WHERE stage='queued'").fetchone()[0];h=c.execute('SELECT COUNT(*) FROM healthy_configs').fetchone()[0];c.close();return a,q,h
def run():
 a0,q0,h0=counts();start=time.time();peak=0;samples=[]
 while True:
  time.sleep(5);a,q,h=counts();sys=snapshot();peak=max(peak,sys['load_ratio']);samples.append({'t':round(time.time()-start),'done':a-a0,'load':round(sys['load_ratio'],2),'ram':round(sys['mem_free_ratio'],2)})
  if a-a0>=500 or time.time()-start>=300:break
 elapsed=time.time()-start;a,q,h=counts();r={'target':500,'completed':a-a0,'seconds':round(elapsed,1),'tpm':round((a-a0)/(elapsed/60),1),'goal_met':a-a0>=500 and elapsed<=240,'queue_delta':q0-q,'healthy_delta':h-h0,'peak_load_ratio':round(peak,2),'samples':samples};P.write_text(json.dumps(r));return r
if __name__=='__main__':print(json.dumps(run()))