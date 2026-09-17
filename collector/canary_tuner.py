import json,time,pathlib,statistics
from .db import connect
from .system_pressure import snapshot as pressure
P=pathlib.Path('/root/Config_manager/data/canary_tuning.json');BASE=(1,6);C=[(1,5),(1,7),(2,5)]
def sample():
 c=connect();r=c.execute('SELECT total_ms FROM profile_samples WHERE ts>? AND total_ms>0 ORDER BY ts DESC LIMIT 80',(time.time()-300,)).fetchall();c.close();v=[x[0] for x in r];return round(statistics.median(v),1) if v else 0,len(v)
def step():
 med,n=sample();psi=pressure()['psi']['cpu'].get('avg10',0)
 try:s=json.loads(P.read_text())
 except:s={'candidate':0,'baseline_ms':med or 9999,'wins':0,'losses':0}
 base=s.get('baseline_ms',med or 9999);idx=s.get('candidate',0)%len(C);cand=C[idx];improve=(base-med)/max(1,base) if med else 0;wins=s.get('wins',0);loss=s.get('losses',0)
 if n>=30 and psi<55 and improve>.05:wins+=1
 elif n>=30 and (improve<-.05 or psi>65):loss+=1
 promote=wins>=3 and wins>=loss+2;out={'baseline':list(BASE),'candidate':idx,'candidate_limits':list(cand),'baseline_ms':round(base,1),'observed_median_ms':med,'samples':n,'cpu_psi':psi,'wins':wins,'losses':loss,'promote_ready':promote,'updated':time.time()};P.write_text(json.dumps(out));return out
if __name__=='__main__':print(json.dumps(step()))