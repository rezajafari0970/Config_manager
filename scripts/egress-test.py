#!/usr/bin/env python3
"""Offline adversarial protocol/privacy contract tests; no live classifications."""
import asyncio,importlib.util,json,pathlib,tempfile,unittest
HERE=pathlib.Path(__file__).parent
def load(name,file):
 spec=importlib.util.spec_from_file_location(name,HERE/file);m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m
session=load('egress_session','egress-session.py');driver=load('egress_driver','egress-probe.py')
class Contracts(unittest.TestCase):
 def test_ipv6_scope_error_keeps_digit_without_leaking_payload(self):
  self.assertEqual(driver.error_code(ValueError('ipv6_only_probe_pending')),'ipv6_only_probe_pending')
  self.assertEqual(driver.error_code(ValueError('endpoint_dns_failed')),'endpoint_dns_failed')
  for error in [ValueError('secret.example.com'),ValueError('8.8.8.8'),RuntimeError('private payload')]:self.assertEqual(driver.error_code(error),'driver_infrastructure_error')
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
  original=session.measure
  async def run():
   arrived=[];all_started=asyncio.Event()
   async def fake(domain,variant,wave,timeout,control_ipv4=None):
    arrived.append((domain,wave))
    if len(arrived)==21:all_started.set()
    await asyncio.wait_for(all_started.wait(),5)
    return dict(domain=domain,wave=wave,echo=True,family=4,actual_ads=domain in session.DOMAINS)
   session.measure=fake
   rows=await session.session({'variant':'raw','waves':3})
   self.assertEqual(len(arrived),21);self.assertEqual(len(rows),15)
   self.assertEqual({x['wave'] for x in rows},{1,2,3})
  try:asyncio.run(run())
  finally:session.measure=original
 def test_ipv6_controls_share_primary_deadline(self):
  original=session.measure
  async def run():
   arrived=[];all_started=asyncio.Event()
   async def fake(domain,variant,wave,timeout,control_ipv4=None):
    arrived.append(domain)
    if len(arrived)==7:all_started.set()
    await asyncio.wait_for(all_started.wait(),5)
    return dict(domain=domain,wave=wave,echo=True,family=6 if domain in session.DOMAINS else 4,actual_ads=domain in session.DOMAINS)
   session.measure=fake
   rows=await session.session({'variant':'raw','waves':1})
   self.assertEqual(len(arrived),7);self.assertEqual(len(rows),7)
  try:asyncio.run(run())
  finally:session.measure=original
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
class RegressionContracts(unittest.TestCase):
 def test_dns_only_scope_preserves_raw_and_shared_gate(self):
  import copy
  cfg={'inbounds':[{'listen':'127.0.0.1','port':10808,'protocol':'socks'}],
       'outbounds':[{'protocol':'vless','tag':'proxy'},{'protocol':'freedom','tag':'direct'}],
       'routing':{'rules':[{'type':'field','port':'53','outboundTag':'direct'},
                           {'type':'field','ip':['1.1.1.1','104.16.248.249'],'outboundTag':'direct'}]}}
  before=copy.deepcopy(cfg)
  obj=driver.scope.inspect(cfg,driver.lab.preflight)
  self.assertEqual(obj['routing']['rules'],[]);self.assertEqual(cfg,before)
  driver.lab.preflight.runtime_config(obj,{})
  with self.assertRaisesRegex(ValueError,'direct_route_not_qualified'):driver.lab.preflight.runtime_config(cfg,{})
  for rule in [{'port':'443','outboundTag':'direct'},{'ip':['1.1.1.0/24'],'outboundTag':'direct'},
               {'ip':['1.1.1.1','8.7.6.5'],'outboundTag':'direct'},
               {'domain':['googleads.g.doubleclick.net'],'outboundTag':'direct'},
               {'ip':['1.1.1.1'],'unknown':'unsafe','outboundTag':'direct'}]:
   cfg['routing']['rules']=[rule]
   with self.assertRaisesRegex(ValueError,'direct_route_not_qualified'):driver.lab.preflight.runtime_config(driver.scope.inspect(cfg,driver.lab.preflight),{})
 def test_control_family_is_explicit_and_never_direct(self):
  import ipaddress
  target=session.socks_target('www.cloudflare.com','104.16.123.96')
  self.assertEqual(target,b'\x01'+ipaddress.ip_address('104.16.123.96').packed)
  self.assertEqual(session.socks_target('one.one.one.one')[0],4)
  self.assertEqual(session.socks_target(session.DOMAINS[0])[0],3)
  for ip in (None,'127.0.0.1','::1','2606:4700:4700::1111','not-an-ip'):
   with self.assertRaisesRegex(ValueError,'control_ipv4_unavailable'):session.socks_target('www.cloudflare.com',ip)
 def test_control_does_not_use_source_dns_bypass_ip(self):
  self.assertEqual(session.CONTROLS[:2],['www.cloudflare.com','api.ipify.org'])
 def test_deep_waves_are_bounded_and_sequential(self):
  original=session.measure;active=set();overlap=[]
  async def fake(domain,variant,wave,timeout,control_ipv4=None):
   active.add((domain,wave));overlap.append(len({w for _,w in active}));await asyncio.sleep(.01);active.remove((domain,wave))
   return dict(domain=domain,wave=wave,echo=True,family=4,actual_ads=domain in session.DOMAINS)
  session.measure=fake
  try:
   rows=asyncio.run(session.session({'variant':'raw','waves':3,'sequential':True}))
   self.assertEqual(len(rows),15);self.assertEqual(max(overlap),1)
  finally:session.measure=original
