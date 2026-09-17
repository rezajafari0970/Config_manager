import urllib.parse,re
from .validator import issue,WARN
KNOWN={'tcp','ws','grpc','httpupgrade','xhttp'}
def audit(raw):
 z=[]
 try:p=urllib.parse.urlsplit(raw);q=urllib.parse.parse_qs(p.query,keep_blank_values=True)
 except Exception as e:return [issue('TROJAN_URI_PARSE_ERROR',type(e).__name__)]
 typ=(q.get('type',['tcp'])[0] or 'tcp').lower();sec=(q.get('security',['tls'])[0] or 'tls').lower()
 for k,v in q.items():
  if len(v)>1:z.append(issue('TROJAN_DUPLICATE_QUERY',f'duplicate query {k}; first value used',WARN))
 if re.search(r'%(?![0-9A-Fa-f]{2})',p.query):z.append(issue('TROJAN_BAD_PERCENT_ENCODING','malformed percent encoding; preserved',WARN))
 if typ not in KNOWN:z.append(issue('TROJAN_UNKNOWN_TRANSPORT',f'unknown transport {typ}; preserved',WARN))
 if sec not in ('tls','none'):z.append(issue('TROJAN_UNKNOWN_SECURITY',f'unknown security {sec}; preserved',WARN))
 if typ=='grpc' and not any(q.get(k,[''])[0] for k in ('serviceName','service_name')):z.append(issue('TROJAN_GRPC_SERVICE_MISSING','gRPC serviceName missing',WARN))
 if sec=='tls' and not any(q.get(k,[''])[0] for k in ('sni','serverName')):z.append(issue('TROJAN_SNI_MISSING','TLS SNI missing; may use address',WARN))
 return z