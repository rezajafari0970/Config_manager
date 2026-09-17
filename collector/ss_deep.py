import urllib.parse
from .validator import issue,WARN,b64
KNOWN={'aes-128-gcm','aes-192-gcm','aes-256-gcm','chacha20-ietf-poly1305','xchacha20-ietf-poly1305','2022-blake3-aes-128-gcm','2022-blake3-aes-256-gcm','2022-blake3-chacha20-poly1305','none'}
def parse(raw):
 body=raw.split('://',1)[1];main=body.split('#',1)[0];main=main.split('?',1)[0]
 if '@' not in main:main=b64(main).decode('utf8')
 creds,server=main.rsplit('@',1)
 try:dec=b64(creds).decode('utf8') if ':' not in creds else urllib.parse.unquote(creds)
 except:dec=urllib.parse.unquote(creds)
 method,sep,password=dec.partition(':');host,_,prt=server.rpartition(':');return method,password,host,int(prt)
def audit(raw):
 try:m,p,h,prt=parse(raw)
 except Exception:return [] # base validator owns opaque/non-SIP002 compatibility
 z=[]
 if not m:z.append(issue('SS_METHOD_MISSING','Shadowsocks method missing'))
 elif m.lower() not in KNOWN:z.append(issue('SS_UNKNOWN_METHOD',f'unknown method {m}; preserved',WARN))
 if p=='':z.append(issue('SS_PASSWORD_EMPTY','Shadowsocks password empty',WARN))
 return z