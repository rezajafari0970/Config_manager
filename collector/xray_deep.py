import json
from .validator import issue,WARN
NETWORK_KEYS={'tcp':'tcpSettings','ws':'wsSettings','grpc':'grpcSettings','xhttp':'xhttpSettings','httpupgrade':'httpupgradeSettings','h2':'httpSettings','http':'httpSettings','kcp':'kcpSettings'}
def audit(raw):
 o=json.loads(raw);z=[]
 for i,x in enumerate(o.get('outbounds',[])):
  if not isinstance(x,dict):continue
  p=x.get('protocol'); st=x.get('streamSettings')
  if p in ('vless','vmess','trojan','shadowsocks') and not isinstance(x.get('settings'),dict):z.append(issue('XRAY_PROTOCOL_SETTINGS_SHAPE',f'outbounds[{i}] {p} settings is not object',WARN))
  if not isinstance(st,dict):continue
  net=str(st.get('network') or '').lower(); sec=str(st.get('security') or '').lower()
  if net and net not in NETWORK_KEYS:z.append(issue('XRAY_UNKNOWN_NETWORK',f'outbounds[{i}] unknown network {net}',WARN))
  key=NETWORK_KEYS.get(net)
  if key in st and not isinstance(st[key],dict):z.append(issue('XRAY_TRANSPORT_SETTINGS_SHAPE',f'outbounds[{i}].{key} is not object',WARN))
  if sec=='tls' and 'tlsSettings' in st and not isinstance(st['tlsSettings'],dict):z.append(issue('XRAY_TLS_SETTINGS_SHAPE',f'outbounds[{i}].tlsSettings is not object',WARN))
  if sec=='reality' and 'realitySettings' in st and not isinstance(st['realitySettings'],dict):z.append(issue('XRAY_REALITY_SETTINGS_SHAPE',f'outbounds[{i}].realitySettings is not object',WARN))
 r=o.get('routing')
 if r is not None and not isinstance(r,dict):z.append(issue('XRAY_ROUTING_SHAPE','routing is not object',WARN))
 if isinstance(r,dict) and 'rules' in r and not isinstance(r['rules'],list):z.append(issue('XRAY_ROUTING_RULES_SHAPE','routing.rules is not array',WARN))
 return z