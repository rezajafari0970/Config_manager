import json,subprocess,urllib.parse
def getjson(port,url):
 p=subprocess.run(['/usr/bin/curl','-fsS','--max-time','8','--socks5-hostname',f'127.0.0.1:{port}',url],stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,text=True);return json.loads(p.stdout) if p.returncode==0 else {}
def flag(cc):return ''.join(chr(127397+ord(c)) for c in cc.upper()) if len(cc)==2 else '🌐'
def enrich(port):
 a=getjson(port,'https://ipwho.is/');ip=a.get('ip');cc=a.get('country_code') or '';country=a.get('country') or 'Unknown';conn=a.get('connection') or {}
 b=getjson(port,'https://ipapi.co/json/');asn=str(conn.get('asn') or b.get('asn') or '');org=conn.get('org') or b.get('org') or ''
 hosting=bool(a.get('hosting') or a.get('proxy'));cdn='cdn' if any(x in (org or '').lower() for x in ('cloudflare','akamai','fastly','cloudfront','cdn')) else ('hosting' if hosting else 'non-cdn')
 dc=org or 'Unknown';f=flag(cc);return {'egress_ip':ip,'country_code':cc.upper(),'country_name':country,'country_flag':f,'city':a.get('city') or b.get('city'),'region':a.get('region') or b.get('region'),'asn':asn,'network_org':org,'datacenter':dc,'cdn_state':cdn,'remark':f+' '+country}