#!/usr/bin/env python3
"""Isolated real worker/lease/SIGKILL test. Production is read-only canaries.

No fixture classification is promoted to live residential evidence.
"""
import base64,hashlib,json,os,pathlib,sqlite3,subprocess,tempfile,time,urllib.request,urllib.error
ROOT=pathlib.Path(__file__).resolve().parents[1]
ORIGIN='http://127.0.0.1:18194'
def get(url,prod=False):
    with urllib.request.urlopen(urllib.request.Request(url,headers={'Host':'85.192.63.181:18443'} if prod else {}),timeout=15) as r:return r.read()
def main():
    if os.geteuid()!=0:raise RuntimeError('sandbox supervisor required')
    receipt={'android':'not_run','carriers':'not_run','live_residential_proof':False}
    samples=[b'tuic://00000000-0000-4000-8000-000000000001:synthetic@192.0.2.1:443?sni=example.com',b'hysteria://synthetic@192.0.2.1:443',b'vless://00000000-0000-4000-8000-000000000001@192.0.2.2:443?security=tls&sni=example.com']
    records=json.loads(get('http://127.0.0.1:18191/api/conversions',True))['entries']
    seen=set()
    for item in sorted(records,key=lambda x:x.get('connection')!='passed'):
        if len(samples)>=7:break
        try:
            raw=get('http://127.0.0.1:18191/output/raw/'+item['id'],True)
            digest=hashlib.sha256(raw).hexdigest()
            if digest not in seen and len(raw)<=2*1024*1024:samples.append(raw);seen.add(digest)
        except (OSError,KeyError):continue
    receipt['live_samples']=len(samples)-3
    subprocess.run(['go','test','-c','-o',str(ROOT/'.local/p28-worker-fixture'),'./internal/panel'],cwd=ROOT,check=True)
    with tempfile.TemporaryDirectory(prefix='p28-worker-',dir=ROOT/'.local') as directory:
        work=pathlib.Path(directory);(work/'signing.json').write_text(json.dumps({'master_key':base64.b64encode(bytes([21])*32).decode()}));os.chmod(work/'signing.json',0o600);(work/'samples.json').write_text(json.dumps([base64.b64encode(x).decode() for x in samples]))
        panel=worker=None
        with (work/'panel.log').open('w') as pl,(work/'worker.log').open('w') as wl:
            try:
                panel=subprocess.Popen([str(ROOT/'.local/p28-worker-fixture'),'-test.run=^TestConversionWorkerFixture$'],env=dict(os.environ,CONFIG_CONVERSION_FIXTURE_DIR=directory),stdout=pl,stderr=pl)
                for _ in range(100):
                    try:state=json.loads(get(ORIGIN+'/api/residential'));break
                    except OSError:time.sleep(.1)
                else:raise RuntimeError('fixture not ready')
                def launch():return subprocess.Popen([str(ROOT/'bin/config-residential-worker'),'-signing-config',str(work/'signing.json'),'-socket',str(work/'data/probe.sock'),'-helper',str(ROOT/'scripts/residential-matrix.py'),'-core',os.environ['XRAY_TEST_BINARY'],'-lock-file',str(work/'worker.lock')],env=dict(os.environ,SHEN_RESIDENTIAL_TEST_PREFIX='crt'),stdout=wl,stderr=wl)
                start=time.monotonic();worker=launch();killed=False;maximum=0
                while time.monotonic()-start<420:
                    with sqlite3.connect(work/'data/panel.db') as db:
                        total,completed,active=db.execute("SELECT count(*),sum(tested>0),sum(token!='' AND lease_until>?) FROM residential_records",(int(time.time()*1000),)).fetchone()
                    maximum=max(maximum,active or 0)
                    if not killed and active:worker.kill();worker.wait(timeout=10);worker=launch();killed=True
                    if killed and completed==total and total:break
                    if worker.poll() is not None:raise RuntimeError('worker stopped')
                    time.sleep(.2)
                else:raise RuntimeError('worker recovery timeout')
                state=json.loads(get(ORIGIN+'/api/residential'));assert maximum<=2 and state['eligible_count']==0 and killed
                statuses={}
                for name,url in state['links'].items():
                    try:get(url);raise AssertionError('uncertain config published')
                    except urllib.error.HTTPError as e:statuses[name]=e.code;assert e.code==503
                assert 'UNSUPPORTED' in state['counts']
                receipt.update(passed=True,total=total,completed=completed,max_active=maximum,sigkill_recovered=True,elapsed_s=round(time.monotonic()-start,2),counts=state['counts'],metrics=state['metrics'],export_status=statuses)
                with sqlite3.connect(work/'data/panel.db') as db:
                    # Public structured observations only, never raw configs or URLs.
                    receipt['evidence']=[json.loads(x[0]) for x in db.execute("SELECT report FROM residential_records WHERE tested>0")]
                    assert any(r.get('core_id')=='sing-box-1.14.4' for r in receipt['evidence'])
                    receipt['native_core_selected']=True
            finally:
                # Preserve bounded private failure diagnostics before temp cleanup.
                for name in ['worker.log','panel.log']:
                    if (work/name).exists():(ROOT/'.local'/('p30-last-'+name)).write_bytes((work/name).read_bytes()[-65536:])
                for p in (worker,panel):
                    if p and p.poll() is None:
                        p.terminate()
                        try:p.wait(timeout=35)
                        except subprocess.TimeoutExpired:p.kill();p.wait()
                subprocess.run(['/usr/bin/python3',str(ROOT/'scripts/residential-probe.py'),'--cleanup'],env=dict(os.environ,SHEN_RESIDENTIAL_TEST_PREFIX='crt'),check=True,timeout=30)
    (ROOT/'.local/p28-worker-test.json').write_text(json.dumps(receipt,indent=2))
    print(json.dumps({k:v for k,v in receipt.items() if k!='evidence'}))
if __name__=='__main__':main()