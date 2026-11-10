#!/usr/bin/env python3
"""Trusted isolated residential screening driver. No guessed Ads egress proof.

The same-stream diagnostic is experimental: its returned IP is a candidate
observation, NEVER accepted evidence without an independently verified method.
No advertisement-serving URL, impression, click or identifier is requested.
"""
import argparse, asyncio, base64, contextlib, hashlib, importlib.util, ipaddress
import json, os, pathlib, re, resource, signal, socket, ssl, struct, subprocess
import sys, threading, time, uuid, urllib.request

HERE = pathlib.Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location('residential_adapter', HERE/'probe-one.py')
adapter = importlib.util.module_from_spec(spec); spec.loader.exec_module(adapter)
lab = adapter.lab
PREFIX = os.environ.get('SHEN_RESIDENTIAL_TEST_PREFIX', 'crr')
if PREFIX not in ('crr', 'crt'): raise RuntimeError('invalid resource prefix')
adapter.PREFIX = PREFIX
adapter.OWN = re.compile(r'^'+PREFIX+r'[0-9a-f]{6}20(?:0[0-8])$')
CONTROLS = ['www.cloudflare.com', 'api64.ipify.org']
IPV4_CONTROLS = ['1.1.1.1', 'api.ipify.org']
ALL_CONTROLS = CONTROLS + IPV4_CONTROLS

def control_group(host):
    return 'cloudflare' if host in (CONTROLS[0],IPV4_CONTROLS[0]) else 'ipify' if host in (CONTROLS[1],IPV4_CONTROLS[1]) else ''

def controls_healthy(rows, repeats):
    return all(len({control_group(r['domain']) for r in rows if r['wave']==wave and r.get('outcome')=='passed' and control_group(r['domain'])})==2 for wave in range(1,repeats+1))

def now(): return int(time.time()*1000)

def certificate_authority(cert, host):
    # SAN-only DNS matching for the additional, fixed diagnostic authority.
    # The TLS library already validated the chain and the original Ads SNI.
    host=host.encode('idna').decode('ascii').lower().rstrip('.')
    for kind,name in cert.get('subjectAltName',()):
        if kind!='DNS':continue
        name=name.lower().rstrip('.')
        if name==host:return True
        if name.startswith('*.') and name.count('*')==1 and host.count('.')==name.count('.') and host.endswith(name[1:]):return True
    return False

async def response(reader, allow_close=False):
    header = await reader.readuntil(b'\r\n\r\n')
    if len(header)>32768: raise ValueError('response_headers_limit')
    lines=header.decode('iso-8859-1').split('\r\n'); status=int(lines[0].split()[1]); headers={}
    for line in lines[1:]:
        if ':' in line:
            key,value=line.split(':',1); key=key.lower()
            if key in headers: raise ValueError('duplicate_response_header')
            headers[key]=value.strip()
    if 100<=status<200: raise ValueError('interim_response_unsupported')
    if status in (204,304): body=b''
    elif 'transfer-encoding' in headers:
        if headers['transfer-encoding'].lower()!='chunked' or 'content-length' in headers: raise ValueError('ambiguous_response_framing')
        chunks=[];size=0
        while True:
            line=await reader.readline()
            if len(line)>128:raise ValueError('chunk_header_limit')
            n=int(line.split(b';',1)[0].strip(),16)
            if n==0:
                # Trailers bounded and consumed before stream reuse.
                size=0
                while True:
                    trailer=await reader.readline();size+=len(trailer)
                    if size>4096:raise ValueError('trailer_limit')
                    if trailer==b'\r\n':break
                    if not trailer:raise EOFError()
                break
            size+=n
            if n<0 or size>65536:raise ValueError('response_body_limit')
            chunks.append(await reader.readexactly(n))
            if await reader.readexactly(2)!=b'\r\n':raise ValueError('chunk_framing')
        body=b''.join(chunks)
    elif 'content-length' in headers:
        n=int(headers['content-length'])
        if not 0<=n<=65536:raise ValueError('response_body_limit')
        body=await reader.readexactly(n)
    elif allow_close:
        chunks=[];size=0
        while True:
            chunk=await reader.read(min(8192,65537-size))
            if not chunk:break
            chunks.append(chunk);size+=len(chunk)
            if size>65536:raise ValueError('response_body_limit')
        body=b''.join(chunks)
    else:raise ValueError('persistent_framing_unsupported')
    return status,headers,body

