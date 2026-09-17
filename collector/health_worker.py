import time,json
from .db import connect
from .network_tester import test_one
def pick():
 c=connect();r=c.execute("SELECT * FROM test_candidates WHERE stage='queued' ORDER BY id LIMIT 1").fetchone();c.close();return r
def due_retry():
 c=connect();r=c.execute("SELECT t.* FROM test_candidates t JOIN (SELECT fingerprint,MAX(finished_at) f FROM health_attempts GROUP BY fingerprint) h ON h.fingerprint=t.fingerprint WHERE t.stage='retry_wait' AND h.f<=? ORDER BY h.f LIMIT 1",(time.time()-30,)).fetchone();c.close();return r
def attempt_no(fp):
 c=connect();n=c.execute('SELECT COUNT(*) FROM health_attempts WHERE fingerprint=?',(fp,)).fetchone()[0];c.close();return n+1
def step():
 r=due_retry() or pick()
 if not r:return {'idle':True}
 n=attempt_no(r['fingerprint']);state,details=test_one(r,n)
 if state=='remove':
  c=connect();c.execute("DELETE FROM test_candidates WHERE fingerprint=?",(r['fingerprint'],));c.commit();c.close()
 return {'id':r['id'],'kind':r['kind'],'attempt':n,'state':state,'traffic_verified':details.get('traffic_verified',False)}
if __name__=='__main__':print(json.dumps(step()))