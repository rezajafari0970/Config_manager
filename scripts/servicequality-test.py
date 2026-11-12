#!/usr/bin/env python3
import copy, importlib.util, pathlib, tempfile, time, unittest, json
from unittest.mock import patch
from types import SimpleNamespace
from benchmarks.http import request
from servicequality.probes import service_check
from servicequality.policy import assess, choose, VERSION
from servicequality.catalog import SERVICES, VERSION as DRIVERS
from servicequality.profile import adapt
from benchmarks.quality import PAYLOAD_VERSION
HERE=pathlib.Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('driver',HERE/'service-smart.py');cli=importlib.util.module_from_spec(spec);spec.loader.exec_module(cli)

def check(code=200):
    return dict(status='passed',transport_tls='verified',curl_exit=0,http_status=code,total_ms=500,download_bytes=2097152,upload_bytes=1048576)
CTX={'network_label':'MCI','network_epoch':'1','core_sha256':'core','config_sha256':'config'}
def receipt(t):
    return dict(CTX,version=VERSION,drivers=DRIVERS,started=t,finished=t+1,core_build='passed',runtime='passed',
                payload={'version':PAYLOAD_VERSION,'download':check(),'upload':check(),'post_load':check(204)},
                services={s:[check(code) for _,code in d['probes']] for s,d in SERVICES.items()})
def node(score=50,stamp=1):return dict(eligible=True,score=score,evidence_at=stamp)
def base():
    return dict(inbounds=[dict(tag='socks',port=10808,listen='127.0.0.1',protocol='socks')],
        outbounds=[dict(tag='proxy-01',protocol='vless'),dict(tag='proxy-02',protocol='vless'),dict(tag='block',protocol='blackhole')],
        routing=dict(balancers=[dict(tag='bootstrap-ip-balancer',selector=['proxy-01']),dict(tag='smart-parallel-balancer',selector=['proxy-'])],rules=[
          dict(type='field',domain=['domain:doubleclick.net'],outboundTag='ads'),
          dict(type='field',inboundTag=['dns-module'],balancerTag='smart-parallel-balancer'),
          dict(type='field',network='tcp,udp',balancerTag='smart-parallel-balancer')]))

class Policy(unittest.TestCase):
    def test_complete(self):self.assertTrue(assess([receipt(10),receipt(20)],'instagram',30,CTX)['eligible'])
    def test_old_or_other_scope(self):
        for ctx in [dict(CTX,network_epoch='2'),dict(CTX,core_sha256='other'),dict(CTX,network_label='Irancell')]:
            self.assertFalse(assess([receipt(10),receipt(20)],'instagram',30,ctx)['eligible'])
        self.assertFalse(assess([receipt(10),receipt(20)],'instagram',1000,CTX)['eligible'])
    def test_full_upload_not_boolean(self):
        a=receipt(20);a['payload']['upload']['upload_bytes']=65536;a['payload']['complete']=True
        self.assertFalse(assess([receipt(10),a],'instagram',30,CTX)['eligible'])
    def test_timeout_with_200(self):
        a=receipt(20);a['payload']['download']['curl_exit']=28
        self.assertFalse(assess([receipt(10),a],'instagram',30,CTX)['eligible'])
    def test_interruption_revokes(self):
        a=receipt(25);del a['finished']
        self.assertFalse(assess([receipt(10),receipt(20),a],'instagram',30,CTX)['eligible'])
    def test_separate_services(self):
        a=receipt(20);a['services']['instagram'][0]['status']='failed'
        self.assertFalse(assess([receipt(10),a],'instagram',30,CTX)['eligible'])
        self.assertTrue(assess([receipt(10),a],'telegram',30,CTX)['eligible'])
    def test_duplicate_receipts_do_not_add_confidence(self):
        self.assertFalse(assess([receipt(10),receipt(10)],'instagram',30,CTX)['eligible'])
    def test_nonfinite(self):
        for value in [float('nan'),float('inf'),True,-1]:
            a=receipt(20);a['services']['instagram'][0]['total_ms']=value
            self.assertFalse(assess([receipt(10),a],'instagram',30,CTX)['eligible'])
    def test_small_improvement_does_not_switch(self):
        old=choose({'a':node()},None,10,'scope')
        self.assertEqual(choose({'a':node(),'b':node(55)},old,500,'scope')['target'],'a')
    def test_three_new_waves_and_hold(self):
        old=choose({'a':node()},None,10,'scope')
        for stamp in [1,2,3]:old=choose({'a':node(),'b':node(80,stamp)},old,30,'scope')
        self.assertEqual(old['target'],'a')
        self.assertEqual(choose({'a':node(),'b':node(80,3)},old,150,'scope')['target'],'b')
    def test_polling_is_not_new_evidence(self):
        old=choose({'a':node()},None,10,'scope')
        for _ in range(20):old=choose({'a':node(),'b':node(80)},old,1000,'scope')
        self.assertEqual(old['target'],'a');self.assertEqual(old['wins'],1)
    def test_failover_bypasses_hold_and_no_route_blocks(self):
        old=choose({'a':node()},None,10,'scope')
        self.assertEqual(choose({'b':node()},old,11,'scope')['target'],'b')
        self.assertEqual(choose({},old,11,'scope')['target'],'block')
    def test_scope_resets_incumbent(self):
        old=choose({'a':node()},None,10,'scope')
        self.assertEqual(choose({'a':node(),'b':node(51)},old,11,'new')['target'],'b')