async def connection_probe(host,variant,wave,timeout,path):
    begin=time.monotonic();token=uuid.uuid4().hex;writer=None
    row={'variant':variant,'domain':host,'family':0,'transport':'tcp','path':path,
         'at':now(),'wave':wave,'connection_id':token,'nonce':token,'ip':'',
         'method':'same-tls-candidate-v1','capability':'','tls_verified':False,
         'origin_verified':False,'meaning_verified':False,'nonce_bound':False,
         'same_connection':False,'connections':0,'redirects':0,
         'actual_domain_request':False,'response_sha256':'','outcome':'unknown','code':'not_run',
         'stage':'socks','http_status':0,'diagnostic_status':0,'diagnostic_origin_verified':False,'tls_certificate_sha256':''}
    try:
        async with asyncio.timeout(timeout):
            reader,writer=await asyncio.open_connection('127.0.0.1',10808,limit=32769)
            writer.write(b'\x05\x01\x00');await writer.drain()
            if await reader.readexactly(2)!=b'\x05\x00':raise ValueError('socks_greeting')
            name=host.encode('idna');writer.write(b'\x05\x01\x00\x03'+bytes([len(name)])+name+struct.pack('!H',443));await writer.drain()
            version,rep,_,at=await reader.readexactly(4)
            if version!=5 or rep:raise ValueError('socks_connect')
            n=4 if at==1 else 16 if at==4 else (await reader.readexactly(1))[0] if at==3 else -1
            if n<0:raise ValueError('socks_address')
            await reader.readexactly(n+2)
            row['stage']='tls'
            ctx=ssl.create_default_context();ctx.set_alpn_protocols(['http/1.1'])
            await writer.start_tls(ctx,server_hostname=host,ssl_handshake_timeout=timeout)
            row.update(tls_verified=True,origin_verified=True,connections=1)
            tls=writer.get_extra_info('ssl_object')
            row['tls_certificate_sha256']=hashlib.sha256(tls.getpeercert(binary_form=True)).hexdigest()
            row['stage']='http'
            async def get(authority,url,keep):
                # No forwarded-IP or ad-serving headers, no automatic redirects.
                writer.write(('GET '+url+' HTTP/1.1\r\nHost: '+authority+'\r\nUser-Agent: Shen-Connection-Audit/1\r\nCache-Control: no-cache\r\nConnection: '+('keep-alive' if keep else 'close')+'\r\n\r\n').encode());await writer.drain()
                return await response(reader,allow_close=not keep)
            if host in ALL_CONTROLS:
                url=('/cdn-cgi/trace?shen_probe=' if control_group(host)=='cloudflare' else '/?format=json&shen_probe=')+token
                status,headers,body=await get(host,url,False)
                row['http_status']=status
                if status!=200:raise ValueError('control_http_status')
                if control_group(host)=='cloudflare':
                    fields=dict(line.split('=',1) for line in body.decode().splitlines() if '=' in line)
                    if fields.get('h')!=host:raise ValueError('control_origin')
                    ip=ipaddress.ip_address(fields['ip'])
                else:ip=ipaddress.ip_address(json.loads(body)['ip'])
                if not ip.is_global:raise ValueError('control_ip_invalid')
                row.update(ip=str(ip),family=ip.version,outcome='passed',code='control_only',stage='complete',same_connection=True,actual_domain_request=True,response_sha256=hashlib.sha256(body).hexdigest())
                return row
            status,headers,body=await get(host,'/robots.txt',True)
            row.update(actual_domain_request=True,http_status=status,response_sha256=hashlib.sha256(body).hexdigest(),code='ads_tls_reachable_egress_unknown')
            if 300<=status<400 or headers.get('connection','').lower()=='close':return row
            row['stage']='diagnostic'
            # Authenticate the diagnostic authority on the SAME TLS peer too.
            # This does not establish an undocumented response's IP semantics.
            row['diagnostic_origin_verified']=certificate_authority(tls.getpeercert(),'www.google.com')
            second,headers,body=await get('www.google.com','/sorry/index?shen_probe='+token,False)
            row['diagnostic_status']=second
            # Even a recognizable response and nonce are NOT a verified public
            # diagnostic contract. Preserve candidate observation with UNKNOWN.
            hits=re.findall(rb'IP address:\s*([^<\s]+)',body)
            if len(hits)==1 and ('shen_probe='+token).encode() in body:
                ip=ipaddress.ip_address(hits[0].decode())
                if ip.is_global:row.update(ip=str(ip),family=ip.version,nonce_bound=True,same_connection=True,response_sha256=hashlib.sha256(body).hexdigest(),code='diagnostic_semantics_unverified')
    except Exception as exc:
        # Exception text can include host credentials or response bodies.
        allowed={'socks_greeting','socks_connect','socks_address','control_http_status','control_origin','control_ip_invalid','persistent_framing_unsupported','ambiguous_response_framing','response_body_limit','duplicate_response_header','response_headers_limit','chunk_framing','interim_response_unsupported'}
        row['code']='probe_timeout' if isinstance(exc,TimeoutError) else str(exc) if isinstance(exc,ValueError) and str(exc) in allowed else 'tls_certificate_failure' if isinstance(exc,ssl.SSLCertVerificationError) else 'connection_or_method_unavailable'
        row['outcome']='failed' if not row['tls_verified'] else 'unknown'
    finally:
        if writer:
            writer.close()
            with contextlib.suppress(Exception):await asyncio.wait_for(writer.wait_closed(),.2)
        row['at']=now();row['elapsed_ms']=round((time.monotonic()-begin)*1000,3)
    return row

