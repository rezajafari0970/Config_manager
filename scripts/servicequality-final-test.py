#!/usr/bin/env python3
"""Test the exported service profile as one runtime, including UDP/HTTP3.
This is still native Xray, not Android APK/app UI or a gaming benchmark.
"""
import argparse, concurrent.futures, json, os, pathlib, subprocess, sys, time
from benchmarks.core import Session, encoded, digest
from benchmarks.http import transfers
from benchmarks.quality import complete
from benchmarks.quic import check as http3
from servicequality.catalog import SERVICES, UNMEASURED
from servicequality.probes import service_check, save

def main():
    p=argparse.ArgumentParser();p.add_argument('--config',required=True);p.add_argument('--core',required=True);p.add_argument('--output',required=True);p.add_argument('--network',required=True);p.add_argument('--http3-python');a=p.parse_args()
    os.umask(0o077);out=pathlib.Path(a.output);out.mkdir(parents=True,exist_ok=False,mode=0o700)
    raw=pathlib.Path(a.config).read_bytes();config=json.loads(raw)
    r={'version':'service-final.v1','started':time.time(),'config_bytes_sha256':digest(raw),'config_canonical_sha256':digest(encoded(config)),
       'core_sha256':digest(pathlib.Path(a.core).read_bytes()),'network':a.network,'services':{},'passed':False,'unmeasured':UNMEASURED+['APK/VpnService','other_carriers','long_term_stability']}
    save(out/'result.json',r)
    try:
        validation=subprocess.run([a.core,'run','-test','-c',a.config],capture_output=True,timeout=15)
        r['exact_bytes_core_validation']=validation.returncode==0
        if validation.returncode:raise ValueError('exact_core_rejection')
        with Session(a.core,config,out/'session') as s:
            r['runtime_started']=True;r['startup_ms']=s.startup_ms
            with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
                for name,checks in pool.map(lambda n:(n,service_check(s.port,n,15)),SERVICES):
                    r['services'][name]=checks;save(out/'result.json',r)
            r['payload']=transfers(s.port,out,60,connect_timeout=8)
            script=pathlib.Path(__file__).with_name('transport-checks.py')
            checks=subprocess.run([sys.executable,str(script),'--socks-port',str(s.port),'--timeout','6'],capture_output=True,timeout=30)
            r['transport']=json.loads(checks.stdout)
            if a.http3_python:
                h3=subprocess.run([a.http3_python,str(pathlib.Path(__file__).with_name('http3-check.py')),'12','--socks-port',str(s.port)],capture_output=True,timeout=16)
                r['http3']=json.loads(h3.stdout)
            else:r['http3']=http3(s.port,12)
            r['resources']=s.resources()
            transport={c['name']:c['status'] for c in r['transport'].get('checks',[])}
            r['passed']=(complete(r['payload']) and all(c['status']=='passed' for cs in r['services'].values() for c in cs)
                         and all(transport.get(n)=='passed' for n in ['dns_tcp','dns_udp','doh_cloudflare','doh_google','quic_vn'])
                         and r['http3'].get('status')=='passed' and r['http3'].get('tls')=='verified' and s.process.poll() is None)
    except Exception as e:r['error']=type(e).__name__
    r['finished']=time.time();save(out/'result.json',r);print(json.dumps({k:r.get(k) for k in ['passed','error','startup_ms','exact_bytes_core_validation']}));return 0 if r['passed'] else 4
if __name__=='__main__':raise SystemExit(main())
