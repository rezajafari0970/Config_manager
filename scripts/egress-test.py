#!/usr/bin/env python3
"""Offline adversarial protocol/privacy contract tests; no live classifications."""
import asyncio,importlib.util,json,pathlib,tempfile,unittest
HERE=pathlib.Path(__file__).parent
def load(name,file):
 spec=importlib.util.spec_from_file_location(name,HERE/file);m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m);return m
session=load('egress_session','egress-session.py');driver=load('egress_driver','egress-probe.py')
class DiscoveryContracts(unittest.TestCase):
 def test_discovery_measures_both_variants_and_all_three_waves_despite_overlap(self):
  import base64,contextlib,hashlib,types,time
  from unittest.mock import patch
  calls=[];config=b'{"outbounds":[]}';digest=hashlib.sha256(config).hexdigest()
  class Box:
   def policy(self,value):pass
   def stop(self):pass
   def launch(self,value):return types.SimpleNamespace(poll=lambda:None)
   def net(self,args,**kw):
    if args[-1].endswith('egress-session.py'):
     q=json.loads(kw['input']);calls.append(q);rows=[]
     for wave in range(1,q['waves']+1):
      for i,t in enumerate(q['targets']):
       nonce=f'{wave:016x}{i:016x}'
       rows.append(dict(domain=t['domain'],variant=q['variant'],wave=wave,nonce=nonce,connection=nonce,ip='1.1.1.1',family=4,tls=True,same_stream=True,echo=True,actual_ads=t['role']=='ads'))
     return types.SimpleNamespace(stdout=json.dumps(rows).encode())
    return types.SimpleNamespace(stdout=b'0')
  box=Box();pool=types.SimpleNamespace(verify=lambda *a:None,lease=lambda *a:contextlib.nullcontext(box),geo=types.SimpleNamespace(attach=lambda rows:None))
  args=types.SimpleNamespace(prefix='cgt',run_id='abcdef',slot=31)
  req=dict(hash='synthetic',started=int(time.time()*1000),deep=True,accepted=False,plan=dict(driver='xray',variants=[dict(name=v,hash=digest,config=base64.b64encode(config).decode()) for v in ('raw','template')]))
  with patch.object(driver,'runtime',lambda *a:({},{})),patch.object(driver,'resource_failures',lambda *a:0),patch.object(driver,'diagnostic_delivery',lambda *a:dict(assets=[dict(curl_exit=0,http_status=200,tls_verified=True,complete=True,content_match=True)])):
   old=driver.investigate(req,args,pool)
   self.assertEqual(old['code'],'measured');self.assertEqual(len(old['checks']),2);self.assertEqual(len(old['evidence']),1);self.assertTrue(old['evidence'][0]['selected_samples']);self.assertEqual(old['evidence'][0]['samples'],[])
   calls.clear();req['discovery']=True;new=driver.investigate(req,args,pool)
  self.assertEqual(len(new['checks']),2);self.assertEqual(new['code'],'measured')
  self.assertEqual(len(new['samples']),6*len(session.DEFAULT_TARGETS))
  self.assertEqual([q['variant'] for q in calls],['raw','template'])
  for q in calls:self.assertEqual(q['waves'],3);self.assertTrue(q['sequential']);self.assertFalse(q['adaptive'])
class RenewalBudgetContracts(unittest.TestCase):
 def test_verified_inputs_do_not_silently_use_short_initial_deadline(self):
  self.assertEqual(driver.observation_budget({'deep':True,'accepted':True}),(3,8))
  self.assertEqual(driver.observation_budget({'deep':True,'accepted':False}),(3,8))
  self.assertEqual(driver.observation_budget({}),(2,3))
  self.assertEqual(driver.observation_budget({'discovery':True}),(3,8))
