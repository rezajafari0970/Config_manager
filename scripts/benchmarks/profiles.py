"""Named benchmark-only profiles. No automatic production promotion.
Pinned Xray clamps interval to 10s; its sweep period is interval*sampling.
"""
import copy
PROFILES={
 'baseline-v13':{'interval':'15s','sampling':3,'timeout':'5s'},
 'responsive-v1':{'interval':'10s','sampling':2,'timeout':'3s'},
 'stable-path-v1':{'interval':'15s','sampling':3,'timeout':'5s'},
}
def apply(config,name):
    if name not in PROFILES:raise ValueError('unknown_benchmark_profile')
    c=copy.deepcopy(config)
    balancers=c.get('routing',{}).get('balancers',[])
    if not balancers or any(b.get('strategy',{}).get('type')!='leastLoad' or b.get('fallbackTag')!='block' for b in balancers):raise ValueError('unsupported_benchmark_strategy')
    ping=c['burstObservatory']['pingConfig'];ping.update(PROFILES[name])
    if name=='stable-path-v1':
        for b in balancers:b['strategy']['settings']['expected']=1
    return c
