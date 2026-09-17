import json
from .validator import validate,hard_errors
SAFE_CODES={'XRAY_SETTINGS_NOT_OBJECT','XRAY_STATS_NOT_OBJECT','XRAY_DNS_HOSTS_NOT_OBJECT','XRAY_HTTP_INBOUND_SETTINGS_ARRAY','XRAY_SOCKOPT_ARRAY'}
def repair(kind,raw):
 before=validate(kind,raw); codes={x['code'] for x in before}
 if kind!='json-xray' or not (codes & SAFE_CODES): return None
 o=json.loads(raw); changes=[]
 if 'XRAY_STATS_NOT_OBJECT' in codes and o.get('stats')==[]: o['stats']={}; changes.append('stats: [] -> {}')
 dns=o.get('dns')
 if 'XRAY_DNS_HOSTS_NOT_OBJECT' in codes and isinstance(dns,dict) and dns.get('hosts')==[]: dns['hosts']={}; changes.append('dns.hosts: [] -> {}')
 for i,x in enumerate(o.get('inbounds',[])):
  if 'XRAY_HTTP_INBOUND_SETTINGS_ARRAY' in codes and isinstance(x,dict) and x.get('protocol')=='http' and x.get('settings')==[]: x['settings']={}; changes.append(f'inbounds[{i}].settings: [] -> {{}}')
 for i,x in enumerate(o.get('outbounds',[])):
  if isinstance(x,dict) and isinstance(x.get('streamSettings'),dict):
   st=x['streamSettings']
   if 'XRAY_SOCKOPT_ARRAY' in codes and st.get('sockopt')==[]: st['sockopt']={}; changes.append(f'outbounds[{i}].streamSettings.sockopt: [] -> {{}}')
 if 'XRAY_SETTINGS_NOT_OBJECT' in codes:
  for i,x in enumerate(o.get('outbounds',[])):
   if x.get('settings')==[]: x['settings']={}; changes.append(f'outbounds[{i}].settings: [] -> {{}}')
 if not changes:return None
 fixed=json.dumps(o,separators=(',',':'),ensure_ascii=False)
 after=validate(kind,fixed)
 if hard_errors(after): return None
 return {'raw':fixed,'changes':changes,'remaining':[x for x in after if x['level']=='warning']}
