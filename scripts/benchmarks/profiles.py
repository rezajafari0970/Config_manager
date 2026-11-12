"""Named benchmark-only profiles. No automatic production promotion.
Pinned Xray clamps interval to 10s; its sweep period is interval*sampling.
"""
import copy
PROFILES={
 'baseline-v14':{'interval':'15s','sampling':3,'timeout':'5s'},
 'baseline-v13':{'interval':'15s','sampling':3,'timeout':'5s'},
 'responsive-v1':{'interval':'10s','sampling':2,'timeout':'3s'},
 'stable-path-v1':{'interval':'15s','sampling':3,'timeout':'5s'},
}
def apply(config,name):
    if name not in PROFILES:raise ValueError('unknown_benchmark_profile')
    c=copy.deepcopy(config)
    balancers=c.get('routing',{}).get('balancers',[])
    if not balancers or any(b.get('strategy',{}).get('type')!='leastLoad' for b in balancers):raise ValueError('unsupported_benchmark_strategy')
    pools={b.get('tag'):b for b in balancers}
    out={o.get('tag'):o for o in c.get('outbounds',[])}
    routes=c.get('routing',{}).get('rules',[])
    allowed={'smart-parallel-balancer':'smart-recovery','bootstrap-ip-balancer':'bootstrap-recovery','ads-exit-balancer':'ads-recovery'}
    for b in balancers:
        tag=b.get('fallbackTag')
        if tag=='block':continue
        relay=out.get(tag,{})
        secondary=pools.get(str(tag)+'-balancer',{})
        inbound=str(tag)+'-egress'
        routed=False
        for r in routes:
            if not r.get('inboundTag'):break
            if inbound in r['inboundTag']:
                routed=r.get('balancerTag')==str(tag)+'-balancer';break
        if allowed.get(b.get('tag'))!=tag or relay.get('protocol')!='loopback' or relay.get('settings',{}).get('inboundTag')!=inbound or secondary.get('fallbackTag')!='block' or secondary.get('selector')!=b.get('selector') or not routed:
            raise ValueError('unsupported_benchmark_recovery_graph')
    ping=c['burstObservatory']['pingConfig'];ping.update(PROFILES[name])
    if name=='stable-path-v1':
        for b in balancers:b['strategy']['settings']['expected']=1
    return c
