import json,time,pathlib
from .db import connect
from .workload_gate import state as health
from .resources import sample
P=pathlib.Path('/root/Config_manager/data/governor_feedback.json')
def step():
 now=time.time();c=connect();q=c.execute("SELECT COUNT(*) FROM test_candidates WHERE stage='queued'").fetchone()[0];a=c.execute('SELECT COUNT(*) FROM health_attempts WHERE finished_at>=?',(now-60,)).fetchone()[0];c.close();h=health();r=sample()
 try:s=json.loads(P.read_text())
 except:s={'fetch_limit':2,'last_queue':q,'best_health_tpm':0}
 drain=(s.get('last_queue',q)-q);cur=s.get('fetch_limit',2);best=max(a,s.get('best_health_tpm',0)*.98);target=cur
 if h['cpu_psi']>55 or r['load']>.92:target=max(1,cur-1)
 elif q>300 and a<180 and drain<100:target=max(1,cur-1)
 elif h['cpu_psi']<25 and r['load']<.65 and (q<300 or a>=180):target=min(6,cur+1)
 out={'fetch_limit':target,'health_tpm':a,'queue':q,'queue_drain_per_min':drain,'best_health_tpm':round(best,1),'cpu_psi':h['cpu_psi'],'load':r['load'],'updated':now};P.write_text(json.dumps(out));return out
if __name__=='__main__':print(json.dumps(step()))