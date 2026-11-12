#!/usr/bin/env python3
"""Bounded transport evidence through an existing SOCKS tunnel; no direct fallback."""
import shutil,argparse,concurrent.futures,contextlib,json,os,secrets,socket,struct,subprocess,time
VERSION='transport-evidence.v1'
def exact(s,n):
    b=b''
    while len(b)<n:
        part=s.recv(n-len(b))
        if not part:raise OSError('short_read')
        b+=part
    return b
def address(host,port):
    try:a=b'\x01'+socket.inet_aton(host)
    except OSError:
        raw=host.encode('idna')
        if len(raw)>255:raise ValueError('hostname_length')
        a=b'\x03'+bytes([len(raw)])+raw
    return a+struct.pack('!H',port)
def read_address(s,kind):
    if kind==1:host=socket.inet_ntoa(exact(s,4))
    elif kind==3:host=exact(s,exact(s,1)[0]).decode('ascii')
    elif kind==4:host=socket.inet_ntop(socket.AF_INET6,exact(s,16))
    else:raise OSError('socks_address')
    return host,struct.unpack('!H',exact(s,2))[0]
def socks(command,host,port,timeout,socks_port=10808):
    s=socket.create_connection(('127.0.0.1',socks_port),timeout);s.settimeout(timeout)
    try:
        s.sendall(b'\x05\x01\x00')
        if exact(s,2)!=b'\x05\x00':raise OSError('socks_auth')
        s.sendall(b'\x05'+bytes([command])+b'\x00'+address(host,port))
        head=exact(s,4)
        if head[:3]!=b'\x05\x00\x00':raise OSError('socks_rejected')
        bound=read_address(s,head[3])
        return s,bound
    except BaseException:s.close();raise
def dns_query():
    ident=secrets.token_bytes(2)
    question=b'\x07example\x03com\x00'+struct.pack('!HH',1,1)
    return ident,ident+struct.pack('!HHHHH',0x100,1,0,0,0)+question
def dns_ok(data,ident):
    return len(data)>=12 and data[:2]==ident and data[2]&0x80!=0 and data[3]&15==0 and struct.unpack('!H',data[6:8])[0]>0
def udp_exchange(host,port,data,timeout,socks_port=10808):
    control,bound=socks(3,'0.0.0.0',0,timeout,socks_port)
    with control,socket.socket(socket.AF_INET,socket.SOCK_DGRAM) as u:
        u.settimeout(timeout)
        if bound[0] in ('0.0.0.0','::'):bound=('127.0.0.1',bound[1])
        u.sendto(b'\x00\x00\x00'+address(host,port)+data,bound)
        reply,_=u.recvfrom(65535)
        if len(reply)<4 or reply[:3]!=b'\x00\x00\x00':raise OSError('udp_fragment_or_header')
        kind=reply[3];offset=4+(4 if kind==1 else 16 if kind==4 else 1+reply[4] if kind==3 else 65536)
        if len(reply)<offset+2:raise OSError('udp_address')
        return reply[offset+2:]
def check(kind,timeout,socks_port=10808):
    started=time.monotonic();result={'name':kind,'status':'failed','milliseconds':0}
    try:
        ident,query=dns_query()
        if kind=='dns_tcp':
            s,_=socks(1,'1.1.1.1',53,timeout,socks_port)
            with s:s.sendall(struct.pack('!H',len(query))+query);data=exact(s,struct.unpack('!H',exact(s,2))[0])
            passed=dns_ok(data,ident)
        elif kind=='dns_udp':
            passed=dns_ok(udp_exchange('1.1.1.1',53,query,timeout,socks_port),ident)
        elif kind=='quic_vn':
            dcid=secrets.token_bytes(8);scid=secrets.token_bytes(8)
            packet=b'\xc0'+struct.pack('!I',0x1a2a3a4a)+b'\x08'+dcid+b'\x08'+scid+bytes(1177)
            data=udp_exchange('1.1.1.1',443,packet,timeout,socks_port)
            passed=len(data)>=27 and data[0]&0x80 and data[1:5]==bytes(4) and data[5]==8 and data[6:14]==scid and data[14]==8 and data[15:23]==dcid and (len(data)-23)%4==0
        elif kind in ('doh_cloudflare','doh_google'):
            url='https://'+('1.1.1.1' if kind=='doh_cloudflare' else '8.8.8.8')+'/dns-query'
            r=subprocess.run([shutil.which('curl') or '/usr/bin/curl','--noproxy','','--proxy','socks5h://127.0.0.1:'+str(socks_port),'--max-time',str(timeout),'--connect-timeout',str(timeout),'--max-filesize','65536','--silent','--fail','-H','Content-Type: application/dns-message','--data-binary','@-',url],input=query,capture_output=True,timeout=timeout+1)
            passed=r.returncode==0 and dns_ok(r.stdout,ident)
        else:raise ValueError('unknown_check')
        result['status']='passed' if passed else 'failed'
    except (OSError,ValueError,subprocess.SubprocessError):
        result['status']='failed'
    result['milliseconds']=round((time.monotonic()-started)*1000,2)
    return result
def run(timeout=4,udp=True,socks_port=10808):
    names=['dns_tcp','doh_cloudflare','doh_google']+(['dns_udp','quic_vn'] if udp else [])
    with concurrent.futures.ThreadPoolExecutor(max_workers=3) as pool:checks=list(pool.map(lambda n:check(n,timeout,socks_port),names))
    if not udp:checks += [{'name':n,'status':'unsupported','milliseconds':0} for n in ['dns_udp','quic_vn']]
    return {'version':VERSION,'checks':checks,'http3':'not_run'}
if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('--timeout',type=float,default=4);p.add_argument('--tcp-only',action='store_true');p.add_argument('--socks-port',type=int,default=10808);a=p.parse_args()
    if not 0<a.timeout<=15 or not 1<=a.socks_port<=65535:raise SystemExit('invalid timeout')
    print(json.dumps(run(a.timeout,not a.tcp_only,a.socks_port)))
