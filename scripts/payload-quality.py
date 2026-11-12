#!/usr/bin/env python3
"""Bounded full-payload qualification through SOCKS, never direct.
A finite server-side test, not app/Android/long-term stability evidence.
"""
import json,os,pathlib,subprocess,sys,tempfile,time
VERSION='payload_v1'
DOWNLOAD=2097152
UPLOAD=1048576
def accepted(rc,m,expected,field=None,size=None):
 return rc==0 and m.get('ssl_verify_result')==0 and m.get('http_code') in expected and (field is None or m.get(field)==size)
def request(url,timeout,payload=None,size=None):
 args=['/usr/bin/curl','--noproxy','','--proxy','socks5h://127.0.0.1:10808','--silent','--show-error','--connect-timeout',str(min(timeout,10)),'--max-time',str(timeout),'--max-filesize',str(DOWNLOAD),'--output','/dev/null','--write-out','%{json}','--header','Cache-Control: no-cache']
 if payload:args+=['--header','Content-Type: application/octet-stream','--data-binary','@'+str(payload)]
 args+=[url]
 try:
  r=subprocess.run(args,capture_output=True,timeout=timeout+2,env={'PATH':'/usr/bin:/usr/sbin'})
  try:m=json.loads(r.stdout)
  except ValueError:m={}
  rc=r.returncode
 except (OSError,subprocess.SubprocessError):m={};rc=124
 field='size_upload' if payload else 'size_download' if size is not None else None
 wanted=UPLOAD if payload else size
 passed=accepted(rc,m,{204} if size is None and payload is None else {200},field,wanted)
 return {'status':'passed' if passed else 'failed','bytes':int(m.get(field,0)) if field else 0,'ms':round(m.get('time_total',0)*1000),'curl_exit':rc}
def run(timeout):
 timeout=max(20,min(45,int(timeout)))
 with tempfile.TemporaryDirectory(prefix='config-payload-',dir='/dev/shm') as folder:
  payload=pathlib.Path(folder)/'upload.bin';payload.write_bytes(os.urandom(UPLOAD));payload.chmod(0o600)
  down=request('https://speed.cloudflare.com/__down?bytes='+str(DOWNLOAD)+'&nonce='+str(time.time_ns()),timeout,size=DOWNLOAD)
  up=request('https://speed.cloudflare.com/__up',timeout,payload=payload)
  check=request('https://www.gstatic.com/generate_204',min(timeout,10))
 result={'payload_version':VERSION,'payload_download':down['status'],'payload_upload':up['status'],'payload_recheck':check['status']}
 for key,row in [('download',down),('upload',up)]:
  for field in ('bytes','ms','curl_exit'):result['payload_'+key+'_'+field]=str(row[field])
 return result
def verified(e):
 return e.get('payload_version')==VERSION and all(e.get('payload_'+k)=='passed' for k in ('download','upload','recheck')) and e.get('payload_download_bytes')==str(DOWNLOAD) and e.get('payload_upload_bytes')==str(UPLOAD)
if __name__=='__main__':print(json.dumps(run(int(sys.argv[1]))))
