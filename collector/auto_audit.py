import time
from .db import connect
from .validator import validate,hard_errors
from .repair import repair
from .learning import discover
from .rules import RULEPACK_VERSION
STATE={'last_run':0.0,'duration_ms':0,'configs':0,'hard':0,'bad_repairs':0,'ok':None,'rulepack':RULEPACK_VERSION}
def run():
 t=time.monotonic(); c=connect(); rows=c.execute('SELECT kind,raw FROM configs').fetchall(); c.close(); hard=bad=0
 for kind,raw in rows:
  if hard_errors(validate(kind,raw)): hard+=1
  x=repair(kind,raw)
  if x and hard_errors(validate(kind,x['raw'])): bad+=1
 d=discover(); STATE.update(last_run=time.time(),duration_ms=int((time.monotonic()-t)*1000),configs=len(rows),hard=hard,bad_repairs=bad,ok=(hard==0 and bad==0),rulepack=RULEPACK_VERSION,discovery=d); return dict(STATE)
