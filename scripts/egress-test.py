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
class PerformanceContracts(unittest.TestCase):
 def test_safe_repeated_nonframing_header(self):
  async def check():
   r=asyncio.StreamReader();r.feed_data(b'HTTP/1.1 200 OK\r\nSet-Cookie: a=b\r\nSet-Cookie: c=d\r\nContent-Length: 0\r\n\r\n');r.feed_eof()
   self.assertEqual((await session.read_response(r,False))[0],200)
  asyncio.run(check())
 def test_overlapping_independent_waves(self):
  import time
  original=session.measure
  async def fake(domain,variant,wave,timeout):
   await asyncio.sleep(.15)
   return dict(domain=domain,wave=wave,echo=True,family=4,actual_ads=domain in session.DOMAINS)
  session.measure=fake
  try:
   t=time.monotonic();rows=asyncio.run(session.session({'variant':'raw','waves':3}))
   self.assertEqual(len(rows),15);self.assertLess(time.monotonic()-t,.65)
   self.assertEqual({x['wave'] for x in rows},{1,2,3})
  finally:session.measure=original
 def test_ipv6_controls_share_primary_deadline(self):
  import time
  original=session.measure
  async def fake(domain,variant,wave,timeout):
   await asyncio.sleep(.15)
   return dict(domain=domain,wave=wave,echo=True,family=6 if domain in session.DOMAINS else 4,actual_ads=domain in session.DOMAINS)
  session.measure=fake
  try:
   t=time.monotonic();rows=asyncio.run(session.session({'variant':'raw','waves':1}))
   self.assertEqual(len(rows),7);self.assertLess(time.monotonic()-t,.25)
  finally:session.measure=original
 def test_network_lookup_does_not_hold_budget_lock(self):
  import threading,time
  original=intel.fetch;entered=threading.Event();release=threading.Event();calls=[]
  def fetch(ip):
   calls.append(ip);entered.set();release.wait(2)
   return dict(ip=ip,expires=int(time.time()*1000)+60000,code='explicit_network_type',**{'class':'Residential'})
  intel.fetch=fetch
  try:
   with tempfile.TemporaryDirectory() as directory:
    path=str(pathlib.Path(directory)/'intel.json')
    thread=threading.Thread(target=intel.classify,args=(['8.8.8.8'],path));thread.start();self.assertTrue(entered.wait(1))
    start=time.monotonic();self.assertEqual(intel.classify(['8.8.8.8'],path),[]);self.assertLess(time.monotonic()-start,.25)
    release.set();thread.join(2);self.assertEqual(calls,['8.8.8.8']);self.assertEqual(intel.status(path)['used'],1)
  finally:release.set();intel.fetch=original
 def test_max_slot_is_disjoint_and_owned(self):
  a=driver.lab.Sandbox('/tmp/fixture','abcdef',8192+31,prefix='cge')
  b=driver.lab.Sandbox('/tmp/fixture','abcdef',8192+1,prefix='cgt')
  self.assertNotEqual(a.addr,b.addr);self.assertEqual(a.name,'cgeabcdef201f')
class DispatchContracts(unittest.TestCase):
 def test_plaintext_capability_dispatch_does_not_downgrade_tls(self):
  import base64
  def plan(address,security):
   body={'outbounds':[{'protocol':'vless','settings':{'vnext':[{'address':address,'port':443,'users':[{'encryption':'none'}]}]},'streamSettings':{'security':security}}]}
   return {'variants':[{'config':base64.b64encode(json.dumps(body).encode()).decode()}]}
  self.assertEqual(driver.core_order(plan('8.8.8.8','none'))[0][0],'xray-26.5.9')
  self.assertEqual(driver.core_order(plan('example.com','none'))[0][0],'xray-26.5.9')
  self.assertEqual(driver.core_order(plan('8.8.8.8','tls'))[0][0],'xray-26.11.9')
  self.assertEqual(driver.core_order(plan('127.0.0.1','none'))[0][0],'xray-26.11.9')
 def test_classifier_budget_only_for_same_family_split(self):
  controls=[dict(domain=d,ip='1.1.1.1',family=4,tls=True,echo=True) for d in ('1.1.1.1','api.ipify.org')]
  ads=dict(domain=session.DOMAINS[0],ip='1.1.1.1',family=4,tls=True,echo=True,same_stream=True,actual_ads=True)
  self.assertEqual(driver.classification_candidates(controls+[ads]),[])
  ads['ip']='8.8.8.8';self.assertEqual(driver.classification_candidates(controls+[ads]),['8.8.8.8'])
  ads.update(ip='2001:4860:4860::8888',family=6);self.assertEqual(driver.classification_candidates(controls+[ads]),[])
 def test_failed_firewall_update_is_never_cached(self):
  original=driver.lab.Sandbox.policy;calls=[]
  def policy(box,resolved):
   calls.append(resolved)
   if len(calls)==1:raise OSError('synthetic denied rule update')
  driver.lab.Sandbox.policy=policy
  try:
   box=object.__new__(driver.EgressBox)
   with self.assertRaises(OSError):box.policy({})
   box.policy({});box.policy({});self.assertEqual(len(calls),2)
  finally:driver.lab.Sandbox.policy=original
if __name__=='__main__':unittest.main()


