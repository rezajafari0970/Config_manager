#!/usr/bin/env python3
"""Offline contracts for the real-device agent; no health pass from fixtures."""
import copy,json,os,pathlib,importlib.util,types,subprocess,sys,tempfile,unittest
from unittest.mock import patch
from benchmarks.core import Session,encoded
from benchmarks.profiles import apply
from benchmarks.http import request,transfers
from benchmarks.quality import BENCHMARK_VERSION,PAYLOAD_VERSION,DOWNLOAD_BYTES,UPLOAD_BYTES,complete
class Contracts(unittest.TestCase):
 def setUp(self):
  self.tmp=tempfile.TemporaryDirectory();self.root=pathlib.Path(self.tmp.name)
  self.core=self.root/'fixture-core';self.core.write_text('#!'+sys.executable+'\nimport json,socket,sys,time\nif "-test" in sys.argv:sys.exit(0)\nc=json.load(open(sys.argv[-1]));s=socket.socket();s.bind(("127.0.0.1",c["inbounds"][0]["port"]));s.listen();time.sleep(30)\n');self.core.chmod(0o700)
  self.config={'inbounds':[{'listen':'127.0.0.1','port':10808,'protocol':'socks'},{'listen':'127.0.0.1','port':39081,'protocol':'dokodemo-door'}],'outbounds':[{'tag':'proxy','settings':{'address':'127.0.0.1','port':39081,'id':'preserve'},'streamSettings':{'security':'tls','tlsSettings':{'allowInsecure':False}}}],'metrics':{'listen':'127.0.0.1:11111'},'routing':{'balancers':[{'strategy':{'type':'leastLoad','settings':{'expected':3}},'fallbackTag':'block'}]},'burstObservatory':{'pingConfig':{'interval':'15s','sampling':3,'timeout':'5s'}}}
 def tearDown(self):self.tmp.cleanup()
 def test_owned_session_remaps_graph_and_cleans_after_exception(self):
  before=encoded(self.config);session=Session(self.core,self.config,self.root/'case')
  with self.assertRaisesRegex(RuntimeError,'fixture error'):
   with session:
    runtime=json.loads(session.runtime_bytes);ports=[i['port'] for i in runtime['inbounds']]
    self.assertEqual(len(set(ports+[session.metrics_port])),3)
    self.assertEqual(runtime['outbounds'][0]['settings']['port'],ports[1]);self.assertEqual(runtime['outbounds'][0]['settings']['id'],'preserve')
    self.assertEqual(runtime['routing'],self.config['routing']);self.assertEqual(runtime['outbounds'][0]['streamSettings'],self.config['outbounds'][0]['streamSettings'])
    raise RuntimeError('fixture error')
  self.assertIsNotNone(session.process.poll());self.assertEqual(before,encoded(self.config));self.assertEqual((session.work/'original.json').read_bytes(),before)
 def test_reject_external_listener_before_runtime(self):
  self.config['inbounds'][1]['listen']='0.0.0.0';s=Session(self.core,self.config,self.root/'case')
  with self.assertRaisesRegex(ValueError,'unsupported_benchmark_inbounds'):s.__enter__()
  self.assertIsNone(s.process)
 def test_profile_changes_only_observatory_and_preserves_fail_closed(self):
  baseline=copy.deepcopy(self.config);candidate=apply(baseline,'responsive-v1')
  self.assertEqual(candidate.pop('burstObservatory'),{'pingConfig':{'interval':'10s','sampling':2,'timeout':'3s'}})
  baseline.pop('burstObservatory');self.assertEqual(candidate,baseline)
  with self.assertRaises(ValueError):apply(self.config,'unknown')
  self.config['routing']['balancers'][0]['fallbackTag']='direct'
  with self.assertRaises(ValueError):apply(self.config,'responsive-v1')
 def test_stable_profile_keeps_all_members_and_only_changes_expected(self):
  original=encoded(self.config);c=apply(self.config,'stable-path-v1');self.assertEqual(c['routing']['balancers'][0]['strategy']['settings']['expected'],1)
  self.assertEqual(c['outbounds'],self.config['outbounds']);self.assertEqual(c['burstObservatory'],self.config['burstObservatory']);self.assertEqual(original,encoded(self.config))
 def test_https_status_does_not_override_invalid_certificate(self):
  fake=subprocess.CompletedProcess([],60,json.dumps({'ssl_verify_result':10,'http_code':200}).encode(),b'')
  with patch('benchmarks.http.subprocess.run',return_value=fake) as run:
   r=request(12345,'https://example.com')
   args=run.call_args.args[0];self.assertIn('socks5h://127.0.0.1:12345',args);self.assertNotIn('--insecure',args);self.assertEqual(r['status'],'failed');self.assertEqual(r['transport_tls'],'unverified')
 def test_resume_preserves_evidence_age_and_rejects_changed_network(self):
  spec=importlib.util.spec_from_file_location('android_bench',pathlib.Path(__file__).with_name('android-bench.py'));agent=importlib.util.module_from_spec(spec);spec.loader.exec_module(agent)
  from benchmarks.core import digest
  key=digest(encoded(self.config));work=self.root/('case-000-'+key[:12]);work.mkdir()
  prior={'benchmark_version':BENCHMARK_VERSION,'config_sha256':key,'core_sha256':'fixture','network_label':'network-a','benchmark_stage':'full','observation_seconds':55,'finished':123,'build':'passed'}
  (work/'result.json').write_text(json.dumps(prior))
  args=types.SimpleNamespace(output=str(self.root),core_sha='fixture',network_label='network-a',resume=True,stage='full',observe_seconds=55)
  with patch('benchmarks.core.validate',side_effect=AssertionError('must not fabricate fresh core evidence')):
   got=agent.run_one(args,(0,{'config':self.config}))
  self.assertTrue(got['historical_reused']);self.assertEqual(got['finished'],123)
  args.network_label='network-b'
  with self.assertRaisesRegex(ValueError,'resume_context_mismatch'):agent.run_one(args,(0,{'config':self.config}))
  self.assertEqual(json.loads((work/'result.json').read_text()),prior)
  args.network_label='network-a';old=dict(prior,benchmark_version='android-benchmark.v1');(work/'result.json').write_text(json.dumps(old))
  with self.assertRaisesRegex(ValueError,'resume_context_mismatch'):agent.run_one(args,(0,{'config':self.config}))
  self.assertEqual(json.loads((work/'result.json').read_text()),old)
 def test_transfer_requires_exact_payload_size(self):
  fake=subprocess.CompletedProcess([],0,json.dumps({'ssl_verify_result':0,'http_code':200,'size_download':1}).encode(),b'')
  with patch('benchmarks.http.subprocess.run',return_value=fake):self.assertEqual(request(12345,'https://example.com',download_bytes=100)['status'],'failed')
