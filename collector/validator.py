import base64,json,urllib.parse,uuid,re
HARD='hard'; WARN='warning'
def issue(code,msg,level=HARD): return {'code':code,'message':msg,'level':level}
def port(v):
 try: return 1<=int(v)<=65535
 except: return False
def uuid_ok(v):
 try: uuid.UUID(str(v)); return True
 except: return False
def b64(s): return base64.urlsafe_b64decode(s+'='*((4-len(s)%4)%4))
def link(kind,raw):
 z=[]
 try:
  if kind=='ss':
   payload=raw.split('://',1)[1].split('#',1)[0].split('?',1)[0]
   # SIP002 supports both userinfo-base64@host:port and legacy full-base64 method:password@host:port.
   if '@' not in payload:
    try: payload=b64(payload).decode('utf8')
    except Exception: return [issue('SS_INVALID_BASE64','Shadowsocks payload is not valid base64')]
   if '@' not in payload: return [issue('SS_MISSING_SERVER','Shadowsocks server is missing')]
   creds,server=payload.rsplit('@',1)
   if ':' not in creds:
    try: decoded=b64(creds).decode('utf8')
    except Exception: decoded=''
    if ':' not in decoded: return [issue('SS_OPAQUE_USERINFO','Shadowsocks userinfo is opaque/non-SIP002; preserved unchanged for compatible clients',WARN)]
   host,sep,prt=server.rpartition(':')
   if not host: z.append(issue('MISSING_HOST','Server host is missing'))
   if not sep or not port(prt): z.append(issue('INVALID_PORT','Server port is missing or invalid'))
   from .ss_deep import audit as ss_audit
   z.extend(ss_audit(raw))
   return z
  if kind=='vmess':
   o=json.loads(b64(raw.split('://',1)[1]).decode());
   if not isinstance(o,dict): return [issue('VMESS_NOT_OBJECT','VMess payload is not an object')]
   if not str(o.get('add','')).strip(): z.append(issue('VMESS_MISSING_ADDRESS','VMess address is missing'))
   if not port(o.get('port')): z.append(issue('VMESS_INVALID_PORT','VMess port is missing or invalid'))
   if not uuid_ok(o.get('id')): z.append(issue('VMESS_INVALID_UUID','VMess UUID is invalid'))
   from .vmess_deep import audit as vmess_audit
   z.extend(vmess_audit(raw))
   return z
  p=urllib.parse.urlsplit(raw); host=p.hostname
  if not host: z.append(issue('MISSING_HOST','Server host is missing'))
  try:
   if p.port is None or not port(p.port): z.append(issue('INVALID_PORT','Server port is missing or invalid'))
  except: z.append(issue('INVALID_PORT','Server port is invalid'))
  user=urllib.parse.unquote(p.username or '')
  if kind=='vless' and not uuid_ok(user): z.append(issue('VLESS_NONSTANDARD_ID','VLESS user id is non-standard; preserved for client compatibility',WARN))
  if kind=='trojan' and not user: z.append(issue('TROJAN_MISSING_PASSWORD','Trojan password is missing'))
  if kind in ('hy','hy2') and not user: z.append(issue('HY_MISSING_AUTH','Hysteria authentication is missing'))
  if kind=='ss' and not (p.username or p.netloc): z.append(issue('SS_MISSING_CREDENTIALS','Shadowsocks credentials are missing'))
 except Exception as e: z.append(issue('LINK_PARSE_ERROR',type(e).__name__))
 if kind=='vless':
  from .vless_deep import audit as vless_audit
  z.extend(vless_audit(raw))
 if kind=='trojan':
  from .trojan_deep import audit as trojan_audit
  z.extend(trojan_audit(raw))
 return z
def json_config(raw):
 try: o=json.loads(raw)
 except Exception as e: return 'json-invalid',[issue('JSON_PARSE_ERROR',type(e).__name__)]
 # Preserve arbitrary valid JSON. Only claim Xray when its core shape is unambiguous.
 if not isinstance(o,dict): return 'json-custom',[]
 if 'outbounds' not in o: return 'json-custom',[]
 if not isinstance(o['outbounds'],list): return 'json-custom',[issue('XRAY_OUTBOUNDS_NOT_ARRAY','outbounds exists but is not an array',WARN)]
 if not o['outbounds']: return 'json-xray',[issue('XRAY_NO_OUTBOUNDS','Xray has no outbound')]
 hard=[]
 for i,x in enumerate(o['outbounds']):
  if not isinstance(x,dict): hard.append(issue('XRAY_OUTBOUND_NOT_OBJECT',f'outbounds[{i}] is not an object')); continue
  proto=x.get('protocol')
  if not isinstance(proto,str) or not proto.strip(): hard.append(issue('XRAY_MISSING_PROTOCOL',f'outbounds[{i}] protocol is missing'))
 # Unknown fields and client-specific shapes are deliberately preserved.
 return 'json-xray',hard
