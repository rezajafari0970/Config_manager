#!/usr/bin/env python3
"""Build PRIVATE Simple exports and a Smart candidate from device-qualified members.
Newly composed bytes are never marked healthy here; run the final core/traffic gate.
"""
import argparse,base64,hashlib,json,os,pathlib,subprocess,time
from benchmarks.core import encoded,digest
from benchmarks.quality import BENCHMARK_VERSION, complete, timestamp
VERSION='device-quality.v2'
SITES={'google','instagram','telegram','telegram_web','youtube','youtube_image','play'}
TRANSPORT={'dns_tcp','dns_udp','doh_cloudflare','doh_google','quic_vn'}
def qualify(row,core,network,now,max_age):
    if not network or network.lower() in ('unverified','unknown','unspecified'):return 'network_scope_unverified'
    if row.get('core_sha256')!=core:return 'other_target_core'
    if row.get('network_label')!=network:return 'other_network'
    tested=timestamp(row.get('finished'))
    if tested is None or not 0<=now-tested<=max_age:return 'stale_or_future_receipt'
    if row.get('benchmark_version')!=BENCHMARK_VERSION or row.get('benchmark_stage')!='full':return 'other_benchmark_contract'
    if row.get('build')!='passed' or row.get('startup')!='passed':return 'core_or_startup_unverified'
    if row.get('infrastructure_or_startup_error') or row.get('runtime_failure'):return 'incomplete_run'
    checks={x.get('name'):x for x in row.get('https',[])}
    if not SITES.issubset(checks) or any(checks[n].get('status')!='passed' or checks[n].get('transport_tls')!='verified' or checks[n].get('curl_exit')!=0 for n in SITES):return 'destination_transfer_incomplete'
    if not complete(row.get('transfers')):return 'bidirectional_payload_incomplete'
    transport={x.get('name'):x.get('status') for x in row.get('transport',{}).get('checks',[])}
    if any(transport.get(n)!='passed' for n in TRANSPORT):return 'dns_or_udp_incomplete'
    if row.get('http3',{}).get('status')!='passed' or row.get('http3',{}).get('tls')!='verified':return 'http3_unverified'
    return 'qualified_finite_observation'
def select(corpora,reports,core,network,now,max_age):
    candidates={digest(encoded(r['config'])):r['config'] for c in corpora for r in c['rows'] if r.get('family')!='smart'}
    latest={}
    for report in reports:
        for r in report['results']:
            h=r['config_sha256']
            if r.get('core_sha256')!=core or r.get('network_label')!=network:continue
            stamp=max((v for v in (timestamp(r.get('started')),timestamp(r.get('finished'))) if v is not None),default=float('-inf'))
            if h not in latest or stamp>=latest[h][0]:latest[h]=(stamp,r)
    accepted=[];decisions=[]
    for h,c in candidates.items():
        r=latest.get(h,(None,None))[1];code=qualify(r,core,network,now,max_age) if r else 'no_target_network_receipt'
        decisions.append({'sha256':h,'code':code,'tested':r.get('finished') if r else None})
        if code=='qualified_finite_observation':accepted.append(c)
    return accepted,decisions
if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('--corpus',action='append',required=True);p.add_argument('--report',action='append',required=True);p.add_argument('--core-sha',required=True);p.add_argument('--network-label',required=True);p.add_argument('--max-age',type=int,default=1800);p.add_argument('--compiler',required=True);p.add_argument('--output-dir',required=True);a=p.parse_args()
    if not 60<=a.max_age<=3600:raise SystemExit('max-age must be 60..3600 seconds')
    os.umask(0o077);out=pathlib.Path(a.output_dir);out.mkdir(parents=True,exist_ok=False)
    members,decisions=select([json.loads(pathlib.Path(f).read_text()) for f in a.corpus],[json.loads(pathlib.Path(f).read_text()) for f in a.report],a.core_sha,a.network_label,int(time.time()),a.max_age)
    report={'version':VERSION,'target_core_sha256':a.core_sha,'network_label':a.network_label,'decisions':decisions,'qualified_members':len(members),'final_smart_status':'not_built','apk_status':'not_run'}
    # Fresh directories prevent a failed recheck leaving an old healthy artifact.
    simple=out/'simple';simple.mkdir()
    for member in members:
        body=encoded(member);(simple/(digest(body)+'.json')).write_bytes(body)
    (out/'qualified-simple-array.json').write_bytes(encoded(members))
    if members:
        run=subprocess.run([a.compiler,'--smart'],input=encoded(members),capture_output=True,check=True,timeout=30);plan=json.loads(run.stdout)
        if plan['code']!='prepared':raise SystemExit('composition: '+plan['code'])
        c=plan['candidates'][0];body=base64.b64decode(c['config'],validate=True);assert digest(body)==c['sha256']
        (out/'candidate-smart.json').write_bytes(body);report.update(final_smart_status='requires_exact_core_and_runtime_test',final_smart_sha256=c['sha256'])
    (out/'qualification.json').write_text(json.dumps(report,indent=2));print(json.dumps({'qualified_members':len(members),'final_smart_status':report['final_smart_status']}));raise SystemExit(0 if members else 4)
