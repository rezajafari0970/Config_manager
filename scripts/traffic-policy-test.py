#!/usr/bin/env python3
"""Synthetic TCP/UDP forwarding regression in the existing closed sandbox.
Called by TestPinnedCoreTrafficForwarding with XRAY_TRAFFIC_TEST=1 as root.
No live subscription is accepted or probed. QUIC handshake is not claimed:
UDP/443 echo demonstrates removal of the local QUIC transport gate.
"""
import copy,hashlib,importlib.util,json,os,pathlib,secrets,socket
import struct,subprocess,sys,tempfile,threading,time

HERE=pathlib.Path(__file__).resolve().parent

def echo():
    def udp(port):
        s=socket.socket(socket.AF_INET,socket.SOCK_DGRAM);s.bind(('127.0.0.1',port))
        while True:
            b,peer=s.recvfrom(8192);s.sendto(b,peer)
    for port in (443,24445):threading.Thread(target=udp,args=(port,),daemon=True).start()
    s=socket.socket();s.setsockopt(socket.SOL_SOCKET,socket.SO_REUSEADDR,1);s.bind(('127.0.0.1',24444));s.listen()
    def client(c):
        with c:
            c.settimeout(3)
            try:
                b=c.recv(8192)
                if b:c.sendall(b)
            except OSError:pass
    while True:threading.Thread(target=client,args=(s.accept()[0],),daemon=True).start()

def exact(s,n):
    out=b''
    while len(out)<n:
        b=s.recv(n-len(out))
        if not b:raise OSError('closed SOCKS connection')
        out+=b
    return out

def socks(command,port):
    s=socket.create_connection(('127.0.0.1',10808),timeout=3);s.settimeout(10)
    s.sendall(b'\x05\x01\x00')
    assert exact(s,2)==b'\x05\x00'
    s.sendall(bytes([5,command,0,1])+socket.inet_aton('127.0.0.1')+struct.pack('!H',port))
    h=exact(s,4);assert h[:2]==b'\x05\x00'
    if h[3]==1:host=socket.inet_ntoa(exact(s,4))
    elif h[3]==4:host=socket.inet_ntop(socket.AF_INET6,exact(s,16))
    else:host=exact(s,exact(s,1)[0]).decode()
    return s,(host,struct.unpack('!H',exact(s,2))[0])

def exercise():
    token=secrets.token_bytes(32);s,_=socks(1,24444)
    with s:s.sendall(token);tcp=exact(s,len(token))==token
    results={'tcp':tcp}
    # Fresh association per destination: reusing an existing UDP session could
    # bypass a first-packet policy gate and yield a false-positive control.
    for port,key in [(24445,'udp'),(443,'udp443')]:
        udp=socket.socket(socket.AF_INET,socket.SOCK_DGRAM);udp.bind(('127.0.0.1',0));udp.settimeout(2)
        control,relay=socks(3,udp.getsockname()[1])
        with control,udp:
            token=secrets.token_bytes(32)
            packet=b'\x00\x00\x00\x01'+socket.inet_aton('127.0.0.1')+struct.pack('!H',port)+token
            udp.sendto(packet,relay)
            try:
                reply,_=udp.recvfrom(8192)
                results[key]=reply[:3]==b'\x00\x00\x00' and reply.endswith(token)
            except socket.timeout:results[key]=False
    print(json.dumps(results))

