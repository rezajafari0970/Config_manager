import copy,json,sqlite3,collections
from collector.validator import validate
c=sqlite3.connect('data/collector.db');row=c.execute("select raw from configs where kind='json-xray' and raw like '%\"vnext\"%' limit 1").fetchone();base=json.loads(row[0])
def mutate(o,name):
 x=copy.deepcopy(o)
 if name=='outbounds_not_array':x['outbounds']={}
 elif name=='empty_outbounds':x['outbounds']=[]
 elif name=='outbound_no_protocol':x['outbounds'][0].pop('protocol',None)
 elif name=='settings_array':x['outbounds'][0]['settings']=[]
 elif name=='stats_array':x['stats']=[]
 elif name=='dns_hosts_array':x.setdefault('dns',{})['hosts']=[]
 elif name=='routing_rules_object':x.setdefault('routing',{})['rules']={}
 elif name=='duplicate_outbound_tag':x['outbounds'].append(copy.deepcopy(x['outbounds'][0]))
 elif name=='missing_route_ref':x.setdefault('routing',{})['rules']=[{'type':'field','outboundTag':'definitely-missing'}]
 elif name=='stream_array':x['outbounds'][0]['streamSettings']=[]
 return x
names=['outbounds_not_array','empty_outbounds','outbound_no_protocol','settings_array','stats_array','dns_hosts_array','routing_rules_object','duplicate_outbound_tag','missing_route_ref','stream_array']
R=[]
for n in names:
 raw=json.dumps(mutate(base,n),separators=(',',':'));z=validate('json-xray',raw);R.append((n,[i['code'] for i in z]))
print(json.dumps(R,indent=2));print('detected',sum(bool(x[1]) for x in R),'of',len(R))