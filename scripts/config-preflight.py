"""Static egress checks for exact final configs. Does not certify health."""
import copy,ipaddress
VERSION='exact-runtime-preflight.v1'
def fail(code):raise ValueError(code)
def relay_endpoints(c):
    obs=c.get('outbounds',[]);ins=c.get('inbounds',[])
    bytag={x.get('tag'):x for x in obs};rules=c.get('routing',{}).get('rules',[])
    relays={};legs=set();routes=set()
    for ob in obs:
        if ob.get('protocol')!='vless':continue
        s=ob.get('settings',{})
        node=s if 'address' in s else (s.get('vnext') or [{}])[0]
        if node.get('address')!='127.0.0.1':continue
        stream=ob.get('streamSettings',{})
        if stream.get('security','none') not in ('','none') or stream.get('method',stream.get('network','tcp')) not in ('','tcp','raw'):fail('relay_family_not_qualified')
        users=[s] if node is s else node.get('users',[])
        if len(users)!=1 or users[0].get('encryption','none') not in ('','none') or users[0].get('flow'):fail('relay_family_not_qualified')
        if stream.get('sockopt') or ob.get('proxySettings') or ob.get('sendThrough'):fail('relay_local_carrier_invalid')
        if any(k in stream for k in ('finalmask','address','port')):fail('relay_family_not_qualified')
        found=[x for x in ins if x.get('listen')=='127.0.0.1' and x.get('port')==node.get('port') and x.get('protocol') in ('dokodemo-door','tunnel')]
        if len(found)!=1:fail('relay_listener_missing')
        inbound=found[0];tag=inbound.get('tag');cfg=inbound.get('settings',{})
        if set(inbound)-{'tag','listen','port','protocol','settings'} or set(cfg)-{'address','port','network','followRedirect'} or cfg.get('network')!='tcp' or cfg.get('followRedirect') is not False:fail('relay_listener_unbounded')
        host,port=cfg.get('address'),cfg.get('port')
        if not isinstance(host,str) or not isinstance(port,int) or not 1<=port<=65535:fail('relay_destination_invalid')
        try:ipaddress.ip_address(host);key,value='ip',host
        except ValueError:key,value='domain','full:'+host
        scoped=[]
        for i,r in enumerate(rules):
            if tag in r.get('inboundTag',[]):scoped.append((i,r))
            elif not r.get('inboundTag') and len(scoped)<2:fail('relay_route_shadowed')
            if len(scoped)==2:break
        if len(scoped)!=2:fail('relay_deny_missing')
        (ai,a),(di,d)=scoped;leg=a.get('outboundTag');block=d.get('outboundTag')
        if a!={'type':'field','inboundTag':[tag],'network':'tcp',key:[value],'port':str(port),'outboundTag':leg}:fail('relay_route_unbounded')
        if d!={'type':'field','inboundTag':[tag],'network':'tcp,udp','outboundTag':block} or bytag.get(block,{}).get('protocol')!='blackhole':fail('relay_deny_missing')
        carrier=bytag.get(leg,{})
        if carrier.get('protocol')!='freedom' or carrier.get('settings')!={} or set(carrier)-{'tag','protocol','settings','streamSettings','sendThrough'} or set(carrier.get('streamSettings',{}))-{'sockopt'}:fail('relay_carrier_invalid')
        ident=(node['address'],node['port'])
        if ident in relays:fail('relay_listener_shared')
        relays[ident]=(host,port);legs.add(leg);routes.update((ai,di))
    # Any direct route outside the fixed carrier is disallowed in a proxy-health
    # certificate; unused original helpers remain intact.
    for i,r in enumerate(rules):
        if bytag.get(r.get('outboundTag'),{}).get('protocol')=='freedom' and i not in routes:fail('direct_route_not_qualified')
    if obs and obs[0].get('protocol')=='freedom':fail('direct_default_not_qualified')
    for ob in obs:
        for ref in (ob.get('proxySettings',{}).get('tag'),ob.get('streamSettings',{}).get('sockopt',{}).get('dialerProxy')):
            if ref in legs:fail('relay_carrier_shared')
    return relays
def runtime_config(config,resolved):
    relay_endpoints(config)
    listeners=set()
    for x in config.get('inbounds',[]):
        if x.get('listen')!='127.0.0.1':fail('nonloopback_listener_not_qualified')
        port=x.get('port')
        if not isinstance(port,int) or port in listeners:fail('listener_port_conflict')
        listeners.add(port)
    socks=[x for x in config.get('inbounds',[]) if x.get('protocol')=='socks' and x.get('port')==10808]
    if len(socks)!=1 or socks[0].get('settings',{}).get('auth','noauth')!='noauth':fail('probe_socks_contract_missing')
    return copy.deepcopy(config)
