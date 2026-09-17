import json,pathlib,time
from .health_settings import load,save
from .resource_scheduler import snapshot
from .db import connect
P=pathlib.Path('/root/Config_manager/data/autotune.json')
def throughput(sec=90):
 c=connect();now=time.time();n=c.execute('SELECT COUNT(*) FROM health_attempts WHERE finished_at>=?',(now-sec,)).fetchone()[0];c.close();return n/(sec/60)
def step():
 cfg=load();sys=snapshot();t=throughput();
 try:st=json.loads(P.read_text())
 except:st={'best_tpm':0,'last_concurrency':cfg['max_concurrency']}
 cur=cfg['max_concurrency'];target=cur
 if sys['load_ratio']>.82 or sys['mem_free_ratio']<.20:target=max(2,cur-2)
 elif t>=st.get('best_tpm',0)*.97 and sys['load_ratio']<.65 and sys['mem_free_ratio']>.35:target=min(16,cur+1)
 elif t<st.get('best_tpm',0)*.85 and cur>4:target=max(4,cur-1)
 if target!=cur:save({'max_concurrency':target})
 st={'best_tpm':max(t,st.get('best_tpm',0)*.98),'tpm':round(t,1),'last_concurrency':target,'load_ratio':sys['load_ratio'],'mem_free_ratio':sys['mem_free_ratio'],'updated':time.time()};P.write_text(json.dumps(st));return st
if __name__=='__main__':print(json.dumps(step()))