import json,time,pathlib
from .db import connect
from .system_pressure import snapshot as pressure
P=pathlib.Path('/root/Config_manager/data/stage_tuning.json');C=[(1,5),(1,6),(1,7),(2,5)]
def step():
 now=time.time();c=connect();tpm=c.execute('SELECT COUNT(*) FROM health_attempts WHERE finished_at>=?',(now-60,)).fetchone()[0];c.close();psi=pressure()['psi']['cpu'].get('avg10',0)
 try:s=json.loads(P.read_text())
 except:s={'index':1,'best_tpm':0,'best':[1,6],'scores':{}}
 cur=C[s.get('index',1)%len(C)];scores=s.get('scores',{});scores[f'{cur[0]}/{cur[1]}']={'tpm':tpm,'psi':psi,'ts':now};best=s.get('best',[1,6]);best_tpm=s.get('best_tpm',0)
 if psi<65 and tpm>best_tpm:best=list(cur);best_tpm=tpm
 idx=(s.get('index',1)+1)%len(C);nxt=C[idx] if psi<70 else tuple(best);out={'startup':nxt[0],'probe':nxt[1],'index':idx,'best':best,'best_tpm':best_tpm,'last_tpm':tpm,'cpu_psi':psi,'scores':scores,'updated':now};P.write_text(json.dumps(out));return out
if __name__=='__main__':print(json.dumps(step()))