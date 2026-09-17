import base64,hashlib,json,re,urllib.parse,uuid
SCHEMES={'ss':'ss','trojan':'trojan','vless':'vless','vmess':'vmess','wireguard':'wireguard','wg':'wireguard','hy2':'hy2','hysteria2':'hy2','hy':'hy','hysteria':'hy'}
def b64d(s): return base64.urlsafe_b64decode(s.strip()+'='*((4-len(s.strip())%4)%4)).decode('utf8','strict')
def validate(kind,raw):
 issues=[]; fixed=raw.strip()
 try:
  if kind=='vmess':
   payload=raw.split('://',1)[1]; o=json.loads(b64d(payload));
   if not o.get('add'): issues.append('VMESS_MISSING_ADDRESS')
   if not o.get('port'): issues.append('VMESS_MISSING_PORT')
   if not o.get('id'): issues.append('VMESS_MISSING_UUID')
   elif not _uuid(o['id']): issues.append('VMESS_INVALID_UUID')
  elif kind in ('json-xray','json-custom'): json.loads(raw)
  else:
   p=urllib.parse.urlsplit(raw)
   if not p.scheme or not p.hostname: issues.append('MISSING_HOST')
   try:
    if p.port is not None and not 1<=p.port<=65535: issues.append('INVALID_PORT')
   except ValueError: issues.append('INVALID_PORT')
   if kind=='vless' and (not p.username or not _uuid(urllib.parse.unquote(p.username))): issues.append('VLESS_INVALID_UUID')
   if kind=='trojan' and not p.username: issues.append('TROJAN_MISSING_PASSWORD')
 except Exception as e: issues.append('PARSE_ERROR:'+type(e).__name__)
 return fixed,issues
def _uuid(x):
 try: uuid.UUID(str(x)); return True
 except: return False
def classify_json(o):
 if isinstance(o,dict) and isinstance(o.get('outbounds'),list): return 'json-xray'
 if isinstance(o,(dict,list)): return 'json-custom'
def extract(text):
 original=text; bodies=[text]
 if text.strip() and not any(s+'://' in text.lower() for s in SCHEMES):
  try:
   d=b64d(text.strip())
   if '://' in d or d.lstrip().startswith(('{','[')): bodies.append(d)
  except: pass
 found=[]
 for body in bodies:
  for m in re.finditer(r'(?i)(?:vless|vmess|trojan|ss|wireguard|wg|hy2|hysteria2|hy|hysteria)://[^\s<>"\']+',body):
   raw=m.group(0).strip().rstrip(',;'); kind=SCHEMES[raw.split(':',1)[0].lower()]; fixed,issues=validate(kind,raw); found.append((kind,raw,fixed,issues))
  try:
   o=json.loads(body); kind=classify_json(o)
   if kind:
    raw=body.strip(); fixed,issues=validate(kind,raw); found.append((kind,raw,fixed,issues))
  except: pass
 out=[]; seen=set()
 for kind,raw,fixed,issues in found:
  fp=hashlib.sha256(raw.encode()).hexdigest()
  if fp not in seen: seen.add(fp); out.append({'fingerprint':fp,'kind':kind,'raw':raw,'fixed':fixed,'issues':issues})
 return out
