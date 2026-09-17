import time
from .db import connect
from .rules import RULES,RULEPACK_VERSION
def discover():
 c=connect(); rows=c.execute('SELECT kind,code,COUNT(*) n,MIN(raw) sample FROM warnings GROUP BY kind,code').fetchall(); now=time.time(); made=0
 for kind,code,n,sample in rows:
  if code in RULES: continue
  c.execute('INSERT INTO proposals(kind,code,occurrences,sample_raw,status,first_seen,last_seen) VALUES(?,?,?,?,?,?,?) ON CONFLICT(kind,code) DO UPDATE SET occurrences=excluded.occurrences,last_seen=excluded.last_seen',(kind,code,n,sample[:16000],'candidate',now,now)); made+=1
 c.commit(); total=c.execute('SELECT COUNT(*) FROM proposals').fetchone()[0]; c.close(); return {'rulepack':RULEPACK_VERSION,'new_patterns_seen':made,'proposals':total}
def list_proposals(limit=100):
 c=connect(); rows=[dict(x) for x in c.execute('SELECT * FROM proposals ORDER BY occurrences DESC,last_seen DESC LIMIT ?',(limit,))];c.close();return rows
