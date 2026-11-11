#!/usr/bin/env python3
"""Live isolated worker crash recovery plus mobile UI, never production mutations."""
import os,pathlib,subprocess,json,time,sqlite3,signal,secrets
ROOT=pathlib.Path(__file__).resolve().parents[1]
WORK=pathlib.Path('/tmp/shen-egress-p39-worker-'+secrets.token_hex(4));WORK.mkdir(mode=0o700)
subprocess.run(['go','test','-c','-o',str(WORK/'fixture'),'./internal/panel'],cwd=ROOT,check=True)
subprocess.run(['go','build','-o',str(WORK/'worker'),'./cmd/config-egress-worker'],cwd=ROOT,check=True)
env=dict(os.environ,EGRESS_WORKER_FIXTURE=str(WORK))
with open(WORK/'fixture.log','w') as log:
 fixture=subprocess.Popen([str(WORK/'fixture'),'-test.run=^TestEgressWorkerFixture$','-test.v'],env=env,stdout=log,stderr=log)
 worker=None
 try:
  for _ in range(100):
   if fixture.poll() is not None:raise RuntimeError('fixture startup failed')
   try:
    ready=json.loads((WORK/'ready.json').read_text())
    if ready['pid']==fixture.pid:break
   except (OSError,ValueError):pass
   time.sleep(.1)
  else:raise RuntimeError('fixture not ready')
  args=[str(WORK/'worker'),'--socket',ready['socket'],'--helper',str(ROOT/'scripts/egress-probe.py'),'--prefix','cgt','--lock-file',str(WORK/'worker.lock'),'--workers','2']
  def start():return subprocess.Popen(args,stdout=subprocess.DEVNULL,stderr=open(WORK/'worker.log','a'))
  worker=start();db=sqlite3.connect(WORK/'panel.db')
  for _ in range(100):
   active=db.execute("SELECT count(*) FROM egress_jobs WHERE token!='' AND lease_until>?",(int(time.time()*1000),)).fetchone()[0]
   if active==2:break
   time.sleep(.1)
  assert active==2
  killed=[x[0] for x in db.execute("SELECT hash FROM egress_jobs WHERE token!=''").fetchall()];assert len(killed)==2;worker.kill();worker.wait();time.sleep(1);worker=start()
  browserEnv=dict(os.environ,PANEL_TEST_ORIGIN=ready['origin'],PANEL_TEST_DIR=str(WORK))
  subprocess.run(['node',str(ROOT/'scripts/egress-browser.cjs')],env=browserEnv,check=True,timeout=90)
  begin=time.monotonic();maximum=0
  while time.monotonic()-begin<300:
   total,tested,active=db.execute("SELECT count(*),sum(tested>0),sum(token!='' AND lease_until>?) FROM egress_jobs",(int(time.time()*1000),)).fetchone();maximum=max(maximum,len(list(pathlib.Path("/run/netns").glob("cgt*"))))
   if tested==total:break
   time.sleep(2)
  assert tested==total,(tested,total)
  assert maximum<=2
  assert all(db.execute('SELECT attempts,tested FROM egress_jobs WHERE hash=?',(h,)).fetchone()[0]>=2 for h in killed)
  states=dict(db.execute('SELECT state,count(*) FROM egress_jobs GROUP BY state').fetchall())
  receipt=dict(directory=str(WORK),passed=True,total=total,tested=tested,max_active=maximum,states=states,crash_recovered=True,seconds=round(time.monotonic()-begin,2),live_inputs=3)
  (WORK/'receipt.json').write_text(json.dumps(receipt,indent=2));print(json.dumps(receipt));db.close()
 finally:
  if worker and worker.poll() is None:worker.terminate();worker.wait(timeout=40)
  fixture.terminate();fixture.wait(timeout=20)
  subprocess.run(['/usr/bin/python3',str(ROOT/'scripts/egress-probe.py'),'--prefix','cgt','--cleanup','--slot','0'],check=True)