async def session(request):
    policy=request['policy'];variants=request['variant'];sem=asyncio.Semaphore(6);rows=[]
    # Small initial screen; a healthy first wave expands to every explicit domain.
    # Reaching every domain still does not establish egress/network classification.
    priority=['googleads.g.doubleclick.net','pubads.g.doubleclick.net','securepubads.g.doubleclick.net']
    screening=[h for h in priority if h in policy['domains']]
    screening=(screening+[h for h in policy['domains'] if h not in screening])[:3]
    domains=policy['domains'] if request.get('deep') else screening
    for wave in range(1,policy['repeats']+1):
        async def run(host):
            async with sem:return await connection_probe(host,variants['name'],wave,policy['timeout_seconds'],variants['paths'][0])
        wave_rows=await asyncio.gather(*(run(h) for h in CONTROLS+domains))
        if wave==1 and not request.get('deep') and controls_healthy([r for r in wave_rows if r['domain'] in ALL_CONTROLS],1):
            remaining=[h for h in policy['domains'] if h not in domains]
            wave_rows.extend(await asyncio.gather(*(run(h) for h in remaining)))
            domains=policy['domains']
        # An IPv4 Ads observation must not be compared with IPv6 controls.
        # Add two independent IPv4 controls only when that gap is observed.
        if any(r.get('family')==4 and r['domain'] not in ALL_CONTROLS for r in wave_rows):
            groups={control_group(r['domain']) for r in wave_rows if r.get('family')==4 and r.get('outcome')=='passed' and control_group(r['domain'])}
            if len(groups)<2:wave_rows.extend(await asyncio.gather(*(run(h) for h in IPV4_CONTROLS)))
        # One bounded fresh retry for a transient control failure; never borrow
        # an IP from a different domain/wave or call it Ads evidence.
        if any(r.get('tls_verified') for r in wave_rows):
            async def retry(row):
                if row['domain'] not in ALL_CONTROLS or row.get('outcome')=='passed':return row
                await asyncio.sleep(.2)
                second=await run(row['domain']);second['attempts']=2
                return second
            wave_rows=await asyncio.gather(*(retry(r) for r in wave_rows))
        rows.extend(wave_rows)
        if wave==1 and not request.get('deep') and all(not r.get('tls_verified') for r in wave_rows):break
        if wave<policy['repeats']:await asyncio.sleep(.15)
    return {'observations':[r for r in rows if r['domain'] not in ALL_CONTROLS], 'controls':[r for r in rows if r['domain'] in ALL_CONTROLS]}

def strict_policy(box,resolved):
    # Residential tests have no host/system DNS allowance. Proxy hostname answers
    # are resolved once by the trusted supervisor and pinned before launch.
    pairs=sorted({(ip,str(p)) for v in resolved.values() for ip in v['ips'] for p in v.get('ports',[v['port']])})
    allow='\n'.join(f'ip daddr {ip} tcp dport {p} accept\nip daddr {ip} udp dport {p} accept' for ip,p in pairs)
    rules='flush ruleset\ntable inet guard { chain out { type filter hook output priority 0; policy drop; oifname "lo" accept; '+allow+'\n}; chain in { type filter hook input priority 0; policy drop; iifname "lo" accept; ct state established,related accept; }; }\n'
    box.net([lab.NFT,'-f','-'],input=rules.encode())