class Contracts(unittest.TestCase):
 def test_host_dns_failure_does_not_rotate_compatible_cores(self):
  self.assertTrue(driver.core_independent_failure('endpoint_dns_failed'))
  self.assertTrue(driver.core_independent_failure('ipv6_only_probe_pending'))
  for code in ('core_rejected','runtime_start_failed','core_schema_mismatch','driver_subprocess_timeout'):
   self.assertFalse(driver.core_independent_failure(code))
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
   async def fake(domain,variant,wave,timeout,control_ipv4=None,target=None):
    arrived.append((domain,wave))
    if len(arrived)==min(12,3*len(session.DEFAULT_TARGETS)):all_started.set()
    await asyncio.wait_for(all_started.wait(),5)
    return dict(domain=domain,wave=wave,echo=True,family=4,actual_ads=domain in session.DOMAINS)
   session.measure=fake
   rows=await session.session({'variant':'raw','waves':3})
   self.assertEqual(len(arrived),3*len(session.DEFAULT_TARGETS));self.assertEqual(len(rows),3*len(session.DEFAULT_TARGETS))
   self.assertEqual({x['wave'] for x in rows},{1,2,3})
  try:asyncio.run(run())
  finally:session.measure=original
 def test_ipv6_controls_share_primary_deadline(self):
  original=session.measure
  async def run():
   arrived=[];all_started=asyncio.Event()
   async def fake(domain,variant,wave,timeout,control_ipv4=None,target=None):
    arrived.append(domain)
    if len(arrived)==len(session.DEFAULT_TARGETS):all_started.set()
    await asyncio.wait_for(all_started.wait(),5)
    return dict(domain=domain,wave=wave,echo=True,family=6 if domain in session.DOMAINS else 4,actual_ads=domain in session.DOMAINS)
   session.measure=fake
   rows=await session.session({'variant':'raw','waves':1})
   self.assertEqual(len(arrived),len(session.DEFAULT_TARGETS));self.assertEqual(len(rows),len(session.DEFAULT_TARGETS))
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
 def test_domain_controls_preserve_domain_routing(self):
  for name in session.CONTROLS:
   for supplied in [None,'104.16.123.96','127.0.0.1']:
    target=session.socks_target(name,supplied)
    self.assertEqual(target[0],3)
    self.assertEqual(target[2:].decode(),name)

 def test_control_does_not_use_source_dns_bypass_ip(self):
  self.assertIn('www.google.com',session.CONTROLS);self.assertIn('www.cloudflare.com',session.CONTROLS)
 def test_deep_waves_are_bounded_and_sequential(self):
  original=session.measure;active=set();overlap=[]
  async def fake(domain,variant,wave,timeout,control_ipv4=None,target=None):
   active.add((domain,wave));overlap.append(len({w for _,w in active}));await asyncio.sleep(.01);active.remove((domain,wave))
   return dict(domain=domain,wave=wave,echo=True,family=4,actual_ads=domain in session.DOMAINS)
  session.measure=fake
  try:
   rows=asyncio.run(session.session({'variant':'raw','waves':3,'sequential':True}))
   self.assertEqual(len(rows),3*len(session.DEFAULT_TARGETS));self.assertEqual(max(overlap),1)
  finally:session.measure=original
class PrivacyContracts(unittest.TestCase):
 def test_only_job_local_opaque_equality_survives(self):
  rows=[dict(ip='8.8.8.8',echo=True),dict(ip='8.8.8.8',echo=True),dict(ip='1.1.1.1',echo=True),dict(ip='',echo=False)]
  out=driver.private_sources(rows,b'a'*32)
  self.assertTrue(all('ip' not in r for r in out));self.assertEqual(out[0]['source'],out[1]['source']);self.assertNotEqual(out[0]['source'],out[2]['source']);self.assertEqual(out[3]['source'],'')
  other=driver.private_sources([dict(ip='8.8.8.8',echo=True)],b'b'*32)
  self.assertNotEqual(other[0]['source'],out[0]['source'])
 def test_driver_has_no_network_class_dependency(self):
  self.assertFalse(hasattr(driver,'intel'));self.assertFalse((HERE/'egress-classify.py').exists())
class ReachabilityHeadContracts(unittest.TestCase):
 def test_head_large_content_length_does_not_read_a_body(self):
  async def run():
   reader=asyncio.StreamReader();reader.feed_data(b'HTTP/1.1 200 OK\r\nContent-Length: 2000000\r\n\r\n');reader.feed_eof()
   status,headers,body=await session.read_response(reader,True,head=True)
   self.assertEqual(status,200);self.assertEqual(body,b'');self.assertEqual(headers['content-length'],'2000000')
  asyncio.run(run())
