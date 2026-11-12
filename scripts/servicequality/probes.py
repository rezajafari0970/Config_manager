"""Bounded read-only service probes and acknowledged synthetic CF uploads."""
import concurrent.futures, json, pathlib, time
from benchmarks.core import Session, encoded, digest
from benchmarks.http import request, transfers
from .catalog import SERVICES, VERSION as DRIVERS, UNMEASURED
from .policy import VERSION

def save(path,obj):
    path=pathlib.Path(path);path.parent.mkdir(parents=True,exist_ok=True,mode=0o700)
    tmp=path.with_suffix('.tmp');tmp.write_text(json.dumps(obj,indent=2));tmp.replace(path)

def service_check(port, name, timeout):
    driver=SERVICES[name];checks=[]
    for url,status in driver['probes']:
        c=request(port,url,(status,),timeout=timeout,connect_timeout=8,follow_redirects=True)
        allowed=driver['domains'] or ['gstatic.com'];host=c.get('effective_host') or ''
        if not any(host==d or host.endswith('.'+d) for d in allowed):c.update(status='failed',driver_error='redirect_left_service_scope')
        checks.append(c)
    return checks

def measure(core, config, output, context, timeout=15):
    out=pathlib.Path(output);out.mkdir(parents=True,exist_ok=False,mode=0o700)
    r=dict(context,version=VERSION,drivers=DRIVERS,started=time.time(),
           config_sha256=digest(encoded(config)),core_build='not_run',runtime='not_run',
           services={},unmeasured=UNMEASURED)
    save(out/'receipt.json',r) # interruption revokes old success; never silently skipped
    session=Session(core,config,out/'core')
    try:
        with session as s:
            r.update(core_build='passed',runtime='passed',startup_ms=s.startup_ms)
            def service(item):
                name,driver=item
                return name,service_check(s.port,name,timeout)
            with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
                for name,checks in pool.map(service,SERVICES.items()):
                    r['services'][name]=checks;save(out/'receipt.json',r)
            # Do not equate <1.4 Mbps with a dead connection. Keep exact sizes
            # and acknowledgements, allow a bounded 60s payload window on weak
            # networks, and let measured goodput reduce the ranking honestly.
            r['payload']=transfers(s.port,out,60,connect_timeout=8)
            r['probe_policy']={'https_timeout':timeout,'connect_timeout':8,'payload_timeout':60,'upload_retries':0}
            r['resources']=s.resources()
            if s.process.poll() is not None:r['runtime']='failed'
    except Exception as e:
        r['error']=type(e).__name__;r['runtime']='failed';r['core_build']=getattr(session,'build',{}).get('status','not_run')
    checks=[c for values in r['services'].values() for c in values]+list(r.get('payload',{}).get(k,{}) for k in ['download','upload','post_load'])
    r['observed_http_bytes']=sum(int(c.get('download_bytes',0))+int(c.get('upload_bytes',0)) for c in checks)
    r['finished']=time.time();save(out/'receipt.json',r);return r
