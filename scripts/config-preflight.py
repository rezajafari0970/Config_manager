"""Static egress checks for exact final configs. Does not certify health."""
import copy,ipaddress
VERSION='exact-runtime-preflight.v2'
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
    relay_endpoints(logical_for_inspection(config))
    listeners=set()
    for x in config.get('inbounds',[]):
        if x.get('listen')!='127.0.0.1':fail('nonloopback_listener_not_qualified')
        port=x.get('port')
        if not isinstance(port,int) or port in listeners:fail('listener_port_conflict')
        listeners.add(port)
    socks=[x for x in config.get('inbounds',[]) if x.get('protocol')=='socks' and x.get('port')==10808]
    if len(socks)!=1 or socks[0].get('settings',{}).get('auth','noauth')!='noauth':fail('probe_socks_contract_missing')
    return copy.deepcopy(config)

def logical_for_inspection(config):
    """Inspect real destinations without modifying any bytes used by the core."""
    c=copy.deepcopy(config);obs=c.get('outbounds',[])
    bytag={}
    for ob in obs:
        tag=ob.get('tag')
        if tag and tag in bytag:fail('duplicate_outbound_tag')
        if tag:bytag[tag]=ob
    remove=set()
    for ob in obs:
        if ob.get('protocol')!='vless':continue
        settings=ob.get('settings',{})
        node=settings if 'address' in settings else (settings.get('vnext') or [{}])[0]
        if node.get('address')!='127.0.0.1':continue
        stream=ob.get('streamSettings',{});opt=stream.get('sockopt',{})
        ref=opt.get('dialerProxy');leg=bytag.get(ref,{})
        dest=leg.get('settings',{}).get('redirect')
        if leg.get('protocol')!='freedom' or not isinstance(dest,str):continue
        if stream.get('security','none') not in ('','none') or stream.get('method',stream.get('network','tcp')) not in ('','tcp','raw','ws','websocket','httpupgrade','grpc','xhttp','splithttp','kcp','mkcp'):fail('relay_family_not_qualified')
        users=[settings] if node is settings else node.get('users',[])
        if len(users)!=1 or users[0].get('encryption','none') not in ('','none') or users[0].get('flow'):fail('relay_family_not_qualified')
        if set(opt)!={'dialerProxy'} or ob.get('proxySettings') or ob.get('sendThrough') or any(k in stream for k in ('finalmask','address','port')):fail('relay_local_carrier_invalid')
        if set(leg)-{'tag','protocol','settings','streamSettings','sendThrough'} or set(leg.get('settings',{}))!={'redirect'} or set(leg.get('streamSettings',{}))-{'sockopt'}:fail('relay_carrier_invalid')
        host,sep,port=dest.rpartition(':');host=host.strip('[]')
        if not sep or not port.isdigit() or not 1<=int(port)<=65535 or int(port)!=node.get('port') or not host:fail('relay_destination_invalid')
        try:ip=ipaddress.ip_address(host)
        except ValueError:ip=None
        if ip is not None:
            if ip.is_loopback or ip.is_link_local or ip.is_unspecified or ip.is_multicast:fail('relay_destination_invalid')
        else:
            if any(x in host for x in '/\\:@ '+chr(9)+chr(10)+chr(13)):fail('relay_destination_invalid')
        refs=[]
        def walk(x):
            if isinstance(x,dict):
                for key,value in x.items():
                    if key in ('dialerProxy','outboundTag','fallbackTag') and value==ref:refs.append(key)
                    if key=='proxySettings' and isinstance(value,dict) and value.get('tag')==ref:refs.append(key)
                    if key in ('selector','subjectSelector') and isinstance(value,list) and any(isinstance(v,str) and ref.startswith(v) for v in value):refs.append(key)
                    walk(value)
            elif isinstance(x,list):
                for value in x:walk(value)
        walk(config)
        if refs!=['dialerProxy'] or ref in remove:fail('relay_carrier_shared')
        node['address']=host
        if 'sockopt' in leg.get('streamSettings',{}):stream['sockopt']=copy.deepcopy(leg['streamSettings']['sockopt'])
        else:stream.pop('sockopt',None)
        if 'sendThrough' in leg:ob['sendThrough']=leg['sendThrough']
        remove.add(ref)
    c['outbounds']=[x for x in obs if x.get('tag') not in remove]
    return c
