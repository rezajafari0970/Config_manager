#!/usr/bin/env python3
"""Bounded, strict HTTPS bootstrap-asset checks. No ad requests or impressions.
An available asset is necessary network evidence, not an ad-fill/render claim.
"""
import concurrent.futures,hashlib,json,os,pathlib,secrets,re,subprocess,sys,tempfile,time
VERSION='ads-delivery.v1'
LIMIT=524288
def registry(values):
 if not isinstance(values,list) or not 1<=len(values)<=8:raise ValueError('delivery_registry_invalid')
 seen=set()
 for t in values:
  if not isinstance(t,dict) or set(t)!={'profile','domain','path','marker'} or not re.fullmatch(r'[a-z0-9.-]{1,80}',t['profile']) or t['profile'] in seen or not re.fullmatch(r'[a-z0-9.-]{1,253}',t['domain']) or not re.fullmatch(r'/[a-zA-Z0-9/_.-]{1,160}',t['path']) or not re.fullmatch(r'[a-zA-Z0-9_]{1,64}',t['marker']):raise ValueError('delivery_registry_invalid')
  seen.add(t['profile'])
 return values
def transfer(t,timeout,host=False):
 row=dict(profile=t['profile'],domain=t['domain'],path=t['path'],nonce=secrets.token_hex(16),http_status=0,curl_exit=124,tls_verified=False,complete=False,content_match=False,bytes=0,body_sha='',code='transport_unavailable')
 with tempfile.TemporaryDirectory(prefix='ads-asset-',dir='/dev/shm') as folder:
  body=pathlib.Path(folder)/'body'
  args=['/usr/bin/curl','--noproxy','*' if host else '','--silent','--http1.1','--connect-timeout',str(timeout),'--max-time',str(timeout),'--max-filesize',str(LIMIT),'--output',str(body),'--write-out','%{json}','--header','Accept-Encoding: identity','--header','Cache-Control: no-cache']
  if not host:args+=['--proxy','socks5h://127.0.0.1:10808']
  args+=['https://'+t['domain']+t['path']]
  try:
   p=subprocess.run(args,capture_output=True,timeout=timeout+1,env={'PATH':'/usr/bin:/usr/sbin'});m=json.loads(p.stdout);b=body.read_bytes() if body.exists() else b''
   if len(b)>LIMIT:raise ValueError('body_limit')
   mime=str(m.get('content_type','')).split(';')[0].strip().lower()
   complete=p.returncode==0 and int(m.get('size_download',0))==len(b)
   matched=mime in ('text/javascript','application/javascript','application/x-javascript') and t['marker'].encode() in b and len(b)>=1024
   row.update(curl_exit=p.returncode,http_status=int(m.get('http_code',0)),tls_verified=m.get('ssl_verify_result')==0,complete=complete,content_match=matched,bytes=len(b),body_sha=hashlib.sha256(b).hexdigest())
   row['code']='asset_available' if complete and matched and row['http_status']==200 and row['tls_verified'] else 'asset_unavailable'
  except Exception:pass
 return row
def run(req):
 targets=registry(req.get('targets'));timeout=req.get('timeout',3)
 if timeout not in (3,6,8):raise ValueError('delivery_timeout')
 host=req.get('host_control') is True
 with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:assets=list(pool.map(lambda t:transfer(t,timeout,host),targets))
 return dict(version=VERSION,variant='host' if host else req['variant'],host_control=host,at=int(time.time()*1000),assets=assets)
if __name__=='__main__':print(json.dumps(run(json.load(sys.stdin))))
