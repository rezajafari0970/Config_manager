#!/usr/bin/env python3
"""Exact-IP intelligence plugins, finite caches and independent circuit breakers.

No built-in ASN/country/provider-to-Residential table. Provider configuration is
private, administrator-owned JSON. A positive classification is supplied only by
an explicitly configured specialist schema, never absence of a hosting flag.
"""
import asyncio, fcntl, os, tempfile, hashlib, ipaddress, json, pathlib, ssl, time, urllib.parse, urllib.request

class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self,*args,**kwargs): return None

def field(value,path):
    for part in path.split('.'):
        if not isinstance(value,dict):return None
        value=value.get(part)
    return value

class JSONDriver:
    """HTTPS JSON contract: echoed exact IP, explicit class, country and ASN.

    Mapping a provider's documented enum is configuration, not an ASN heuristic.
    A missing/unknown value remains Unknown. No boolean non-hosting inference.
    """
    def __init__(self,config):self.config=config
    async def lookup(self,ip):
        c=self.config;parsed=ipaddress.ip_address(ip)
        if not parsed.is_global:raise ValueError('nonpublic_ip')
        url=c['url'].replace('{ip}',urllib.parse.quote(str(parsed),safe=''))
        u=urllib.parse.urlsplit(url)
        if u.scheme!='https' or not u.hostname or u.username or u.password or '{ip}' not in c['url']:raise ValueError('invalid_provider_url')
        def request():
            headers={'Accept':'application/json','User-Agent':'Shen-IP-Evidence/1'}
            # Token stays in a private file/header, never in a URL or receipt.
            if c.get('token_file'):
                path=pathlib.Path(c['token_file']);info=path.stat()
                if not path.is_absolute() or info.st_mode&0o077:raise ValueError('private_provider_token_required')
                headers['Authorization']='Bearer '+path.read_text().strip()
            opener=urllib.request.build_opener(urllib.request.ProxyHandler({}),urllib.request.HTTPSHandler(context=ssl.create_default_context()),NoRedirect())
            with opener.open(urllib.request.Request(url,headers=headers),timeout=min(10,c.get('timeout',3))) as response:
                if response.status!=200:raise ValueError('provider_status')
                raw=response.read(65537)
            if len(raw)>65536:raise ValueError('provider_response_limit')
            data=json.loads(raw);fields=c['fields'];echo=field(data,fields['ip'])
            if ipaddress.ip_address(echo)!=parsed:raise ValueError('provider_ip_mismatch')
            kind=c.get('classes',{}).get(str(field(data,fields.get('class',''))),'Unknown')
            if kind not in ('Residential','Mobile','Datacenter'):kind='Unknown'
            # Specialist capability is established in the trusted registry;
            # ordinary public geolocation never gains one here.
            country=field(data,fields.get('country','')) or ''
            if not isinstance(country,str) or len(country)>3:country=''
            asn=field(data,fields.get('asn',''))
            if isinstance(asn,str):
                token=asn.split(' ',1)[0];asn=int(token[2:]) if token.startswith('AS') and token[2:].isdigit() else int(token) if token.isdigit() else 0
            if not isinstance(asn,int) or isinstance(asn,bool) or asn<0:asn=0
            stamp=int(time.time()*1000)
            return {'ip':str(parsed),'source':c['id'],'at':stamp,'expires':stamp+min(3600,max(30,c.get('ttl_seconds',600)))*1000,'class':kind,'country':country,'asn':asn,'evidence_sha256':hashlib.sha256(raw).hexdigest()}
        return await asyncio.to_thread(request)

