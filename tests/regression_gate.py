import sqlite3,sys
from collector.validator import validate,hard_errors
from collector.repair import repair
c=sqlite3.connect('data/collector.db'); rows=c.execute('select kind,raw from configs').fetchall(); hard=changed=repair_bad=0
for k,r in rows:
 if hard_errors(validate(k,r)): hard+=1
 x=repair(k,r)
 if x:
  if hard_errors(validate(k,x['raw'])): repair_bad+=1
  if x['raw']==r: changed+=1
print({'configs':len(rows),'hard_in_pool':hard,'invalid_repairs':repair_bad,'noop_repairs':changed})
sys.exit(1 if hard or repair_bad or changed else 0)
