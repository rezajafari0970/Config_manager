#!/usr/bin/env python3
"""Exact-IP Proxycheck v3 driver with durable free-tier budget and shared cache.
No credentials/configurations are sent. Missing service/classification is UNKNOWN.
"""
import concurrent.futures,contextlib,datetime,fcntl,hashlib,ipaddress,json,os,pathlib,time,urllib.request
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
DEFAULT_PATH='/var/lib/config-repair-egress/intel.json'
DAILY_LIMIT=90
@contextlib.contextmanager
def transaction(path):
 p=pathlib.Path(path);p.parent.mkdir(mode=0o700,parents=True,exist_ok=True)
 with open(str(p)+'.lock','a') as lock:
  os.chmod(lock.name,0o600);fcntl.flock(lock,fcntl.LOCK_EX)
  try:state=json.loads(p.read_text())
  except (OSError,ValueError):state={'day':'','used':0,'entries':{}}
  day=datetime.datetime.now(datetime.timezone.utc).date().isoformat()
  if state.get('day')!=day:state.update(day=day,used=0)
  yield state
  now=int(time.time()*1000)
  state['entries']={k:v for k,v in state['entries'].items() if v['expires']>now}
  tmp=p.with_suffix('.tmp');tmp.write_text(json.dumps(state));os.chmod(tmp,0o600);os.replace(tmp,p)
def status(path=DEFAULT_PATH):
 try:s=json.loads(pathlib.Path(path).read_text())
 except (OSError,ValueError):s={}
 used=s.get('used',0) if s.get('day')==datetime.datetime.now(datetime.timezone.utc).date().isoformat() else 0
 return {'provider':PROVIDER,'daily_limit':DAILY_LIMIT,'used':used,'remaining':max(0,DAILY_LIMIT-used),'code':'quota_exhausted' if used>=DAILY_LIMIT else 'available'}
def fetch(key):
 at=int(time.time()*1000)
 try:
  req=urllib.request.Request('https://proxycheck.io/v3/'+key+'?ver=24-June-2026',headers={'User-Agent':'Shen-Egress-Audit/3','Accept':'application/json'})
  with urllib.request.build_opener(NoRedirect(),urllib.request.ProxyHandler({})).open(req,timeout=4) as response:
   body=response.read(131073)
   if len(body)>131072:raise ValueError('provider_body_limit')
  fact=parse(key,body,at);fact['class']=fact.pop('class_');return fact
 except Exception:return dict(ip=key,provider=PROVIDER,**{'class':'Unknown'},asn='',country='',at=at,expires=at+300000,digest='',code='provider_unavailable')
def classify(ips,path=DEFAULT_PATH):
 # Reserve under lock, perform HTTPS outside it, and merge atomically. Concurrent
 # workers share exact-IP single-flight leases and a crash-safe daily budget.
 now=int(time.time()*1000);keys=set();todo=[];out=[]
 for raw in ips:
  try:ip=ipaddress.ip_address(raw)
  except ValueError:continue
  if ip.is_global and not ip.is_multicast and not ip.is_reserved:keys.add(str(ip))
 with transaction(path) as state:
  for key in sorted(keys):
   cached=state['entries'].get(key)
   if cached and cached['expires']>now:
    if cached.get('code')!='lookup_pending':out.append(cached)
    continue
   if state['used']>=DAILY_LIMIT:continue
   state['used']+=1;todo.append(key)
   state['entries'][key]=dict(ip=key,provider=PROVIDER,**{'class':'Unknown'},asn='',country='',at=now,expires=now+15000,digest='',code='lookup_pending')
 if todo:
  with concurrent.futures.ThreadPoolExecutor(max_workers=4) as pool:found=list(pool.map(fetch,todo))
  with transaction(path) as state:
   for fact in found:state['entries'][fact['ip']]=fact
  out+=found
 return out
