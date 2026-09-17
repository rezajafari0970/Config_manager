import time,json,pathlib
from .db import connect
from .system_pressure import snapshot as pressure
from .lane_budget import snapshot as lanes
P=pathlib.Path('/root/Config_manager/data/benchmark1000-v3.json')
def run(fps):
 start=time.time();samples=[];marks=','.join('?'*len(fps));peak=0
 while time.time()-start<305:
  time.sleep(5);c=connect();done=c.execute(f"select count(*) from test_candidates where fingerprint in ({marks}) and stage='tested'",fps).fetchone()[0];q=dict(c.execute("select coalesce(lane,'fast'),count(*) from test_candidates where stage in ('queued','retry_wait') group by coalesce(lane,'fast')").fetchall());c.close();psi=pressure()['psi']['cpu'].get('avg10',0);peak=max(peak,psi);samples.append({'sec':round(time.time()-start),'done':done,'queue_lanes':q,'lane_budget':lanes()['limits'],'psi':psi})
  if done>=len(fps):break
 sec=time.time()-start;out={'target':len(fps),'unique_done':done,'seconds':round(sec,1),'unique_per_min':round(done/sec*60,1),'goal_met':done>=1000 and sec<=300,'peak_cpu_psi':peak,'samples':samples};P.write_text(json.dumps(out));return out
if __name__=='__main__':
 c=connect();fps=[r[0] for r in c.execute("select fingerprint from test_candidates where stage='queued' order by id limit 1000")];c.close();print(json.dumps(run(fps)))