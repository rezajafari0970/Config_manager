#!/usr/bin/env python3
"""Private immutable-snapshot repair audit; never grants publication.
All distinct final candidates build on pinned cores. Network representatives
are bounded per protocol/transport/security family; exact bytes are retained.
"""
import argparse,base64,collections,concurrent.futures,hashlib,importlib.util,json,os,pathlib,queue,secrets,subprocess,time
HERE=pathlib.Path(__file__).resolve().parent
def module(name,file):
 s=importlib.util.spec_from_file_location(name,HERE/file);m=importlib.util.module_from_spec(s);s.loader.exec_module(m);return m
audit=module('repair_family','template-audit.py');lab=audit.lab;driver=audit.driver
CORES=[
 ('26.11.9','e87bce586acdb9136aa5b1f1d9c8db9f1ea1d145b6d86368c7f6f662f9f6a66d'),
 ('26.10.31','cc4b84d3069569f297a61f04b914f2bb37e9d202fd7b84cb50def1fc30fd40e1'),
 ('26.7.28','64d46afb80adea1bf97a0d467e83f4a9ac1ebd0995891e84bca3f1a1d1affb1d'),
 ('26.5.9','b2c525e082cf2fef460499c88838d355f0b9bfb5a00bdb3eaa99b6af63825006')]
def plan(compiler,raw,mode='--template'):
 return json.loads(subprocess.run([compiler,mode],input=raw,capture_output=True,timeout=15,check=True).stdout)
def validate(box,data,version):
 box.core=pathlib.Path('/opt/config-repair/core-'+version+'/xray')
 box.policy({});p=box.launch(data,True)
 try:
  p.wait(timeout=15);log=box.logs()
  if p.returncode==0:return {'core':'passed'}
  if p.returncode<0 or not log.startswith(b'Xray '):return {'core':'blocked','code':'validation_infrastructure'}
  code=lab.classify(log,'core_rejected')
  return {'core':'blocked' if code=='missing_runtime_asset' else 'failed','code':code,'diagnostic':log.decode(errors='replace')[-6000:]}
 except subprocess.TimeoutExpired:return {'core':'blocked','code':'validation_timeout'}
 finally:box.stop()
