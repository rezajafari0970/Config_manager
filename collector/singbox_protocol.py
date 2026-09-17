import json,uuid
from .validator import issue,WARN
def _uuid(v):
 try:uuid.UUID(str(v));return True
 except:return False
def audit(raw):
 o=json.loads(raw);z=[]
 for i,x in enumerate(o.get('outbounds',[])):
  if not isinstance(x,dict):continue
  t=x.get('type')
  if t in ('vless','vmess') and not _uuid(x.get('uuid')):z.append(issue('SINGBOX_NONSTANDARD_UUID',f'outbounds[{i}] {t} uuid non-standard',WARN))
  if t=='trojan' and not str(x.get('password','')):z.append(issue('SINGBOX_TROJAN_PASSWORD_MISSING',f'outbounds[{i}] password missing',WARN))
  if t=='shadowsocks':
   if not str(x.get('method','')):z.append(issue('SINGBOX_SS_METHOD_MISSING',f'outbounds[{i}] method missing',WARN))
   if 'password' not in x:z.append(issue('SINGBOX_SS_PASSWORD_MISSING',f'outbounds[{i}] password missing',WARN))
  if t in ('hysteria','hysteria2') and not any(x.get(k) for k in ('auth','auth_str','password')):z.append(issue('SINGBOX_HY_AUTH_MISSING',f'outbounds[{i}] auth missing',WARN))
  if t=='tuic' and not (x.get('uuid') and x.get('password')):z.append(issue('SINGBOX_TUIC_CREDENTIALS_MISSING',f'outbounds[{i}] credentials incomplete',WARN))
  tr=x.get('transport')
  if isinstance(tr,dict) and not tr.get('type'):z.append(issue('SINGBOX_TRANSPORT_TYPE_MISSING',f'outbounds[{i}] transport.type missing',WARN))
 return z