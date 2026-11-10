#!/usr/bin/env python3
import asyncio, importlib.util, pathlib, unittest
from unittest.mock import patch
spec=importlib.util.spec_from_file_location('res',pathlib.Path(__file__).with_name('residential-probe.py'))
res=importlib.util.module_from_spec(spec);spec.loader.exec_module(res)

class Tests(unittest.IsolatedAsyncioTestCase):
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
