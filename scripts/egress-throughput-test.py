#!/usr/bin/env python3
"""Bounded isolated throughput run over unique, real, retained source inputs.
No production queue, source, subscription or eligibility proof is changed.
No IP classification service is used.
"""
import argparse, collections, json, os, pathlib, signal, sqlite3, subprocess, time, urllib.request

ROOT = pathlib.Path(__file__).resolve().parents[1]

def available():
    for line in pathlib.Path('/proc/meminfo').read_text().splitlines():
        if line.startswith('MemAvailable:'):
            return int(line.split()[1]) * 1024
    return 0

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--reuse')
    parser.add_argument('--count', type=int, default=1000)
    parser.add_argument('--workers', type=int, default=31)
    parser.add_argument('--seconds', type=int, default=1200)
    args = parser.parse_args()
    assert 1 <= args.count <= 2000 and 1 <= args.workers <= 31
    work = pathlib.Path(args.reuse).resolve() if args.reuse else ROOT / '.local' / ('throughput-' + time.strftime('%Y%m%d-%H%M%S'))
    assert work.parent == (ROOT/'.local').resolve() and work.name.startswith('throughput-')
    work.mkdir(mode=0o700,exist_ok=bool(args.reuse))
    (work/'ready.json').unlink(missing_ok=True)
    if args.reuse:
        conn=sqlite3.connect(work/'panel.db')
        conn.execute("UPDATE egress_jobs SET tested=0,attempts=0,state='PENDING',reason='new_input',token='',lease_until=0,next_run=0,report='{}',decision='{}'")
        conn.execute('DELETE FROM egress_events');conn.commit();conn.close()
    subprocess.run(['go', 'test', '-c', '-o', str(work/'fixture'), './internal/panel'], cwd=ROOT, check=True)
    env = dict(os.environ, EGRESS_WORKER_FIXTURE=str(work), EGRESS_BENCHMARK_INPUTS=str(args.count))
    fixture_log = open(work/'fixture.log', 'w')
    worker_log = open(work/'worker.log', 'w')
    fixture = subprocess.Popen([str(work/'fixture'), '-test.run=^TestEgressWorkerFixture$', '-test.v'], env=env, stdout=fixture_log, stderr=fixture_log)
    worker = None
    db = None
    try:
        deadline = time.monotonic()+180
        while time.monotonic()<deadline:
            if fixture.poll() is not None: raise RuntimeError('fixture startup failed')
            try:
                ready = json.loads((work/'ready.json').read_text())
                if ready['pid']==fixture.pid: break
            except (OSError, ValueError): pass
            time.sleep(.2)
        else: raise RuntimeError('fixture startup deadline')
        db = sqlite3.connect('file:'+str(work/'panel.db')+'?mode=ro', uri=True)
        total = db.execute('SELECT count(*) FROM egress_jobs').fetchone()[0]
        assert total>0
        command = [str(ROOT/'bin/config-egress-worker'), '--socket', ready['socket'], '--helper', str(ROOT/'scripts/egress-probe.py'), '--prefix', 'cgt', '--lock-file', str(work/'worker.lock'), '--workers', str(args.workers)]
        command = ['systemd-run', '--scope', '--quiet', '--unit=shen-egress-throughput-'+str(os.getpid()), '-p', 'MemoryMax=2G', '-p', 'TasksMax=4096'] + command
        begin = time.monotonic(); begin_wall_ms = int(time.time()*1000)
        worker = subprocess.Popen(command, stdout=worker_log, stderr=worker_log, start_new_session=True)
        maximum = 0; minimum_memory = available(); at_300 = None; timeline = []; done = 0
        while time.monotonic()-begin<args.seconds:
            if worker.poll() is not None: raise RuntimeError('worker exited')
            elapsed = time.monotonic()-begin
            now = int(time.time()*1000)
            done, active = db.execute("SELECT coalesce(sum(tested>0),0),coalesce(sum(token!='' AND lease_until>?),0) FROM egress_jobs", (now,)).fetchone()
            maximum=max(maximum,active);minimum_memory=min(minimum_memory,available())
            if elapsed>=300 and at_300 is None: at_300=db.execute('SELECT count(DISTINCT hash) FROM egress_events WHERE at>=? AND at<=?',(begin_wall_ms,begin_wall_ms+300000)).fetchone()[0]
            if not timeline or elapsed-timeline[-1]['seconds']>=10:
                row=dict(seconds=round(elapsed,2),assessed=done,total=total,active=active,available_mib=available()//1048576)
                timeline.append(row);print(json.dumps(row),flush=True)
            if done==total: break
            if available()<256*1048576: raise RuntimeError('host memory safety guard')
            time.sleep(1)
        elapsed=time.monotonic()-begin
        if at_300 is None:at_300=done
        states=dict(db.execute('SELECT state,count(*) FROM egress_jobs GROUP BY state'))
        reasons=dict(db.execute('SELECT reason,count(*) FROM egress_jobs GROUP BY reason'))
        samples=collections.Counter(); attempts=[]
        for text, in db.execute("SELECT report FROM egress_jobs WHERE tested>0"):
            report=json.loads(text)
            samples.update(x['code'] for x in (report.get('samples') or []))
            if report.get('wall_ms') is not None: attempts.append(report['wall_ms'])
        attempts.sort()
        receipt=dict(version='ads-egress.v5',requested=args.count,total=total,assessed=done,complete=done==total,seconds=round(elapsed,3),assessed_at_300s=at_300,workers=args.workers,max_active=maximum,min_available_mib=minimum_memory//1048576,states=states,reasons=reasons,sample_codes=dict(samples),definitive=states.get('RESIDENTIAL',0)+states.get('NO_SPLIT',0),job_ms_p50=attempts[len(attempts)//2] if attempts else None,job_ms_p95=attempts[min(len(attempts)-1,int(len(attempts)*.95))] if attempts else None,timeline=timeline,fixture_origin=ready['origin'],scope='Distinct real retained source hashes; initial assessments include inconclusive and unsupported results. Not 1000 definitive residential classifications.')
        (work/'receipt.json').write_text(json.dumps(receipt,indent=2))
        (ROOT/'.local/throughput-latest.json').write_text(json.dumps({'directory':str(work),**receipt},indent=2))
        if os.environ.get('PLAYWRIGHT_MODULE') and os.environ.get('CHROME_PATH'):
            browser_env=dict(os.environ,PANEL_TEST_ORIGIN=ready['origin'],PANEL_TEST_DIR=str(work))
            subprocess.run(['node',str(ROOT/'scripts/egress-browser.cjs')],env=browser_env,check=True,timeout=90)
        print(json.dumps({k:v for k,v in receipt.items() if k!='timeline'}),flush=True)
    finally:
        if worker and worker.poll() is None:
            os.killpg(worker.pid,signal.SIGTERM)
            try:worker.wait(timeout=40)
            except subprocess.TimeoutExpired:os.killpg(worker.pid,signal.SIGKILL);worker.wait(timeout=5)
        subprocess.run(['/usr/bin/python3',str(ROOT/'scripts/egress-probe.py'),'--prefix','cgt','--cleanup','--slot','0'],check=True)
        if db:db.close()
        if fixture.poll() is None:fixture.terminate();fixture.wait(timeout=20)
        fixture_log.close();worker_log.close()

if __name__=='__main__':main()
