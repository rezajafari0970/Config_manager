import sqlite3,collections,time
from collector.validator import validate,hard_errors
DB='data/collector.db'; c=sqlite3.connect(DB); c.row_factory=sqlite3.Row
rows=c.execute('select id,fingerprint,kind,raw from configs').fetchall(); kinds=collections.Counter(); hard=collections.Counter(); warn=collections.Counter(); raw_changed=0
for r in rows:
 kinds[r['kind']]+=1; z=validate(r['kind'],r['raw'])
 for x in hard_errors(z): hard[(r['kind'],x['code'])]+=1
 for x in z:
  if x['level']=='warning': warn[(r['kind'],x['code'])]+=1
print('TOTAL',len(rows));print('KINDS',dict(kinds));print('HARD',dict(hard));print('WARN',dict(warn));print('QUARANTINE',c.execute('select count(*) from quarantine').fetchone()[0]);print('SOURCES',c.execute('select status,count(*) from sources group by status').fetchall())
