#!/usr/bin/env python3
"""Independent bounded upload/download health contract for Residential Ads.
Run inside its sandbox. Host mode is an outage control, never VPN health proof.
"""
import concurrent.futures,json,os,pathlib,subprocess,sys,tempfile,time
VERSION='ads-health.v1';SIZE=65536
TARGETS={'cloudflare':('https://speed.cloudflare.com/__down?bytes=65536','https://speed.cloudflare.com/__up'),'httpbingo':('https://httpbingo.org/bytes/65536','https://httpbingo.org/post')}
def transfer(url,payload=None,host=False):
 args=['/usr/bin/curl','--noproxy','*' if host else '','--silent','--connect-timeout','3','--max-time','7','--max-filesize','262144','--output','/dev/null','--write-out','%{json}','--header','Cache-Control: no-cache']
 if not host:args+=['--proxy','socks5h://127.0.0.1:10808']
 if payload:args+=['--header','Content-Type: application/octet-stream','--data-binary','@'+str(payload)]
 args+=[url]
 try:
  p=subprocess.run(args,capture_output=True,timeout=9,env={'PATH':'/usr/bin:/usr/sbin'});m=json.loads(p.stdout);rc=p.returncode
 except Exception:m={};rc=124
 field='size_upload' if payload else 'size_download'
 return dict(bytes=int(m.get(field,0)),curl_exit=rc,http_status=int(m.get('http_code',0)),tls_verified=m.get('ssl_verify_result')==0,ms=round(m.get('time_total',0)*1000))
def passed(x):return x['curl_exit']==0 and x['tls_verified'] and x['http_status']==200 and x['bytes']==SIZE
def run(host=False):
 with tempfile.TemporaryDirectory(prefix='ads-health-',dir='/dev/shm') as folder:
  payload=pathlib.Path(folder)/'upload';payload.write_bytes(os.urandom(SIZE));payload.chmod(0o600)
  def provider(item):
   name,(down,up)=item
   with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
    a=pool.submit(transfer,down,None,host);b=pool.submit(transfer,up,payload,host);d,u=a.result(),b.result()
   return dict(provider=name,download=d,upload=u,passed=passed(d) and passed(u))
  with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:rows=list(pool.map(provider,TARGETS.items()))
 return dict(version=VERSION,at=int(time.time()*1000),host_control=host,providers=rows)
if __name__=='__main__':print(json.dumps(run('--host-control' in sys.argv)))
