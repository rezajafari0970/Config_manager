"""Project Smart adapter. Versioned explicit route groups; raw inputs stay intact."""
import copy, re
from .catalog import SERVICES

def adapt(base, choices=None, api_port=None):
    c=copy.deepcopy(base)
    if c.get('api'):raise ValueError('existing_api_requires_explicit_adapter')
    tags=[o.get('tag') for o in c['outbounds']]
    if len(set(tags))!=len(tags) or any(str(t).startswith('sq-') for t in tags):raise ValueError('tag_collision')
    entries={t for t in tags if re.fullmatch(r'proxy-[0-9]+',t or '')}
    if not entries or 'block' not in tags:raise ValueError('not_project_smart_profile')
    bootstrap=next(b['selector'] for b in c['routing']['balancers'] if b['tag']=='bootstrap-ip-balancer')
    if choices is not None:
        if set(choices)!=set(SERVICES):raise ValueError('missing_service_decision')
        if any(t not in entries|{'block'} for t in choices.values()):raise ValueError('invalid_target')
        if choices['default'] not in set(bootstrap)|{'block'}:raise ValueError('default_requires_independent_ip_for_bootstrap')
    # Existing Ads and source-specific routes retain precedence. Only the original
    # universal Smart rule is replaced, after API/DNS/scoped routing.
    rules=c['routing']['rules']; universal=[i for i,r in enumerate(rules) if set(r)=={'type','network','balancerTag'} and r['network']=='tcp,udp']
    if len(universal)!=1:raise ValueError('ambiguous_catchall')
    index=universal[0]
    if index!=len(rules)-1:raise ValueError('rules_after_catchall')
    def target(service):
        return {'outboundTag':choices[service]} if choices is not None else {'balancerTag':'sq-'+service}
    for r in rules[:index]:
        # Preserve Ads independently; only main/default and IP bootstrap go to
        # the quality-selected independent IP. No system resolver fallback.
        if r.get('balancerTag') in ('smart-parallel-balancer','bootstrap-ip-balancer'):
            r.pop('balancerTag');r.update(target('default'))
    specific=[]
    for name,driver in SERVICES.items():
        if not driver['domains']:continue
        specific.append(dict(type='field',domain=['domain:'+d for d in driver['domains']],network='tcp,udp',**target(name)))
    c['routing']['rules']=rules[:index]+specific+[dict(type='field',network='tcp,udp',**target('default'))]
    c['remarks']='@shenvpn-service-quality-v1 | '+('measured snapshot' if choices is not None else 'controller required')
    if choices is None:
        if not isinstance(api_port,int) or not 1024<=api_port<=65535:raise ValueError('invalid_api_port')
        ports=[i['port'] for i in c.get('inbounds',[])]
        if api_port in ports:raise ValueError('api_port_collision')
        c['api']={'tag':'sq-api','services':['RoutingService']}
        c['inbounds'].append({'tag':'sq-api-in','listen':'127.0.0.1','port':api_port,'protocol':'dokodemo-door','settings':{'address':'127.0.0.1'}})
        c['routing']['rules'].insert(0,{'type':'field','inboundTag':['sq-api-in'],'outboundTag':'sq-api'})
        for name in SERVICES:
            c['routing']['balancers'].append({'tag':'sq-'+name,'selector':['sq-unassigned-'+name+'-'],
                'strategy':{'type':'leastLoad','settings':{'expected':1,'maxRTT':'2s','tolerance':.15}},'fallbackTag':'block'})
    return c