def run(a):
 os.umask(0o077)
 snapshot=json.loads(pathlib.Path(a.snapshot).read_text());rows=snapshot['rows'];known={x['hash'] for x in rows}
 previous=json.loads(pathlib.Path(a.resume).read_text()) if a.resume else {}
 if previous:assert previous.get('stage')=='complete' and previous.get('cores')==dict(CORES) and 0<=time.time()-previous['started']<7200,'resume evidence expired or incomplete'
 cached=previous.get('candidates',{}).copy()
 for row in previous.get('smart',[]):
  if row.get('sha256'):cached[row['sha256']]=row
 prior=json.loads(pathlib.Path(a.prior).read_text()) if a.prior else {'precore':[]}
 for x in prior['precore']:
  if x['hash'] not in known:rows.append({'hash':x['hash'],'raw':x['raw_base64'],'kind':x.get('classification',{}).get('protocol','unknown'),'sources':','.join(v['source'] for v in x['identities']),'identities':','.join(v['id'] for v in x['identities']),'historical':True})
 for v,sha in CORES:assert lab.digest(pathlib.Path('/opt/config-repair/core-'+v+'/xray').read_bytes())==sha
 report={'audit_version':'repair-audit.v1','snapshot':snapshot['snapshot'],'resumed_evidence_started':previous.get('started'),'sources':snapshot['sources'],'started':int(time.time()),'target':'26.11.9','cores':dict(CORES),'android_apk':'not_run','carriers':'not_run','compiler_sha256':lab.digest(pathlib.Path(a.compiler).read_bytes()),'old_compiler_sha256':lab.digest(pathlib.Path(a.old_compiler).read_bytes()),'per_family':a.per_family,'records':[],'candidates':{},'smart':[]}
 def save(stage):
  report['stage']=stage;tmp=pathlib.Path(a.report+'.tmp');tmp.write_text(json.dumps(report,ensure_ascii=False,indent=2));tmp.replace(a.report)
 candidates={}
 def prepare(r):
  if r.get('error'):return {'hash':r['hash'],'stage':'snapshot_unavailable','code':r['error']}
  raw=base64.b64decode(r['raw'],validate=True);assert lab.digest(raw)==r['hash']
  before=plan(a.old_compiler,raw);after=plan(a.compiler,raw)
  row={k:r.get(k,'') for k in ['hash','kind','sources','identities','historical']};row.update(before=before['code'],after=after['code'],family=audit.raw_family(raw,r['kind']),candidates=[],stage='preparation_blocked')
  for c in after.get('candidates',[]):
   data=base64.b64decode(c['config'],validate=True);assert lab.digest(data)==c['sha256']
   row['family']=audit.family(json.loads(data));row['candidates'].append(c['sha256'])
   row['stage']='prepared'
  return row,after.get('candidates',[])
 with concurrent.futures.ThreadPoolExecutor(max_workers=a.workers) as pool:
  for result in pool.map(prepare,rows):
   if isinstance(result,dict):report['records'].append(result);continue
   row,cs=result;report['records'].append(row)
   for c in cs:candidates.setdefault(c['sha256'],{'candidate':c,'family':row['family'],'rows':[]})['rows'].append(row)
 save('prepared');print(json.dumps({'stage':'prepared','inputs':len(rows),'candidates':len(candidates),'codes':dict(collections.Counter(r.get('after',r.get('code')) for r in report['records']))}),flush=True)
 slots=queue.Queue()
 for i in range(a.workers):slots.put(i)
 def build(item):
  slot=slots.get()
  try:
   data=base64.b64decode(item['candidate']['config'],validate=True);results={}
   old=cached.get(item['candidate']['sha256'],{})
   if old.get('build') and set(old['build'])==set(dict(CORES)):
    item['build']=old['build'];item['accepted']=all(old['build'][v]['core']=='passed' for v in ['26.11.9','26.10.31']);item['reused_build']=True;return item
   with lab.Sandbox('/opt/config-repair/core-26.11.9/xray',secrets.token_hex(3),8192+slot,prefix='ctf') as box:
    for v,_ in CORES:results[v]=validate(box,data,v)
   item['build']=results;item['accepted']=all(results[v]['core']=='passed' for v in ['26.11.9','26.10.31'])
   return item
  finally:slots.put(slot)
 with concurrent.futures.ThreadPoolExecutor(max_workers=a.workers) as pool:
  for i,item in enumerate(pool.map(build,candidates.values()),1):
   c=item['candidate'];report['candidates'][c['sha256']]={'family':item['family'],'rule':c['rule'],'build':item['build'],'reused_build':item.get('reused_build',False)}
   if i%50==0:save('build');print(json.dumps({'stage':'build','completed':i,'total':len(candidates)}),flush=True)
 for row in report['records']:
  if not row.get('candidates'):continue
  statuses=[candidates[h]['build']['26.11.9']['core'] for h in row['candidates']]
  row['stage']='core_passed' if 'passed' in statuses else 'core_rejected' if all(s=='failed' for s in statuses) else 'validation_incomplete'
 save('build_complete')
 groups=collections.defaultdict(list)
 for item in candidates.values():
  if item['accepted']:groups[item['family']].append(item)
 prior_hashes={x['hash'] for x in prior['precore']}
 selected=[]
 for family,items in sorted(groups.items()):
  # Prior parser failures first; diversify transport/security option shapes.
  def priority(item):
   c=json.loads(base64.b64decode(item['candidate']['config']));shape=json.dumps([{k:v for k,v in ob.items() if k in ['mux','streamSettings']} for ob in c.get('outbounds',[])],sort_keys=True)
   return (not any(r['hash'] in prior_hashes for r in item['rows']),hashlib.sha256(shape.encode()).hexdigest())
  selected+=sorted(items,key=priority)[:a.per_family]
 def connect(item):
  slot=slots.get()
  try:
   results={}
   old=cached.get(item['candidate']['sha256'],{})
   if set(old.get('runtime',{}))=={'26.11.9','26.10.31'}:
    item['runtime']=old['runtime'];item['reused_runtime']=True;return item
   for v in ['26.11.9','26.10.31']:
    with lab.Sandbox('/opt/config-repair/core-'+v+'/xray',secrets.token_hex(3),8192+slot,prefix='ctf') as box:
     results[v]=lab.test_candidate(box,item['candidate'],a.timeout)
   item['runtime']=results
   print(json.dumps({'stage':'connection','family':item['family'],'results':{v:r['code'] for v,r in results.items()}}),flush=True)
   return item
  except Exception as e:
   item['runtime']={'audit_error':type(e).__name__};return item
  finally:slots.put(slot)
 with concurrent.futures.ThreadPoolExecutor(max_workers=a.workers) as pool:
  for item in pool.map(connect,selected):
   report['candidates'][item['candidate']['sha256']]['runtime']=item['runtime'];report['candidates'][item['candidate']['sha256']]['reused_runtime']=item.get('reused_runtime',False);save('connection')
 for family,items in sorted(groups.items()):
  reps=sorted(items,key=lambda x:(x.get('runtime',{}).get('26.11.9',{}).get('connection')!='passed',x['candidate']['sha256']))[:a.per_family]
  p=plan(a.compiler,json.dumps([json.loads(base64.b64decode(x['candidate']['config'])) for x in reps]).encode(),'--smart')
  row={'family':family,'members':len(reps),'prepare':p['code']}
  if p.get('candidates'):
   item={'candidate':p['candidates'][0],'family':family,'rows':[]};build(item);row['build']=item['build'];row['sha256']=item['candidate']['sha256']
   if item['accepted']:connect(item);row['runtime']=item['runtime']
  report['smart'].append(row);save('smart')
 summary={'inputs':len(report['records']),'current_inputs':sum(not x.get('historical') for x in rows),'unique_candidates':len(candidates),'stages':dict(collections.Counter(r['stage'] for r in report['records'])),'repaired_prior':sum(r.get('hash') in prior_hashes and r['stage']=='core_passed' for r in report['records']),'network_samples':len(selected)}
 report['summary']=summary;report['finished']=int(time.time());save('complete');print(json.dumps({'stage':'complete',**summary}),flush=True)
if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('--snapshot',required=True);p.add_argument('--resume');p.add_argument('--prior');p.add_argument('--compiler',required=True);p.add_argument('--old-compiler',required=True);p.add_argument('--workers',type=int,choices=(1,2,3),default=2);p.add_argument('--per-family',type=int,choices=(1,2,3),default=2);p.add_argument('--timeout',type=int,default=6);p.add_argument('--report',required=True);run(p.parse_args())
