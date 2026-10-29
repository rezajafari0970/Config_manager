#!/usr/bin/env python3
"""Isolated queue/worker/restart acceptance. Optional authorized read-only canary."""
import argparse, base64, hashlib, http.cookiejar, json, os, pathlib, re, secrets, signal, sqlite3, subprocess, tempfile, time, urllib.request

ROOT=pathlib.Path(__file__).resolve().parents[1]
ORIGIN='http://127.0.0.1:18194'
def main():
    parser=argparse.ArgumentParser();parser.add_argument('--live-canary',type=int,default=0);a=parser.parse_args()
    if os.geteuid()!=0:raise SystemExit('root required for isolated sandbox acceptance')
    receipt={'android':'not_run'}
    with tempfile.TemporaryDirectory(prefix='p6-worker-',dir=ROOT/'.local') as temp:
        work=pathlib.Path(temp);os.chmod(work,0o700);cfg=work/'panel.json';cfg.write_text(json.dumps({'listen':'127.0.0.1:18194','public_origin':ORIGIN,'database':str(work/'data/panel.db'),'master_key':base64.b64encode(secrets.token_bytes(32)).decode()}));os.chmod(cfg,0o600)
        panel_log=(work/'panel.log').open('w');worker_log=(work/'worker.log').open('w');panel=None;worker=None
        try:
            panel=subprocess.Popen([str(ROOT/'bin/config-panel'),'-config',str(cfg)],stdout=panel_log,stderr=panel_log)
            opener=urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))
            for _ in range(100):
                try:
                    with opener.open(ORIGIN+'/output',timeout=2) as f:html=f.read().decode()
                    break
                except OSError:time.sleep(.1)
            else:raise RuntimeError('isolated_panel_unavailable')
            csrf=re.search(r'name="csrf-token" content="([^"]+)"',html).group(1)
            def get(path):
                with opener.open(ORIGIN+path,timeout=20) as f:return json.load(f)
            def post(path,value,raw=False):
                req=urllib.request.Request(ORIGIN+path,data=value if raw else json.dumps(value).encode(),headers={'Origin':ORIGIN,'X-CSRF-Token':csrf,'Content-Type':'text/plain' if raw else 'application/json'})
                with opener.open(req,timeout=30) as f:return json.load(f)
            key=base64.b64encode(bytes([1])*32).decode();pub=base64.b64encode(bytes([2])*32).decode()
            samples=[b'socks5://user:synthetic@192.0.2.1:1080',b'hysteria://synthetic@192.0.2.1:443',f'wireguard://{key}@192.0.2.1:51820?publickey={pub}&address=10.0.0.2%2F32'.encode(),b'hy2://synthetic@192.0.2.1:443?sni=example.com',b'{"stats":[],"outbounds":[{"protocol":"blackhole","settings":{}}]}']
            if a.live_canary:
                known=set()
                old=ROOT/'.local/probe-bank/evidence.db'
                if old.exists():
                    db=sqlite3.connect('file:'+str(old)+'?mode=ro',uri=True);known={r[0] for r in db.execute("SELECT original_sha256 FROM attempts WHERE connection='passed'")};db.close()
                records=[];page=0
                while True:
                    req=urllib.request.Request('http://127.0.0.1:18191/api/state?page='+str(page),headers={'Host':'85.192.63.181:18443'})
                    with urllib.request.urlopen(req,timeout=20) as f:state=json.load(f)
                    if not state['outputs']:break
                    records.extend(x for x in state['outputs'] if not x['disabled']);page+=1
                records.sort(key=lambda x:(x['hash'] not in known,x['kind'],x['hash']))
                added=0
                for item in records:
                    if added>=a.live_canary:break
                    try:
                        req=urllib.request.Request('http://127.0.0.1:18191/output/raw/'+item['id'],headers={'Host':'85.192.63.181:18443'})
                        with urllib.request.urlopen(req,timeout=20) as f:raw=f.read(2*1024*1024+1)
                        if hashlib.sha256(raw).hexdigest()!=item['hash']:continue
                        samples.append(raw);added+=1
                    except OSError:continue
                receipt['live_samples']=added
            for raw in samples:post('/api/bank/import',raw,True)
            post('/api/probe/settings',{'enabled':True,'concurrency':3,'timeout_seconds':6,'retest_seconds':3600,'health_ttl_seconds':7200})
            def launch():return subprocess.Popen([str(ROOT/'bin/config-probe-worker'),'-socket',str(work/'data/probe.sock'),'-helper',str(ROOT/'scripts/probe-one.py'),'-core',str(ROOT/'.local/xray-26.7.28/xray')],stdout=worker_log,stderr=worker_log)
            worker=launch();killed=False;retries=0;start=time.monotonic();max_active=0
            while time.monotonic()-start<420:
                state=get('/api/bank');max_active=max(max_active,state['stats']['running'])
                if not killed and state['stats']['running']:
                    worker.kill();worker.wait(timeout=10);worker=launch();killed=True;receipt['supervisor_sigkill_recovery']='exercised'
                rows=state['entries'];completed=sum(v['tested']>0 for v in rows)
                if completed==len(rows) and rows:break
                if worker.poll() is not None:
                    retries+=1
                    if retries>3:raise RuntimeError('isolated_probe_supervisor_stopped')
                    time.sleep(30);worker=launch();receipt['preflight_retries']=retries
                if int(time.monotonic()-start)%20==0:print(json.dumps({'completed':completed,'total':len(rows),'active':state['stats']['running']}),flush=True)
                time.sleep(1)
            else:raise RuntimeError('durable_queue_did_not_finish')
            receipt.update(total=len(rows),completed=completed,max_active=max_active,verified=state['stats']['verified'],codes={code:sum(v['code']==code for v in rows) for code in {v['code'] for v in rows}})
            assert max_active<=3 and killed
            assert any(v['code']=='hysteria1_unsupported_by_target_core' for v in rows)
            assert all(v['android']=='not_run' for v in rows)
            with opener.open(state['subscriptions']['all']) as f:export=json.load(f)
            receipt['subscription_configs']=len(export)
            assert len(export)<=state['stats']['verified']
            keep=rows[0]['hash'];post('/api/bank/batch',{'action':'disable','ids':[keep]});assert not next(v for v in get('/api/bank')['entries'] if v['hash']==keep)['enabled']
            receipt['passed']=True
        finally:
            for process in [worker,panel]:
                if process and process.poll() is None:
                    process.terminate()
                    try:process.wait(timeout=35)
                    except subprocess.TimeoutExpired:process.kill();process.wait()
            subprocess.run(['/usr/bin/python3',str(ROOT/'scripts/probe-one.py'),'--cleanup'],check=True,timeout=30)
            panel_log.close();worker_log.close()
            (ROOT/'.local/p6-worker-diagnostic.log').write_text((work/'worker.log').read_text()[-8000:])
    (ROOT/'.local/p6-worker-test.json').write_text(json.dumps(receipt,indent=2));print(json.dumps(receipt),flush=True)
if __name__=='__main__':main()
