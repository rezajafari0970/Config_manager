from .workload_gate import state as health_state
from .resources import sample
import json,pathlib,time
def fetch_limit():
 try:
  p=pathlib.Path('/root/Config_manager/data/governor_feedback.json');d=json.loads(p.read_text())
  if time.time()-d['updated']<120:return max(1,min(6,int(d['fetch_limit'])))
 except:pass
 h=health_state();r=sample();return 1 if h['cpu_psi']>55 else (2 if h['queue']>1000 else min(4,r['limit']))
def snapshot():
 h=health_state();r=sample();return {'fetch_limit':fetch_limit(),'health':h,'resources':r}