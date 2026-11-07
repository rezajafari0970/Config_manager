#!/usr/bin/env python3
"""Real pinned-core smart worker, isolated queue, restart, sources and inline exports."""
import argparse,hashlib,json,os,pathlib,sqlite3,subprocess,tempfile,time,urllib.request
ROOT=pathlib.Path(__file__).resolve().parents[1]
ORIGIN='http://127.0.0.1:18195'
def get(url,prod=False):
 req=urllib.request.Request(url,headers={'Host':'85.192.63.181:18443'} if prod else {})
 with urllib.request.urlopen(req,timeout=30) as f:return f.read(),dict(f.headers)
def normalize(body):
 value=json.loads(body);value['outbounds'].sort(key=lambda x:x['tag'])
 return json.dumps(value,sort_keys=True,separators=(',',':'))
def main():
 parser=argparse.ArgumentParser();parser.add_argument('--live-canary',type=int,default=0);args=parser.parse_args()
 if os.geteuid()!=0:raise SystemExit('root required for sandbox integration')
 subprocess.run(['go','test','-c','-o',str(ROOT/'.local/smart-worker-fixture'),'./internal/panel'],cwd=ROOT,check=True)
 receipt={'android':'not_run','whole_profile_connection':'not_run'}
 with tempfile.TemporaryDirectory(prefix='smart-worker-',dir=ROOT/'.local') as tmp:
  work=pathlib.Path(tmp);os.chmod(work,0o700)
  samples={name:[] for name in ['Alpha','Beta','Gamma']}
  for i in range(1,56):
   name='Alpha' if i<50 else 'Beta' if i<54 else 'Gamma'
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
  for name in ['probe-one.py','live-probe.py']:(work/name).write_text((ROOT/'scripts'/name).read_text())
  (work/'smart-probe-one.py').write_text((ROOT/'scripts/smart-probe-one.py').read_text().replace("PREFIX='csm'","PREFIX='cst'"))
  panel=worker=None
  with (work/'panel.log').open('w') as pl,(work/'worker.log').open('w') as wl:
   try:
    panel=subprocess.Popen([str(ROOT/'.local/smart-worker-fixture'),'-test.run=^TestSmartWorkerFixture$','-test.v'],env=dict(os.environ,CONFIG_SMART_FIXTURE_DIR=str(work)),stdout=pl,stderr=pl)
    for _ in range(150):
     if panel.poll() is not None:raise RuntimeError('fixture exited; '+(work/'panel.log').read_text()[-2000:])
     try:json.loads(get(ORIGIN+'/api/smart')[0]);break
     except OSError:time.sleep(.1)
    else:raise RuntimeError('fixture unavailable')
    def launch():return subprocess.Popen([str(ROOT/'bin/config-probe-worker'),'-mode','smart','-socket',str(work/'data/probe.sock'),'-helper',str(work/'smart-probe-one.py'),'-core',str(ROOT/'.local/xray-26.7.28/xray'),'-lock-file',str(work/'smart.lock')],stdout=wl,stderr=wl)
    worker=launch();killed=False;start=time.monotonic();last=-1;max_active=0
    while time.monotonic()-start<480:
     state=json.loads(get(ORIGIN+'/api/smart')[0]);active=state['worker']['active'];max_active=max(max_active,active)
     running=sum(r['state']=='running' for r in state['entries'])
     with sqlite3.connect(str(work/'data/panel.db')) as claims:live_claims=claims.execute("SELECT count(*) FROM smart_heads WHERE token!='' AND lease_until>?",(int(time.time()*1000),)).fetchone()[0]
     max_active=max(max_active,live_claims)
     if not killed and running:worker.kill();worker.wait(timeout=10);worker=launch();killed=True
     if state['stats']['total']>=55 and state['stats']['ready']==state['stats']['total']:break
     if worker.poll() is not None:raise RuntimeError('worker exited; '+(work/'worker.log').read_text()[-1000:])
     now=int(time.monotonic()-start)//20
     if now!=last:last=now;print(json.dumps({'ready':state['stats']['ready'],'total':state['stats']['total'],'active':active,'blocked':state['stats']['blocked']}),flush=True)
     time.sleep(.25)
    else:raise RuntimeError('smart queue timeout: '+json.dumps(state['stats'])+' codes '+str({r['code'] for r in state['entries']}))
    assert killed and max_active<=2
    lines=get(state['export_url'])[0].splitlines();assert len(lines)==state['stats']['ready']==len(set(lines))
    for line in lines:
     j=json.loads(line);assert j['remarks']=='@shenvpn-smart-parallel-v6b'
     assert list(j)==['remarks','log','dns','inbounds','outbounds','routing','burstObservatory','policy','stats','metrics']
     assert j['log']=={'loglevel':'none','access':'none','error':'none','dnsLog':False}
     assert len(j['dns']['servers'])==4 and len(j['routing']['balancers'])==2 and len(j['routing']['rules'])==4
     assert j['outbounds'][0]['tag'] in j['routing']['balancers'][1]['selector'] and j['routing']['balancers'][0]['fallbackTag']=='block'
     assert len([x for x in j['outbounds'] if x['tag'].startswith('proxy-')])>=50
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
     covered=db.execute('SELECT head,count(DISTINCT source) FROM smart_members GROUP BY head').fetchall()
     assert all(n==len(samples) for _,n in covered)
     assert db.execute('SELECT status FROM conversion_worker').fetchone()[0]=='offline'
    receipt.update(passed=True,profiles=len(lines),members_per_profile=50,sources=len(samples),max_active=max_active,sigkill_recovered=True,unique_head_round=True,inline=True,core='xray-26.7.28')
   finally:
    for process in [worker,panel]:
     if process and process.poll() is None:
      process.terminate()
      try:process.wait(timeout=35)
      except subprocess.TimeoutExpired:process.kill();process.wait()
    subprocess.run(['/usr/bin/python3',str(work/'smart-probe-one.py'),'--cleanup'],check=True,timeout=30)
    (ROOT/'.local/smart-worker-diagnostic.log').write_text((work/'worker.log').read_text()[-8000:])
 (ROOT/'.local/smart-worker-test.json').write_text(json.dumps(receipt,indent=2));print(json.dumps(receipt),flush=True)
if __name__=='__main__':main()
