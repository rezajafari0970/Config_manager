"""Verified HTTP/3 DoH through a loopback SOCKS UDP relay using HTTP/3 curl.
No TCP/direct fallback. The bridge admits only the fixed resolver reply.
"""
import importlib.util,json,pathlib,select,shutil,socket,struct,subprocess,tempfile,threading
spec=importlib.util.spec_from_file_location('bench_transport',pathlib.Path(__file__).resolve().parents[1]/'transport-checks.py')
t=importlib.util.module_from_spec(spec);spec.loader.exec_module(t)

def check(port,timeout=8):
    curl=shutil.which('curl')
    if not curl or 'HTTP3' not in subprocess.run([curl,'--version'],capture_output=True,text=True,timeout=3).stdout:
        return {'status':'not_run','code':'curl_http3_unavailable'}
    halt=threading.Event();worker=None;control=None
    try:
        control,relay=t.socks(3,'0.0.0.0',0,timeout,port)
        if relay[0] in ('0.0.0.0','::'):relay=('127.0.0.1',relay[1])
        if relay[0]!='127.0.0.1':raise ValueError('non_loopback_socks_relay')
        ident,query=t.dns_query();prefix=b'\x00\x00\x00\x01'+socket.inet_aton('1.1.1.1')+struct.pack('!H',443)
        with socket.socket(socket.AF_INET,socket.SOCK_DGRAM) as upstream,socket.socket(socket.AF_INET,socket.SOCK_DGRAM) as local,tempfile.TemporaryDirectory(prefix='shen-h3-') as tmp:
            upstream.connect(relay);local.bind(('127.0.0.1',0));peer=[None];count=[0]
            def bridge():
                while not halt.is_set() and count[0]<4096:
                    for sock in select.select([local,upstream],[],[],.1)[0]:
                        try:
                            data,addr=sock.recvfrom(65535);count[0]+=1
                            if sock is local:
                                if peer[0] is None:peer[0]=addr
                                if addr==peer[0]:upstream.send(prefix+data)
                            elif peer[0] and data.startswith(prefix):local.sendto(data[len(prefix):],peer[0])
                        except OSError:return
            worker=threading.Thread(target=bridge,daemon=True);worker.start();body=pathlib.Path(tmp)/'answer'
            cmd=[curl,'--http3-only','--proxy','','--noproxy','*','--connect-to','cloudflare-dns.com:443:127.0.0.1:'+str(local.getsockname()[1]),'--max-time',str(timeout),'--connect-timeout',str(timeout),'--silent','--max-filesize','65536','--output',str(body),'--write-out','%{json}','-H','Content-Type: application/dns-message','--data-binary','@-','https://cloudflare-dns.com/dns-query']
            try:
                p=subprocess.run(cmd,input=query,capture_output=True,timeout=timeout+2);m=json.loads(p.stdout or '{}');payload=body.read_bytes() if body.exists() else b''
                passed=p.returncode==0 and str(m.get('http_version'))=='3' and m.get('http_code')==200 and m.get('ssl_verify_result')==0 and t.dns_ok(payload,ident)
                return {'status':'passed' if passed else 'failed','code':'http3_doh_response','driver':'curl-socks-udp.v1','http_version':m.get('http_version'),'http_status':m.get('http_code'),'curl_exit':p.returncode,'tls':'verified' if p.returncode==0 and m.get('ssl_verify_result')==0 else 'unverified','milliseconds':round(m.get('time_total',0)*1000,2),'packets_forwarded':count[0]}
            finally:
                halt.set();worker.join(timeout=1)
    except (OSError,ValueError,subprocess.SubprocessError):return {'status':'failed','code':'http3_probe_error','driver':'curl-socks-udp.v1'}
    finally:
        halt.set()
        if control:control.close()
