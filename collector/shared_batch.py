import time,json,concurrent.futures
from .db import connect
from .multi_sandbox import start,stop
from .dynamic_probes import pair
from .health_policy import initial_action
from .socket_traffic import snapshot as traffic_snapshot,verified as traffic_verified
SUPPORTED=('vless','vmess','trojan','ss')
def claim_batch(n):
 c=connect();c.execute('BEGIN IMMEDIATE');rows=c.execute("SELECT * FROM test_candidates t WHERE stage='queued' AND kind IN ('vless','vmess','trojan') AND NOT EXISTS(SELECT 1 FROM health_claims h WHERE h.fingerprint=t.fingerprint) ORDER BY id LIMIT ?",(n,)).fetchall();now=time.time()
 for r in rows:c.execute('INSERT OR IGNORE INTO health_claims(fingerprint,claimed_at) VALUES(?,?)',(r['fingerprint'],now))
 c.commit();c.close();return rows
def finish(row,pr,traffic_ok,attempt=1):
 u=bool(pr.get('upload',{}).get('ok'));d=bool(pr.get('download',{}).get('ok'));action=initial_action(attempt,u,d and traffic_ok);state={'promote_healthy':'healthy','retry_after_30s':'retry','delete':'remove'}[action];now=time.time();c=connect();c.execute('INSERT INTO health_attempts(fingerprint,attempt,phase,upload_ok,download_ok,external_ok,server_traffic_ok,providers,bytes_up,bytes_down,started_at,finished_at,result) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)',(row['fingerprint'],attempt,'shared',int(u),int(d),int(u and d),int(traffic_ok),json.dumps({'probe':pr,'shared':True}),pr.get('upload',{}).get('bytes',0),pr.get('download',{}).get('bytes',0),now,now,state));c.execute('UPDATE test_candidates SET upload_ok=?,download_ok=?,healthy=?,stage=?,updated_at=? WHERE fingerprint=?',(int(u),int(d),int(state=='healthy'),'tested' if state!='retry' else 'retry_wait',now,row['fingerprint']));c.execute('DELETE FROM health_claims WHERE fingerprint=?',(row['fingerprint'],));c.commit();c.close();return state
def run(n=8):
 rows=claim_batch(n)
 if not rows:return {'idle':True}
 h=start(rows)
 if not h.get('ok'):
  c=connect();c.executemany('DELETE FROM health_claims WHERE fingerprint=?',[(r['fingerprint'],) for r in rows]);c.commit();c.close();return {'error':h.get('error')}
 before=traffic_snapshot(h['process'].pid,h['ports']);t=time.time()
 try:
  with concurrent.futures.ThreadPoolExecutor(max_workers=len(rows)) as e:prs=list(e.map(pair,h['ports']))
 finally:
  after=traffic_snapshot(h['process'].pid,h['ports']);stop(h)
 states=[]
 for r,p in zip(rows,prs):
  if p.get('defer'):
   c=connect();c.execute('DELETE FROM health_claims WHERE fingerprint=?',(r['fingerprint'],));c.commit();c.close();states.append('defer')
  else:states.append(finish(r,p,traffic_verified(before,after,h['ports'][len(states)])))
 return {'count':len(rows),'ms':round((time.time()-t)*1000),'states':states}