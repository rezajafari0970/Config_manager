import time
from .db import connect
def sync():
 c=connect();now=time.time();rows=c.execute("SELECT original_fingerprint,candidate_fingerprint,candidate_kind,candidate_raw FROM recovery_registry WHERE recoverable=1 AND validator_ok=1 AND xray_ok=1 AND singbox_ok=1").fetchall();n=0
 for r in rows:
  c.execute("INSERT INTO recovered_candidates(original_fingerprint,candidate_fingerprint,kind,raw,source,status,first_seen,last_seen) VALUES(?,?,?,?,?,'structurally_verified',?,?) ON CONFLICT(original_fingerprint) DO UPDATE SET candidate_fingerprint=excluded.candidate_fingerprint,kind=excluded.kind,raw=excluded.raw,status='structurally_verified',last_seen=excluded.last_seen",(r['original_fingerprint'],r['candidate_fingerprint'],r['candidate_kind'],r['candidate_raw'],'ss-recovery',now,now));n+=1
 c.commit();total=c.execute('SELECT COUNT(*) FROM recovered_candidates').fetchone()[0];c.close();return {'synced':n,'total':total,'status':'structurally_verified'}
if __name__=='__main__':print(sync())