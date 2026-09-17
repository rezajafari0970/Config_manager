import json
from .validator import issue,WARN,b64,port,uuid_ok
KNOWN_NET={'tcp','ws','grpc','h2','http','kcp','quic'}
def decode(raw):return json.loads(b64(raw.split('://',1)[1].split('#',1)[0]).decode('utf8'))
def audit(raw):
 z=[]
 try:o=decode(raw)
 except Exception as e:return [issue('VMESS_DECODE_ERROR',type(e).__name__)]
 if not isinstance(o,dict):return [issue('VMESS_NOT_OBJECT','VMess payload is not object')]
 net=str(o.get('net') or 'tcp').lower();tls=str(o.get('tls') or '').lower()
 if net not in KNOWN_NET:z.append(issue('VMESS_UNKNOWN_NETWORK',f'unknown network {net}; preserved',WARN))
 if tls not in ('','none','tls'):z.append(issue('VMESS_UNKNOWN_TLS',f'unknown tls mode {tls}; preserved',WARN))
 if net=='grpc' and not (o.get('path') or o.get('serviceName')):z.append(issue('VMESS_GRPC_SERVICE_MISSING','gRPC service name missing',WARN))
 if net in ('ws','h2','http') and 'path' in o and not isinstance(o.get('path'),str):z.append(issue('VMESS_PATH_INVALID','transport path invalid'))
 aid=o.get('aid',0)
 try:int(aid)
 except:z.append(issue('VMESS_ALTERID_INVALID','alterId is not integer',WARN))
 scy=str(o.get('scy') or o.get('security') or 'auto').lower()
 if not scy:z.append(issue('VMESS_SECURITY_EMPTY','VMess security empty',WARN))
 return z