from .workload_gate import state as health_state
from .resources import sample
def fetch_limit():
 h=health_state();r=sample();q=h['queue'];psi=h['cpu_psi']
 if psi>55 or q>2500:return 1
 if psi>35 or q>1000:return 2
 if psi>20 or q>400:return 3
 return min(6,r['limit'])
def snapshot():
 h=health_state();r=sample();return {'fetch_limit':fetch_limit(),'health':h,'resources':r}