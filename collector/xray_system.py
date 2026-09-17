import json
from .validator import issue,WARN
def audit(raw):
 o=json.loads(raw);z=[]
 dns=o.get('dns')
 if dns is not None and not isinstance(dns,dict):z.append(issue('XRAY_DNS_SHAPE','dns is not object',WARN))
 if isinstance(dns,dict):
  if 'servers' in dns and not isinstance(dns['servers'],list):z.append(issue('XRAY_DNS_SERVERS_SHAPE','dns.servers is not array',WARN))
  if 'hosts' in dns and not isinstance(dns['hosts'],dict):z.append(issue('XRAY_DNS_HOSTS_SHAPE','dns.hosts is not object',WARN))
 ins=o.get('inbounds')
 if ins is not None and not isinstance(ins,list):z.append(issue('XRAY_INBOUNDS_SHAPE','inbounds is not array',WARN))
 if isinstance(ins,list):
  tags=set()
  for i,x in enumerate(ins):
   if not isinstance(x,dict):continue
   if not x.get('protocol'):z.append(issue('XRAY_INBOUND_PROTOCOL_MISSING',f'inbounds[{i}] protocol missing',WARN))
   tag=x.get('tag')
   if tag and tag in tags:z.append(issue('XRAY_DUPLICATE_INBOUND_TAG',f'duplicate inbound tag {tag}',WARN))
   if tag:tags.add(tag)
 policy=o.get('policy')
 if policy is not None and not isinstance(policy,dict):z.append(issue('XRAY_POLICY_SHAPE','policy is not object',WARN))
 stats=o.get('stats')
 if stats is not None and not isinstance(stats,dict):z.append(issue('XRAY_STATS_SHAPE','stats is not object',WARN))
 api=o.get('api')
 if api is not None and not isinstance(api,dict):z.append(issue('XRAY_API_SHAPE','api is not object',WARN))
 return z