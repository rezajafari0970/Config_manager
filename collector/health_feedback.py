import json,time,pathlib
from .db import connect
from .health_settings import load,save
from .system_pressure import snapshot as pressure
P=pathlib.Path('/root/Config_manager/data/health_feedback.json')
def step():
 now=time.time();c=connect();tpm=c.execute('SELECT COUNT(*) FROM health_attempts WHERE finished_at>=?',(now-60,)).fetchone()[0];q=c.execute("SELECT COUNT(*) FROM test_candidates WHERE stage='queued'").fetchone()[0];c.close();p=pressure();psi=p['psi']['cpu'].get('avg10',0);cfg=load()
 try:s=json.loads(P.read_text())
 except:s={'best_tpm':0,'direction':1,'batch_size':8,'lanes':1}
 best=s.get('best_tpm',0);direction=s.get('direction',1);conc=cfg['max_concurrency']
 if psi>60:direction=-1
 elif best and tpm<best*.88:direction=-1
 elif tpm>=best*.97 and psi<45:direction=1
 conc=max(2,min(16,conc+direction));batch=max(4,min(16,s.get('batch_size',8)+(2*direction)));lanes=1 if psi>45 else max(1,min(3,s.get('lanes',1)+(1 if direction>0 and tpm<200 else 0)))
 save({'max_concurrency':conc});out={'target_tpm':200,'tpm':tpm,'queue':q,'cpu_psi':psi,'max_concurrency':conc,'shared_batch_size':batch,'shared_lanes':lanes,'best_tpm':max(tpm,best*.985),'direction':direction,'updated':now};P.write_text(json.dumps(out));return out
if __name__=='__main__':print(json.dumps(step()))