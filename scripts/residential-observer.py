#!/usr/bin/env python3
"""Optional authorized server-flow observer bridge; never a public reflector.

Only a Go-verified Ed25519 receipt can qualify. This transport performs no
classification and sends no raw configuration, credentials, or private URLs.
The operator must supply a server observer that really correlates each flow.
"""
import asyncio, json, pathlib, ssl, urllib.parse, urllib.request

class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self,*args,**kwargs):return None

def load(path):
    path=pathlib.Path(path)
    if not path.exists():return None
    if path.is_symlink() or path.stat().st_mode&0o077 or path.stat().st_size>32768:raise ValueError('private_observer_config_required')
    config=json.loads(path.read_text());url=urllib.parse.urlsplit(config['url'])
    if config.get('version')!='shen.residential.observer.v1' or url.scheme!='https' or not url.hostname or url.username or url.password or url.query or url.fragment:raise ValueError('invalid_observer_contract')
    return config

async def attest(config,report):
    if not config:return report['observations']
    variants={v['name']:v for v in report['variants']};sem=asyncio.Semaphore(4)
    async def one(row):
        if not row.get('tls_verified') or not row.get('actual_domain_request'):return row
        v=variants[row['variant']]
        request={k:row[k] for k in ('variant','domain','transport','path','at','wave','nonce','connection_id','response_sha256')}
        request.update(version='shen.residential.flow.v1',config_hash=v['hash'],runtime_hash=v['runtime_hash'],policy_hash=report['policy_hash'])
        def call():
            headers={'Content-Type':'application/json'}
            if config.get('token_file'):
                token=pathlib.Path(config['token_file'])
                if not token.is_absolute() or token.is_symlink() or token.stat().st_mode&0o077:raise ValueError('private_observer_token_required')
                headers['Authorization']='Bearer '+token.read_text().strip()
            opener=urllib.request.build_opener(urllib.request.ProxyHandler({}),urllib.request.HTTPSHandler(context=ssl.create_default_context()),NoRedirect())
            with opener.open(urllib.request.Request(config['url'],data=json.dumps(request).encode(),headers=headers),timeout=3) as response:
                if response.status!=200:raise ValueError('observer_unavailable')
                raw=response.read(16385)
            if len(raw)>16384:raise ValueError('observer_response_limit')
            signed=json.loads(raw)
            # Do not allow a broker to substitute another run, domain, path,
            # control result, config, time, response, or runtime.
            if any(signed.get(k)!=value for k,value in request.items() if k!='version'):raise ValueError('observer_scope_mismatch')
            if signed.get('method')!='server-flow-ed25519-v1' or not signed.get('attestation'):raise ValueError('observer_signature_missing')
            return signed
        try:
            async with sem:return await asyncio.to_thread(call)
        except Exception:return row
    return await asyncio.gather(*(one(row) for row in report['observations']))