class DeviceQuality(unittest.TestCase):
 def setUp(self):
  spec=importlib.util.spec_from_file_location('device_quality',pathlib.Path(__file__).with_name('build-device-smart.py'));self.gate=importlib.util.module_from_spec(spec);spec.loader.exec_module(self.gate)
  self.config={'outbounds':[{'protocol':'vless','settings':{'id':'unchanged','address':'example.test','encryption':'none'}}]}
  from benchmarks.core import digest
  self.key=digest(encoded(self.config));self.corpus={'rows':[{'family':'vless','config':self.config}]}
  ok={'status':'passed','transport_tls':'verified','curl_exit':0,'http_status':200}
  self.row={'benchmark_version':BENCHMARK_VERSION,'benchmark_stage':'full','config_sha256':self.key,'core_sha256':'target','network_label':'mci','finished':990,'started':980,'build':'passed','startup':'passed','https':[dict(ok,name=n) for n in self.gate.SITES],'transport':{'checks':[{'name':n,'status':'passed'} for n in self.gate.TRANSPORT]},'http3':{'status':'passed','tls':'verified'},'transfers':{'version':PAYLOAD_VERSION,'both_passed':True,'download':dict(ok,download_bytes=DOWNLOAD_BYTES),'upload':dict(ok,upload_bytes=UPLOAD_BYTES),'post_load':dict(ok,http_status=204)}}
 def select(self,rows):return self.gate.select([self.corpus],[{'results':rows}],'target','mci',1000,60)
 def test_qualified_complete_graph_is_preserved(self):
  original=encoded(self.corpus);accepted,reasons=self.select([self.row]);self.assertEqual(accepted,[self.config]);self.assertEqual(reasons[0]['code'],'qualified_finite_observation');self.assertEqual(encoded(self.corpus),original)
 def test_latest_failure_overrides_old_success_and_other_network_cannot_refresh(self):
  bad=copy.deepcopy(self.row);bad.update(finished=995);bad['transfers']['download'].update(status='failed',curl_exit=28,http_status=200,download_bytes=1200)
  accepted,reasons=self.select([self.row,bad]);self.assertEqual(accepted,[]);self.assertEqual(reasons[0]['code'],'bidirectional_payload_incomplete')
  other=copy.deepcopy(self.row);other.update(finished=999,network_label='wifi')
  self.assertEqual(self.select([self.row,bad,other])[0],[])
 def test_unverified_network_label_cannot_qualify(self):
  row=copy.deepcopy(self.row);row['network_label']='unverified'
  self.assertEqual(self.gate.qualify(row,'target','unverified',1000,60),'network_scope_unverified')
 def test_stale_partial_and_wrong_core_never_qualify(self):
  for patchrow in [{'finished':900},{'finished':None},{'core_sha256':'old'},{'network_label':'wifi'},{'runtime_failure':'core_exited'},{'http3':{'status':'failed'}}]:
   with self.subTest(patchrow=patchrow):
    row=copy.deepcopy(self.row);row.update(patchrow);self.assertNotEqual(self.gate.qualify(row,'target','mci',1000,60),'qualified_finite_observation')
  self.row['https'][0].update(status='failed',curl_exit=28,http_status=200);self.assertEqual(self.select([self.row])[0],[])

 def test_old_small_transfer_and_incomplete_postload_do_not_qualify(self):
  for kind in ('old_version','small_download','small_upload','post_load','missing_contract'):
   with self.subTest(kind=kind):
    row=copy.deepcopy(self.row)
    if kind=='old_version':row['benchmark_version']='android-benchmark.v1'
    if kind=='small_download':row['transfers']['download']['download_bytes']=524288
    if kind=='small_upload':row['transfers']['upload']['upload_bytes']=131072
    if kind=='post_load':row['transfers']['post_load'].update(status='failed',curl_exit=28)
    if kind=='missing_contract':row['transfers'].pop('version')
    self.assertEqual(self.select([row])[0],[])
 def test_latest_interrupted_attempt_invalidates_previous_pass_without_exception(self):
  newer=copy.deepcopy(self.row);newer.update(started=999,finished=None)
  self.assertEqual(self.select([self.row,newer])[0],[])
  for value in (True,float('nan'),float('inf'),10**1000):
   bad=copy.deepcopy(self.row);bad['finished']=value
   self.assertNotEqual(self.gate.qualify(bad,'target','mci',1000,60),'qualified_finite_observation')