def main():
    if os.geteuid()!=0:raise SystemExit('root required for closed network fixture')
    request=json.load(sys.stdin);core=pathlib.Path(request['core']).resolve()
    assert hashlib.sha256(core.read_bytes()).hexdigest()=='64d46afb80adea1bf97a0d467e83f4a9ac1ebd0995891e84bca3f1a1d1affb1d'
    spec=importlib.util.spec_from_file_location('traffic_sandbox',HERE/'live-probe.py')
    lab=importlib.util.module_from_spec(spec);spec.loader.exec_module(lab)
    with tempfile.TemporaryDirectory(prefix='synthetic-traffic-') as directory:
        work=pathlib.Path(directory);key=work/'key.pem';cert=work/'cert.pem'
        subprocess.run(['openssl','req','-x509','-newkey','rsa:2048','-nodes','-days','1','-subj','/CN=localhost','-keyout',str(key),'-out',str(cert)],check=True,capture_output=True,timeout=10)
        der=subprocess.check_output(['openssl','x509','-in',str(cert),'-outform','DER'])
        pin=hashlib.sha256(der).hexdigest()
        server={'tag':'fixture-server','listen':'127.0.0.1','port':24443,'protocol':'vless','settings':{'decryption':'none','clients':[{'id':'11111111-1111-4111-8111-111111111111','flow':'xtls-rprx-vision'}]},'streamSettings':{'network':'tcp','security':'tls','tlsSettings':{'certificates':[{'certificate':cert.read_text().strip().splitlines(),'key':key.read_text().strip().splitlines()}]}}}
        report={}
        for phase in ('before','after'):
            config=copy.deepcopy(request[phase])
            proxy=config['outbounds'][0];settings=proxy['settings']
            endpoint=settings.get('address') or settings['vnext'][0]['address']
            assert proxy['protocol']=='vless' and endpoint=='127.0.0.1'
            proxy['streamSettings']['tlsSettings']={'serverName':'localhost','pinnedPeerCertSha256':pin}
            # Controlled fixture additions only. The client's normal SOCKS
            # routing still goes through VLESS/TLS before the echo destination.
            config['inbounds'].append(server)
            config['outbounds'].append({'tag':'fixture-direct','protocol':'freedom','settings':{'finalRules':[{'action':'allow','network':'tcp,udp','ip':['127.0.0.1/32'],'port':'443,24444,24445'}]}})
            config['routing']['rules'].insert(0,{'type':'field','inboundTag':['fixture-server'],'outboundTag':'fixture-direct'})
            config['log']={'loglevel':'warning'}
            with lab.Sandbox(str(core),secrets.token_hex(3),8199,prefix='ctf') as box:
                # Retain the standard endpoint-deny guard and fixed DNS exception;
                # this synthetic fixture sends traffic only over loopback.
                box.policy({})
                echoer=subprocess.Popen([lab.IP,'netns','exec',box.name,sys.executable,__file__,'--echo'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
                try:
                    process=box.launch(json.dumps(config).encode())
                    for _ in range(50):
                        if process.poll() is not None:raise RuntimeError('synthetic core exited: '+box.logs().decode(errors='replace')[-1200:])
                        check=box.net([sys.executable,'-c','import socket; s=socket.socket(); print(s.connect_ex(("127.0.0.1",10808)))'])
                        if check.stdout.strip()==b'0':break
                        time.sleep(.05)
                    else:raise RuntimeError('synthetic core not ready')
                    try:result=box.net([sys.executable,__file__,'--exercise'],timeout=15)
                    except subprocess.CalledProcessError as e:raise RuntimeError(e.stderr.decode(errors='replace')[-1200:]+' '+box.logs().decode(errors='replace')[-2400:]) from None
                    report[phase]=json.loads(result.stdout)
                    expected={'tcp':True,'udp':True,'udp443':phase=='after'}
                    if report[phase]!=expected:raise RuntimeError('forwarding mismatch '+json.dumps(report)+' '+box.logs().decode(errors='replace')[-1200:])
                finally:
                    echoer.terminate()
                    try:echoer.wait(timeout=3)
                    except subprocess.TimeoutExpired:echoer.kill();echoer.wait(timeout=3)
        print(json.dumps({'passed':True,'traffic':report,'quic_handshake':'not_run'}))

if __name__=='__main__':
    if len(sys.argv)>1 and sys.argv[1]=='--echo':echo()
    elif len(sys.argv)>1 and sys.argv[1]=='--exercise':exercise()
    else:main()