class Profile(unittest.TestCase):
    def test_preserves_original_and_ads_precedence(self):
        b=base();before=copy.deepcopy(b);c=adapt(b,{s:'proxy-01' for s in SERVICES})
        self.assertEqual(b,before);self.assertEqual(c['routing']['rules'][0],b['routing']['rules'][0])
        self.assertEqual(c['outbounds'],b['outbounds']);self.assertNotIn('api',c)
        self.assertEqual(c['routing']['rules'][1]['outboundTag'],'proxy-01')
    def test_no_general_direct_or_udp443_block(self):
        c=adapt(base(),{s:'proxy-01' for s in SERVICES})
        self.assertFalse(any(r.get('outboundTag')=='direct' or r.get('port')=='443' for r in c['routing']['rules']))
    def test_startup_failclosed_api_loopback(self):
        c=adapt(base(),api_port=39000)
        self.assertEqual(c['inbounds'][-1]['listen'],'127.0.0.1')
        self.assertEqual(len([b for b in c['routing']['balancers'] if b.get('fallbackTag')=='block']),len(SERVICES))
    def test_port_and_tag_collisions(self):
        with self.assertRaises(ValueError):adapt(base(),api_port=10808)
        b=base();b['outbounds'].append(dict(tag='sq-x'))
        with self.assertRaises(ValueError):adapt(b,api_port=39000)
    def test_default_must_bootstrap_without_recursive_dns(self):
        with self.assertRaises(ValueError):adapt(base(),{s:'proxy-02' for s in SERVICES})
    def test_incomplete_decisions_rejected(self):
        with self.assertRaises(ValueError):adapt(base(),{'default':'proxy-01'})
    def test_target_freedom_not_allowed(self):
        with self.assertRaises(ValueError):adapt(base(),{s:'direct' for s in SERVICES})

class HTTPContract(unittest.TestCase):
    def run_request(self,**kwargs):
        with patch('benchmarks.http.subprocess.run',return_value=SimpleNamespace(returncode=0,stdout=json.dumps({'http_code':200,'ssl_verify_result':0}).encode())) as run:
            request(10808,'https://example.com/',**kwargs)
            return run.call_args.args[0]
    def test_legacy_benchmark_timeout_unchanged(self):
        cmd=self.run_request(timeout=60)
        self.assertEqual(cmd[cmd.index('--connect-timeout')+1],'3');self.assertNotIn('--location',cmd)
    def test_slow_profile_keeps_strict_tls(self):
        cmd=self.run_request(timeout=60,connect_timeout=8)
        self.assertEqual(cmd[cmd.index('--connect-timeout')+1],'8');self.assertNotIn('-k',cmd)
    def test_redirects_bounded_https_only(self):
        cmd=self.run_request(follow_redirects=True)
        self.assertEqual(cmd[cmd.index('--max-redirs')+1],'3');self.assertEqual(cmd[cmd.index('--proto-redir')+1],'=https')
    def test_upload_never_redirected_or_replayed(self):
        with self.assertRaises(ValueError):self.run_request(payload='unused',follow_redirects=True)
    def test_cross_service_redirect_is_not_evidence(self):
        with patch('servicequality.probes.request',return_value=dict(check(),effective_host='whatsapp.com.example.org')):
            self.assertEqual(service_check(10808,'whatsapp',15)[0]['status'],'failed')
    def test_real_service_redirect_allowed(self):
        with patch('servicequality.probes.request',return_value=dict(check(),effective_host='www.whatsapp.com')):
            self.assertEqual(service_check(10808,'whatsapp',15)[0]['status'],'passed')

class Resume(unittest.TestCase):
    def setup_case(self,d):
        root=pathlib.Path(d);(root/'core').write_bytes(b'fixture-not-executed');(root/'corpus').write_text(json.dumps({'rows':[{'config':{}}]}))
        out=root/'out';out.mkdir();ctx=dict(core_sha256=cli.digest(b'fixture-not-executed'),network_label='MCI',network_epoch='1');(out/'context.json').write_text(json.dumps(ctx))
        for n in range(2):
            c=out/('round-%d-member-000'%n);c.mkdir();r=dict(receipt(10+n),**ctx);r['config_sha256']=cli.digest(cli.encoded({}));(c/'receipt.json').write_text(json.dumps(r))
        argv=['driver','measure','--corpus',str(root/'corpus'),'--core',str(root/'core'),'--network','MCI','--epoch','1','--output',str(out),'--resume']
        return out,argv
    def test_complete_receipts_retained_without_retesting(self):
        with tempfile.TemporaryDirectory() as d:
            out,argv=self.setup_case(d);before=(out/'round-0-member-000/receipt.json').read_bytes()
            with patch('sys.argv',argv),patch.object(cli,'measure') as measure:cli.main();measure.assert_not_called()
            self.assertEqual(before,(out/'round-0-member-000/receipt.json').read_bytes())
    def test_changed_contract_refuses_resume(self):
        with tempfile.TemporaryDirectory() as d:
            out,argv=self.setup_case(d);p=out/'round-0-member-000/receipt.json';r=json.loads(p.read_text());r['version']='old';p.write_text(json.dumps(r))
            with patch('sys.argv',argv),self.assertRaises(SystemExit):cli.main()
    def test_only_interrupted_case_retried_and_archived(self):
        with tempfile.TemporaryDirectory() as d:
            out,argv=self.setup_case(d);p=out/'round-0-member-000/receipt.json';r=json.loads(p.read_text());r.pop('finished');p.write_text(json.dumps(r))
            with patch('sys.argv',argv),patch.object(cli,'measure',return_value={'runtime':'passed','observed_http_bytes':0}) as m:cli.main();self.assertEqual(m.call_count,1)
            self.assertEqual(len(list(out.glob('*.interrupted-*'))),1)

if __name__=='__main__':unittest.main()
