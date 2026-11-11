#!/usr/bin/env python3
"""All current unique inputs: exact builds; bounded representatives: live traffic.
No raw bytes, addresses, credentials or subscription tokens are written to receipts.
Uses the existing isolated core sandbox. This audit does not grant publication.
"""
import argparse,re,base64,collections,concurrent.futures,hashlib,importlib.util,json,pathlib,queue,secrets,subprocess,time,urllib.request,urllib.parse
HERE=pathlib.Path(__file__).resolve().parent
def module(name,file):
 s=importlib.util.spec_from_file_location(name,HERE/file);m=importlib.util.module_from_spec(s);s.loader.exec_module(m);return m
lab=module('audit_lab','live-probe.py');driver=module('audit_driver','probe-one.py')
def safe_label(value):
 return value if isinstance(value,str) and re.fullmatch(r'[a-z0-9-]{1,24}',value) else 'unclassified'
def family(config):
 labels=set()
 for ob in config.get('outbounds',[]):
  p=ob.get('protocol','unknown')
  if p in ('freedom','blackhole','dns','loopback'):continue
  stream=ob.get('streamSettings',{});n=stream.get('method',stream.get('network','tcp')) or 'tcp';n={'raw':'tcp','websocket':'ws','splithttp':'xhttp','mkcp':'kcp'}.get(n,n)
  sec=stream.get('security','none') or 'none'
  if n=='tcp' and stream.get('rawSettings',stream.get('tcpSettings',{})).get('header',{}).get('type')=='http':n='tcp-http'
  if p=='wireguard':n,sec='udp','native'
  if p=='hysteria':p='hysteria'+str(ob.get('settings',{}).get('version','?'))
  labels.add('+'.join(map(safe_label,(p,n,sec))))
 return ','.join(sorted(labels))[:500] or 'unknown'
def raw_family(raw,kind):
 try:
  s=raw.decode().strip()
  if s.startswith('{'):return family(json.loads(s))
  u=urllib.parse.urlsplit(s);q=urllib.parse.parse_qs(u.query,keep_blank_values=True)
  n=q.get('type',['tcp'])[0];sec=q.get('security',['none'])[0] or 'none'
  if n in ('raw','tcp') and q.get('headerType')==['http']:n='tcp-http'
  if u.scheme in ('vless','trojan'):return '+'.join(map(safe_label,(u.scheme,n,sec)))
 except (ValueError,UnicodeError,TypeError):pass
 return safe_label(kind)+'+unparsed'
