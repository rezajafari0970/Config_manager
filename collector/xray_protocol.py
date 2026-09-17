import json,uuid
from .validator import issue,WARN
def _port(v):
 try:return 1<=int(v)<=65535
 except:return False
def _uuid(v):
 try:uuid.UUID(str(v));return True
 except:return False
def audit(raw):
 o=json.loads(raw);z=[]
 for i,x in enumerate(o.get('outbounds',[])):
  if not isinstance(x,dict):continue
  p=x.get('protocol');s=x.get('settings')
  if not isinstance(s,dict):continue
  if p in ('vless','vmess'):
   vs=s.get('vnext');
   if not isinstance(vs,list) or not vs:
    if all(k in s for k in ('address','port')): continue # client/fork flattened outbound; preserve
    z.append(issue('XRAY_VNEXT_MISSING',f'outbounds[{i}] {p} vnext missing',WARN));continue
   for j,v in enumerate(vs):
    if not isinstance(v,dict):continue
    if not str(v.get('address','')).strip():z.append(issue('XRAY_SERVER_ADDRESS_MISSING',f'outbounds[{i}].vnext[{j}] address missing',WARN))
    if not _port(v.get('port')):z.append(issue('XRAY_SERVER_PORT_INVALID',f'outbounds[{i}].vnext[{j}] port invalid',WARN))
    users=v.get('users');
    if not isinstance(users,list) or not users:z.append(issue('XRAY_USERS_MISSING',f'outbounds[{i}].vnext[{j}] users missing',WARN));continue
    for u in users:
     if isinstance(u,dict) and p=='vless' and not _uuid(u.get('id')):z.append(issue('XRAY_VLESS_NONSTANDARD_ID','VLESS JSON id is non-standard; preserved',WARN))
  if p=='trojan':
   ss=s.get('servers')
   if not isinstance(ss,list) or not ss:z.append(issue('XRAY_TROJAN_SERVERS_MISSING',f'outbounds[{i}] trojan servers missing',WARN));continue
   for j,v in enumerate(ss):
    if not isinstance(v,dict):continue
    if not str(v.get('address','')).strip():z.append(issue('XRAY_SERVER_ADDRESS_MISSING',f'outbounds[{i}].servers[{j}] address missing',WARN))
    if not _port(v.get('port')):z.append(issue('XRAY_SERVER_PORT_INVALID',f'outbounds[{i}].servers[{j}] port invalid',WARN))
    if not str(v.get('password','')):z.append(issue('XRAY_TROJAN_PASSWORD_MISSING',f'outbounds[{i}].servers[{j}] password missing',WARN))
  st=x.get('streamSettings')
  if isinstance(st,dict):
   sec=str(st.get('security') or '').lower()
   if sec=='tls' and isinstance(st.get('tlsSettings'),dict):
    t=st['tlsSettings']
    if not (t.get('serverName') or t.get('serverNameToVerify')):z.append(issue('XRAY_TLS_SNI_MISSING',f'outbounds[{i}] TLS SNI missing; may still be valid by address',WARN))
   if sec=='reality' and isinstance(st.get('realitySettings'),dict):
    r=st['realitySettings']
    if not (r.get('publicKey') or r.get('password')):z.append(issue('XRAY_REALITY_KEY_MISSING',f'outbounds[{i}] Reality publicKey/password missing',WARN))
    if not r.get('serverName'):z.append(issue('XRAY_REALITY_SNI_MISSING',f'outbounds[{i}] Reality serverName missing',WARN))
 return z