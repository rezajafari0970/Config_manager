#!/usr/bin/env python3
"""Independent country drivers with bounded consensus and last-good loading.
Only authenticated Ads peer addresses enter lookup. Persistent receipts contain
no addresses. GeoJS (MaxMind lineage) is consulted only for local uncertainty.
"""
import collections, hashlib, hmac, ipaddress, json, os, pathlib, re, sqlite3, time, urllib.request
VERSION='ads-country.v2'
MAX_AGE_MS=45*86400000
CONFIG='/etc/config-repair/egress-country.json'
def code(c):return isinstance(c,str) and bool(re.fullmatch('[A-Z]{2}',c)) and c not in ('ZZ','XX','EU','AP')
class SQLiteRanges:
 def __init__(self,path):
  self.db=sqlite3.connect(pathlib.Path(path).resolve().as_uri()+'?mode=ro&immutable=1',uri=True)
  self.db.execute('PRAGMA query_only=ON');self.db.execute('PRAGMA cache_size=-512')
 def lookup(self,ip):
  r=self.db.execute('SELECT finish,country FROM ranges WHERE family=? AND start<=? ORDER BY start DESC LIMIT 1',(ip.version,ip.packed)).fetchone()
  return r[1] if r and r[0]>=ip.packed else ''
 def close(self):self.db.close()
class MMDB:
 def __init__(self,path):
  import maxminddb
  self.reader=maxminddb.open_database(path)
 def lookup(self,ip):
  d=self.reader.get(str(ip)) or {};c=d.get('country',{})
  return d.get('country_code','') or (c.get('iso_code','') if isinstance(c,dict) else '')
 def close(self):self.reader.close()
DRIVERS={'sqlite-ranges':SQLiteRanges,'mmdb':MMDB}
def consensus(facts):
 groups={}
 for f in facts:
  if code(f['country']):groups.setdefault(f['lineage'],set()).add(f['country'])
 votes=collections.Counter(next(iter(v)) for v in groups.values() if len(v)==1)
 if votes:
  c,n=votes.most_common(1)[0]
  if n>=2 and n>sum(votes.values())/2:return c
 return ''
