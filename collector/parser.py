import base64,hashlib,json,re
from .validator import validate,json_config,hard_errors
SCHEMES={'ss':'ss','trojan':'trojan','vless':'vless','vmess':'vmess','wireguard':'wireguard','wg':'wireguard','hy2':'hy2','hysteria2':'hy2','hy':'hy','hysteria':'hy'}
def b64d(s): return base64.urlsafe_b64decode(s.strip()+'='*((4-len(s.strip())%4)%4)).decode('utf8','strict')
def extract(text):
 bodies=[text]
 if text.strip() and not any(s+'://' in text.lower() for s in SCHEMES):
  try:
   d=b64d(text.strip())
   if '://' in d or d.lstrip().startswith(('{','[')): bodies.append(d)
  except: pass
 found=[]
 for body in bodies:
  for m in re.finditer(r'(?i)(?:vless|vmess|trojan|ss|wireguard|wg|hy2|hysteria2|hy|hysteria)://[^\s<>"\']+',body):
   raw=m.group(0).strip().rstrip(',;'); kind=SCHEMES[raw.split(':',1)[0].lower()]; found.append((kind,raw,validate(kind,raw)))
  b=body.strip()
  if b.startswith(('{','[')):
   kind,issues=json_config(b)
   if kind!='json-invalid':
    try:
     jo=json.loads(b)
     if isinstance(jo,dict) and isinstance(jo.get('outbounds'),list) and jo['outbounds'] and all(isinstance(x,dict) for x in jo['outbounds']) and any('type' in x for x in jo['outbounds']) and not any('protocol' in x for x in jo['outbounds']): kind='json-singbox'
    except: pass
    found.append((kind,b,validate(kind,b)))
 out=[]; seen=set()
 for kind,raw,issues in found:
  fp=hashlib.sha256(raw.encode()).hexdigest()
  if fp in seen: continue
  seen.add(fp); out.append({'fingerprint':fp,'kind':kind,'raw':raw,'issues':issues,'hard':hard_errors(issues)})
 return out
