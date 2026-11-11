"""Source-derived v2rayNG 2.3.15 Custom Config contract, NOT Android execution.
Pinned source: CoreConfigManager.kt@57907cf354d9769200ed4717b4430d605abe0952.
AngConfigManager tries array then object; its first exception is not final failure.
"""
import copy,json
VERSION='v2rayng-2.3.15-custom.v1'
def profiles(data):
    root=json.loads(data)
    values=root if isinstance(root,list) else [root]
    if not values or any(not isinstance(v,dict) or not all(k in v for k in ('inbounds','outbounds','routing')) for v in values):raise ValueError('custom_import_shape_invalid')
    return values
def transform(config,speed,tun,mtu=1500):
    root=copy.deepcopy(config)
    if speed:
        root.setdefault('stats',{})
        root.setdefault('policy',{'system':{'statsOutboundUplink':True,'statsOutboundDownlink':True}})
    else:
        root.pop('stats',None)
        if isinstance(root.get('policy'),dict):
            root['policy'].pop('system',None)
            if not root['policy']:root.pop('policy')
    if tun and not any(x.get('protocol')=='tun' for x in root['inbounds']):
        root['inbounds'].append({'tag':'tun','protocol':'tun','settings':{'name':'xray0','MTU':mtu,'userLevel':8},'sniffing':{'enabled':True,'destOverride':['http','tls','quic']}})
    return root
def variants(data):
    p=profiles(data)
    if len(p)!=1:raise ValueError('individual_profile_required')
    for speed,tun in [(False,False),(True,False),(False,True),(True,True)]:
        yield json.dumps(transform(p[0],speed,tun),separators=(',',':')).encode()
