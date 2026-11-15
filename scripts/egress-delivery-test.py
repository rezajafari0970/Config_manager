#!/usr/bin/env python3
"""Hermetic contracts for the real bootstrap-asset driver, without ad traffic."""
import importlib.util,json,pathlib,subprocess,unittest
from unittest.mock import patch
spec=importlib.util.spec_from_file_location('delivery',pathlib.Path(__file__).with_name('egress-delivery.py'))
d=importlib.util.module_from_spec(spec);spec.loader.exec_module(d)
TARGET={'profile':'doubleclick.v1','domain':'securepubads.g.doubleclick.net','path':'/tag/js/gpt.js','marker':'googletag'}
class Contracts(unittest.TestCase):
 def result(self,body=b'googletag'+b' '*1200,meta=None,exit=0,host=False):
  metadata={'http_code':200,'ssl_verify_result':0,'content_type':'text/javascript; charset=UTF-8','size_download':len(body)}
  metadata.update(meta or {})
  def fake(args,**kw):
   self.assertNotIn('--insecure',args);self.assertNotIn('--location',args)
   self.assertEqual(args[args.index('--max-filesize')+1],str(d.LIMIT))
   self.assertEqual(args[-1],'https://securepubads.g.doubleclick.net/tag/js/gpt.js')
   self.assertEqual('--proxy' in args,not host)
   if not host:self.assertEqual(args[args.index('--proxy')+1],'socks5h://127.0.0.1:10808')
   pathlib.Path(args[args.index('--output')+1]).write_bytes(body)
   return subprocess.CompletedProcess(args,exit,json.dumps(metadata).encode(),b'')
  with patch.object(d.subprocess,'run',side_effect=fake):return d.transfer(TARGET,3,host)
 def test_bounded_verified_script(self):
  a=self.result();self.assertEqual(a['code'],'asset_available');self.assertEqual(len(a['nonce']),32);self.assertEqual(len(a['body_sha']),64)
 def test_wrong_mime_block_page_is_not_available(self):
  self.assertFalse(self.result(meta={'content_type':'text/html'})['content_match'])
 def test_empty_truncated_unverified_or_failed_never_available(self):
  for body,meta,exit in [(b'',{},0),(b'googletag'+b' '*1100,{'size_download':9999},0),(b'googletag'+b' '*1100,{'ssl_verify_result':20},0),(b'googletag'+b' '*1100,{},28)]:
   with self.subTest(meta=meta,exit=exit):self.assertNotEqual(self.result(body,meta,exit)['code'],'asset_available')
 def test_resource_limit(self):
  self.assertEqual(self.result(b'x'*(d.LIMIT+1))['curl_exit'],124)
 def test_bad_metadata_stays_unknown(self):
  with patch.object(d.subprocess,'run',return_value=subprocess.CompletedProcess([],0,b'not-json',b'')):
   self.assertEqual(d.transfer(TARGET,3)['curl_exit'],124)
 def test_host_control_is_explicit_direct(self):self.assertEqual(self.result(host=True)['code'],'asset_available')
 def test_registry_rejects_malformed_duplicate_and_injected_targets(self):
  for value in [None,[],[TARGET,TARGET],[dict(TARGET,domain='evil/route')],[dict(TARGET,path='/x?y=1')],[dict(TARGET,marker='bad marker')]]:
   with self.subTest(value=value),self.assertRaises((ValueError,TypeError)):d.registry(value)
 def test_no_implicit_unsupported_timeout(self):
  with self.assertRaises(ValueError):d.run({'targets':[TARGET],'timeout':30})
if __name__=='__main__':unittest.main()
