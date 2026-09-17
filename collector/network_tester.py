import json,time
from .db import connect
from .network_sandbox import start,stop
from .probes import download,upload
from .health_policy import decide
def test_one(row,attempt=1):
 t=time.time();h=start(row['kind'],row['raw']);d=u=False;details={'sandbox':h.get('ok',False)}
 if h.get('ok'):
  try:d,dd=download(h['port']);u,uu=upload(h['port']);details.update(download=dd,upload=uu)
  finally:stop(h)
 state=decide(u,d,attempt);c=connect();c.execute('INSERT INTO health_attempts(fingerprint,attempt,upload_ok,download_ok,traffic_ok,providers,started_at,finished_at) VALUES(?,?,?,?,?,?,?,?)',(row['fingerprint'],attempt,int(u),int(d),int(u and d),json.dumps(details),t,time.time()));c.execute('UPDATE test_candidates SET upload_ok=?,download_ok=?,healthy=?,stage=?,updated_at=? WHERE fingerprint=?',(int(u),int(d),int(state=='healthy'),'tested' if state!='retry' else 'retry_wait',time.time(),row['fingerprint']));c.commit();c.close();return state,details
def next_one():
 c=connect();r=c.execute("SELECT * FROM test_candidates WHERE stage='queued' ORDER BY id LIMIT 1").fetchone();c.close();return r
if __name__=='__main__':
 r=next_one();print(test_one(r) if r else 'empty')