class PayloadProof(unittest.TestCase):
 def test_transfer_runs_complete_sizes_post_load_and_erases_synthetic_payload(self):
  seen=[]
  def fake(port,url,**kw):
   if kw.get('payload') is not None:
    body=pathlib.Path(kw['payload']);seen.append(body.stat().st_size)
   n=204 if 'generate_204' in url else 200
   return {'status':'passed','transport_tls':'verified','curl_exit':0,'http_status':n,'download_bytes':kw.get('download_bytes',0),'upload_bytes':UPLOAD_BYTES if kw.get('payload') else 0}
  with tempfile.TemporaryDirectory() as root,patch('benchmarks.http.request',side_effect=fake) as call:
   result=transfers(12345,root,9)
   self.assertTrue(complete(result));self.assertEqual(seen,[UPLOAD_BYTES]);self.assertEqual(len(call.call_args_list),3)
   self.assertEqual(call.call_args_list[0].kwargs['download_bytes'],DOWNLOAD_BYTES)
   self.assertFalse((pathlib.Path(root)/'upload.bin').exists())
 def test_timeout_with_http200_and_full_local_write_is_not_upload_proof(self):
  ok={'status':'passed','transport_tls':'verified','curl_exit':0,'http_status':200}
  result={'version':PAYLOAD_VERSION,'both_passed':True,'complete':True,'download':dict(ok,download_bytes=DOWNLOAD_BYTES),'upload':dict(ok,upload_bytes=UPLOAD_BYTES,curl_exit=28),'post_load':dict(ok,http_status=204)}
  self.assertFalse(complete(result))

 def test_failed_local_write_does_not_become_reported_upload_speed(self):
  spec=importlib.util.spec_from_file_location('comparison',pathlib.Path(__file__).with_name('compare-benchmarks.py'));m=importlib.util.module_from_spec(spec);spec.loader.exec_module(m)
  report={'network_label':'mci','core_sha256':'test','results':[{'label':'case','build':'passed','startup':'passed','connection':'partial','transfers':{'upload':{'status':'failed','upload_mbps':99},'download':{'status':'passed','download_mbps':3}}}]}
  row=m.summarize(report)['rows'][0]
  self.assertIsNone(row['upload_mbps']);self.assertEqual(row['download_mbps'],3)

 def test_simple_exports_are_exact_and_existing_snapshot_is_never_reused(self):
  import time,base64
  from benchmarks.core import digest
  spec=importlib.util.spec_from_file_location('quality_export',pathlib.Path(__file__).with_name('build-device-smart.py'));gate=importlib.util.module_from_spec(spec);spec.loader.exec_module(gate)
  with tempfile.TemporaryDirectory() as root:
   r=pathlib.Path(root);conf={'outbounds':[{'protocol':'vless','settings':{'id':'fixture'}}]};body=encoded(conf);key=digest(body);now=int(time.time())
   ok={'status':'passed','transport_tls':'verified','curl_exit':0,'http_status':200}
   row={'config_sha256':key,'core_sha256':'target','network_label':'fixture-net','benchmark_version':BENCHMARK_VERSION,'benchmark_stage':'full','started':now-1,'finished':now,'build':'passed','startup':'passed','https':[dict(ok,name=n) for n in gate.SITES],'transport':{'checks':[{'name':n,'status':'passed'} for n in gate.TRANSPORT]},'http3':{'status':'passed','tls':'verified'},'transfers':{'version':PAYLOAD_VERSION,'download':dict(ok,download_bytes=DOWNLOAD_BYTES),'upload':dict(ok,upload_bytes=UPLOAD_BYTES),'post_load':dict(ok,http_status=204)}}
   (r/'corpus').write_text(json.dumps({'rows':[{'family':'vless','config':conf}]}));(r/'report').write_text(json.dumps({'results':[row]}))
   plan={'code':'prepared','candidates':[{'config':base64.b64encode(body).decode(),'sha256':key}]}
   compiler=r/'compiler';compiler.write_text('#!'+sys.executable+'\nimport sys,json\nsys.stdin.read()\nprint('+repr(json.dumps(plan))+')\n');compiler.chmod(0o700)
   args=[sys.executable,str(pathlib.Path(__file__).with_name('build-device-smart.py')),'--corpus',str(r/'corpus'),'--report',str(r/'report'),'--core-sha','target','--network-label','fixture-net','--compiler',str(compiler),'--output-dir',str(r/'output')]
   first=subprocess.run(args,capture_output=True);self.assertEqual(first.returncode,0,first.stderr)
   self.assertEqual((r/'output/simple'/ (key+'.json')).read_bytes(),body)
   self.assertEqual(json.loads((r/'output/qualified-simple-array.json').read_bytes()),[conf])
   (r/'report').write_text(json.dumps({'results':[]}))
   again=subprocess.run(args,capture_output=True);self.assertNotEqual(again.returncode,0)
   self.assertEqual((r/'output/simple'/ (key+'.json')).read_bytes(),body)

if __name__=='__main__':unittest.main()
