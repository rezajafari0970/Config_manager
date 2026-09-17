import time
from .db import connect
def sync():
 c=connect();now=time.time();n=0
 for r in c.execute('SELECT fingerprint,kind,raw FROM configs').fetchall():
  c.execute("INSERT INTO test_candidates(fingerprint,kind,raw,origin,stage,created_at,updated_at) VALUES(?,?,?,'healthy-structure-pool','queued',?,?) ON CONFLICT(fingerprint) DO UPDATE SET raw=excluded.raw,kind=excluded.kind,updated_at=excluded.updated_at",(r['fingerprint'],r['kind'],r['raw'],now,now));n+=1
 for r in c.execute("SELECT candidate_fingerprint,kind,raw FROM recovered_candidates WHERE status='structurally_verified'").fetchall():
  c.execute("INSERT INTO test_candidates(fingerprint,kind,raw,origin,stage,created_at,updated_at) VALUES(?,?,?,'recovered','queued',?,?) ON CONFLICT(fingerprint) DO UPDATE SET raw=excluded.raw,kind=excluded.kind,updated_at=excluded.updated_at",(r['candidate_fingerprint'],r['kind'],r['raw'],now,now));n+=1
 c.commit();total=c.execute('SELECT COUNT(*) FROM test_candidates').fetchone()[0];c.close();return {'synced':n,'total':total,'health_rule':'upload_ok AND download_ok'}
def mark_result(fid,upload_ok,download_ok):
 healthy=int(bool(upload_ok) and bool(download_ok));c=connect();c.execute("UPDATE test_candidates SET upload_ok=?,download_ok=?,healthy=?,stage='tested',updated_at=? WHERE fingerprint=?",(int(bool(upload_ok)),int(bool(download_ok)),healthy,time.time(),fid));c.commit();c.close();return healthy
if __name__=='__main__':print(sync())