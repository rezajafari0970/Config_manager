import urllib.parse
from .validator import issue,WARN,port,uuid_ok
KNOWN_TYPES={'tcp','ws','grpc','xhttp','httpupgrade','http','h2','kcp','splithttp'}
def audit(raw):
 z=[]
 try:p=urllib.parse.urlsplit(raw);q=urllib.parse.parse_qs(p.query,keep_blank_values=True)
 except Exception as e:return [issue('VLESS_URI_PARSE_ERROR',type(e).__name__)]
 uid=urllib.parse.unquote(p.username or '')
 if not uid:z.append(issue('VLESS_ID_MISSING','VLESS user id missing'))
 elif not uuid_ok(uid): pass # base validator already emits VLESS_NONSTANDARD_ID
 for k,v in q.items():
  if len(v)>1:z.append(issue('VLESS_DUPLICATE_QUERY',f'duplicate query parameter {k}; first value used',WARN))
 if '%' in p.query:
  import re
  if re.search(r'%(?![0-9A-Fa-f]{2})',p.query):z.append(issue('VLESS_BAD_PERCENT_ENCODING','malformed percent encoding; preserved',WARN))
 typ=(q.get('type',['tcp'])[0] or 'tcp').lower();sec=(q.get('security',['none'])[0] or 'none').lower()
 if typ not in KNOWN_TYPES:z.append(issue('VLESS_UNKNOWN_TRANSPORT',f'unknown transport {typ}; preserved',WARN))
 if sec not in ('none','tls','reality'):z.append(issue('VLESS_UNKNOWN_SECURITY',f'unknown security {sec}; preserved',WARN))
 if typ=='grpc' and not any(k in q for k in ('serviceName','service_name')):z.append(issue('VLESS_GRPC_SERVICE_MISSING','gRPC serviceName missing',WARN))
 if typ in ('ws','xhttp','splithttp','httpupgrade') and 'path' in q and not isinstance(q['path'][0],str):z.append(issue('VLESS_PATH_INVALID','transport path invalid',WARN))
 if sec in ('tls','reality') and not any(q.get(k,[''])[0] for k in ('sni','serverName')):z.append(issue('VLESS_SNI_MISSING',f'{sec} SNI missing; may use address',WARN))
 if sec=='reality' and not any(q.get(k,[''])[0] for k in ('pbk','publicKey','password')):z.append(issue('VLESS_REALITY_KEY_MISSING','Reality public key/password missing',WARN))
 return z