import json
from .validator import issue,WARN
def audit(raw):
 o=json.loads(raw);z=[]
 outs={x.get('tag') for x in o.get('outbounds',[]) if isinstance(x,dict) and x.get('tag')}
 ins={x.get('tag') for x in o.get('inbounds',[]) if isinstance(x,dict) and x.get('tag')}
 for i,x in enumerate(o.get('outbounds',[])):
  if not isinstance(x,dict):continue
  d=x.get('detour')
  if d and d not in outs:z.append(issue('SINGBOX_DETOUR_REF',f'outbounds[{i}] detour references missing {d}',WARN))
 route=o.get('route') if isinstance(o.get('route'),dict) else {}
 rules=route.get('rules',[]) if isinstance(route.get('rules'),list) else []
 for i,r in enumerate(rules):
  if not isinstance(r,dict):continue
  out=r.get('outbound')
  if isinstance(out,str) and out not in outs:z.append(issue('SINGBOX_ROUTE_OUTBOUND_REF',f'route.rules[{i}] missing outbound {out}',WARN))
  inbound=r.get('inbound')
  vals=inbound if isinstance(inbound,list) else ([inbound] if isinstance(inbound,str) else [])
  for tag in vals:
   if tag not in ins:z.append(issue('SINGBOX_ROUTE_INBOUND_REF',f'route.rules[{i}] missing inbound {tag}',WARN))
 dns=o.get('dns') if isinstance(o.get('dns'),dict) else {}
 servers=dns.get('servers',[]) if isinstance(dns.get('servers'),list) else []
 stags={x.get('tag') for x in servers if isinstance(x,dict) and x.get('tag')}
 for i,r in enumerate(dns.get('rules',[]) if isinstance(dns.get('rules'),list) else []):
  if isinstance(r,dict) and r.get('server') and r['server'] not in stags:z.append(issue('SINGBOX_DNS_SERVER_REF',f'dns.rules[{i}] missing server {r["server"]}',WARN))
 return z