class PrivacyContracts(unittest.TestCase):
 def test_only_job_local_opaque_equality_survives(self):
  rows=[dict(ip='8.8.8.8',echo=True),dict(ip='8.8.8.8',echo=True),dict(ip='1.1.1.1',echo=True),dict(ip='',echo=False)]
  out=driver.private_sources(rows,b'a'*32)
  self.assertTrue(all('ip' not in r for r in out));self.assertEqual(out[0]['source'],out[1]['source']);self.assertNotEqual(out[0]['source'],out[2]['source']);self.assertEqual(out[3]['source'],'')
  other=driver.private_sources([dict(ip='8.8.8.8',echo=True)],b'b'*32)
  self.assertNotEqual(other[0]['source'],out[0]['source'])
 def test_raw_shortcut_requires_every_repeated_domain(self):
  rows=[dict(wave=w,domain=d,echo=True,tls=True,same_stream=True,actual_ads=d in session.DOMAINS,family=4,ip='1.1.1.1') for w in (1,2) for d in session.DOMAINS+session.CONTROLS[:2]]
  self.assertTrue(driver.raw_shared_egress(rows,2))
  rows[0]['ip']='8.8.8.8';self.assertFalse(driver.raw_shared_egress(rows,2))
  rows[0]['ip']='1.1.1.1';rows[0]['echo']=False;self.assertFalse(driver.raw_shared_egress(rows,2))
 def test_raw_shortcut_pairs_ipv6_without_comparing_to_ipv4(self):
  rows=[]
  for wave in (1,2):
   for domain in session.DOMAINS+session.CONTROLS:
    family=4 if domain=='www.cloudflare.com' else 6
    rows.append(dict(wave=wave,domain=domain,family=family,ip='1.1.1.1' if family==4 else '2606:4700:4700::1111',echo=True,tls=True,same_stream=True,actual_ads=domain in session.DOMAINS))
  self.assertTrue(driver.raw_shared_egress(rows,2))
  rows[0]['ip']='2001:4860:4860::8888';self.assertFalse(driver.raw_shared_egress(rows,2))
 def test_driver_has_no_network_class_dependency(self):
  self.assertFalse(hasattr(driver,'intel'));self.assertFalse((HERE/'egress-classify.py').exists())
class AdaptiveContracts(unittest.TestCase):
 def simulate(self,change):
  original=session.measure;calls=[]
  async def fake(domain,variant,wave,timeout,control_ipv4=None):
   calls.append((domain,wave))
   row=dict(domain=domain,wave=wave,echo=True,tls=True,same_stream=True,family=6 if domain in session.CONTROLS[2:] else 4,actual_ads=domain in session.DOMAINS)
   change(row);return row
  session.measure=fake
  try:return asyncio.run(session.session(dict(variant='raw',waves=3,sequential=True,adaptive=True))),calls
  finally:session.measure=original
 def test_two_complete_paired_waves_skip_third(self):
  rows,calls=self.simulate(lambda r:None)
  self.assertEqual(len(rows),10);self.assertEqual({w for d,w in calls},{1,2})
 def test_third_wave_retries_only_missing_domain_and_controls(self):
  def change(r):
   if r['domain']==session.DOMAINS[0] and r['wave']==1:r['echo']=False
  rows,calls=self.simulate(change)
  self.assertEqual({d for d,w in calls if w==3 and d in session.DOMAINS},{session.DOMAINS[0]})
  self.assertEqual(len(rows),13)
 def test_family_change_needs_third_wave(self):
  def change(r):
   if r['domain'] in session.DOMAINS and r['wave']==2:r['family']=6
  rows,calls=self.simulate(change)
  self.assertEqual({d for d,w in calls if w==3 and d in session.DOMAINS},set(session.DOMAINS))
 def test_unpaired_controls_do_not_skip_retry(self):
  def change(r):
   if r['domain']=='api.ipify.org' and r['wave']==2:r['echo']=False
  rows,calls=self.simulate(change)
  self.assertEqual({d for d,w in calls if w==3 and d in session.DOMAINS},set(session.DOMAINS))
 def test_flat_vless_dispatch_preserves_tls_and_local_semantics(self):
  import base64
  for address,security,encryption,want in [('8.8.8.8','none','none','26.5.9'),('example.com','none','none','26.5.9'),('127.0.0.1','none','none','26.11.9'),('8.8.8.8','tls','none','26.11.9'),('8.8.8.8','none','mlkem768x25519plus.native.0rtt.key','26.11.9')]:
   cfg={'outbounds':[{'protocol':'vless','settings':{'address':address,'port':443,'id':'fixture','encryption':encryption},'streamSettings':{'security':security}}]}
   plan={'variants':[{'config':base64.b64encode(json.dumps(cfg).encode()).decode()}]}
   self.assertEqual(driver.core_order(plan)[0][0],'xray-'+want)
if __name__=='__main__':unittest.main()


