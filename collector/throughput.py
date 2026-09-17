import time
from .db import connect
def sample():
 c=connect();now=time.time();a=c.execute('SELECT COUNT(*) FROM health_attempts').fetchone()[0];h=c.execute('SELECT COUNT(*) FROM healthy_configs').fetchone()[0];q=c.execute("SELECT COUNT(*) FROM test_candidates WHERE stage='queued'").fetchone()[0];c.execute('INSERT INTO throughput_samples(ts,attempts,healthy,queued) VALUES(?,?,?,?)',(now,a,h,q));c.execute('DELETE FROM throughput_samples WHERE ts<?',(now-3600,));c.commit();old=c.execute('SELECT * FROM throughput_samples WHERE ts<=? ORDER BY ts DESC LIMIT 1',(now-60,)).fetchone();c.close();mins=max((now-old['ts'])/60,.01) if old else 1;return {'attempts_per_min':round((a-old['attempts'])/mins,1) if old else 0,'healthy':h,'queued':q}
