"""Bounded verified HTTPS probes; destination status and transport are separate."""
import json,os,pathlib,shutil,subprocess,time
from .quality import PAYLOAD_VERSION, DOWNLOAD_BYTES, UPLOAD_BYTES, complete
TARGETS={
 'google':('https://www.gstatic.com/generate_204',{204}),
 'instagram':('https://www.instagram.com/robots.txt',{200}),
 'telegram':('https://telegram.org/',{200}),
 'telegram_web':('https://web.telegram.org/',{200}),
 'youtube':('https://www.youtube.com/generate_204',{204}),
 'youtube_image':('https://i.ytimg.com/vi/jNQXAC9IVRw/hqdefault.jpg',{200}),
 'play':('https://play.google.com/robots.txt',{200}),
}
def request(port,url,expected=(200,),timeout=6,payload=None,download_bytes=None,http1=False):
    curl=shutil.which('curl')
    if not curl:return {'status':'unavailable','code':'curl_missing'}
    cmd=[curl,'--noproxy','','--proxy','socks5h://127.0.0.1:'+str(port),'--silent','--show-error','--connect-timeout',str(min(3,timeout)),'--max-time',str(timeout),'--max-filesize','2097152','--output','/dev/null','--write-out','%{json}','-H','Cache-Control: no-cache']
    if http1:cmd+=['--http1.1']
    if payload is not None:cmd+=['-H','Content-Type: application/octet-stream','--data-binary','@'+str(payload)]
    cmd+=[url];started=time.monotonic()
    try:
        p=subprocess.run(cmd,capture_output=True,timeout=timeout+2)
        try:m=json.loads(p.stdout)
        except ValueError:m={}
        rc=p.returncode
    except subprocess.TimeoutExpired:m={};rc=124
    transport=rc==0 and m.get('ssl_verify_result')==0 and m.get('http_code',0)>0
    accepted=transport and m.get('http_code') in expected
    if download_bytes is not None:accepted=accepted and m.get('size_download')==download_bytes
    if payload is not None:accepted=accepted and m.get('size_upload')==pathlib.Path(payload).stat().st_size
    return {'status':'passed' if accepted else 'failed','transport_tls':'verified' if transport else 'unverified','curl_exit':rc,'http_status':m.get('http_code',0),'http_version':m.get('http_version'),'total_ms':round(m.get('time_total',time.monotonic()-started)*1000,2),'ttfb_ms':round(m.get('time_starttransfer',0)*1000,2),'local_socks_connect_ms':round(m.get('time_connect',0)*1000,2),'socks_and_tls_ms':round(m.get('time_appconnect',0)*1000,2),'download_bytes':m.get('size_download',0),'upload_bytes':m.get('size_upload',0),'download_mbps':round(m.get('speed_download',0)*8/1e6,4),'upload_mbps':round(m.get('speed_upload',0)*8/1e6,4),'dns_timing_scope':'curl local/SOCKS setup; remote DNS measured separately','functional_app_test':'not_run'}
def site(port,name,timeout=6):
    url,status=TARGETS[name];return {'name':name,**request(port,url,status,timeout)}
def transfers(port,work,timeout=8):
    work=pathlib.Path(work);payload=work/'upload.bin';payload.write_bytes(os.urandom(UPLOAD_BYTES))
    try:
        down=request(port,'https://speed.cloudflare.com/__down?bytes='+str(DOWNLOAD_BYTES)+'&nonce='+str(time.time_ns()),timeout=timeout,download_bytes=DOWNLOAD_BYTES)
        up=request(port,'https://speed.cloudflare.com/__up',timeout=timeout,payload=payload)
        post=request(port,'https://www.gstatic.com/generate_204',expected=(204,),timeout=min(timeout,8))
        result={'version':PAYLOAD_VERSION,'download':down,'upload':up,'post_load':post,'both_passed':down['status']==up['status']=='passed','upload_evidence':'verified HTTPS 2xx acknowledgement and exact submitted byte count; not a server-side payload hash'}
        result['complete']=complete(result)
        return result
    finally:payload.unlink(missing_ok=True)
