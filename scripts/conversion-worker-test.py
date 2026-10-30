#!/usr/bin/env python3
"""Isolated durable conversion queue and real sandbox acceptance, optional read-only canary."""
import argparse,base64,hashlib,json,os,pathlib,secrets,sqlite3,subprocess,tempfile,time,urllib.request
ROOT=pathlib.Path(__file__).resolve().parents[1]
ORIGIN='http://127.0.0.1:18194'
def get(url,production=False):
 req=urllib.request.Request(url,headers={'Host':'85.192.63.181:18443'} if production else {})
 with urllib.request.urlopen(req,timeout=20) as f:return f.read()
def main():
 a=argparse.ArgumentParser();a.add_argument('--live-canary',type=int,default=0);args=a.parse_args();receipt={'android':'not_run'}
 if os.geteuid()!=0:raise SystemExit('root required for sandbox integration')
 subprocess.run(['go','test','-c','-o',str(ROOT/'.local/p7-worker-fixture'),'./internal/panel'],check=True,cwd=ROOT)
 with tempfile.TemporaryDirectory(prefix='p7-worker-',dir=ROOT/'.local') as tmp:
  work=pathlib.Path(tmp);os.chmod(work,0o700)
  key=base64.b64encode(bytes([1])*32).decode();pub=base64.b64encode(bytes([2])*32).decode()
  samples=[b'socks5://user:synthetic@192.0.2.1:1080',b'hysteria://synthetic@192.0.2.1:443',f'wireguard://{key}@192.0.2.1:51820?publickey={pub}&address=10.0.0.2%2F32'.encode(),b'hy2://synthetic@192.0.2.1:443?sni=example.com',b'vless://00000000-0000-4000-8000-000000000001@192.0.2.1:443?security=tls&sni=example.com']
  if args.live_canary:
   records=[]
   for page in range(100):
    state=json.loads(get('http://127.0.0.1:18191/api/state?page='+str(page),True));records.extend(x for x in state['outputs'] if not x['disabled'])
    if len(state['outputs'])<100:break
   # Prefer currently successful originals while still sampling protocol diversity.
   db=sqlite3.connect('file:/var/lib/config-repair/panel.db?mode=ro',uri=True)
   table='probe_results' if db.execute("SELECT count(*) FROM sqlite_master WHERE name='probe_results'").fetchone()[0] else 'conversion_results'
   good={r[0] for r in db.execute('SELECT hash FROM '+table+" WHERE connection='passed'")};db.close()
   records.sort(key=lambda x:(x['hash'] not in good,x['kind'],x['hash']))
   picked=[];kinds=set()
   for x in records:
    if x['kind'] not in kinds:picked.append(x);kinds.add(x['kind'])
   picked+= [x for x in records if x not in picked]
   n=0
   for item in picked:
    if n>=args.live_canary:break
    try:
     raw=get('http://127.0.0.1:18191/output/raw/'+item['id'],True)
     if len(raw)>2*1024*1024 or hashlib.sha256(raw).hexdigest()!=item['hash']:continue
     samples.append(raw);n+=1
    except OSError:continue
   receipt['live_samples']=n
  (work/'samples.json').write_text(json.dumps([base64.b64encode(b).decode() for b in samples]))
  # Distinct tested resource prefix and lock: cannot clean production namespaces.
  (work/'probe-one.py').write_text((ROOT/'scripts/probe-one.py').read_text().replace('crw','ctw'))
  (work/'live-probe.py').write_text((ROOT/'scripts/live-probe.py').read_text())
  panel=worker=None
  with (work/'panel.log').open('w') as pl,(work/'worker.log').open('w') as wl:
   try:
    env=dict(os.environ,CONFIG_CONVERSION_FIXTURE_DIR=str(work));panel=subprocess.Popen([str(ROOT/'.local/p7-worker-fixture'),'-test.run=^TestConversionWorkerFixture$','-test.v'],env=env,stdout=pl,stderr=pl)
    for _ in range(100):
     try:json.loads(get(ORIGIN+'/api/conversions'));break
     except OSError:time.sleep(.1)
    else:raise RuntimeError('fixture unavailable')
    def launch():return subprocess.Popen([str(ROOT/'bin/config-probe-worker'),'-socket',str(work/'data/probe.sock'),'-helper',str(work/'probe-one.py'),'-core',str(ROOT/'.local/xray-26.7.28/xray'),'-lock-file',str(work/'worker.lock')],stdout=wl,stderr=wl)
    worker=launch();killed=False;start=time.monotonic();max_active=0;last=-1
    while time.monotonic()-start<420:
     state=json.loads(get(ORIGIN+'/api/conversions'));rows=state['entries'];active=sum(r['state']=='running' for r in rows);max_active=max(max_active,active);completed=sum(r['tested']>0 for r in rows)
     if not killed and active:worker.kill();worker.wait(timeout=10);worker=launch();killed=True
     if completed==len(rows) and rows:break
     if worker.poll() is not None:raise RuntimeError('worker stopped; '+(work/'worker.log').read_text()[-1000:])
     if int(time.monotonic()-start)//20!=last:last=int(time.monotonic()-start)//20;print(json.dumps({'completed':completed,'total':len(rows),'active':active}),flush=True)
     time.sleep(1)
    else:raise RuntimeError('durable conversion queue did not finish')
    lines=get(state['export_url']).splitlines();configs=[json.loads(b) for b in lines]
    assert killed and max_active<=3 and any(r['code']=='hysteria1_unsupported_by_target_core' for r in rows)
    assert len(lines)==len(set(lines)) and all(c['remarks']=='@shenvpn' and len(c['outbounds'])>=4 for c in configs)
    receipt.update(passed=True,total=len(rows),completed=completed,core_passed=sum(r['core']=='passed' for r in rows),connected=sum(r['connection']=='passed' for r in rows),jsonl_configs=len(lines),max_active=max_active,sigkill_recovered=True,codes={c:sum(r['code']==c for r in rows) for c in sorted({r['code'] for r in rows})})
   finally:
    for process in [worker,panel]:
     if process and process.poll() is None:
      process.terminate()
      try:process.wait(timeout=35)
      except subprocess.TimeoutExpired:process.kill();process.wait()
    subprocess.run(['/usr/bin/python3',str(work/'probe-one.py'),'--cleanup'],check=True,timeout=30)
    (ROOT/'.local/p7-worker-diagnostic.log').write_text((work/'worker.log').read_text()[-8000:])
 (ROOT/'.local/p7-worker-test.json').write_text(json.dumps(receipt,indent=2));print(json.dumps(receipt),flush=True)
if __name__=='__main__':main()