def run(a):
 host=a.host or urllib.parse.urlsplit(json.loads(pathlib.Path(a.runtime_config).read_text())['public_origin']).netloc
 headers={'Host':host,'X-Forwarded-Proto':'https'}
 def fetch(path):
  with urllib.request.urlopen(urllib.request.Request(a.base+path,headers=headers),timeout=20) as response:return response.read(8*1024*1024)
 entries=[];seen=set()
 for page in range(100):
  rows=json.loads(fetch('/api/conversions?page='+str(page)))['entries']
  for e in rows:
   if e['id'] not in seen:entries.append(e);seen.add(e['id'])
  if len(rows)<100:break
 records=[];candidates={};raw_seen=set()
 def prepare(e):
  try:
   raw=fetch('/output/raw/'+e['id']+'?preview=1')
   h=lab.digest(raw);plan=json.loads(subprocess.run([a.compiler,'--template'],input=raw,capture_output=True,check=True,timeout=12).stdout)
   return e,h,plan,raw_family(raw,e['kind'])
  except urllib.error.HTTPError as ex:return e,'',{'code':'snapshot_http_'+str(ex.code)},safe_label(e['kind'])+'+unparsed'
  except Exception as ex:return e,'',{'code':'snapshot_or_prepare_'+type(ex).__name__},safe_label(e['kind'])+'+unparsed'
 with concurrent.futures.ThreadPoolExecutor(max_workers=3) as pool:
  for e,h,plan,f in pool.map(prepare,entries):
   if h and h in raw_seen:continue
   raw_seen.add(h)
   row={'identity':e['id'],'raw_sha256':h,'family':f,'prepare':plan.get('code'),'normal':'not_run','smart':'not_run','connection':'not_run'}
   records.append(row)
   for c in plan.get('candidates',[]):
    data=base64.b64decode(c['config'],validate=True);assert lab.digest(data)==c['sha256']
    f=family(json.loads(data));row['family']=f
    item=candidates.setdefault(c['sha256'],{'candidate':c,'data':data,'family':f,'rows':[],'previous_connected':e['connection']=='passed','result':{}})
    item['rows'].append(row)
 print(json.dumps({'stage':'prepared','inputs':len(records),'unique_candidates':len(candidates)}),flush=True)
 slots=queue.Queue()
 for i in range(3):slots.put(i)
 pins=[('/opt/config-repair/core-26.10.31/xray',driver.CLIENT_CORE_SHA),('/opt/config-repair/core-26.11.9/xray',driver.CORE_SHA)]
 for path,sha in pins:assert lab.digest(pathlib.Path(path).read_bytes())==sha
 def build(item):
  slot=slots.get()
  try:
   result={}
   with lab.Sandbox(pins[0][0],secrets.token_hex(3),8192+slot,prefix='ctf') as box:
    for path,sha in pins:
     error=driver.validate_client(box,item['data'],path);result[path.split('/')[-2]]=error or {'core':'passed'}
    item['build']=result
    item['accepted']=all(v['core']=='passed' for v in result.values())
    for row in item['rows']:
     row['normal']='passed' if item['accepted'] else 'rejected'
     row['build']=result
   return item
  finally:slots.put(slot)
 with concurrent.futures.ThreadPoolExecutor(max_workers=3) as pool:
  for i,item in enumerate(pool.map(build,candidates.values()),1):
   if i%50==0:print(json.dumps({'stage':'build','completed':i}),flush=True)
 groups=collections.defaultdict(list)
 for item in candidates.values():
  if item.get('accepted'):groups[item['family']].append(item)
 selected=[]
 for f,items in sorted(groups.items()):
  selected+=sorted(items,key=lambda x:(not x['previous_connected'],x['candidate']['sha256']))[:a.per_family]
 def connect(item):
  slot=slots.get()
  try:
   with lab.Sandbox(pins[0][0],secrets.token_hex(3),8192+slot,prefix='ctf') as box:
    result=lab.test_candidate(box,item['candidate'],a.timeout);item['result']=result
    for row in item['rows']:row['connection']=result['connection'];row['runtime']=result
   if a.native_helper:
    class NativeBox(lab.Sandbox):
     def core_mounts(self):
      return ['--ro-bind','/lib64/ld-linux-x86-64.so.2','/lib64/ld-linux-x86-64.so.2','--ro-bind','/usr/lib/x86_64-linux-gnu/libc.so.6','/lib/x86_64-linux-gnu/libc.so.6']
     def core_command(self,test):return ['/xray','/config.json','https://www.gstatic.com/generate_204']
    try:
     resolved=lab.endpoints(json.loads(item['data']))
     with NativeBox(a.native_helper,secrets.token_hex(3),8192+slot,prefix='ctf') as box:
      box.policy(resolved);process=box.launch(item['data']);process.wait(timeout=32)
      native=next(json.loads(line) for line in reversed(box.logs().decode().splitlines()) if line.startswith('{') and '"ok"' in line)
      item['native']={k:native[k] for k in ('ok','delay_ms','elapsed_ms') if k in native}
      logs=box.logs().decode()
      if native.get('ok') is not True:item['native']['diagnostic']='native_dns_blocked_by_sandbox' if 'lookup' in logs and 'operation not permitted' in logs else 'native_ping_failed'
    except Exception as ex:item['native']={'ok':False,'diagnostic':type(ex).__name__}
   print(json.dumps({'stage':'connection','family':item['family'],'status':result['connection'],'code':result['code']}),flush=True)
   return item
  except Exception as ex:
   item['result']={'connection':'not_run','code':'audit_'+type(ex).__name__};return item
  finally:slots.put(slot)
 with concurrent.futures.ThreadPoolExecutor(max_workers=3) as pool:list(pool.map(connect,selected))
 smart=[]
 for f,items in sorted(groups.items()):
  representatives=sorted(items,key=lambda x:(x.get('result',{}).get('connection')!='passed',not x['previous_connected'],x['candidate']['sha256']))[:a.per_family]
  plan=json.loads(subprocess.run([a.compiler,'--smart'],input=json.dumps([json.loads(x['data']) for x in representatives]).encode(),capture_output=True,check=True,timeout=15).stdout)
  row={'family':f,'members':len(representatives),'code':plan.get('code'),'build':'not_run','connection':'not_run'}
  if plan.get('candidates'):
   c=plan['candidates'][0];item={'candidate':c,'data':base64.b64decode(c['config']),'family':f,'rows':[]}
   build(item);row['build']='passed' if item['accepted'] else 'rejected';row['checks']=item['build']
   if item['accepted']:
    connect(item);row['connection']=item['result']['connection'];row['runtime']=item['result'];row['native']=item.get('native',{})
  smart.append(row)
  for item in items:
   for record in item['rows']:record['smart']=row['build'] if row['build']!='not_run' else row['code']
 summary=[]
 for f in sorted({x['family'] for x in records}):
  rows=[x for x in records if x['family']==f]
  summary.append({'family':f,'inputs':len(rows),'build_passed':sum(x['normal']=='passed' for x in rows),'prepare_rejected':sum(x['normal']=='not_run' and not str(x['prepare']).startswith('snapshot_') for x in rows),'snapshot_unavailable':sum(str(x['prepare']).startswith('snapshot_') for x in rows),'build_rejected':sum(x['normal']=='rejected' for x in rows),'connection_samples':sum(x['family']==f for x in selected),'connected_samples':sum(x['family']==f and x.get('result',{}).get('connection')=='passed' for x in selected),'native_ping_passed':sum(x['family']==f and x.get('native',{}).get('ok') is True for x in selected)})
 report={'version':'template-audit.v1','tested':int(time.time()),'template':'v7','smart_template':'v11','target':'v2rayNG2.3.15','android_apk':'not_run','carriers':'not_run','inputs':len(records),'unique_candidates':len(candidates),'per_family':a.per_family,'families':summary,'smart':smart,'records':records,'native_samples':[{'family':x['family'],'sha256':x['candidate']['sha256'],**x.get('native',{})} for x in selected]}
 pathlib.Path(a.report).write_text(json.dumps(report,indent=2));print(json.dumps({'stage':'complete','families':summary,'report':a.report}),flush=True)
if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('--base',default='http://127.0.0.1:18191');p.add_argument('--host');p.add_argument('--runtime-config',default='/etc/config-repair/panel.json');p.add_argument('--compiler',default=str(HERE.parent/'config-probe-plan' if (HERE.parent/'config-probe-plan').exists() else HERE.parent/'.local/config-probe-plan'));p.add_argument('--per-family',type=int,choices=(1,2,3),default=3);p.add_argument('--timeout',type=int,default=7);p.add_argument('--native-helper');p.add_argument('--report',required=True);run(p.parse_args())