class AdaptiveContracts(unittest.TestCase):
 def simulate(self,change):
  original=session.measure;calls=[]
  async def fake(domain,variant,wave,timeout,control_ipv4=None,target=None):
   calls.append((domain,wave))
   row=dict(domain=domain,wave=wave,echo=True,tls=True,same_stream=True,family=4,ip=domain,actual_ads=domain in session.DOMAINS)
   change(row);return row
  session.measure=fake
  try:return asyncio.run(session.session(dict(variant='raw',waves=3,sequential=True,adaptive=True))),calls
  finally:session.measure=original
 def test_two_complete_paired_waves_skip_third(self):
  rows,calls=self.simulate(lambda r:None)
  self.assertEqual(len(rows),2*len(session.DEFAULT_TARGETS));self.assertEqual({w for d,w in calls},{1,2})
 def test_third_wave_repeats_required_ads_and_comparison_controls(self):
  def change(r):
   if r['domain']==session.DOMAINS[0] and r['wave']==1:r['echo']=False
  rows,calls=self.simulate(change)
  self.assertEqual({d for d,w in calls if w==3},{t['domain'] for t in session.DEFAULT_TARGETS if t.get('required')})
  self.assertEqual(len(rows),2*len(session.DEFAULT_TARGETS)+sum(t.get("required",False) for t in session.DEFAULT_TARGETS))
 def test_family_change_needs_third_wave(self):
  def change(r):
   if r['domain'] in session.DOMAINS and r['wave']==2:r['family']=6
  rows,calls=self.simulate(change)
  self.assertEqual({d for d,w in calls if w==3},{t['domain'] for t in session.DEFAULT_TARGETS if t.get('required')})
 def test_all_echoes_with_insufficient_same_family_controls_retry(self):
  def change(r):
   if r['domain'] in session.DOMAINS or r['domain']=='www.google.com':r['family']=6
  rows,calls=self.simulate(change)
  self.assertTrue(any(w==3 for d,w in calls))
 def test_shared_same_family_does_not_need_separation_retry(self):
  def change(r):r['ip']='shared-control'
  rows,calls=self.simulate(change)
  self.assertEqual({w for d,w in calls},{1,2})
 def test_redundant_unpaired_control_does_not_repeat_complete_evidence(self):
  def change(r):
   if r['domain']=='api.ipify.org' and r['wave']==2:r['echo']=False
  rows,calls=self.simulate(change)
  self.assertEqual({w for d,w in calls},{1,2})
 def test_missing_indispensable_control_still_retries(self):
  def change(r):
   if r['domain'] in ('api.ipify.org','www.cloudflare.com') and r['wave']==2:r['echo']=False
  rows,calls=self.simulate(change)
  self.assertTrue(any(w==3 for d,w in calls))
 def test_adaptive_deep_pair_stays_serial_and_keeps_wave_ids(self):
  original=session.measure;active=0;peak=0;overlap=False;active_waves={}
  async def fake(domain,variant,wave,timeout,control_ipv4=None,target=None):
   nonlocal active,peak,overlap
   active+=1;peak=max(peak,active);active_waves[wave]=active_waves.get(wave,0)+1;overlap=overlap or len(active_waves)>1
   await asyncio.sleep(.2);active-=1;active_waves[wave]-=1
   if not active_waves[wave]:del active_waves[wave]
   return dict(domain=domain,wave=wave,echo=True,family=4,ip=domain)
  session.measure=fake
  try:
   rows=asyncio.run(session.session(dict(variant='raw',waves=3,sequential=False,adaptive=True,targets=[t for t in session.DEFAULT_TARGETS if t.get('required')])) )
   self.assertFalse(overlap);self.assertLessEqual(peak,12);self.assertEqual({r['wave'] for r in rows},{1,2})
  finally:session.measure=original
 def test_flat_vless_dispatch_preserves_tls_and_local_semantics(self):
  import base64
  for address,security,encryption,want in [('8.8.8.8','none','none','26.5.9'),('example.com','none','none','26.5.9'),('127.0.0.1','none','none','26.11.9'),('8.8.8.8','tls','none','26.11.9'),('8.8.8.8','none','mlkem768x25519plus.native.0rtt.key','26.11.9')]:
   cfg={'outbounds':[{'protocol':'vless','settings':{'address':address,'port':443,'id':'fixture','encryption':encryption},'streamSettings':{'security':security}}]}
   plan={'variants':[{'config':base64.b64encode(json.dumps(cfg).encode()).decode()}]}
   self.assertEqual(driver.core_order(plan)[0][0],'xray-'+want)
