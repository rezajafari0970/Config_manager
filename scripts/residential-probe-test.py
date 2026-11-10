#!/usr/bin/env python3
import asyncio, importlib.util, pathlib, unittest
from unittest.mock import patch
spec=importlib.util.spec_from_file_location('res',pathlib.Path(__file__).with_name('residential-probe.py'))
res=importlib.util.module_from_spec(spec);spec.loader.exec_module(res)

class Tests(unittest.IsolatedAsyncioTestCase):
 async def test_healthy_screen_expands_all_explicit_domains_in_first_wave(self):
  async def fake(host,variant,wave,timeout,path):return {'domain':host,'tls_verified':True,'wave':wave,'outcome':'passed','family':6}
  domains=['googleads.g.doubleclick.net','pubads.g.doubleclick.net','securepubads.g.doubleclick.net','www.googleadservices.com','admob.com']
  with patch.object(res,'connection_probe',fake):
   result=await res.session({'policy':{'domains':domains,'repeats':3,'timeout_seconds':1},'variant':{'name':'raw','paths':['single']}})
  self.assertEqual(len(result['observations']),15)
  self.assertEqual({r['domain'] for r in result['observations'] if r['wave']==1},set(domains))
 async def test_ipv4_ads_gets_independent_ipv4_controls_without_borrowing_ipv6(self):
  async def fake(host,variant,wave,timeout,path):
   return {'domain':host,'tls_verified':True,'wave':wave,'outcome':'passed','family':6 if host in res.CONTROLS else 4}
  with patch.object(res,'connection_probe',fake):
   result=await res.session({'policy':{'domains':['www.googleadservices.com'],'repeats':1,'timeout_seconds':1},'variant':{'name':'raw','paths':['single']}})
  self.assertEqual(len(result['controls']),4);self.assertEqual(len(result['observations']),1)
  self.assertEqual({res.control_group(x['domain']) for x in result['controls'] if x['family']==4},{'cloudflare','ipify'})
  self.assertTrue(res.controls_healthy(result['controls'],1))
 def test_second_authority_requires_matching_certificate_san(self):
  self.assertTrue(res.certificate_authority({'subjectAltName':[('DNS','*.google.com')]},'www.google.com'))
  for name in ['*.doubleclick.net','www.google.com.evil','*.*.com','*.www.google.com']:
   self.assertFalse(res.certificate_authority({'subjectAltName':[('DNS',name)]},'www.google.com'))
  self.assertFalse(res.certificate_authority({'subject':(('commonName','www.google.com'),)},'www.google.com'))
 async def test_parallel_screening_covers_three_domains_and_two_controls(self):
  active=maximum=0
  async def fake(host,variant,wave,timeout,path):
   nonlocal active,maximum
   active+=1;maximum=max(maximum,active);await asyncio.sleep(.001);active-=1
   return {'domain':host,'tls_verified':True,'wave':wave}
  request={'policy':{'domains':['pagead2.googlesyndication.com','googleads.g.doubleclick.net','pubads.g.doubleclick.net','securepubads.g.doubleclick.net'],'repeats':3,'timeout_seconds':1},'variant':{'name':'raw','paths':['single']}}
  with patch.object(res,'connection_probe',fake):result=await res.session(request)
  self.assertEqual(len(result['observations']),9);self.assertEqual(len(result['controls']),6)
  self.assertEqual({x['domain'] for x in result['observations']},{'googleads.g.doubleclick.net','pubads.g.doubleclick.net','securepubads.g.doubleclick.net'})
  self.assertEqual(maximum,5)
 async def read(self,data):
  r=asyncio.StreamReader(limit=32769);r.feed_data(data);r.feed_eof();return await res.response(r)
 async def test_exact_framing_for_same_stream(self):
  status,_,body=await self.read(b'HTTP/1.1 200 OK\r\nContent-Length: 4\r\n\r\ntest')
  self.assertEqual((status,body),(200,b'test'))
  self.assertEqual((await self.read(b'HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n4\r\ntest\r\n0\r\n\r\n'))[2],b'test')
 async def test_close_delimited_control_is_bounded_and_cannot_reuse_stream(self):
  reader=asyncio.StreamReader();reader.feed_data(b'HTTP/1.1 200 OK\r\nConnection: close\r\n\r\n{"ip":"192.0.2.1"}');reader.feed_eof()
  self.assertEqual((await res.response(reader,allow_close=True))[2],b'{"ip":"192.0.2.1"}')
  reader=asyncio.StreamReader();reader.feed_data(b'HTTP/1.1 200 OK\r\n\r\n'+b'x'*65537);reader.feed_eof()
  with self.assertRaises(ValueError):await res.response(reader,allow_close=True)
 async def test_one_control_retry_never_replaces_ads_observation(self):
  calls={}
  async def fake(host,variant,wave,timeout,path):
   calls[host]=calls.get(host,0)+1
   return {'domain':host,'tls_verified':True,'wave':wave,'outcome':'failed' if host==res.CONTROLS[0] and calls[host]==1 else 'passed'}
  with patch.object(res,'connection_probe',fake):
   result=await res.session({'policy':{'domains':['googleads.g.doubleclick.net'],'repeats':1,'timeout_seconds':1},'variant':{'name':'raw','paths':['single']}})
  self.assertEqual(calls[res.CONTROLS[0]],2);self.assertEqual(calls['googleads.g.doubleclick.net'],1)
  self.assertEqual(len(result['controls']),2);self.assertEqual(len(result['observations']),1)
 async def test_ambiguous_oversized_and_unframed_rejected(self):
  for data in [b'HTTP/1.1 200 OK\r\n\r\nhello',b'HTTP/1.1 200 OK\r\nContent-Length: 1\r\nContent-Length: 2\r\n\r\nx',b'HTTP/1.1 200 OK\r\nContent-Length: 999999\r\n\r\n',b'HTTP/1.1 200 OK\r\nContent-Length: 1\r\nTransfer-Encoding: chunked\r\n\r\n']:
   with self.assertRaises(ValueError):await self.read(data)
 def test_strict_policy_has_no_dns_or_direct_allowance(self):
  class Box:
   def net(self,args,**kwargs):self.rules=kwargs['input'].decode()
  box=Box();res.strict_policy(box,{'proxy':{'ips':['192.0.2.1'],'port':8443}})
  self.assertIn('policy drop',box.rules);self.assertNotIn('dport 53',box.rules);self.assertNotIn('1.1.1.1',box.rules)
  self.assertIn('ip daddr 192.0.2.1 tcp dport 8443 accept',box.rules)
 def test_resource_scopes_disjoint(self):
  self.assertNotEqual(res.lab.SANDBOX_ADDRESS_BASES['crr'],res.lab.SANDBOX_ADDRESS_BASES['crw'])
  self.assertNotEqual(res.lab.SANDBOX_ADDRESS_BASES['crr'],res.lab.SANDBOX_ADDRESS_BASES['crt'])

if __name__=='__main__':unittest.main()