class Registry:
 def __init__(self):
  self.key=None;self.providers=[];self.errors=[];self.retry_at=0;self.remote=False;self.cache=collections.OrderedDict();self.secret=os.urandom(32);self.remote_retry=0
 def close(self):
  for _,driver in self.providers:driver.close()
  self.providers=[]
 def refresh(self,config=CONFIG):
  if time.monotonic()<self.retry_at:return
  try:
   p=pathlib.Path(config);st=p.stat();key=(str(p),st.st_ino,st.st_mtime_ns,st.st_size)
   if key==self.key:return
   if st.st_size>32768:raise ValueError()
   m=json.loads(p.read_text());entries=m.get('providers',[])
   if m.get('version') not in (VERSION,'ads-country.v1') or not 1<=len(entries)<=8:raise ValueError()
   names=[e.get('name','') for e in entries]
   if len(names)!=len(set(names)) or any(not re.fullmatch('[a-z0-9-]{1,48}',n) for n in names) or any(not re.fullmatch('[a-z0-9-]{1,48}',e.get('lineage',e['name'])) for e in entries):raise ValueError()
   loaded=[];errors=[];seen=set();old={e['name']:(e,d) for e,d in self.providers}
   for entry in entries:
    name=entry.get('name','');lineage=entry.get('lineage',name)
    if not re.fullmatch('[a-z0-9-]{1,48}',name) or name in seen or not re.fullmatch('[a-z0-9-]{1,48}',lineage):raise ValueError()
    seen.add(name);entry={**entry,'lineage':lineage}
    try:
     path=pathlib.Path(entry['path'])
     if not path.is_absolute() or not path.is_file() or path.stat().st_size>512*1024*1024:raise ValueError()
     if name in old and old[name][0]==entry:loaded.append(old.pop(name));continue
     with path.open('rb') as f:digest=hashlib.file_digest(f,'sha256').hexdigest()
     if digest!=entry['sha256'] or not isinstance(entry['released_at'],int):raise ValueError()
     loaded.append((entry,DRIVERS[entry['driver']](path)))
    except Exception:
     errors.append(name)
     if name in old:loaded.append(old.pop(name)) # independent last-good, freshness still enforced
   for _,d in old.values():d.close()
   self.providers=loaded;self.errors=errors;self.key=key if not errors else None;self.remote=m.get('geojs',False) is True
   self.retry_at=time.monotonic()+30 if errors else 0
  except Exception:self.retry_at=time.monotonic()+30;self.errors=['manifest_unavailable']
 def lookup(self,value,at=None):
  at=int(time.time()*1000) if at is None else at
  r=dict(version=VERSION,status='country_not_found',country='',at=at,providers=[])
  try:
   ip=ipaddress.ip_address(value)
   if not ip.is_global or ip.is_multicast or ip.is_reserved:raise ValueError()
  except ValueError:r['status']='invalid_source';return r
  for e,d in self.providers:
   if e['released_at']>at+86400000 or at-e['released_at']>=MAX_AGE_MS:continue
   try:c=d.lookup(ip)
   except Exception:continue
   if code(c):r['providers'].append(dict(name=e['name'],lineage=e['lineage'],dataset=e['sha256'],released_at=e['released_at'],country=c,kind='database'))
  self.finish(r);return r
 @staticmethod
 def finish(r):
  r['country']=consensus(r['providers'])
  r['status']='located' if r['country'] else 'country_provider_disagreement' if len({p['country'] for p in r['providers']})>1 else 'country_insufficient_sources'
 def remote_lookup(self,values):
  results={};missing=[];now=time.monotonic()
  for v in values:
   k=hmac.new(self.secret,v.encode(),hashlib.sha256).digest();cached=self.cache.get(k)
   if cached and cached[0]>now:results[v]=cached[1];self.cache.move_to_end(k)
   else:missing.append(v)
  if missing and now>=self.remote_retry:
   # Bounded batch; strict default TLS, no redirects, no proxy environment.
   try:
    if len(missing)>128:raise ValueError()
    class NoRedirect(urllib.request.HTTPRedirectHandler):
     def redirect_request(self,*a,**k):raise ValueError('redirect')
    opener=urllib.request.build_opener(urllib.request.ProxyHandler({}),NoRedirect())
    req=urllib.request.Request('https://get.geojs.io/v1/ip/country.json?ip='+','.join(missing),headers={'User-Agent':'Shen-Ads-Country/2.0'})
    with opener.open(req,timeout=3) as response:raw=response.read(65537)
    if len(raw)>65536:raise ValueError()
    data=json.loads(raw);data=[data] if isinstance(data,dict) else data
    if not isinstance(data,list):raise ValueError()
    for x in data:
     v=x.get('ip');c=x.get('country')
     if v not in missing or not code(c):continue
     fact=dict(name='geojs',lineage='maxmind',dataset=hashlib.sha256(json.dumps(x,sort_keys=True).encode()).hexdigest(),released_at=0,checked_at=int(time.time()*1000),country=c,kind='live')
     results[v]=fact;k=hmac.new(self.secret,v.encode(),hashlib.sha256).digest();self.cache[k]=(now+3600,fact)
    while len(self.cache)>4096:self.cache.popitem(last=False)
   except Exception:self.remote_retry=now+30
  return results
 def attach(self,rows,config=CONFIG):
  self.refresh(config);cache={}
  for r in rows:
   if r.get('actual_ads') and all(r.get(k) for k in ('echo','tls','same_stream')):
    v=r.get('ip','')
    if v not in cache:cache[v]=self.lookup(v)
  pending=[v for v,g in cache.items() if not g['country'] and g['status']!='invalid_source']
  if self.remote and pending:
   for v,f in self.remote_lookup(pending).items():cache[v]['providers'].append(f);cache[v]['at']=int(time.time()*1000);self.finish(cache[v])
  for r in rows:
   if r.get('actual_ads') and all(r.get(k) for k in ('echo','tls','same_stream')):r['geo']=cache[r.get('ip','')]
  return rows
