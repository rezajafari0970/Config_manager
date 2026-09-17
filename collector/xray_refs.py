import json
from .validator import issue,WARN
def audit(raw):
 o=json.loads(raw);z=[]
 outs={x.get('tag') for x in o.get('outbounds',[]) if isinstance(x,dict) and x.get('tag')}
 ins={x.get('tag') for x in o.get('inbounds',[]) if isinstance(x,dict) and x.get('tag')}
 r=o.get('routing') if isinstance(o.get('routing'),dict) else {}
 bals={x.get('tag') for x in r.get('balancers',[]) if isinstance(x,dict) and x.get('tag')} if isinstance(r.get('balancers'),list) else set()
 for i,x in enumerate(r.get('rules',[]) if isinstance(r.get('rules'),list) else []):
  if not isinstance(x,dict):continue
  ot=x.get('outboundTag');bt=x.get('balancerTag');it=x.get('inboundTag')
  if ot and ot not in outs and ot!='api':z.append(issue('XRAY_ROUTE_OUTBOUND_REF',f'routing.rules[{i}] references missing outbound {ot}',WARN))
  if bt and bt not in bals:z.append(issue('XRAY_ROUTE_BALANCER_REF',f'routing.rules[{i}] references missing balancer {bt}',WARN))
  if isinstance(it,list):
   for tag in it:
    if tag not in ins and not (tag.startswith('dns-') or tag.startswith('domestic-dns_') or tag=='dns-module' or tag=='api'):z.append(issue('XRAY_ROUTE_INBOUND_REF',f'routing.rules[{i}] references missing inbound {tag}',WARN))
 for i,x in enumerate(o.get('outbounds',[])):
  if not isinstance(x,dict):continue
  ps=x.get('proxySettings')
  if isinstance(ps,dict) and ps.get('tag') and ps['tag'] not in outs:z.append(issue('XRAY_PROXY_REF',f'outbounds[{i}] proxySettings references missing {ps["tag"]}',WARN))
 return z