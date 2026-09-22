import json,time
from .db import connect
from .network_sandbox import start,stop
from .dynamic_probes import pair
from .health_policy import initial_action
from .traffic_verify import snapshot,verified
from .profiler import record
from .stage_metrics import enter,leave
from .stage_limits import startup_enter,startup_leave,probe_enter,probe_leave
from .canary_route import group as canary_group,limits as canary_limits,record as canary_record
from .canary_stats import add as canary_sample
from .protocol_timeout import get as protocol_timeout
from .stage_profiler import add as sp_add
from .enrichment import enrich
from .network_intel import classify
def test_one(row,attempt=1):
 t=time.time();cg=canary_group(row['fingerprint']);cl=canary_limits(cg);x=time.time();w=time.time();startup_enter(cl[0]);sp_add('startup_wait_ms',(time.time()-w)*1000);enter('startup')
 try:h=start(row['kind'],row['raw'])
 finally:leave('startup');startup_leave()
 sandbox_ms=(time.time()-x)*1000;sp_add('startup_exec_ms',sandbox_ms);d=u=False;details={'sandbox':h.get('ok',False)}
 if h.get('ok'):
  try:
   before=snapshot(h);x=time.time();w=time.time();probe_enter(cl[1]);sp_add('probe_wait_ms',(time.time()-w)*1000)
   try:pr=pair(h['port'],protocol_timeout(row['kind'],row['raw']))
   finally:probe_leave()
   probe_ms=(time.time()-x)*1000;sp_add('probe_exec_ms',probe_ms);after=snapshot(h)
   if pr.get('defer'):return 'defer',{'sandbox':True,'probe':pr}
   traffic=verified(before,after);d=bool(pr.get('download',{}).get('ok'));u=bool(pr.get('upload',{}).get('ok'));details.update(probe=pr,download=[pr.get('download',{})],upload=[pr.get('upload',{})],traffic_verified=traffic,io_delta=after['io']-before['io']);d=d and traffic;u=u and traffic
   if d and u:
    meta=enrich(h['port']);ni=classify(row['fingerprint'],meta);meta['network_intelligence']=ni;meta['datacenter']=ni['facility'];meta['cdn_state']='cdn' if ni['cdn'] else ('hosting' if ni['hosting'] else 'non-cdn');details['enrichment_meta']=meta
  finally:stop(h)
 probe_ms=locals().get('probe_ms',0);action=initial_action(attempt,u,d);state={'promote_healthy':'healthy','retry_after_30s':'retry','delete':'remove'}[action];c=connect();c.execute('INSERT INTO health_attempts(fingerprint,attempt,phase,upload_ok,download_ok,external_ok,server_traffic_ok,providers,bytes_up,bytes_down,started_at,finished_at,result) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)',(row['fingerprint'],attempt,'initial',int(u),int(d),int(u and d),int(details.get('traffic_verified',False)),json.dumps(details),sum(x.get('sent_bytes',0) for x in details.get('upload',[])),sum(x.get('bytes',0) for x in details.get('download',[])),t,time.time(),state));c.execute('UPDATE test_candidates SET upload_ok=?,download_ok=?,healthy=?,stage=?,updated_at=? WHERE fingerprint=?',(int(u),int(d),int(state=='healthy'),'tested' if state!='retry' else 'retry_wait',time.time(),row['fingerprint']));c.commit();c.close();db_ms=(time.time()-t)*1000-sandbox_ms-probe_ms;total=(time.time()-t)*1000;record(row['fingerprint'],row['kind'],sandbox_ms,probe_ms,max(0,db_ms),0,total,state);canary_record(cg,total,state=='healthy');canary_sample(cg,row['kind'],row['raw'],total,state=='healthy');return state,details
def next_one():
 c=connect();r=c.execute("SELECT * FROM test_candidates WHERE stage='queued' ORDER BY id LIMIT 1").fetchone();c.close();return r
if __name__=='__main__':
 r=next_one();print(test_one(r) if r else 'empty')