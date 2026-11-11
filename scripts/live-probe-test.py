#!/usr/bin/env python3
"""Unprivileged regression tests. Kernel sandbox acceptance is --self-test."""
import base64, copy, importlib.util, json, pathlib, tempfile, unittest, subprocess
from types import SimpleNamespace
from unittest.mock import patch
spec=importlib.util.spec_from_file_location('probe',pathlib.Path(__file__).with_name('live-probe.py'))
p=importlib.util.module_from_spec(spec);spec.loader.exec_module(p)

class ProbeTests(unittest.TestCase):
 def test_worker_and_lab_networks_are_disjoint(self):
  seen=set()
  for prefix in p.SANDBOX_ADDRESS_BASES:
   for slot in range(9):
    box=p.Sandbox('/synthetic/core','abcdef',8192+slot,prefix=prefix)
    self.assertEqual(box.name,prefix+'abcdef'+format(8192+slot,'x'))
    self.assertNotIn(box.gateway,seen);self.assertNotIn(box.addr,seen)
    seen.update([box.gateway,box.addr])
  self.assertEqual(p.Sandbox('/synthetic/core','abcdef',8192,prefix='crw').gateway,'198.18.128.5')
  with self.assertRaisesRegex(ValueError,'namespace_slot_capacity'):p.Sandbox('/synthetic/core','abcdef',8191,prefix='csp')
  with self.assertRaisesRegex(ValueError,'namespace_address_capacity'):p.Sandbox('/synthetic/core','abcdef',-1)

 def test_site_health_requires_successful_tls_and_expected_http(self):
  box=SimpleNamespace(name='synthetic')
  for rc,status,expected in [(0,b'200',True),(0,b'206',True),(35,b'200',False),(0,b'403',False),(0,b'302',False),(63,b'200',False),(0,b'000',False)]:
   with patch.object(p.subprocess,'run',return_value=SimpleNamespace(returncode=rc,stdout=status)) as run:
    self.assertEqual(p.smart_site_check(box,7),expected)
    args=run.call_args.args[0]
    self.assertIn('socks5h://127.0.0.1:10808',args)
    self.assertNotIn('--insecure',args)
    self.assertEqual(args[-1],'https://www.instagram.com/')
    self.assertEqual(run.call_args.kwargs['timeout'],9)
  with patch.object(p.subprocess,'run',side_effect=subprocess.TimeoutExpired('curl',7)):
   self.assertFalse(p.smart_site_check(box,7))

 def test_classification_removed_tls(self):
  self.assertEqual(p.classify('The feature "allowInsecure" has been removed','unknown'),'removed_allow_insecure')
  self.assertEqual(p.classify('trojan without TLS is prohibited','unknown'),'public_endpoint_requires_tls')
  self.assertEqual(p.classify('vless without TLS or other encryption is prohibited','unknown'),'public_endpoint_requires_tls')
 def test_runtime_diagnosis_uses_specific_auth_error(self):
  self.assertEqual(p.classify('REALITY: received real certificate (potential MITM or redirection)','unknown'),'reality_authentication_rejected')
  self.assertEqual(p.classify('certificate loaded successfully','unknown'),'unknown')
  self.assertEqual(p.classify('curl: (35) Recv failure: Connection reset by peer','unknown'),'peer_reset_connection')

 def test_download_endpoint_allowlist_and_one_resolution_per_host(self):
  class R:stdout=b'["1.1.1.1"]'
  stream={'xhttpSettings':{'downloadSettings':{'address':'ignored.example','port':443},'extra':{'downloadSettings':{'address':'download.example','port':8443,'xhttpSettings':{'extra':{'downloadSettings':{'address':'download.example','port':8444}}}}}}}
  cfg={'outbounds':[{'protocol':'vless','settings':{'address':'upload.example','port':443},'streamSettings':stream}]}
  with patch.object(p,'cmd',return_value=R()) as run:
   r=p.endpoints(cfg)
   self.assertEqual(set(r),{'download.example','upload.example'})
   self.assertEqual(set(r['download.example']['ports']),{'8443','8444'})
   self.assertEqual(run.call_count,2)
  cfg['outbounds'][0]['streamSettings']['xhttpSettings']['extra']['downloadSettings']['address']='127.0.0.1'
  class Private:stdout=b'["127.0.0.1"]'
  with patch.object(p,'cmd',return_value=Private()):
   with self.assertRaisesRegex(ValueError,'nonpublic_endpoint_blocked'):p.endpoints(cfg)
  with self.assertRaisesRegex(ValueError,'invalid_download_endpoint'):
   p.download_endpoints({'xhttpSettings':{'downloadSettings':{'host':'not-a-dial-target'}}})

 def test_no_direct_success(self):
  c={'inbounds':[],'outbounds':[{'protocol':'freedom','settings':{},'tag':'direct'},{'protocol':'trojan','settings':{'address':'example.com','port':443,'password':'synthetic'}}],'routing':{'rules':[{'outboundTag':'direct','domain':['example.org']}]}}
  before=copy.deepcopy(c)
  with self.assertRaisesRegex(ValueError,'direct_route_not_qualified'):p.runtime_config(c,{'example.com':{'ips':['1.1.1.1'],'port':443}})
  self.assertEqual(c,before)
 def test_runtime_preserves_custom_dns_without_rewriting(self):
  c={'inbounds':[{'tag':'socks','listen':'127.0.0.1','port':10808,'protocol':'socks','settings':{'udp':True}}],'outbounds':[{'tag':'proxy','protocol':'trojan','settings':{'address':'example.com','port':443}}],'routing':{'rules':[]},'dns':{'hosts':{'example.com':'8.8.8.8'}}}
  self.assertEqual(p.runtime_config(c,{'example.com':{'ips':['1.1.1.1'],'port':443}}),c)
 def test_relay_scope_and_exact_runtime(self):
  c={'inbounds':[{'tag':'socks','listen':'127.0.0.1','port':10808,'protocol':'socks','settings':{'udp':True}}, {'tag':'relay','listen':'127.0.0.1','port':39081,'protocol':'dokodemo-door','settings':{'address':'1.1.1.1','port':443,'network':'tcp','followRedirect':False}}], 'outbounds':[{'tag':'proxy','protocol':'vless','settings':{'address':'127.0.0.1','port':39081,'id':'synthetic','encryption':'none'},'streamSettings':{'network':'tcp','security':'none'}},{'tag':'transport','protocol':'freedom','settings':{}},{'tag':'block','protocol':'blackhole','settings':{}}], 'routing':{'rules':[{'type':'field','inboundTag':['relay'],'network':'tcp','ip':['1.1.1.1'],'port':'443','outboundTag':'transport'},{'type':'field','inboundTag':['relay'],'network':'tcp,udp','outboundTag':'block'},{'type':'field','network':'tcp,udp','outboundTag':'proxy'}]}}
  self.assertEqual(p.runtime_config(c,{}),c)
  cases=[]
  bad=copy.deepcopy(c);bad['routing']['rules'][0]['port']='1-65535';cases.append(bad)
  bad=copy.deepcopy(c);bad['routing']['rules'].pop(1);cases.append(bad)
  bad=copy.deepcopy(c);bad['inbounds'][1]['listen']='0.0.0.0';cases.append(bad)
  bad=copy.deepcopy(c);bad['routing']['rules'].insert(0,{'outboundTag':'transport'});cases.append(bad)
  bad=copy.deepcopy(c);bad['inbounds'].append(copy.deepcopy(bad['inbounds'][0]));cases.append(bad)
  for bad in cases:
   with self.assertRaises(ValueError):p.runtime_config(bad,{})
 def test_mixed_public_private_dns_fails_closed(self):
  class R:stdout=b'["1.1.1.1","127.0.0.1"]'
  with patch.object(p,'cmd',return_value=R()):
   with self.assertRaisesRegex(ValueError,'nonpublic_endpoint_blocked'):
    p.endpoints({'outbounds':[{'protocol':'trojan','settings':{'address':'example.com','port':443}}]})
 def test_multicast_is_not_a_public_proxy(self):
  class R:stdout=b'["224.0.0.1"]'
  with patch.object(p,'cmd',return_value=R()):
   with self.assertRaisesRegex(ValueError,'nonpublic_endpoint_blocked'):
    p.endpoints({'outbounds':[{'protocol':'trojan','settings':{'address':'example.com','port':443}}]})
 def test_unsupported_does_not_guess(self):
  with self.assertRaisesRegex(ValueError,'outbound_driver_pending'):
   p.endpoints({'outbounds':[{'protocol':'future-protocol','settings':{'address':'example.com','port':443}}]})
 def test_same_host_multiple_ports_and_hopping_preserved(self):
  class R:stdout=b'["1.1.1.1"]'
  with patch.object(p,'cmd',return_value=R()):
   r=p.endpoints({'outbounds':[{'protocol':'hysteria','settings':{'address':'example.com','port':443},'streamSettings':{'finalmask':{'quicParams':{'udpHop':{'ports':'8443,9000-9010'}}}}},{'protocol':'trojan','settings':{'address':'example.com','port':444}}]})
   self.assertEqual(set(r['example.com']['ports']),{'443','444','8443','9000-9010'})
 def test_hopping_cannot_inject_firewall_syntax(self):
  with self.assertRaisesRegex(ValueError,'port_hopping_invalid'):
   p.endpoints({'outbounds':[{'protocol':'hysteria','settings':{'address':'example.com','port':443},'streamSettings':{'finalmask':{'quicParams':{'udpHop':{'ports':'443;accept'}}}}}]})
 def test_exact_candidate_hash_before_execution(self):
  with self.assertRaisesRegex(ValueError,'candidate_hash_mismatch'):
   p.test_candidate(None,{'sha256':'wrong','config':base64.b64encode(b'{}').decode()},1)

 def test_mixed_ipv4_ipv6_pool_keeps_all_original_outbounds(self):
  class V4:stdout=b'["1.1.1.1"]'
  class V6:stdout=b'["2606:4700:4700::1111"]'
  c={'outbounds':[{'tag':'a','protocol':'trojan','settings':{'address':'a.example','port':443}}, {'tag':'b','protocol':'trojan','settings':{'address':'b.example','port':443}}]}
  with patch.object(p,'cmd',side_effect=[V4(),V6()]):
   resolved=p.endpoints(c)
  self.assertEqual(set(resolved),{'a.example'})
  c['inbounds']=[{'listen':'127.0.0.1','port':10808,'protocol':'socks','settings':{'udp':True}}]
  runtime=p.runtime_config(c,resolved)
  self.assertEqual(runtime['outbounds'],c['outbounds'])
  with patch.object(p,'cmd',return_value=V6()):
   with self.assertRaisesRegex(ValueError,'ipv6_only_probe_pending'):p.endpoints(c)
 def test_mixed_pool_still_rejects_nonpublic_ipv6(self):
  class V4:stdout=b'["1.1.1.1"]'
  class V6:stdout=b'["fd00::1"]'
  c={'outbounds':[{'protocol':'trojan','settings':{'address':'a.example','port':443}}, {'protocol':'trojan','settings':{'address':'b.example','port':443}}]}
  with patch.object(p,'cmd',side_effect=[V4(),V6()]):
   with self.assertRaisesRegex(ValueError,'nonpublic_endpoint_blocked'):p.endpoints(c)

if __name__=='__main__':unittest.main()