class Registry:
    def __init__(self,configs):
        self.drivers={};self.cache={};self.inflight={};self.failures={};self.breakers={};self.sem=asyncio.Semaphore(4);self.cache_path=None;self.config_hash=""
        for c in configs:
            if c.get('driver')=='https-json-v1':self.register(c['id'],JSONDriver(c))
    def register(self,name,driver):
        if name in self.drivers:raise ValueError('duplicate_provider')
        self.drivers[name]=driver
    async def lookup(self,source,ip):
        key=(source,str(ipaddress.ip_address(ip)));stamp=int(time.time()*1000)
        if key in self.cache and self.cache[key]['expires']>stamp:return self.cache[key]
        if self.breakers.get(source,0)>stamp:return None
        if key in self.inflight:return await self.inflight[key]
        async def run():
            try:
                async with self.sem:fact=await self.drivers[source].lookup(key[1])
                self.cache[key]=fact;self.failures[source]=0
                if len(self.cache)>4096:
                    for old in sorted(self.cache,key=lambda k:self.cache[k]['expires'])[:1024]:del self.cache[old]
                return fact
            except Exception:
                self.failures[source]=self.failures.get(source,0)+1
                if self.failures[source]>=3:self.breakers[source]=int(time.time()*1000)+60000
                return None
        task=asyncio.create_task(run());self.inflight[key]=task
        try:return await task
        finally:self.inflight.pop(key,None)
    async def collect(self,ips):
        facts=await asyncio.gather(*(self.lookup(source,ip) for source in self.drivers for ip in sorted(set(ips))))
        self.flush()
        return [f for f in facts if f is not None]

    def persisted(self,path,config_hash):
        self.cache_path=pathlib.Path(path);self.config_hash=config_hash
        if not self.cache_path.is_absolute():raise ValueError('absolute_cache_required')
        try:
            data=json.loads(self.cache_path.read_text())
            if data.get('config_hash')!=config_hash:return
            stamp=int(time.time()*1000)
            for fact in data.get('facts',[])[:4096]:
                if fact['source'] in self.drivers and fact['expires']>stamp and fact['at']<=stamp+2000:self.cache[(fact['source'],str(ipaddress.ip_address(fact['ip'])))]=fact
            self.failures={k:int(v) for k,v in data.get('failures',{}).items() if k in self.drivers}
            self.breakers={k:int(v) for k,v in data.get('breakers',{}).items() if k in self.drivers}
        except (OSError,ValueError,KeyError,TypeError):return
    def flush(self):
        if self.cache_path is None:return
        self.cache_path.parent.mkdir(mode=0o700,parents=True,exist_ok=True)
        fd=os.open(str(self.cache_path)+'.lock',os.O_CREAT|os.O_RDWR,0o600)
        try:
            fcntl.flock(fd,fcntl.LOCK_EX);stamp=int(time.time()*1000);merged={}
            try:
                previous=json.loads(self.cache_path.read_text())
                if previous.get('config_hash')==self.config_hash:
                    merged={(v['source'],v['ip']):v for v in previous.get('facts',[]) if v['expires']>stamp and v['at']<=stamp+2000}
            except (OSError,ValueError,KeyError,TypeError):pass
            merged.update(self.cache);facts=sorted(merged.values(),key=lambda v:v['expires'],reverse=True)[:4096]
            data={'config_hash':self.config_hash,'facts':facts,'breakers':self.breakers,'failures':self.failures}
            handle,path=tempfile.mkstemp(dir=self.cache_path.parent,prefix='.intel-')
            try:
                with os.fdopen(handle,'w') as out:json.dump(data,out);out.flush();os.fsync(out.fileno())
                os.replace(path,self.cache_path)
            finally:
                if os.path.exists(path):os.unlink(path)
        finally:os.close(fd)

def load(path):
    path=pathlib.Path(path)
    if not path.exists():return Registry([])
    info=path.stat()
    if info.st_mode&0o077:raise ValueError('private_provider_config_required')
    data=json.loads(path.read_text())
    if data.get('version')!='shen.residential.intel.v1' or len(data.get('providers',[]))>8:raise ValueError('provider_contract_version')
    registry=Registry(data['providers']);registry.persisted(data.get('cache_file','/var/lib/config-repair-residential-intel/cache.json'),hashlib.sha256(json.dumps(data,sort_keys=True).encode()).hexdigest());return registry
