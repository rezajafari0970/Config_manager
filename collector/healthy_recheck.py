import time,json
from .db import connect
from .network_sandbox import start,stop
from .dynamic_probes import pair
from .traffic_verify import snapshot,verified
def one():
 c=connect();r=c.execute('SELECT * FROM healthy_configs WHERE next_check<=? ORDER BY next_check LIMIT 1',(time.time(),)).fetchone();c.close()
 if not r:return {'idle':True}
 h=start(r['kind'],r['raw']);ok=False;pr={};traffic=False
 if h.get('ok'):
  try:b=snapshot(h);pr=pair(h['port']);a=snapshot(h);traffic=verified(b,a);ok=traffic and pr.get('download',{}).get('ok',False) and pr.get('upload',{}).get('ok',False)
  finally:stop(h)
 c=connect()
 if ok:c.execute('UPDATE healthy_configs SET upload_ok=1,download_ok=1,last_healthy=?,next_check=? WHERE fingerprint=?',(time.time(),time.time()+300,r['fingerprint']))
 else:c.execute('DELETE FROM healthy_configs WHERE fingerprint=?',(r['fingerprint'],))
 c.commit();c.close();return {'fingerprint':r['fingerprint'],'keep':bool(ok),'traffic':traffic,'probe':pr}
if __name__=='__main__':print(json.dumps(one()))