class SchedulingContracts(unittest.TestCase):
 def test_core_pin_streams_bytes_and_rejects_changed_binary(self):
  import tempfile,hashlib
  from unittest import mock
  with tempfile.TemporaryDirectory() as directory:
   p=pathlib.Path(directory)/'core';p.write_bytes(b'core-fixture'*1024)
   pin=hashlib.sha256(b'core-fixture'*1024).hexdigest();pool=driver.ResourcePool(None)
   with mock.patch.object(pathlib.Path,'read_bytes',side_effect=AssertionError('whole binary allocation')):
    pool.verify(str(p),pin)
    p.write_bytes(b'changed-core')
    with self.assertRaisesRegex(ValueError,'core_pin_mismatch'):pool.verify(str(p),pin)
   pool.close();pool.geo.close()
 def test_outer_deadline_allows_queued_pairs_and_recovery(self):
  # Ten targets exceed the cap when two waves overlap: 16 seconds for
  # the pair plus 8 for recovery must not hit the old 26-second wrapper.
  self.assertGreaterEqual(driver.scheduling.session_deadline(10,3,8,False),34)
  self.assertGreaterEqual(driver.scheduling.session_deadline(32,3,8,False),82)
  self.assertEqual(driver.scheduling.session_deadline(10,2,3,False),25)
  self.assertEqual(driver.scheduling.session_deadline(10,3,8,True),34)
 def test_transport_driver_controls_ordering_without_shortening_budget(self):
  for network,want in [('tcp',False),('ws',False),('grpc',False),('xhttp',True),('splithttp',True)]:
   cfg={'outbounds':[{'streamSettings':{'network':network}}]}
   self.assertEqual(driver.scheduling.sequential({},cfg),want)
   self.assertTrue(driver.scheduling.sequential({'deep':True,'accepted':True},cfg))
   self.assertTrue(driver.scheduling.sequential({'discovery':True},cfg))
  self.assertEqual(driver.observation_budget({'deep':True,'accepted':True}),(3,8))
class OverlapContracts(unittest.TestCase):
 def test_same_runtime_parallel_join_preserves_both_results(self):
  import threading
  barrier=threading.Barrier(2);finished=threading.Event()
  def health():barrier.wait(timeout=2);finished.set();return {'providers':['diagnostic']}
  def observe():barrier.wait(timeout=2);return ['route-proof']
  rows,h=driver.overlap_assessment(observe,health)
  self.assertEqual(rows,['route-proof']);self.assertEqual(h['providers'],['diagnostic']);self.assertTrue(finished.is_set())
 def test_observer_failure_joins_health_before_next_runtime(self):
  import threading
  started=threading.Event();release=threading.Event();finished=threading.Event()
  def health():started.set();release.wait(timeout=2);finished.set();return {}
  def observe():self.assertTrue(started.wait(timeout=2));release.set();raise ValueError('observation-failed')
  with self.assertRaisesRegex(ValueError,'observation-failed'):driver.overlap_assessment(observe,health)
  self.assertTrue(finished.is_set())
 def test_health_subprocess_failure_does_not_destroy_route_evidence(self):
  class Box:
   def net(self,*a,**kw):raise TimeoutError()
  h=driver.diagnostic_delivery(Box(),'raw',[],3)
  self.assertEqual(h['variant'],'raw');self.assertEqual(h['assets'],[])
if __name__=='__main__':unittest.main()


