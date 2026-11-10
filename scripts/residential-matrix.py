#!/usr/bin/env python3
"""Adaptive official-core compatibility driver; one complete report wins.

Latest core first, then two bounded parallel compatibility attempts. Never mix
raw/template evidence across cores, edit credentials, or substitute Direct.
"""
import argparse, concurrent.futures, contextlib, copy, json, os, pathlib
import select as select_io
import selectors, signal, subprocess, sys, threading, time

HERE = pathlib.Path(__file__).resolve().parent
PINS = json.loads((HERE/'residential-cores.json').read_text())['cores']
PROCESSES = set()
LOCK = threading.RLock()
STOP = threading.Event()
MAX_OUTPUT = 4*1024*1024

def healthy(report):
    variants = report.get('variants', [])
    return report.get('infrastructure_ok', False) and len(variants) == 2 and all(v.get('core') == 'passed' and v.get('health') == 'HEALTHY' for v in variants)

def rank(report):
    return (healthy(report), sum(v.get('core') == 'passed' for v in report.get('variants', [])),
            sum(c.get('outcome') == 'passed' for c in report.get('controls', [])),
            sum(bool(o.get('ip')) for o in report.get('observations', [])))

def select(reports):
    # Stable ties prefer the latest core, never the fastest failed response.
    winner = copy.deepcopy(max(reports, key=rank))
    winner['compatibility'] = [{
        'core_id': r['core_id'], 'core_sha256': r['core_sha256'],
        'started': r['started'], 'finished': r['finished'],
        'infrastructure_ok': r['infrastructure_ok'], 'variants': r['variants'],
        'controls_passed': sum(c.get('outcome') == 'passed' for c in r.get('controls', [])),
        'wall_ms': r.get('wall_ms', 0),
    } for r in reports]
    if not healthy(winner) and any(not r['infrastructure_ok'] for r in reports):
        winner['infrastructure_ok'] = False  # Missing driver is not a deletion vote.
    return winner

def stop(*_):
    STOP.set()
    with LOCK:
        for p in PROCESSES:
            if p.poll() is None:
                with contextlib.suppress(ProcessLookupError):p.terminate()

def terminate(p):
    if p.poll() is None:
        with contextlib.suppress(ProcessLookupError):p.terminate()
        try:p.wait(timeout=1)
        except subprocess.TimeoutExpired:
            with contextlib.suppress(ProcessLookupError):p.kill()
            p.wait(timeout=2)

def run_bounded(command, payload, timeout):
    p=None
    try:
        with LOCK:
            if STOP.is_set():raise InterruptedError()
            p=subprocess.Popen(command,stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,
                               env=os.environ|{'SHEN_RESIDENTIAL_PARENT_PID':str(os.getpid())})
            PROCESSES.add(p)
        if STOP.is_set():raise InterruptedError()
        output=bytearray();position=0;deadline=time.monotonic()+timeout;stdout_open=True
        with selectors.DefaultSelector() as poll:
            for pipe,event in [(p.stdin,selectors.EVENT_WRITE),(p.stdout,selectors.EVENT_READ)]:
                os.set_blocking(pipe.fileno(),False);poll.register(pipe,event)
            while stdout_open or p.poll() is None:
                if STOP.is_set():raise InterruptedError()
                if time.monotonic()>deadline:raise TimeoutError()
                for key,event in poll.select(.1):
                    if event==selectors.EVENT_WRITE:
                        try:position+=os.write(p.stdin.fileno(),payload[position:position+65536])
                        except BrokenPipeError:position=len(payload)
                        if position==len(payload):poll.unregister(p.stdin);p.stdin.close()
                    else:
                        chunk=os.read(p.stdout.fileno(),65536)
                        if not chunk:poll.unregister(p.stdout);stdout_open=False
                        output.extend(chunk)
                        if len(output)>MAX_OUTPUT:raise ValueError('driver_output_limit')
        if STOP.is_set():raise InterruptedError()
        return p.wait(timeout=1),bytes(output)
    finally:
        if p is not None:
            terminate(p)
            for pipe in [p.stdin,p.stdout]:
                if pipe and not pipe.closed:pipe.close()
            with LOCK:PROCESSES.discard(p)

