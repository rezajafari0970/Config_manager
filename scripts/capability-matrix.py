#!/usr/bin/env python3
"""Reproducible official-core build matrix. Build acceptance is NOT transport proof."""
import base64,copy,importlib.util,json,pathlib,secrets,sys
HERE=pathlib.Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('lab',HERE/'live-probe.py')
lab=importlib.util.module_from_spec(spec);spec.loader.exec_module(lab)
UUID='11111111-1111-4111-8111-111111111111'
def outbound(protocol,network,security):
    common={'address':'203.0.113.81','port':443}
    if protocol=='vless':settings=common|{'id':UUID,'encryption':'none'}
    elif protocol=='vmess':settings={'vnext':[common|{'users':[{'id':UUID,'alterId':0,'security':'auto'}]}]}
    elif protocol=='trojan':settings=common|{'password':'synthetic-test-only'}
    elif protocol=='shadowsocks':settings=common|{'method':'aes-128-gcm','password':'synthetic-test-only'}
    elif protocol=='hysteria':settings=common|{'version':2}
    elif protocol=='hysteria1':settings=common|{'version':1};protocol='hysteria'
    elif protocol=='wireguard':settings={'secretKey':base64.b64encode(bytes(range(32))).decode(),'address':['10.23.0.2/32'],'noKernelTun':True,'peers':[{'endpoint':'203.0.113.81:443','publicKey':base64.b64encode(bytes(range(1,33))).decode()}]}
    elif protocol in ('socks','http'):settings={'servers':[common]}
    else:settings=common
    stream={'network':network,'security':security}
    if security=='tls':stream['tlsSettings']={'serverName':'fixture.invalid'}
    if security=='reality':stream['realitySettings']={'serverName':'fixture.invalid','fingerprint':'chrome','publicKey':base64.urlsafe_b64encode(bytes(range(1,33))).decode().rstrip('='),'shortId':'12'}
    if network=='tcp-http':stream['network']='tcp';stream['tcpSettings']={'header':{'type':'http','request':{'headers':{'Host':['fixture.invalid']},'method':'GET','path':['/'],'version':'1.1'}}}
    if network in ('xhttp','splithttp'):stream['xhttpSettings']={'path':'/fixture','mode':'auto'}
    if network=='grpc':stream['grpcSettings']={'serviceName':'fixture'}
    if network=='hysteria':stream['hysteriaSettings']={'version':2,'auth':'synthetic-test-only'}
    return {'tag':'proxy','protocol':protocol,'settings':settings,**({'streamSettings':stream} if protocol!='wireguard' else {})}
def cases():
    for p in ('vless','vmess','trojan','shadowsocks'):
        for n in ('tcp','raw','tcp-http','ws','xhttp','grpc','httpupgrade'):
            for s in ('none','tls','reality'):yield p,n,s,outbound(p,n,s)
    for p,n,s in [('hysteria','hysteria','tls'),('hysteria1','hysteria','tls'),('wireguard','native','native'),('socks','tcp','none'),('http','tcp','tls'),('tuic','quic','tls'),('vless','http','tls'),('vless','quic','tls')]:yield p,n,s,outbound(p,n,s)
def run():
    rows=[]
    pins={'26.10.31':'cc4b84d3069569f297a61f04b914f2bb37e9d202fd7b84cb50def1fc30fd40e1','26.11.9':'e87bce586acdb9136aa5b1f1d9c8db9f1ea1d145b6d86368c7f6f662f9f6a66d'}
    for version,sha in pins.items():
        core=pathlib.Path('/opt/config-repair/core-'+version+'/xray');assert lab.digest(core.read_bytes())==sha
        with lab.Sandbox(str(core),secrets.token_hex(3),8201,prefix='ctf') as box:
            for protocol,network,security,ob in cases():
                config={'inbounds':[],'outbounds':[ob],'routing':{}}
                data=json.dumps(config,separators=(',',':')).encode()
                process=box.launch(data,True);process.wait(timeout=10)
                accepted=process.returncode==0;log=box.logs();box.stop()
                rows.append({'core':version,'protocol':protocol,'transport':network,'security':security,'build':'passed' if accepted else 'failed','diagnostic':'core_build_only' if accepted else lab.classify(log,'core_rejected'),'sha256':lab.digest(data),'connection':'not_run'})
    print(json.dumps({'version':'official-core-matrix.2026-11-11.v1','pins':pins,'rows':rows},indent=2))
if __name__=='__main__':run()
