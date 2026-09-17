import base64,hashlib,json,re,urllib.parse
SCHEMES={'ss':'ss','trojan':'trojan','vless':'vless','vmess':'vmess','wireguard':'wireguard','wg':'wireguard','hy2':'hy2','hysteria2':'hy2','hy':'hy','hysteria':'hy'}
def b64decode(s):
 s=s.strip(); return base64.urlsafe_b64decode(s+'='*((4-len(s)%4)%4)).decode('utf-8','ignore')
def valid_link(x):
 try:
  p=urllib.parse.urlsplit(x); return bool(p.scheme.lower() in SCHEMES and (p.hostname or p.scheme.lower()=='vmess'))
 except: return False
def classify_json(o):
 if isinstance(o,dict) and isinstance(o.get('outbounds'),list): return 'json-xray'
 if isinstance(o,(dict,list)): return 'json-custom'
def extract(text):
 text=text.strip(); candidates=[text]
 if text and not any(s+'://' in text.lower() for s in SCHEMES):
  try:
   d=b64decode(text)
   if '://' in d or d.lstrip().startswith(('{','[')): candidates.append(d)
  except: pass
 out=[]
 for body in candidates:
  for m in re.finditer(r'(?i)(?:vless|vmess|trojan|ss|wireguard|wg|hy2|hysteria2|hy|hysteria)://[^\s<>"\']+',body):
   raw=m.group(0).strip().rstrip(',;'); scheme=raw.split(':',1)[0].lower()
   if valid_link(raw): out.append((SCHEMES[scheme],raw))
  try:
   o=json.loads(body); k=classify_json(o)
   if k: out.append((k,json.dumps(o,separators=(',',':'),ensure_ascii=False,sort_keys=True)))
  except: pass
 seen={};
 for k,r in out: seen.setdefault(hashlib.sha256(r.encode()).hexdigest(),(k,r))
 return [(fp,*v) for fp,v in seen.items()]
