import time,json
from .db import connect
from .network_tester import test_one
from .promote import promote
from .health_settings import load
def claim(where,args=()):
 c=connect();c.execute('BEGIN IMMEDIATE');r=c.execute('SELECT * FROM test_candidates t WHERE '+where+' AND NOT EXISTS(SELECT 1 FROM health_claims h WHERE h.fingerprint=t.fingerprint) ORDER BY t.id LIMIT 1',args).fetchone()
 if r:c.execute('INSERT OR IGNORE INTO health_claims(fingerprint,claimed_at) VALUES(?,?)',(r['fingerprint'],time.time()))
 c.commit();c.close();return r
def pick():return claim("t.stage='queued'")
def due_retry():return claim("t.stage='retry_wait' AND EXISTS(SELECT 1 FROM health_attempts a WHERE a.fingerprint=t.fingerprint GROUP BY a.fingerprint HAVING MAX(a.finished_at)<=?)",(time.time()-load()['retry_seconds'],))
def attempt_no(fp):
 c=connect();n=c.execute('SELECT COUNT(*) FROM health_attempts WHERE fingerprint=?',(fp,)).fetchone()[0];c.close();return n+1
def step():
 r=due_retry() or pick()
 if not r:return {'idle':True}
 n=attempt_no(r['fingerprint']);state,details=test_one(r,n)
 if state=='healthy':details['enrichment']=promote(r)
 c=connect()
 if state=='remove':c.execute("DELETE FROM test_candidates WHERE fingerprint=?",(r['fingerprint'],))
 c.execute('DELETE FROM health_claims WHERE fingerprint=?',(r['fingerprint'],));c.commit();c.close()
 return {'id':r['id'],'kind':r['kind'],'attempt':n,'state':state,'traffic_verified':details.get('traffic_verified',False)}
if __name__=='__main__':print(json.dumps(step()))