import json,pathlib,time
from .canary_stats import analyze
from .system_pressure import snapshot as pressure
A=pathlib.Path('/root/Config_manager/data/canary_analysis.json');S=pathlib.Path('/root/Config_manager/data/canary_promotion.json');T=pathlib.Path('/root/Config_manager/data/stage_tuning.json')
def step():
 analyze();d=json.loads(A.read_text()).get('groups',{});pairs=[]
 for k,b in d.items():
  if not k.startswith('baseline:'):continue
  c=d.get('canary:'+k.split(':',1)[1]);
  if not c or b['n']<30 or c['n']<15:continue
  good=c['p50']<=b['p50']*.95 and c['p90']<=b['p90']*.98 and c['success']>=b['success']-.02;pairs.append(good)
 psi=pressure()['psi']['cpu'].get('avg10',0)
 try:s=json.loads(S.read_text())
 except:s={'wins':0,'losses':0,'promoted':False}
 if pairs and all(pairs) and psi<55:s['wins']+=1
 elif pairs:s['losses']+=1
 promote=len(pairs)>=1 and s['wins']>=3 and s['wins']>=s['losses']+2 and psi<55
 s.update({'matched_families':len(pairs),'cpu_psi':psi,'eligible':promote,'updated':time.time()});S.write_text(json.dumps(s));return s
if __name__=='__main__':print(json.dumps(step()))