def validate(kind,raw):
 if kind.startswith('json-'): return json_config(raw)[1]
 return link(kind,raw)
def hard_errors(issues): return [x for x in issues if x['level']==HARD]
def xray_deep(raw):
 kind,z=json_config(raw)
 if kind!='json-xray' or hard_errors(z): return z
 o=json.loads(raw); tags=set()
 for i,x in enumerate(o.get('outbounds',[])):
  tag=x.get('tag')
  if tag:
   if tag in tags: z.append(issue('XRAY_DUPLICATE_OUTBOUND_TAG',f'duplicate outbound tag: {tag}'))
   tags.add(tag)
  settings=x.get('settings')
  if settings is not None and not isinstance(settings,dict): z.append(issue('XRAY_SETTINGS_NOT_OBJECT',f'outbounds[{i}].settings should be object',WARN))
  stream=x.get('streamSettings')
  if stream is not None and not isinstance(stream,dict): z.append(issue('XRAY_STREAM_NOT_OBJECT',f'outbounds[{i}].streamSettings must be object'))
 for key in ('inbounds','routing'):
  if key=='inbounds' and key in o and not isinstance(o[key],list): z.append(issue('XRAY_INBOUNDS_NOT_ARRAY','inbounds must be an array'))
 if 'stats' in o and not isinstance(o['stats'],dict): z.append(issue('XRAY_STATS_NOT_OBJECT','stats should be an object',WARN))
 for i,x in enumerate(o.get('inbounds',[])):
  if isinstance(x,dict) and x.get('protocol')=='http' and x.get('settings')==[]: z.append(issue('XRAY_HTTP_INBOUND_SETTINGS_ARRAY',f'inbounds[{i}].settings should be object',WARN))
 for i,x in enumerate(o.get('outbounds',[])):
  if not isinstance(x,dict): continue
  st=x.get('streamSettings')
  if isinstance(st,dict):
   if st.get('sockopt')==[]: z.append(issue('XRAY_SOCKOPT_ARRAY',f'outbounds[{i}].streamSettings.sockopt should be object',WARN))
   if st.get('realitySettings')==[]: z.append(issue('XRAY_REALITY_SETTINGS_ARRAY',f'outbounds[{i}].streamSettings.realitySettings should be object',WARN))
   fm=st.get('finalmask')
   if isinstance(fm,dict):
    for part in fm.get('tcp',[]):
     if isinstance(part,dict) and part.get('type')=='fragment' and isinstance(part.get('settings'),dict):
      fs=part['settings']; vals=list(fs.get('lengths',[]))+list(fs.get('delays',[]))+[fs.get('maxSplit')]
      if any(str(v).strip() in ('0','0-0') for v in vals if v is not None): z.append(issue('XRAY_FINALMASK_ZERO','finalmask fragment contains zero values rejected by current Xray Core; preserved unchanged',WARN)); break
 dns=o.get('dns')
 if isinstance(dns,dict) and 'hosts' in dns and not isinstance(dns['hosts'],dict): z.append(issue('XRAY_DNS_HOSTS_NOT_OBJECT','dns.hosts should be an object',WARN))
 return z
_old_validate=validate
def singbox_config(raw):
 try:o=json.loads(raw)
 except Exception as e:return [issue('SINGBOX_JSON_PARSE_ERROR',type(e).__name__)]
 if not isinstance(o,dict):return [issue('SINGBOX_NOT_OBJECT','sing-box config must be an object')]
 z=[]; outs=o.get('outbounds')
 if not isinstance(outs,list) or not outs:return [issue('SINGBOX_NO_OUTBOUNDS','sing-box outbounds are missing')]
 tags=set()
 for i,x in enumerate(outs):
  if not isinstance(x,dict):z.append(issue('SINGBOX_OUTBOUND_NOT_OBJECT',f'outbounds[{i}] is not an object'));continue
  typ=x.get('type')
  if not isinstance(typ,str) or not typ:z.append(issue('SINGBOX_MISSING_TYPE',f'outbounds[{i}].type is missing'))
  tag=x.get('tag')
  if tag and tag in tags:z.append(issue('SINGBOX_DUPLICATE_TAG',f'duplicate outbound tag: {tag}'))
  if tag:tags.add(tag)
 return z

def validate(kind,raw):
 if kind=='json-xray':
  from .xray_deep import audit
  from .xray_protocol import audit as protocol_audit
  from .xray_system import audit as system_audit
  from .xray_refs import audit as refs_audit
  return xray_deep(raw)+audit(raw)+protocol_audit(raw)+system_audit(raw)+refs_audit(raw)
 if kind=='json-singbox':
  from .singbox_deep import audit as singbox_deep
  from .singbox_protocol import audit as singbox_protocol
  from .singbox_refs import audit as singbox_refs
  return singbox_config(raw)+singbox_deep(raw)+singbox_protocol(raw)+singbox_refs(raw)
 if kind=='json-custom': return json_config(raw)[1]
 return link(kind,raw)
