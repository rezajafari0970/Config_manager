import time,json,pathlib
from .db import connect
from .system_pressure import snapshot as pressure
P=pathlib.Path('/root/Config_manager/data/benchmark1000-v2.json')
def run(fps):
 start=time.time();peak=0;s=[];marks=','.join('?'*len(fps))
 while time.time()-start<305:
  time.sleep(5);c=connect();done=c.execute(f"select count(*) from test_candidates where fingerprint in ({marks}) and stage='tested'",fps).fetchone()[0];retry=c.execute(f"select count(*) from test_candidates where fingerprint in ({marks}) and stage='retry_wait'",fps).fetchone()[0];c.close();psi=pressure()['psi']['cpu'].get('avg10',0);peak=max(peak,psi);s.append({'sec':round(time.time()-start),'unique_done':done,'retry':retry,'psi':psi})
  if done>=len(fps):break
 sec=time.time()-start;out={'target':len(fps),'unique_done':done,'seconds':round(sec,1),'unique_per_min':round(done/sec*60,1),'goal_met':done>=1000 and sec<=300,'peak_cpu_psi':peak,'samples':s};P.write_text(json.dumps(out));return out
if __name__=='__main__':
 c=connect();fps=[r[0] for r in c.execute("select fingerprint from test_candidates where stage='queued' order by id limit 1000")];c.close();print(json.dumps(run(fps)))