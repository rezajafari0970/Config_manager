import urllib.parse,base64
from .validator import issue,WARN,port
def b64key(v):
 try:return len(base64.b64decode(urllib.parse.unquote(v)+'==='))==32
 except:return False
def audit(raw):
 z=[]
 try:p=urllib.parse.urlsplit(raw);q=urllib.parse.parse_qs(p.query,keep_blank_values=True)
 except Exception as e:return [issue('WG_URI_PARSE_ERROR',type(e).__name__)]
 try:prt=p.port
 except:prt=None
 if not p.hostname:z.append(issue('WG_HOST_MISSING','WireGuard endpoint host missing'))
 if not port(prt):z.append(issue('WG_PORT_INVALID','WireGuard endpoint port invalid'))
 private=urllib.parse.unquote(p.username or '') or q.get('privatekey',[''])[0];public=q.get('publickey',[''])[0] or q.get('publicKey',[''])[0]
 if not private:z.append(issue('WG_PRIVATE_KEY_MISSING','WireGuard private key missing'))
 elif not b64key(private):z.append(issue('WG_PRIVATE_KEY_NONSTANDARD','WireGuard private key is non-standard',WARN))
 if not public:z.append(issue('WG_PUBLIC_KEY_MISSING','WireGuard peer public key missing',WARN))
 elif not b64key(public):z.append(issue('WG_PUBLIC_KEY_NONSTANDARD','WireGuard public key is non-standard',WARN))
 return z