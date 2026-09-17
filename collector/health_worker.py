import time,json
from .db import connect
from .network_tester import test_one
from .promote import promote
from .health_settings import load
from .lane_budget import enter as lane_enter,leave as lane_leave
from .lifecycle_trace import event
from .dispatch_metrics import add as dm_add,start as dm_start,finish as dm_finish
def claim(where,args=()):
 c=connect();c.execute('BEGIN IMMEDIATE');r=c.execute('SELECT * FROM test_candidates t WHERE '+where+' AND NOT EXISTS(SELECT 1 FROM health_claims h WHERE h.fingerprint=t.fingerprint) ORDER BY t.updated_at ASC,t.id ASC LIMIT 1',args).fetchone()
 if r:c.execute('INSERT OR IGNORE INTO health_claims(fingerprint,claimed_at) VALUES(?,?)',(r['fingerprint'],time.time()))
 c.commit();c.close();return r
def pick():
 slot=int(time.time()*10)%10
 lane='slow' if slot==0 else ('normal' if slot in (1,2) else 'fast')
 r=claim("t.stage='queued' AND COALESCE(t.lane,'fast')=?",(lane,))
 return r or claim("t.stage='queued'")
def due_retry():return claim("t.stage='retry_wait' AND EXISTS(SELECT 1 FROM health_attempts a WHERE a.fingerprint=t.fingerprint GROUP BY a.fingerprint HAVING MAX(a.finished_at)<=?)",(time.time()-load()['retry_seconds'],))
def attempt_no(fp):
 c=connect();n=c.execute('SELECT COUNT(*) FROM health_attempts WHERE fingerprint=?',(fp,)).fetchone()[0];c.close();return n+1
def step():
 ct=time.time();r=due_retry() or pick();dm_add('claim_ms',(time.time()-ct)*1000)
 if not r:return {'idle':True}
 event(r['fingerprint'],'PICK',stage=r['stage'],lane=r['lane'])
 n=attempt_no(r['fingerprint']);wt=time.time();budget=lane_enter(r['lane'] or 'fast');dm_add('lane_wait_ms',(time.time()-wt)*1000);dm_start();event(r['fingerprint'],'START',attempt=n);started=time.time()
 try:state,details=test_one(r,n)
 finally:lane_leave(budget);dm_finish()
 cost=(time.time()-started)*1000;event(r['fingerprint'],'RESULT',attempt=n,state=state,cost_ms=round(cost))
 if state=='defer':
  c=connect();c.execute('DELETE FROM health_claims WHERE fingerprint=?',(r['fingerprint'],));c.commit();c.close();return {'id':r['id'],'kind':r['kind'],'attempt':n,'state':'defer','reason':details.get('probe',{}).get('reason')}
 if state=='healthy':details['enrichment']=promote(r)
 lane='fast' if cost<1500 else ('normal' if cost<3500 else 'slow')
 c=connect();c.execute('UPDATE test_candidates SET lane=?,cost_ms=? WHERE fingerprint=?',(lane,cost,r['fingerprint']))
 if state=='remove':c.execute("DELETE FROM test_candidates WHERE fingerprint=?",(r['fingerprint'],))
 c.execute('DELETE FROM health_claims WHERE fingerprint=?',(r['fingerprint'],));c.commit();c.close()
 return {'id':r['id'],'kind':r['kind'],'attempt':n,'state':state,'traffic_verified':details.get('traffic_verified',False)}
if __name__=='__main__':print(json.dumps(step()))