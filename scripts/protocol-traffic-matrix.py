#!/usr/bin/env python3
"""Closed-network real forwarding across independent protocol/transport fixtures.
No production credentials. Failed fixtures are recorded, never auto-qualified.
"""
import base64,copy,hashlib,importlib.util,json,pathlib,secrets,subprocess,sys,tempfile,time
HERE=pathlib.Path(__file__).resolve().parent
def module(name,file):
 s=importlib.util.spec_from_file_location(name,HERE/file);m=importlib.util.module_from_spec(s);s.loader.exec_module(m);return m
lab=module('lab','live-probe.py');matrix=module('matrix','capability-matrix.py')
def run():
 report=[]
 with tempfile.TemporaryDirectory(prefix='protocol-fixture-') as directory:
  work=pathlib.Path(directory);key=work/'key.pem';cert=work/'cert.pem'
  subprocess.run(['openssl','req','-x509','-newkey','rsa:2048','-nodes','-days','1','-subj','/CN=fixture.invalid','-addext','subjectAltName=DNS:fixture.invalid','-keyout',str(key),'-out',str(cert)],check=True,capture_output=True)
  pin=hashlib.sha256(subprocess.check_output(['openssl','x509','-in',str(cert),'-outform','DER'])).hexdigest()
  for version in ('26.10.31','26.11.9'):
   core='/opt/config-repair/core-'+version+'/xray'
   for protocol,network,security in ([('wireguard','native','native')] if '--wireguard' in sys.argv else
    [(p,n,s) for p in ('vless','vmess','trojan') for n in ('tcp','tcp-http','ws','xhttp','grpc','httpupgrade','kcp') for s in (('none','tls') if p!='trojan' else ('tls',))]
    + [('shadowsocks','tcp','none'),('hysteria','hysteria','tls')]):
    if '--case' in sys.argv and ','.join((protocol,network,security)) != sys.argv[sys.argv.index('--case')+1]:continue
    item={'core':version,'protocol':protocol,'transport':network,'security':security,'tcp':'failed','udp':'not_run','udp443':'not_run'}
    ob=matrix.outbound(protocol,network,security);ob['settings']['port']=24443
    if protocol=='vmess':ob['settings']['vnext'][0]['port']=24443
    if protocol=='shadowsocks' and '--ss-method' in sys.argv:
     method=sys.argv[sys.argv.index('--ss-method')+1]
     if method not in ('aes-128-gcm','aes-256-gcm','chacha20-ietf-poly1305','2022-blake3-aes-128-gcm','2022-blake3-aes-256-gcm','2022-blake3-chacha20-poly1305'):raise ValueError('unknown_ss_cipher')
     password=base64.b64encode(bytes(range(16 if method=='2022-blake3-aes-128-gcm' else 32))).decode() if method.startswith('2022-') else 'synthetic-test-only'
     ob['settings'].update(method=method,password=password);item['cipher']=method
    stream=ob.get('streamSettings',{});serverStream=copy.deepcopy(stream)
    if security=='tls':
     stream['tlsSettings']={'serverName':'fixture.invalid','pinnedPeerCertSha256':pin}
     serverStream['tlsSettings']={'certificates':[{'certificate':cert.read_text().strip().splitlines(),'key':key.read_text().strip().splitlines()}]}
    if protocol=='vless':settings={'decryption':'none','clients':[{'id':matrix.UUID}]}
    elif protocol=='vmess':settings={'clients':[{'id':matrix.UUID,'alterId':0}]}
    elif protocol=='trojan':settings={'clients':[{'password':'synthetic-test-only'}]}
    elif protocol=='shadowsocks':settings={'method':ob['settings']['method'],'password':ob['settings']['password'],'network':'tcp,udp'}
    else:settings={'version':2,'clients':[{'auth':'synthetic-test-only'}]}
    if protocol=='wireguard':
     def pair(seed):
      encoded=base64.urlsafe_b64encode(seed).decode().rstrip('=')
      lines=subprocess.check_output([core,'x25519','-i',encoded]).decode().splitlines()
      values={k.strip():v.strip() for k,v in (line.split(':',1) for line in lines if ':' in line)}
      public=values.get('Password (PublicKey)',values.get('Password',values.get('Public key')))
      return base64.b64encode(seed).decode(),base64.b64encode(base64.urlsafe_b64decode(public+'='*((-len(public))%4))).decode()
     cp,cu=pair(bytes(range(32)));sp,su=pair(bytes(range(1,33)))
     ob['settings']={'secretKey':cp,'address':['10.23.0.2/32'],'noKernelTun':True,'peers':[{'endpoint':'203.0.113.81:24443','publicKey':su,'allowedIPs':['0.0.0.0/0']}] }
     settings={'secretKey':sp,'address':['10.23.0.1/32'],'noKernelTun':True,'peers':[{'publicKey':cu,'allowedIPs':['10.23.0.2/32']}] }
    if network=='tcp-http':serverStream['tcpSettings']={'header':{'type':'http'}}
    server={'log':{'loglevel':'warning'},'inbounds':[{'tag':'fixture','listen':'203.0.113.81','port':24443,'protocol':protocol,'settings':settings,'streamSettings':serverStream}],'outbounds':[{'protocol':'freedom','settings':{'finalRules':[{'action':'allow','network':'tcp,udp','ip':['127.0.0.1/32'],'port':'443,24444,24445'}]}}]}
    client={'log':{'loglevel':'warning'},'inbounds':[{'tag':'socks','listen':'127.0.0.1','port':10808,'protocol':'socks','settings':{'auth':'noauth','udp':True}}],'outbounds':[ob],'routing':{'rules':[{'type':'field','network':'tcp,udp','outboundTag':'proxy'}]}}
    # Exercise the common template actually exported by production.
    plan=json.loads(subprocess.run([str(HERE.parent/'.local/config-probe-plan'),'--template'],input=json.dumps(client).encode(),capture_output=True,check=True).stdout)
    if not plan.get('candidates'):
     item['diagnostic']=plan.get('code','conversion_failed');report.append(item);continue
    data=base64.b64decode(plan['candidates'][0]['config'])
    with lab.Sandbox(core,secrets.token_hex(3),8202,prefix='ctf') as box:
     box.net([lab.IP,'addr','add','203.0.113.81/32','dev','lo'])
     path=box.work/'fixture.json';path.write_text(json.dumps(server))
     prefix=[lab.IP,'netns','exec',box.name]
     traffic=HERE/'traffic-policy-test.py'
     if protocol=='wireguard':
      box.net([lab.IP,'addr','add','203.0.113.82/32','dev','lo'])
      server['outbounds'][0]['settings']['finalRules'][0]['ip']=['203.0.113.82/32']
      path.write_text(json.dumps(server))
      traffic=box.work/'traffic.py';traffic.write_text((HERE/'traffic-policy-test.py').read_text().replace("socket.inet_aton('127.0.0.1')","socket.inet_aton('203.0.113.82')").replace("bind(('127.0.0.1',","bind(('0.0.0.0',").replace("bytes([5,command,0,1])+socket.inet_aton('203.0.113.82')","bytes([5,command,0,1])+socket.inet_aton('203.0.113.82' if command==1 else '127.0.0.1')"))
     log=tempfile.TemporaryFile()
     fixture=subprocess.Popen(['/usr/bin/python3',str(HERE/'live-probe.py'),'--sandbox-launch',str(box.cg)]+prefix+[core,'run','-c',str(path)],stdout=log,stderr=subprocess.STDOUT)
     echo=subprocess.Popen(prefix+[sys.executable,str(traffic),'--echo'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
     try:
      process=box.launch(data)
      for _ in range(50):
       if process.poll() is not None:raise RuntimeError('client_start_failed')
       ready=box.net([sys.executable,'-c','import socket;s=socket.socket();print(s.connect_ex(("127.0.0.1",10808)))'])
       if ready.stdout.strip()==b'0':break
       time.sleep(.05)
      try:r=box.net([sys.executable,str(traffic),'--exercise'],timeout=18)
      except subprocess.CalledProcessError:raise RuntimeError('traffic_failed')
      item.update({k:'passed' if v else 'failed' for k,v in json.loads(r.stdout).items()})
      item['config_sha256']=lab.digest(data);item['runtime_sha256']=lab.digest((box.work/'config.json').read_bytes())
     except Exception as e:
      log.seek(0);item['diagnostic']=lab.classify(box.logs()+log.read(8192),str(e))
     finally:
      for p in (fixture,echo):
       p.terminate()
       try:p.wait(timeout=2)
       except subprocess.TimeoutExpired:p.kill();p.wait()
      log.close()
    report.append(item)
 print(json.dumps({'version':'protocol-traffic.2026-11-11.v1','scope':'synthetic-closed-network','https':'separate_test','android':'not_run','http3':'not_run','rows':report},indent=2))
if __name__=='__main__':run()
