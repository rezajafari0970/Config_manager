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
  if kind=='vmess':
   o=json.loads(b64(raw.split('://',1)[1]).decode());
   if not isinstance(o,dict): return [issue('VMESS_NOT_OBJECT','VMess payload is not an object')]
   if not str(o.get('add','')).strip(): z.append(issue('VMESS_MISSING_ADDRESS','VMess address is missing'))
   if not port(o.get('port')): z.append(issue('VMESS_INVALID_PORT','VMess port is missing or invalid'))
   if not uuid_ok(o.get('id')): z.append(issue('VMESS_INVALID_UUID','VMess UUID is invalid'))
   return z
  p=urllib.parse.urlsplit(raw); host=p.hostname
  if not host: z.append(issue('MISSING_HOST','Server host is missing'))
  try:
   if p.port is None or not port(p.port): z.append(issue('INVALID_PORT','Server port is missing or invalid'))
  except: z.append(issue('INVALID_PORT','Server port is invalid'))
  user=urllib.parse.unquote(p.username or '')
  if kind=='vless' and not uuid_ok(user): z.append(issue('VLESS_INVALID_UUID','VLESS UUID is invalid'))
  if kind=='trojan' and not user: z.append(issue('TROJAN_MISSING_PASSWORD','Trojan password is missing'))
  if kind in ('hy','hy2') and not user: z.append(issue('HY_MISSING_AUTH','Hysteria authentication is missing'))
  if kind=='ss' and not (p.username or p.netloc): z.append(issue('SS_MISSING_CREDENTIALS','Shadowsocks credentials are missing'))
 except Exception as e: z.append(issue('LINK_PARSE_ERROR',type(e).__name__))
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
