#!/usr/bin/env python3
"""Unprivileged regression tests. Kernel sandbox acceptance is --self-test."""
import base64, copy, importlib.util, json, pathlib, tempfile, unittest
from unittest.mock import patch
spec=importlib.util.spec_from_file_location('probe',pathlib.Path(__file__).with_name('live-probe.py'))
p=importlib.util.module_from_spec(spec);spec.loader.exec_module(p)

class ProbeTests(unittest.TestCase):
 def test_classification_removed_tls(self):
  self.assertEqual(p.classify('The feature "allowInsecure" has been removed','unknown'),'removed_allow_insecure')
  self.assertEqual(p.classify('trojan without TLS is prohibited','unknown'),'public_endpoint_requires_tls')
  self.assertEqual(p.classify('vless without TLS or other encryption is prohibited','unknown'),'public_endpoint_requires_tls')
 def test_no_direct_success(self):
  c={'inbounds':[],'outbounds':[{'protocol':'freedom','settings':{},'tag':'direct'},{'protocol':'trojan','settings':{'address':'example.com','port':443,'password':'synthetic'}}],'routing':{'rules':[{'outboundTag':'direct','domain':['example.org']}]}}
  before=copy.deepcopy(c);r=p.runtime_config(c,{'example.com':{'ips':['1.1.1.1'],'port':443}})
  self.assertEqual(c,before);self.assertEqual(r['outbounds'][0]['protocol'],'blackhole');self.assertEqual(r['routing'],c['routing'])
 def test_dns_override_is_not_silent(self):
  with self.assertRaisesRegex(ValueError,'custom_dns_mapping_requires_review'):
   p.runtime_config({'dns':{'hosts':{'example.com':'8.8.8.8'}}},{'example.com':{'ips':['1.1.1.1'],'port':443}})
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

if __name__=='__main__':unittest.main()
