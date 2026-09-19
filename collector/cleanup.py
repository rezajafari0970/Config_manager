import time,json
from .db import connect
from .health_settings import load
def run():
 now=time.time();ttl=load().get('config_lifetime_seconds',7200);cut=now-ttl;c=connect()
 stale=[r[0] for r in c.execute('SELECT fingerprint FROM configs WHERE last_seen<?',(cut,)).fetchall()]
 if stale:
  q=','.join('?'*len(stale));c.execute(f'DELETE FROM healthy_configs WHERE fingerprint IN ({q})',stale);c.execute(f'DELETE FROM health_claims WHERE fingerprint IN ({q})',stale);c.execute(f'DELETE FROM test_candidates WHERE fingerprint IN ({q})',stale);c.execute(f'DELETE FROM configs WHERE fingerprint IN ({q})',stale)
 c.execute('DELETE FROM source_configs WHERE last_seen<?',(cut,));c.execute('DELETE FROM health_claims WHERE claimed_at<?',(now-120,));c.commit();c.close();return {'expired':len(stale),'ttl':ttl}
if __name__=='__main__':print(json.dumps(run()))