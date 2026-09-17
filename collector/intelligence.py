import time,collections
from .validator import validate,hard_errors
from .repair import repair
ENGINE_VERSION='1.0.0'
def analyze(kind,raw):
 issues=validate(kind,raw); hard=hard_errors(issues); warnings=[x for x in issues if x['level']=='warning']; rep=repair(kind,raw)
 if hard: state,confidence='invalid',0.99
 elif rep: state,confidence='repairable',0.98
 elif warnings: state,confidence='compatible-nonstandard',0.75
 else: state,confidence='valid',0.95
 return {'version':ENGINE_VERSION,'state':state,'confidence':confidence,'issues':issues,'repair':rep,'raw':raw}
def learn_snapshot(conn):
 kinds=collections.Counter(); warnings=collections.Counter(); total=0
 for k,r in conn.execute('SELECT kind,raw FROM configs'):
  total+=1;kinds[k]+=1
  for x in validate(k,r):
   if x['level']=='warning':warnings[(k,x['code'])]+=1
 return {'version':ENGINE_VERSION,'at':time.time(),'total':total,'kinds':dict(kinds),'warnings':{f'{k}:{c}':n for (k,c),n in warnings.items()}}
