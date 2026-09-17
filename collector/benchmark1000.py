import time,json,pathlib
from .db import connect
from .system_pressure import snapshot as pressure
P=pathlib.Path('/root/Config_manager/data/benchmark1000.json')
def run():
 c=connect();start=time.time();a0=c.execute('select count(*) from health_attempts').fetchone()[0];c.close();peak=0;samples=[]
 while time.time()-start<360:
  time.sleep(5);c=connect();a=c.execute('select count(*) from health_attempts').fetchone()[0];q=c.execute("select count(*) from test_candidates where stage in ('queued','retry_wait')").fetchone()[0];c.close();p=pressure();psi=p['psi']['cpu'].get('avg10',0);peak=max(peak,psi);samples.append({'s':round(time.time()-start),'done':a-a0,'queue':q,'psi':psi})
  if a-a0>=1000:break
 sec=time.time()-start;done=a-a0;out={'target':1000,'done':done,'seconds':round(sec,1),'tpm':round(done/sec*60,1),'goal_met':done>=1000 and sec<=300,'peak_cpu_psi':peak,'samples':samples};P.write_text(json.dumps(out));return out
if __name__=='__main__':print(json.dumps(run()))