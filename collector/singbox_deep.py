import json
from .validator import issue,WARN
def audit(raw):
 o=json.loads(raw);z=[];outs=o.get('outbounds',[]);tags=set()
 for i,x in enumerate(outs if isinstance(outs,list) else []):
  if not isinstance(x,dict):continue
  typ=x.get('type');tag=x.get('tag')
  if tag and tag in tags:z.append(issue('SINGBOX_DUPLICATE_TAG',f'duplicate outbound tag {tag}',WARN))
  if tag:tags.add(tag)
  if typ in ('vless','vmess','trojan','shadowsocks','hysteria','hysteria2','tuic'):
   if not str(x.get('server','')).strip():z.append(issue('SINGBOX_SERVER_MISSING',f'outbounds[{i}] server missing',WARN))
   try:p=int(x.get('server_port',0));ok=1<=p<=65535
   except:ok=False
   if not ok:z.append(issue('SINGBOX_PORT_INVALID',f'outbounds[{i}] server_port invalid',WARN))
  tls=x.get('tls')
  if tls is not None and not isinstance(tls,dict):z.append(issue('SINGBOX_TLS_SHAPE',f'outbounds[{i}].tls is not object',WARN))
  tr=x.get('transport')
  if tr is not None and not isinstance(tr,dict):z.append(issue('SINGBOX_TRANSPORT_SHAPE',f'outbounds[{i}].transport is not object',WARN))
 route=o.get('route')
 if route is not None and not isinstance(route,dict):z.append(issue('SINGBOX_ROUTE_SHAPE','route is not object',WARN))
 dns=o.get('dns')
 if dns is not None and not isinstance(dns,dict):z.append(issue('SINGBOX_DNS_SHAPE','dns is not object',WARN))
 return z