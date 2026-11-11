#!/usr/bin/env python3
"""Exact-IP Proxycheck v3 driver with durable free-tier budget and shared cache.
No credentials/configurations are sent. Missing service/classification is UNKNOWN.
"""
import datetime,fcntl,hashlib,ipaddress,json,os,pathlib,time,urllib.request
PROVIDER='proxycheck-v3'
class NoRedirect(urllib.request.HTTPRedirectHandler):
 def redirect_request(self,*args,**kwargs):return None
def parse(ip,body,at):
 obj=json.loads(body)
 if obj.get('status') not in ('ok','warning'):raise ValueError('provider_unavailable')
 item=obj.get(ip)
 if not isinstance(item,dict):raise ValueError('exact_ip_missing')
 net=item.get('network',{});loc=item.get('location',{});kind=net.get('type')
 if kind not in ('Residential','Wireless','Business','Hosting'):kind='Unknown'
 return dict(ip=ip,provider=PROVIDER,class_=kind,asn=str(net.get('asn') or ''),country=str(loc.get('country_code') or loc.get('code') or ''),at=at,expires=at+3600000,digest=hashlib.sha256(body).hexdigest(),code='explicit_network_type')
def classify(ips,path='/var/lib/config-repair-egress/intel.json'):
 p=pathlib.Path(path);p.parent.mkdir(mode=0o700,parents=True,exist_ok=True)
 with open(str(p)+'.lock','a') as lock:
  os.chmod(lock.name,0o600);fcntl.flock(lock,fcntl.LOCK_EX)
  try:state=json.loads(p.read_text())
  except (OSError,ValueError):state={'day':'','used':0,'entries':{}}
  day=datetime.datetime.now(datetime.timezone.utc).date().isoformat()
  if state['day']!=day:state.update(day=day,used=0)
  now=int(time.time()*1000);out=[]
  for raw in sorted(set(ips)):
   try:ip=ipaddress.ip_address(raw)
   except ValueError:continue
   if not ip.is_global or ip.is_multicast or ip.is_reserved:continue
   key=str(ip);cached=state['entries'].get(key)
   if cached and cached['expires']>now:out.append(cached);continue
   if state['used']>=90:continue # leave ten queries for operator diagnosis
   state['used']+=1
   # Commit reservation before request; a crash cannot reset the daily budget.
   def save():
    state['entries']={k:v for k,v in state['entries'].items() if v['expires']>now}
    tmp=p.with_suffix('.tmp');tmp.write_text(json.dumps(state));os.chmod(tmp,0o600);os.replace(tmp,p)
   save()
   try:
    req=urllib.request.Request('https://proxycheck.io/v3/'+key+'?ver=24-June-2026',headers={'User-Agent':'Shen-Egress-Audit/2','Accept':'application/json'})
    with urllib.request.build_opener(NoRedirect(),urllib.request.ProxyHandler({})).open(req,timeout=6) as response:
     body=response.read(131073)
     if len(body)>131072:raise ValueError('provider_body_limit')
    fact=parse(key,body,now);fact['class']=fact.pop('class_');state['entries'][key]=fact;out.append(fact)
   except Exception:
    state['entries'][key]=dict(ip=key,provider=PROVIDER,**{'class':'Unknown'},asn='',country='',at=now,expires=now+300000,digest='',code='provider_unavailable')
   save()
  return out
