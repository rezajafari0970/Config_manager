#!/usr/bin/env python3
"""Offline adversarial protocol/classifier contract tests; no live classifications."""
import asyncio,importlib.util,json,pathlib,tempfile,unittest
HERE=pathlib.Path(__file__).parent
def load(name,file):
 spec=importlib.util.spec_from_file_location(name,HERE/file);m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m
session=load('egress_session','egress-session.py');intel=load('egress_intel','egress-classify.py');driver=load('egress_driver','egress-probe.py')
class Contracts(unittest.TestCase):
 def response(self,data,close=False):
  async def run():
   reader=asyncio.StreamReader();reader.feed_data(data);reader.feed_eof();return await session.read_response(reader,close)
  return asyncio.run(run())
 def test_framed_redirect_does_not_reconnect(self):
  self.assertEqual(self.response(b'HTTP/1.1 301 Moved\r\nContent-Length: 0\r\nLocation: https://elsewhere/\r\n\r\n')[0],301)
 def test_ambiguous_framing_rejected(self):
  for data in [b'Content-Length: 0\r\nTransfer-Encoding: chunked',b'Content-Length: 0\r\nContent-Length: 2']:
   with self.assertRaises(ValueError):self.response(b'HTTP/1.1 200 OK\r\n'+data+b'\r\n\r\n')
 def test_persistent_unframed_rejected(self):
  with self.assertRaises(ValueError):self.response(b'HTTP/1.1 200 OK\r\n\r\nbody')
 def test_chunked_consumes_trailers(self):
  self.assertEqual(self.response(b'HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n3\r\nabc\r\n0\r\nFoo: bar\r\n\r\n')[2],b'abc')
 def test_explicit_class_and_exact_ip_required(self):
  b=json.dumps({'status':'ok','8.8.8.8':{'network':{'asn':'AS4609','type':None},'location':{'country_code':'MO'}}}).encode();r=intel.parse('8.8.8.8',b,1000);self.assertEqual(r['class_'],'Unknown')
  with self.assertRaises(ValueError):intel.parse('1.1.1.1',b,1000)
 def test_daily_budget_survives_restart(self):
  import datetime
  with tempfile.TemporaryDirectory() as d:
   p=pathlib.Path(d)/'intel.json';p.write_text(json.dumps({'day':datetime.datetime.now(datetime.timezone.utc).date().isoformat(),'used':90,'entries':{}}));self.assertEqual(intel.classify(['8.8.8.8'],str(p)),[]);self.assertEqual(json.loads(p.read_text())['used'],90)
 def test_disjoint_resources(self):
  self.assertNotEqual(driver.lab.SANDBOX_ADDRESS_BASES['cge'],driver.lab.SANDBOX_ADDRESS_BASES['crw']);self.assertNotEqual(driver.lab.SANDBOX_ADDRESS_BASES['cge'],driver.lab.SANDBOX_ADDRESS_BASES['cgt'])
 def test_second_authority_not_assumed(self):
  self.assertFalse(session.authority({'subjectAltName':[('DNS','*.doubleclick.net')]},'www.google.com'))
if __name__=='__main__':unittest.main()
