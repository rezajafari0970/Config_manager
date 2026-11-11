#!/usr/bin/env python3
"""Current-client rejection regression; real checks run only inside the sandbox."""
import argparse,copy,hashlib,importlib.util,json,pathlib,subprocess,unittest,uuid
from unittest.mock import patch
ROOT=pathlib.Path(__file__).resolve().parents[1]
spec=importlib.util.spec_from_file_location('driver',ROOT/'scripts/probe-one.py')
driver=importlib.util.module_from_spec(spec);spec.loader.exec_module(driver)

class FakeBox:
    def __init__(self,rc=0,log=b'Xray synthetic',timeout=False):
        self.core=pathlib.Path('/original');self.returncode=rc;self.log=log
        self.timeout=timeout;self.data=None;self.stops=0
    def stop(self):self.stops+=1
    def policy(self,value):assert value=={}
    def launch(self,data,test):assert test;self.data=data;return self
    def wait(self,timeout):
        if self.timeout:raise subprocess.TimeoutExpired('synthetic',timeout)
    def logs(self):return self.log

class ClientGateTest(unittest.TestCase):
    def test_exact_bytes_no_repair_and_restore(self):
        body=b'{"security":"none","private-fixture":"unchanged"}'
        box=FakeBox()
        self.assertIsNone(driver.validate_client(box,body,'/new'))
        self.assertEqual(box.data,body);self.assertEqual(box.core,pathlib.Path('/original'))
    def test_semantic_rejection_is_not_connection_failure(self):
        for message,code in [
            (b'vless without TLS or other encryption is prohibited','client_vless_encryption_required'),
            (b'trojan without TLS is prohibited','client_trojan_tls_required'),
            (b'something unsupported','client_core_incompatible')]:
            box=FakeBox(23,b'Xray synthetic '+message+b' SECRET')
            result=driver.validate_client(box,b'{}','/new')
            self.assertEqual(result['code'],code);self.assertEqual(result['core'],'failed')
            self.assertEqual(result['connection'],'not_run')
            self.assertNotIn('SECRET',json.dumps(result));self.assertEqual(box.core,pathlib.Path('/original'))
    def test_infrastructure_and_timeout_fail_closed(self):
        for box in [FakeBox(-9),FakeBox(1,b'bwrap unavailable'),FakeBox(timeout=True)]:
            result=driver.validate_client(box,b'{}','/new')
            self.assertEqual(result['core'],'blocked');self.assertEqual(result['connection'],'not_run')
            self.assertEqual(box.core,pathlib.Path('/original'))
    def test_runtime_uses_bundled_core_and_restores_on_failure(self):
        box=FakeBox();candidate={'sha256':'unchanged'}
        def execute(b,c,t):
            self.assertEqual(b.core,pathlib.Path('/bundled'))
            self.assertIs(c,candidate);self.assertEqual(t,7)
            return {'connection':'passed'}
        with patch.object(driver.lab,'test_candidate',side_effect=execute):
            r=driver.run_target(box,candidate,7,'/bundled')
            self.assertEqual(r['evidence']['runtime_core'],'xray_26_10_31')
        self.assertEqual(box.core,pathlib.Path('/original'))
        with patch.object(driver.lab,'test_candidate',side_effect=ValueError('synthetic')):
            with self.assertRaises(ValueError):driver.run_target(box,candidate,7,'/bundled')
        self.assertEqual(box.core,pathlib.Path('/original'));self.assertEqual(box.stops,2)
    def test_app_array_object_and_relay_preservation(self):
        root={'inbounds':[{'tag':'relay','protocol':'dokodemo-door','listen':'127.0.0.1','port':39081}], 'outbounds':[], 'routing':{'rules':[{'inboundTag':['relay'],'outboundTag':'carrier'}]},'stats':{},'policy':{'levels':{'0':{'handshake':12}},'system':{'statsInboundUplink':True}}}
        before=copy.deepcopy(root)
        self.assertEqual(driver.app.profiles(json.dumps(root)),[root])
        self.assertEqual(driver.app.profiles(json.dumps([root])),[root])
        with self.assertRaises(ValueError):driver.app.profiles(json.dumps(root)+'\n'+json.dumps(root))
        variants=list(driver.app.variants(json.dumps(root)));self.assertEqual(len(variants),4)
        for i,data in enumerate(variants):
            v=json.loads(data);self.assertEqual(v['inbounds'][0],before['inbounds'][0]);self.assertEqual(v['routing'],before['routing'])
            self.assertEqual(v['policy']['levels'],before['policy']['levels'])
            self.assertEqual(any(x['protocol']=='tun' for x in v['inbounds']),i>=2)
            self.assertEqual('stats' in v,i in (1,3))
        self.assertEqual(root,before)
    def test_all_export_drivers_use_gate(self):
        for name in ['smart-probe-one.py','splash-probe-one.py','ads-probe-one.py']:
            s=(ROOT/'scripts'/name).read_text()
            self.assertIn("'probe-one.py'",s);self.assertIn('driver.main()',s)