def read_request():
    deadline=time.monotonic()+15;body=bytearray()
    while True:
        if STOP.is_set():raise InterruptedError()
        if time.monotonic()>deadline:raise TimeoutError()
        if not select_io.select([sys.stdin.buffer],[],[],.1)[0]:continue
        chunk=os.read(sys.stdin.fileno(),65536)
        if not chunk:return json.loads(body)
        body.extend(chunk)
        if len(body)>8*1024*1024:raise ValueError('driver_input_limit')

def cleanup_slots(helper,slot,run_id):
    failed=False
    for physical in range(3*slot-2,3*slot+1):
        cmd=[sys.executable,helper,'--cleanup-slot',str(physical)]
        if run_id:cmd+=['--cleanup-run',run_id]
        try:failed=subprocess.run(cmd,timeout=5,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL).returncode!=0 or failed
        except (OSError,subprocess.TimeoutExpired):failed=True
    if failed:raise RuntimeError('matrix_cleanup_incomplete')

def main():
    ap=argparse.ArgumentParser();ap.add_argument('--core');ap.add_argument('--slot',type=int)
    ap.add_argument('--run-id');ap.add_argument('--cleanup',action='store_true')
    ap.add_argument('--cleanup-slot',type=int);ap.add_argument('--cleanup-run');ap.add_argument('--self-test',action='store_true')
    args=ap.parse_args()
    helper=str(HERE/'residential-probe.py')
    if args.cleanup or args.self_test:
        subprocess.run([sys.executable,helper]+sys.argv[1:],check=True,timeout=60);return
    if args.cleanup_slot is not None:
        if not 1<=args.cleanup_slot<=2:raise ValueError('invalid_slot')
        cleanup_slots(helper,args.cleanup_slot,args.cleanup_run)
        return
    if not args.slot or not 1<=args.slot<=2 or not args.core or not args.run_id:raise ValueError('invalid_matrix_request')
    signal.signal(signal.SIGTERM,stop)
    parent=os.getppid()
    if parent<=1 or int(os.environ.get('SHEN_RESIDENTIAL_PARENT_PID',parent))!=parent:raise ValueError('owner_unavailable')
    def watch():
        while os.getppid()==parent:time.sleep(.25)
        os.kill(os.getpid(),signal.SIGTERM)
    threading.Thread(target=watch,daemon=True).start()
    request=read_request();payload=json.dumps(request).encode();begin=time.monotonic()
    def one(index):
        pin=PINS[index];core=pathlib.Path(args.core) if pin['id']=='xray-26.7.28' else pathlib.Path('/opt/config-repair')/('core-'+pin['id'][5:])/'xray'
        start=int(time.time()*1000)
        failure={'version':'shen.residential.v1','policy_hash':request['policy_hash'],'template_version':request['template_version'],
                 'core_id':pin['id'],'core_sha256':pin['sha256'],'worker':request['worker'],'infrastructure_epoch':request['infrastructure_epoch'],
                 'started':start,'finished':start,'infrastructure_ok':False,'variants':[],'observations':[],'controls':[],'ip_facts':[]}
        try:
            rc,out=run_bounded([sys.executable,helper,'--core',str(core),'--core-id',pin['id'],'--slot',str(3*args.slot-2+index),'--run-id',args.run_id],payload,105)
            if rc:return failure
            result=json.loads(out)
            if result['core_sha256']!=pin['sha256'] or result['core_id']!=pin['id']:return failure
            return result
        except (OSError,ValueError,KeyError):return failure
        finally:failure['finished']=int(time.time()*1000)
    reports=[one(0)]
    if not healthy(reports[0]):
        with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:reports+=list(pool.map(one,[1,2]))
    if STOP.is_set():raise SystemExit(143)
    result=select(reports)
    result['wall_ms']=round((time.monotonic()-begin)*1000,3)
    result['cpu_ms']=sum(r.get('cpu_ms',0) for r in reports)
    result['peak_rss_bytes']=max(r.get('peak_rss_bytes',0) for r in reports)
    encoded=json.dumps(result)
    if len(encoded.encode())>MAX_OUTPUT:raise ValueError('matrix_output_limit')
    print(encoded)

if __name__=='__main__':main()
