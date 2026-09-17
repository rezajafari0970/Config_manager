import time
from .db import connect
from .network_sandbox import start,stop
from .enrichment import enrich
from .network_intel import classify
def promote(row):
 h=start(row['kind'],row['raw']);meta={}
 if h.get('ok'):
  try:meta=enrich(h['port']);meta['network_intelligence']=classify(row['fingerprint'],meta);ni=meta['network_intelligence'];meta['datacenter']=ni['facility'];meta['cdn_state']='cdn' if ni['cdn'] else ('hosting' if ni['hosting'] else 'non-cdn')
  finally:stop(h)
 now=time.time();c=connect();cols=['remark','country_code','country_name','country_flag','city','region','asn','network_org','datacenter','egress_ip','cdn_state'];vals=[meta.get(x) for x in cols]
 c.execute('INSERT INTO healthy_configs(fingerprint,kind,raw,remark,country_code,country_name,country_flag,city,region,asn,network_org,datacenter,egress_ip,cdn_state,upload_ok,download_ok,first_healthy,last_healthy,next_check) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(fingerprint) DO UPDATE SET remark=excluded.remark,country_code=excluded.country_code,country_name=excluded.country_name,country_flag=excluded.country_flag,city=excluded.city,region=excluded.region,asn=excluded.asn,network_org=excluded.network_org,datacenter=excluded.datacenter,egress_ip=excluded.egress_ip,cdn_state=excluded.cdn_state,last_healthy=excluded.last_healthy,next_check=excluded.next_check',(row['fingerprint'],row['kind'],row['raw'],*vals,1,1,now,now,now+300));c.commit();c.close();return meta