"""Bounded HTTP/3 payload probes through SOCKS UDP; no TCP/direct fallback."""
import ipaddress,json,pathlib,select,shutil,socket,struct,subprocess,tempfile,threading,time
from .quic import t

VERSION='http3-payload.v1'
DOWNLOAD=2097152
UPLOAD=1048576
HOST='speed.cloudflare.com'

def accepted(rc,metrics,field,size):
    return (rc==0 and str(metrics.get('http_version'))=='3' and
            metrics.get('http_code')==200 and metrics.get('ssl_verify_result')==0 and
            metrics.get(field)==size)

def transfer(port,address,upload=False,timeout=20):
    ip=ipaddress.ip_address(address)
    if ip.version!=4 or not ip.is_global:raise ValueError('public_ipv4_target_required')
    if not 1<=port<=65535 or not 1<=timeout<=45:raise ValueError('invalid_probe_bound')
    curl=shutil.which('curl')
    if not curl or 'HTTP3' not in subprocess.run([curl,'--version'],capture_output=True,text=True,timeout=3).stdout:
        return {'status':'not_run','code':'curl_http3_unavailable','driver':VERSION}
    halt=threading.Event();control=None;worker=None
    counts={'sent':0,'received':0,'unexpected':0}
    try:
        control,relay=t.socks(3,'0.0.0.0',0,timeout,port)
        if relay[0] in ('0.0.0.0','::'):relay=('127.0.0.1',relay[1])
        if relay[0]!='127.0.0.1':raise ValueError('non_loopback_socks_relay')
        prefix=b'\x00\x00\x00\x01'+ip.packed+struct.pack('!H',443)
        with socket.socket(socket.AF_INET,socket.SOCK_DGRAM) as upstream,socket.socket(socket.AF_INET,socket.SOCK_DGRAM) as local,tempfile.TemporaryDirectory(prefix='shen-h3-payload-') as tmp:
            upstream.connect(relay);local.bind(('127.0.0.1',0));peer=[None]
            for sock in (upstream,local):
                for option in (socket.SO_RCVBUF,socket.SO_SNDBUF):
                    try:sock.setsockopt(socket.SOL_SOCKET,option,1048576)
                    except OSError:pass
            def bridge():
                while not halt.is_set() and sum(counts.values())<32768:
                    for sock in select.select([local,upstream],[],[],.1)[0]:
                        try:
                            data,addr=sock.recvfrom(65535)
                            if sock is local:
                                if peer[0] is None:peer[0]=addr
                                if addr==peer[0]:upstream.send(prefix+data);counts['sent']+=1
                                else:counts['unexpected']+=1
                            elif peer[0] and data.startswith(prefix):
                                local.sendto(data[len(prefix):],peer[0]);counts['received']+=1
                            else:counts['unexpected']+=1
                        except OSError:return
            worker=threading.Thread(target=bridge,daemon=True);worker.start()
            body=pathlib.Path(tmp)/'response'
            url='https://'+HOST+('/__up' if upload else '/__down?bytes='+str(DOWNLOAD)+'&nonce='+str(time.time_ns()))
            cmd=[curl,'--http3-only','--proxy','','--noproxy','*','--connect-to',HOST+':443:127.0.0.1:'+str(local.getsockname()[1]),'--max-time',str(timeout),'--connect-timeout',str(min(timeout,8)),'--silent','--show-error','--max-filesize',str(DOWNLOAD),'--output',str(body),'--write-out','%{json}']
            if upload:
                payload=pathlib.Path(tmp)/'upload';payload.write_bytes(b'x'*UPLOAD);payload.chmod(0o600)
                cmd+=['-H','Content-Type: application/octet-stream','--data-binary','@'+str(payload)]
            cmd+=[url]
            try:
                p=subprocess.run(cmd,capture_output=True,timeout=timeout+2);m=json.loads(p.stdout or '{}')
                field='size_upload' if upload else 'size_download';size=UPLOAD if upload else DOWNLOAD
                ok=accepted(p.returncode,m,field,size)
                return {'status':'passed' if ok else 'failed','driver':VERSION,'direction':'upload' if upload else 'download','http_version':m.get('http_version'),'http_status':m.get('http_code'),'curl_exit':p.returncode,'tls':'verified' if p.returncode==0 and m.get('ssl_verify_result')==0 else 'unverified','bytes':m.get(field,0),'milliseconds':round(m.get('time_total',0)*1000,2),'mbps':round(m.get('speed_upload' if upload else 'speed_download',0)*8/1e6,4),'datagrams':dict(counts),'packet_loss':'not_measured','evidence':'verified HTTP/3 exact submitted bytes and 200 acknowledgement; upload is not an echo hash'}
            finally:halt.set();worker.join(timeout=1)
    except (OSError,ValueError,subprocess.SubprocessError) as e:
        return {'status':'failed','code':'http3_payload_probe_error','error_type':type(e).__name__,'driver':VERSION}
    finally:
        halt.set()
        if control:control.close()
