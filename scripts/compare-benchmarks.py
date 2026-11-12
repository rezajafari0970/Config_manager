#!/usr/bin/env python3
"""Conservative measurement report. Never promotes a profile or grants output health."""
import argparse,collections,json,math,pathlib,statistics

def percentile(values,p):
    xs=sorted(values);return round(xs[max(0,math.ceil(len(xs)*p)-1)],2) if xs else None

def summarize(report):
    rows=[]
    for r in report['results']:
        attempts=[s['https'] for s in r.get('observations',[])];ok=[x for x in attempts if x['status']=='passed'];t=r.get('transfers',{});res=r.get('resources',{})
        rows.append({'label':r['label'],'core_build':r['build'],'startup':r['startup'],'first_verified_https_ms':r.get('first_verified_https_ms'),'request_successes':len(ok),'request_attempts':len(attempts),'request_p95_ms':percentile([x['total_ms'] for x in ok],.95),'destination_https':r['connection'],'download':t.get('download',{}).get('status','not_run'),'upload':t.get('upload',{}).get('status','not_run'),'download_mbps':t.get('download',{}).get('download_mbps'),'upload_mbps':t.get('upload',{}).get('upload_mbps'),'http3':r.get('http3',{}).get('status','not_run'),'core_cpu_percent':res.get('core_cpu_one_cpu_percent'),'core_peak_rss_mib':round(res.get('core_peak_rss_bytes',0)/1048576,2) if res else None,'temperature_c':r.get('battery_before',{}).get('temperature')})
    grouped=collections.defaultdict(list)
    for r in report['results']:grouped[r.get('profile','simple')].append(r)
    decisions={}
    for profile,rs in grouped.items():
        if profile in ('simple','baseline-v13'):continue
        reason=[]
        counts=collections.Counter(r.get('member_count') for r in rs)
        if not counts or min(counts.values())<3:reason.append('fewer_than_three_paired_repetitions_per_size')
        if any(r.get('build')!='passed' or r.get('startup')!='passed' for r in rs):reason.append('core_or_startup_not_verified')
        if any(not r.get('transfers',{}).get('both_passed') for r in rs):reason.append('bidirectional_transfer_not_verified_for_every_case')
        if any(r.get('connection')!='passed' for r in rs):reason.append('destination_checks_incomplete')
        # Repeated network observations alone do not authorize a production change.
        decisions[profile]={'state':'not_qualified' if reason else 'requires_paired_review','reasons':reason or ['check_identical_members_core_network_and_confidence_before_promotion']}
    return {'version':'benchmark-comparison.v1','network_label':report['network_label'],'core_sha256':report['core_sha256'],'rows':rows,'profiles':decisions,'production_action':'retain_current_profile','limits':report.get('limitations',[])+['Single randomized failure phase cannot establish worst-case failover','Shared phone load and short payloads limit throughput conclusions']}
if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('report');p.add_argument('--output',required=True);a=p.parse_args();out=summarize(json.loads(pathlib.Path(a.report).read_text()));pathlib.Path(a.output).write_text(json.dumps(out,indent=2));print(json.dumps({'rows':len(out['rows']),'production_action':out['production_action'],'profiles':out['profiles']}))
