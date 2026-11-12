"""Network/core/config-scoped selection; no network I/O or hidden random choice."""
import math, statistics
from benchmarks.quality import complete, timestamp
from .catalog import SERVICES, VERSION as DRIVERS

VERSION = 'service-quality.v2'

def good(check):
    return (isinstance(check,dict) and check.get('status')=='passed'
            and check.get('transport_tls')=='verified' and check.get('curl_exit')==0
            and 200 <= check.get('http_status',0) < 300)

def finite(v):
    n=timestamp(v)
    return n is not None and n >= 0

def assess(history, service, now, context, max_age=900):
    """Two distinct completed rounds; newest failed/incomplete round revokes old pass."""
    samples={}
    for r in history:
        if r.get('version')!=VERSION or r.get('drivers')!=DRIVERS: continue
        if any(r.get(k)!=v for k,v in context.items()): continue
        started=timestamp(r.get('started')); finished=timestamp(r.get('finished'))
        if started is None or not 0<=now-started<=max_age: continue
        # IDs cannot manufacture repetitions, nor may retries share one timestamp.
        samples[started]=r
    rows=[samples[t] for t in sorted(samples)][-5:]
    def reason(r):
        finished=timestamp(r.get('finished'))
        if finished is None or not r['started']<=finished<=now or now-finished>max_age:return 'incomplete_or_invalid_time'
        if r.get('core_build')!='passed' or r.get('runtime')!='passed': return 'core_or_startup_unverified'
        if not complete(r.get('payload')):return 'bidirectional_payload_incomplete'
        checks=r.get('services',{}).get(service,[])
        if len(checks)!=len(SERVICES[service]['probes']):return 'missing_service_probe'
        if not all(good(c) and finite(c.get('total_ms')) for c in checks):
            return 'service_http_denied_or_unavailable' if any(c.get('http_status') in (401,403,429) for c in checks) else 'service_https_incomplete'
        return None
    def eligible(r):return reason(r) is None
    if len(rows)<2:return {'eligible':False,'reason':'insufficient_fresh_repetitions'}
    if not all(eligible(r) for r in rows[-2:]):return {'eligible':False,'reason':next(reason(r) for r in reversed(rows[-2:]) if reason(r))}
    accepted=[r for r in rows if eligible(r)]
    lat=[max(c['total_ms'] for c in r['services'][service]) for r in accepted]
    down=[r['payload']['download']['total_ms'] for r in accepted]
    up=[r['payload']['upload']['total_ms'] for r in accepted]
    if not all(finite(v) and v>0 for v in down+up):return {'eligible':False,'reason':'invalid_transfer_timing'}
    # Setup-inclusive goodput is a conservative ceiling, not service upload proof.
    d=16.777216/(statistics.median(down)/1000)
    u=8.388608/(statistics.median(up)/1000)
    delay=statistics.median(lat); variation=statistics.median([abs(b-a) for a,b in zip(lat,lat[1:])])
    reliability=len(accepted)/len(rows)
    # Bounded logarithms prevent bandwidth from outweighing instability.
    score=100*(.35*reliability+.20*min(1,math.log1p(d)/math.log(31))+
               .20*min(1,math.log1p(u)/math.log(16))+.15/(1+delay/500)+.10/(1+variation/250))
    return {'eligible':True,'reason':'provisional_https_and_generic_bidirectional',
            'score':round(score,4),'download_mbps':round(d,4),'upload_mbps':round(u,4),
            'service_completion_ms':delay,'completion_variation_ms':variation,
            'http_success_fraction':reliability,'samples':len(rows),'evidence_at':rows[-1]['started'],
            'service_upload':'not_measured','packet_jitter':'not_measured'}

def choose(candidates, previous, now, scope, hold=120, margin=10, wins=3):
    """Fast cached decision. Improvement needs three NEW waves and 120s dwell.
    Unsafe/stale incumbent bypasses hold; no qualified candidate means block.
    """
    old=previous if previous and previous.get('scope')==scope else {}
    allowed={k:v for k,v in candidates.items() if v.get('eligible') and finite(v.get('score'))}
    current=old.get('target','block')
    best=max(sorted(allowed),key=lambda k:allowed[k]['score']) if allowed else 'block'
    state=dict(old,scope=scope,target=current)
    if current not in allowed:
        return dict(scope=scope,target=best,since=now,reason='qualified_recovery' if best!='block' else 'no_fresh_qualified_route',challenger=None,wins=0)
    if best==current or allowed[best]['score']<allowed[current]['score']+margin:
        return dict(state,reason='healthy_route_retained',challenger=None,wins=0)
    stamp=allowed[best]['evidence_at']
    count=old.get('wins',0) if old.get('challenger')==best else 0
    if stamp!=old.get('challenger_evidence'):count+=1
    state.update(challenger=best,challenger_evidence=stamp,wins=count,reason='improvement_pending')
    if now-old.get('since',now)>=hold and count>=wins:
        state.update(target=best,since=now,reason='repeated_material_improvement',challenger=None,wins=0)
    return state
