#!/usr/bin/env python3
"""Real core-build and SIGKILL queue tests with SYNTHETIC connection receipts.
Real final-profile traffic is tested separately; no network health is claimed here.
"""
import argparse,hashlib,json,os,pathlib,sqlite3,subprocess,tempfile,time,urllib.request
ROOT=pathlib.Path(__file__).resolve().parents[1]
def get(url,prod=False):
 req=urllib.request.Request(url,headers={'Host':'85.192.63.181:18443'} if prod else {})
 with urllib.request.urlopen(req,timeout=30) as f:return f.read(),dict(f.headers)
def normalize(body):
 value=json.loads(body);value['outbounds'].sort(key=lambda x:x['tag'])
 return json.dumps(value,sort_keys=True,separators=(',',':'))
def main():
 parser=argparse.ArgumentParser();parser.add_argument('--splash',action='store_true');parser.add_argument('--ads',action='store_true');parser.add_argument('--live-canary',type=int,default=0);parser.add_argument('--profiles',type=int,choices=(7,55),default=55);args=parser.parse_args()
 if os.geteuid()!=0:raise SystemExit('root required for sandbox integration')
 scope='splash' if args.splash else 'normal';api='/api/smart-splash' if args.splash else '/api/smart';table='splash_smart_' if args.splash else 'smart_';mode='smart-splash' if args.splash else 'smart'
 if args.ads:scope='ads';api='/api/smart-ads';table='ads_smart_';mode='smart-ads'
 receipt={'scope':scope,'android':'not_run','whole_profile_connection':'not_run','worker_sha256':hashlib.sha256((ROOT/'bin/config-probe-worker').read_bytes()).hexdigest()}
 with tempfile.TemporaryDirectory(prefix='smart-worker-',dir=ROOT/'.local') as tmp:
  work=pathlib.Path(tmp);os.chmod(work,0o700)
  fixture=work/'fixture'
  subprocess.run(['go','test','-c','-o',str(fixture),'./internal/panel'],cwd=ROOT,check=True)
  samples={name:[] for name in ['Alpha','Beta','Gamma']}
  for i in range(1,args.profiles+1):
   name='Alpha' if i<max(2,args.profiles-5) else 'Beta' if i<args.profiles-1 else 'Gamma'
   samples[name].append(f'vless://00000000-0000-4000-8000-{i:012d}@192.0.2.{i}:443?security=tls&sni=example.com')
  if args.live_canary:
   state=json.loads(get('http://127.0.0.1:18191/api/conversions',True)[0])
   body=get(state['export_url'],True)[0];picked=[];kinds=set()
   for line in body.splitlines():
    value=json.loads(line);signature=tuple(sorted(set(x.get('protocol','') for x in value.get('outbounds',[]))))
    if signature not in kinds:picked.append(line.decode());kinds.add(signature)
    if len(picked)>=args.live_canary:break
   samples['Canary']=picked;receipt['live_samples']=len(picked)
  (work/'samples.json').write_text(json.dumps(samples))
  for name in ['probe-one.py','live-probe.py','config-preflight.py','app-import-contract.py','transport-checks.py','http3-check.py']:(work/name).write_text((ROOT/'scripts'/name).read_text())
  (work/'smart-probe-one.py').write_text((ROOT/'scripts'/('ads-probe-one.py' if args.ads else 'splash-probe-one.py' if args.splash else 'smart-probe-one.py')).read_text().replace("PREFIX='csm'","PREFIX='cst'").replace("PREFIX='csp'","PREFIX='cpt'").replace("PREFIX='csa'","PREFIX='cat'"))
  if args.live_canary:raise RuntimeError('Use the separate exact-profile live-canary driver; synthetic queue receipts cannot certify live samples')
  helper=work/'smart-probe-one.py'
  text=helper.read_text();text=text.replace("if __name__=='__main__':driver.main()",(ROOT/'scripts/testdata/synthetic-connection-adapter.py').read_text()+"\nif __name__=='__main__':driver.main()")
  helper.write_text(text)
  panel=worker=None
  with (work/'panel.log').open('w') as pl,(work/'worker.log').open('w') as wl:
   try:
    panel=subprocess.Popen([str(fixture),'-test.run=^TestSmartWorkerFixture$','-test.v'],env=dict(os.environ,CONFIG_SMART_FIXTURE_DIR=str(work),CONFIG_SMART_FIXTURE_SCOPE=scope),stdout=pl,stderr=pl)
    fixture_started=time.monotonic();last_ready_error=""
    for _ in range(600):
     if panel.poll() is not None:raise RuntimeError('fixture exited; '+(work/'panel.log').read_text()[-2000:])
     try:
      ready=json.loads((work/'public.json').read_text());assert ready['scope']==scope
      origin=ready['origin'];json.loads(get(origin+api)[0]);break
     except (OSError,json.JSONDecodeError) as error:last_ready_error=type(error).__name__;time.sleep(.1)
    else:raise RuntimeError("fixture unavailable: "+last_ready_error+"; "+(work/"panel.log").read_text()[-2000:])
    receipt["fixture_ready_seconds"]=round(time.monotonic()-fixture_started,2)
    def launch():return subprocess.Popen([str(ROOT/'bin/config-probe-worker'),'-mode',mode,'-socket',str(work/'data/probe.sock'),'-helper',str(work/'smart-probe-one.py'),'-core','/opt/config-repair/core-26.11.9/xray','-lock-file',str(work/'smart.lock')],stdout=wl,stderr=wl)
    worker=launch();killed=False;start=time.monotonic();last=-1;max_active=0
    while time.monotonic()-start<900:
     state=json.loads(get(origin+api)[0]);active=state['worker']['active'];max_active=max(max_active,active)
     running=sum(r['state']=='running' for r in state['entries'])
     with sqlite3.connect(str(work/'data/panel.db')) as claims:live_claims=claims.execute("SELECT count(*) FROM "+table+"heads WHERE token!='' AND lease_until>?",(int(time.time()*1000),)).fetchone()[0]
     max_active=max(max_active,live_claims)
     if not killed and running:worker.kill();worker.wait(timeout=10);worker=launch();killed=True
     if state['stats']['total']>=args.profiles and state['stats']['ready']==state['stats']['total']:break
     if worker.poll() is not None:raise RuntimeError('worker exited; '+(work/'worker.log').read_text()[-1000:])
     now=int(time.monotonic()-start)//20
     if now!=last:last=now;print(json.dumps({'ready':state['stats']['ready'],'total':state['stats']['total'],'active':active,'blocked':state['stats']['blocked']}),flush=True)
     time.sleep(.25)
    else:raise RuntimeError('smart queue timeout: '+json.dumps(state['stats'])+' codes '+str({r['code'] for r in state['entries']}))
    assert killed and max_active<=2
    values=json.loads(get(state['export_url'])[0]);lines=[json.dumps(v,separators=(',',':')).encode() for v in values];assert len(lines)==state['stats']['ready']==len(set(lines))
    for line in lines:
     j=json.loads(line);assert j['remarks']==('@shenvpn-smart-ads' if args.ads else '@shenvpn-smart-parallel-v6b')
     assert list(j)==['remarks','log','dns','inbounds','outbounds','routing','burstObservatory','policy','stats','metrics']
     assert j['log']=={'loglevel':'none','access':'none','error':'none','dnsLog':False}
     assert len(j['dns']['servers'])==4 and len(j['routing']['balancers'])==2 and len(j['routing']['rules'])==(6 if args.ads else 4)
     assert j['outbounds'][0]['tag'] in j['routing']['balancers'][1]['selector'] and j['routing']['balancers'][0]['fallbackTag']=='block'
     assert len([x for x in j['outbounds'] if x['tag'].startswith('proxy-')])>=(50 if args.ads else min(50,args.profiles))
     assert j['routing']['balancers'][-1]['strategy']=={'type':'leastLoad','settings':{'expected':3,'maxRTT':'2s','tolerance':.15}}
    normalized={normalize(line) for line in lines}
    turns=[]
    for i in range(len(lines)+1):
     body,headers=get(state['single_export_url']);assert normalize(body) in normalized
     assert hashlib.sha256(body.strip()).hexdigest()=={k.lower():v for k,v in headers.items()}['x-content-sha256']
     assert headers['Content-Disposition']=='inline'
     turns.append((int(headers['X-Export-Cycle']),headers['X-Smart-First-Id']))
    assert all(c==1 for c,_ in turns[:-1]) and len(set(h for _,h in turns[:-1]))==len(lines) and turns[-1][0]==2
    with sqlite3.connect(str(work/'data/panel.db')) as db:
     covered=db.execute('SELECT head,count(DISTINCT source) FROM '+table+'members GROUP BY head').fetchall()
     assert all(n==(3 if args.ads else len(samples)) for _,n in covered)
     if args.ads:
      assert db.execute("SELECT count(*) FROM ads_smart_members m JOIN sources s ON s.id=m.source WHERE (m.position=0 AND s.splash!=1) OR(m.position>0 AND s.splash!=0)").fetchone()[0]==0
     assert db.execute('SELECT status FROM conversion_worker').fetchone()[0]=='offline'
    if args.splash:
     other=json.loads(get(origin+'/api/smart')[0]);assert other['stats']['ready']==0 and other['settings']['count']==10
     assert all(b'192.0.2.200' not in line for line in lines)
    assert hashlib.sha256((ROOT/'bin/config-probe-worker').read_bytes()).hexdigest()==receipt['worker_sha256']
    receipt.update(passed=True,profiles=len(lines),members_per_profile=51 if args.ads else min(50,args.profiles),sources=len(samples),max_active=max_active,sigkill_recovered=True,unique_head_round=True,inline=True,core='xray-26.11.9',connection_receipts='synthetic_state_machine_only')
   finally:
    (ROOT/(".local/"+scope+"-fixture-diagnostic.log")).write_text((work/"panel.log").read_text()[-8000:])
    for process in [worker,panel]:
     if process and process.poll() is None:
      process.terminate()
      try:process.wait(timeout=35)
      except subprocess.TimeoutExpired:process.kill();process.wait()
    subprocess.run(['/usr/bin/python3',str(work/'smart-probe-one.py'),'--cleanup'],check=True,timeout=30)
    (ROOT/('.local/ads-worker-diagnostic.log' if args.ads else '.local/splash-worker-diagnostic.log' if args.splash else '.local/smart-worker-diagnostic.log')).write_text((work/'worker.log').read_text()[-8000:])
 (ROOT/('.local/ads-worker-test.json' if args.ads else '.local/splash-worker-test.json' if args.splash else '.local/smart-worker-test.json')).write_text(json.dumps(receipt,indent=2));print(json.dumps(receipt),flush=True)
if __name__=='__main__':main()