def integration():
    core=pathlib.Path(driver.CLIENT_CORE)
    assert hashlib.sha256(core.read_bytes()).hexdigest()==driver.CLIENT_CORE_SHA
    old=pathlib.Path('/opt/config-repair/core-26.7.28/xray')
    assert hashlib.sha256(old.read_bytes()).hexdigest()=='64d46afb80adea1bf97a0d467e83f4a9ac1ebd0995891e84bca3f1a1d1affb1d'
    uid='00000000-0000-4000-8000-000000000001'
    def config(address='vpn.example.com',security='none',flat=False,protocol='vless'):
        settings={'address':address,'port':443,'id':uid,'encryption':'none'}
        if not flat:settings={'vnext':[{'address':address,'port':443,'users':[{'id':uid,'encryption':'none'}]}]}
        stream={'network':'tcp','security':security,'tlsSettings':{'serverName':'example.com','allowInsecure':False}}
        return {'inbounds':[],'outbounds':[{'tag':'proxy','protocol':protocol,'settings':settings,'streamSettings':stream}],'routing':{}}
    cases=[('legacy_public_none',config(),False),
           ('flat_public_none',config(flat=True),False),
           ('inactive_tls_settings',config(),False),
           ('vless_tls',config(security='tls'),True),
           ('private_vless_none',config(address='127.0.0.1'),True)]
    reality=config(security='reality')
    reality['outbounds'][0]['streamSettings']['realitySettings']={'serverName':'example.com','fingerprint':'chrome','publicKey':'AgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgI','shortId':'ab'}
    cases.append(('vless_reality',reality,True))
    for security in ['none','tls']:
        c=config(security=security);ob=c['outbounds'][0];ob['protocol']='trojan'
        ob['settings']={'servers':[{'address':'vpn.example.com','port':443,'password':'synthetic'}]}
        cases.append(('trojan_'+security,c,security=='tls'))
    vmess=config();vmess['outbounds'][0]['protocol']='vmess'
    vmess['outbounds'][0]['settings']['vnext'][0]['users'][0]={'id':uid,'security':'auto'}
    cases.append(('vmess_no_transport_tls',vmess,True))
    bad=config(security='tls');bad['stats']=[];cases.append(('bad_object_shape',bad,False))
    mixed=config(security='tls');mixed['outbounds']+=config()['outbounds']
    mixed['outbounds'][1]['tag']='unused-but-parsed'
    cases.append(('inactive_incompatible_outbound',mixed,False))
    results=[];runid=uuid.uuid4().hex[:6]
    with driver.lab.Sandbox(str(old),runid,8192,prefix='ctw') as box:
        for name,c,expected in cases:
            body=json.dumps(c,separators=(',',':')).encode()
            result=driver.validate_client(box,body,str(core))
            passed=result is None
            results.append({'case':name,'expected':expected,'accepted':passed,'code':result['code'] if result else 'accepted'})
            assert passed==expected,results[-1]
        # The previous pinned core accepted the nested public VLESS shape.
        process=box.launch(json.dumps(config()).encode(),True);process.wait(timeout=15)
        assert process.returncode==0,'old-core regression premise changed'
        box.stop()
    receipt={'passed':True,'bundled':'26.10.31','old':'26.7.28','old_vnext_accepted':True,'cases':results,'android':'not_run','network_connectivity':'not_run'}
    (ROOT/'.local/p33-client-core.json').write_text(json.dumps(receipt,indent=2))
    print(json.dumps(receipt))

if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('--integration',action='store_true');a=p.parse_args()
    suite=unittest.defaultTestLoader.loadTestsFromTestCase(ClientGateTest)
    if not unittest.TextTestRunner().run(suite).wasSuccessful():raise SystemExit(1)
    if a.integration:integration()
