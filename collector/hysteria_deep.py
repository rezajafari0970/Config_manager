import urllib.parse,re
from .validator import issue,WARN,port
def audit(kind,raw):
 z=[]
 try:p=urllib.parse.urlsplit(raw);q=urllib.parse.parse_qs(p.query,keep_blank_values=True)
 except Exception as e:return [issue('HY_URI_PARSE_ERROR',type(e).__name__)]
 if not p.hostname:z.append(issue('HY_MISSING_HOST','Hysteria host missing'))
 try:prt=p.port
 except:prt=None
 if not port(prt):z.append(issue('HY_INVALID_PORT','Hysteria port invalid'))
 for k,v in q.items():
  if len(v)>1:z.append(issue('HY_DUPLICATE_QUERY',f'duplicate query {k}; preserved',WARN))
 if re.search(r'%(?![0-9A-Fa-f]{2})',p.query):z.append(issue('HY_BAD_PERCENT_ENCODING','malformed percent encoding; preserved',WARN))
 if kind in ('hy2','hysteria2'):
  pw=urllib.parse.unquote(p.username or '') or (q.get('password',[''])[0])
  if not pw:z.append(issue('HY2_AUTH_MISSING','Hysteria2 password/auth missing'))
 else:
  if not any(q.get(k,[''])[0] for k in ('auth','auth_str','authString')):z.append(issue('HY1_AUTH_MISSING','Hysteria auth missing',WARN))
 if not any(q.get(k,[''])[0] for k in ('sni','peer')):z.append(issue('HY_SNI_MISSING','TLS SNI/peer missing; may use address',WARN))
 return z