def variant_test(box,v,policy,deep=False,engine=None):
    result={'name':v['name'],'import_rule':v.get('import_rule','original-json'),'hash':v['sha256'],'runtime_hash':'','core':'not_run','health':'UNKNOWN','paths':v['paths'],'paths_complete':v['paths_complete'],'code':'not_run'}
    data=base64.b64decode(v['config'],validate=True)
    if len(data)>2*1024*1024 or lab.digest(data)!=v['sha256']:raise ValueError('invalid_candidate')
    strict_policy(box,{})
    process=box.launch(data,True)
    try:process.wait(timeout=15)
    except subprocess.TimeoutExpired:box.stop();result['code']='core_timeout';return result,{},{}
    rc=process.returncode;core_log=box.logs();box.stop()
    if rc!=0:result.update(core='failed',code=lab.classify(core_log,'core_validation_rejected'));return result,{},{}
    result['core']='passed';config=json.loads(data)
    try:
        if engine:resolved,runtime=engine.prepare(config)
        else:resolved=lab.endpoints(config);runtime=lab.runtime_config(config,resolved)
    except (ValueError,KeyError,TypeError) as exc:
        unsupported=str(exc) in ('outbound_driver_pending','ipv6_only_probe_pending','invalid_download_settings','transport_dependency_depth')
        result.update(core='unsupported' if unsupported else 'passed',code='transport_driver_unsupported' if unsupported else 'endpoint_preparation_unavailable');return result,{},{}
    # Keep the selected inbound's sniffing semantics; no extra listener/API.
    original=next((b for b in config.get('inbounds',[]) if b.get('protocol')=='socks'),{})
    if not engine and 'sniffing' in original:runtime['inbounds'][0]['sniffing']=original['sniffing']
    strict_policy(box,resolved);runtime_bytes=json.dumps(runtime,separators=(',',':')).encode();result['runtime_hash']=lab.digest(runtime_bytes)
    process=box.launch(runtime_bytes)
    ready=False
    for _ in range(30):
        if process.poll() is not None:break
        x=box.net(['/usr/bin/python3','-c','import socket;s=socket.socket();s.settimeout(.1);print(s.connect_ex(("127.0.0.1",10808)))'],timeout=2)
        if x.stdout.strip()==b'0':ready=True;break
        time.sleep(.1)
    observed={}
    if ready:
        request={'policy':policy,'deep':deep,'variant':{k:v[k] for k in ['name','paths']}}
        reply=box.net(['/usr/bin/python3',str(pathlib.Path(__file__).resolve()),'--session'],input=json.dumps(request).encode(),timeout=policy['repeats']*((len(policy['domains'])+5)//4)*(policy['timeout_seconds']+2)+5)
        observed=json.loads(reply.stdout)
    runtime_code=lab.classify(box.logs(),'controls_or_connection_unavailable')
    box.stop()
    controls=observed.get('controls',[])
    good=controls_healthy(controls,policy['repeats'])
    result.update(health='HEALTHY' if good else 'UNKNOWN',code='screening_complete' if good else runtime_code)
    return result,observed.get('observations',[]),controls

def main():
    if '--session' in sys.argv:
        request=json.load(sys.stdin);print(json.dumps(asyncio.run(session(request))));return
    # Reuse the scoped supervisor preflight and cleanup, never production crw.
    if any(a in sys.argv for a in ['--cleanup','--cleanup-slot','--self-test']):adapter.main();return
    ap=argparse.ArgumentParser();ap.add_argument('--core',required=True);ap.add_argument('--core-id',default='xray-26.7.28');ap.add_argument('--slot',type=int,required=True);ap.add_argument('--run-id',required=True);args=ap.parse_args()
    if os.geteuid()!=0 or not 1<=args.slot<=8 or not re.fullmatch('[0-9a-f]{6}',args.run_id):raise ValueError('invalid_supervisor')
    pins=json.loads((HERE/'residential-cores.json').read_text())
    core_sha=next((c['sha256'] for c in pins['cores'] if c['id']==args.core_id),None)
    if not core_sha or lab.digest(pathlib.Path(args.core).read_bytes())!=core_sha:raise ValueError('pinned_core_mismatch')
    parent=os.getppid()
    if parent<=1 or int(os.environ.get('SHEN_RESIDENTIAL_PARENT_PID',parent))!=parent:raise ValueError('owner_unavailable')
    signal.signal(signal.SIGTERM,lambda *_:sys.exit(143))
    def watch():
        while os.getppid()==parent:time.sleep(.25)
        os.kill(os.getpid(),signal.SIGTERM)
    threading.Thread(target=watch,daemon=True).start()
    request=json.load(sys.stdin);plan=request['plan'];policy=request['policy'];started=now();begin=time.monotonic();cpu=time.process_time()
    report={'version':'shen.residential.v1','policy_hash':request['policy_hash'],'template_version':request['template_version'],'core_sha256':core_sha,'core_id':args.core_id,'worker':request['worker'],'infrastructure_epoch':request['infrastructure_epoch'],'started':started,'finished':started,'infrastructure_ok':True,'variants':[],'observations':[],'controls':[],'ip_facts':[]}
    engine=None;sandbox=lab.Sandbox
    if args.core_id=='sing-box-1.14.4':
        if plan.get('driver')!='sing-box':raise ValueError('core_driver_mismatch')
        native_spec=importlib.util.spec_from_file_location('residential_native',HERE/'residential-native.py')
        engine=importlib.util.module_from_spec(native_spec);native_spec.loader.exec_module(engine);sandbox=engine.sandbox_type(lab)
    elif plan.get('driver','xray')!='xray':raise ValueError('core_driver_mismatch')
    try:
        with sandbox(args.core,args.run_id,8192+args.slot,prefix=PREFIX) as box:
            for v in plan['variants']:
                try:variant,observations,controls=variant_test(box,v,policy,request.get('deep',False) or pathlib.Path('/etc/config-repair/residential-observer.json').exists(),engine)
                except Exception:
                    box.stop();variant={'name':v['name'],'hash':v['sha256'],'core':'not_run','health':'UNKNOWN','paths':v['paths'],'paths_complete':False,'code':'probe_infrastructure_error'};observations=[];controls=[];report['infrastructure_ok']=False
                report['variants'].append(variant);report['observations'].extend(observations or []);report['controls'].extend(controls or [])
            stats=dict(line.split() for line in (box.cg/'cpu.stat').read_text().splitlines())
            report['cpu_ms']=int(stats.get('usage_usec',0))/1000
            report['peak_rss_bytes']=int((box.cg/'memory.peak').read_text()) if (box.cg/'memory.peak').exists() else 0
    finally:adapter.cleanup(args.slot,args.run_id)
    report.update(finished=now(),wall_ms=round((time.monotonic()-begin)*1000,3),cpu_ms=report.get('cpu_ms',0)+round((time.process_time()-cpu)*1000,3),peak_rss_bytes=max(report.get('peak_rss_bytes',0),resource.getrusage(resource.RUSAGE_SELF).ru_maxrss*1024))
    intel_spec=importlib.util.spec_from_file_location('residential_intel',HERE/'residential-intel.py')
    intel=importlib.util.module_from_spec(intel_spec);intel_spec.loader.exec_module(intel)
    observer_spec=importlib.util.spec_from_file_location('residential_observer',HERE/'residential-observer.py')
    observer=importlib.util.module_from_spec(observer_spec);observer_spec.loader.exec_module(observer)
    try:report['observations']=asyncio.run(observer.attest(observer.load('/etc/config-repair/residential-observer.json'),report))
    except Exception:pass
    try:registry=intel.load('/etc/config-repair/residential-intel.json')
    except Exception:registry=intel.Registry([])
    candidates=[r['ip'] for r in report['observations']+report['controls'] if r.get('ip')]
    async def collect_intel():
        specialist,metadata=await asyncio.gather(registry.collect(candidates),intel.metadata_registry().collect(candidates))
        return specialist+metadata
    try:report['ip_facts']=asyncio.run(collect_intel())
    except Exception:report['ip_facts']=[]
    report['finished']=now()
    async def infrastructure():
        async def check(host):
            row={'variant':'infrastructure','domain':host,'at':now(),'outcome':'unknown','code':'independent_infrastructure_check'}
            def run():
                url='https://'+host+('/cdn-cgi/trace' if host==CONTROLS[0] else '/?format=json')
                opener=urllib.request.build_opener(urllib.request.ProxyHandler({}),urllib.request.HTTPSHandler(context=ssl.create_default_context()),intel.NoRedirect())
                with opener.open(url,timeout=3) as response:
                    if response.status!=200:raise ValueError('control_status')
                    if len(response.read(65537))>65536:raise ValueError('control_limit')
            try:
                await asyncio.to_thread(run);row.update(outcome='passed',tls_verified=True,origin_verified=True,at=now())
            except Exception:pass
            return row
        return await asyncio.gather(*(check(h) for h in CONTROLS))
    report['infrastructure_controls']=asyncio.run(infrastructure())
    report['infrastructure_ok']=report['infrastructure_ok'] and all(x['outcome']=='passed' for x in report['infrastructure_controls'])
    if report['infrastructure_ok']:
        for variant in report['variants']:
            controls=[x for x in report['controls'] if x['variant']==variant['name']]
            ads=[x for x in report['observations'] if x['variant']==variant['name']]
            if len(controls)==2*policy['repeats'] and all(x['outcome']=='failed' for x in controls) and not any(x.get('tls_verified') for x in ads):variant['health']='UNHEALTHY'
    report.update(finished=now(),wall_ms=round((time.monotonic()-begin)*1000,3))
    print(json.dumps(report))

if __name__=='__main__':main()
