#!/usr/bin/env python3
"""Native preparation never widens the endpoint firewall or invents TLS values."""
import copy,importlib.util,pathlib,socket,unittest
from unittest.mock import patch
spec=importlib.util.spec_from_file_location('native',pathlib.Path(__file__).with_name('residential-native.py'))
native=importlib.util.module_from_spec(spec);spec.loader.exec_module(native)
class Tests(unittest.TestCase):
 def config(self):return {'inbounds':[{'type':'socks','tag':'custom'}],'outbounds':[{'type':'tuic','server':'proxy.example','server_port':8443,'tls':{'enabled':True,'server_name':'auth.example'}}],'route':{'final':'proxy'}}
 def test_pin_only_upstream_and_keep_original(self):
  original=self.config();before=copy.deepcopy(original)
  with patch.object(native.socket,'getaddrinfo',return_value=[(socket.AF_INET,socket.SOCK_DGRAM,0,'',('8.8.8.8',8443))]):resolved,runtime=native.prepare(original)
  self.assertEqual(original,before);self.assertEqual(runtime['outbounds'][0]['server'],'8.8.8.8');self.assertEqual(runtime['outbounds'][0]['tls']['server_name'],'auth.example')
  self.assertEqual(set(resolved),{'proxy.example'});self.assertEqual(runtime['route'],before['route']);self.assertEqual(runtime['inbounds'][0]['tag'],'custom')
 def test_native_ipv6_without_transport_driver_is_unsupported(self):
  c=self.config();c['outbounds'][0]['server']='2606:4700::1111'
  with self.assertRaisesRegex(ValueError,'ipv6_only_probe_pending'):native.prepare(c)
 def test_private_mixed_resolution_never_runs(self):
  answers=[(socket.AF_INET,socket.SOCK_DGRAM,0,'',(ip,8443)) for ip in ['8.8.8.8','127.0.0.1']]
  with patch.object(native.socket,'getaddrinfo',return_value=answers):
   with self.assertRaisesRegex(ValueError,'endpoint_not_public'):native.prepare(self.config())
 def test_unknown_dependencies_and_port_hopping_are_explicit(self):
  for key in ['detour','domain_resolver','server_ports']:
   c=self.config();c['outbounds'][0][key]='unsupported'
   with self.assertRaisesRegex(ValueError,'outbound_driver_pending'):native.prepare(c)
if __name__=='__main__':unittest.main()
