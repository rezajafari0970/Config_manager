import copy,json
from collector.validator import validate
BASE={'inbounds':[{'type':'mixed','tag':'in','listen':'127.0.0.1','listen_port':1080}],'outbounds':[{'type':'vless','tag':'proxy','server':'example.com','server_port':443,'uuid':'00000000-0000-0000-0000-000000000000'},{'type':'direct','tag':'direct'}],'route':{'rules':[{'inbound':['in'],'outbound':'proxy'}]},'dns':{'servers':[{'tag':'cf','address':'1.1.1.1'}],'rules':[{'server':'cf'}]}}
def m(name):
 x=copy.deepcopy(BASE);o=x['outbounds'][0]
 if name=='server_empty':o['server']=''
 elif name=='port_high':o['server_port']=70000
 elif name=='uuid_bad':o['uuid']='bad'
 elif name=='tls_array':o['tls']=[]
 elif name=='transport_array':o['transport']=[]
 elif name=='transport_no_type':o['transport']={}
 elif name=='detour_missing':o['detour']='gone'
 elif name=='route_missing':x['route']['rules'][0]['outbound']='gone'
 elif name=='inbound_missing':x['route']['rules'][0]['inbound']=['gone']
 elif name=='dns_missing':x['dns']['rules'][0]['server']='gone'
 return x
names=['server_empty','port_high','uuid_bad','tls_array','transport_array','transport_no_type','detour_missing','route_missing','inbound_missing','dns_missing']
for n in names:
 z=validate('json-singbox',json.dumps(m(n)));print(n,bool(z),[i['code'] for i in z])
assert all(validate('json-singbox',json.dumps(m(n))) for n in names)