import time
from .db import connect
def record(fp,kind,sandbox_ms=0,probe_ms=0,db_ms=0,enrich_ms=0,total_ms=0,state=''):
 c=connect();c.execute('INSERT INTO profile_samples(ts,fingerprint,kind,sandbox_ms,probe_ms,db_ms,enrich_ms,total_ms,state) VALUES(?,?,?,?,?,?,?,?,?)',(time.time(),fp,kind,sandbox_ms,probe_ms,db_ms,enrich_ms,total_ms,state));c.execute('DELETE FROM profile_samples WHERE ts<?',(time.time()-3600,));c.commit();c.close()
def summary(sec=300):
 c=connect();rows=c.execute('SELECT * FROM profile_samples WHERE ts>=?',(time.time()-sec,)).fetchall();c.close()
 def avg(k):return round(sum(r[k] or 0 for r in rows)/max(1,len(rows)),1)
 vals={k:avg(k) for k in ('sandbox_ms','probe_ms','db_ms','enrich_ms','total_ms')};vals['samples']=len(rows);vals['bottleneck']=max(('sandbox_ms','probe_ms','db_ms','enrich_ms'),key=lambda k:vals[k]);return vals