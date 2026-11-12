#!/usr/bin/env python3
"""Trusted one-job adapter. Payload only on stdin; receipts never contain secrets."""
import argparse, threading, time, signal, base64, contextlib, importlib.util, json, os, pathlib, re, shutil, subprocess, sys, uuid

HERE=pathlib.Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('probe_sandbox',HERE/'live-probe.py')
lab=importlib.util.module_from_spec(spec);spec.loader.exec_module(lab)
spec=importlib.util.spec_from_file_location("app_contract",HERE/"app-import-contract.py")
app=importlib.util.module_from_spec(spec);spec.loader.exec_module(app)
CORE_SHA='e87bce586acdb9136aa5b1f1d9c8db9f1ea1d145b6d86368c7f6f662f9f6a66d'
# Client exports must also build on the current official core. Older-core
# connectivity (including legacy VLESS vnext) is not current-client acceptance.
CLIENT_CORE_SHA='cc4b84d3069569f297a61f04b914f2bb37e9d202fd7b84cb50def1fc30fd40e1'
CLIENT_CORE='/opt/config-repair/core-26.10.31/xray'
PREFIX='crw'
OWN=re.compile(r'^'+PREFIX+r'[0-9a-f]{6}20(?:0[0-8])$')

def validate_client(box, data, core):
    """Build the exact exported bytes in the existing sandbox, before networking.

    Return a bounded diagnostic, never raw logs/endpoints. A successful build
    proves syntax/core compatibility only. Never rewrite security or credentials.
    """
    previous = box.core
    result = {'core':'blocked','connection':'not_run','code':'client_core_infrastructure_error',
              'runtime_sha256':'','endpoints_passed':0}
    try:
        box.stop()
        box.core = pathlib.Path(core).resolve()
        box.policy({})
        process = box.launch(data, True)
        try:
            process.wait(timeout=15)
        except subprocess.TimeoutExpired:
            result['code'] = 'client_core_validation_timeout'
            return result
        log = box.logs()
        if process.returncode == 0:
            return None
        if process.returncode < 0 or not log.startswith(b'Xray '):
            return result
        code = lab.classify(log, 'client_core_incompatible')
        if code == 'missing_runtime_asset':
            result['code'] = 'client_core_asset_missing'
            return result
        result['core'] = 'failed'
        if b'vless without TLS or other encryption is prohibited' in log:
            result['code'] = 'client_vless_encryption_required'
        elif b'trojan without TLS is prohibited' in log:
            result['code'] = 'client_trojan_tls_required'
        else:
            result['code'] = 'client_core_incompatible'
        return result
    finally:
        box.stop()
        box.core = previous

def run_target(box,candidate,timeout,core):
    previous=box.core
    try:
        box.core=pathlib.Path(core).resolve()
        result=lab.test_candidate(box,candidate,timeout)
        result.setdefault('evidence',{})['runtime_core']='xray_26_10_31'
        return result
    finally:
        box.stop();box.core=previous

def attempt_score(result,candidate):
    eligible=result['core']=='passed' and (not candidate.get('requires_connection') or result['connection']=='passed')
    payload=result['connection']=='passed' and result.get('code')=='https_payload_verified'
    return (eligible,payload,{'not_run':0,'failed':1,'partial':2,'passed':3}[result['connection']],result['core']=='passed')

def cleanup(slot=None,runid=None):
    names=set()
    for folder in ['/run/netns','/sys/fs/cgroup','/dev/shm']:
        for p in pathlib.Path(folder).glob(PREFIX+'*'):
            name=p.name.split('-')[0]
            if OWN.fullmatch(name) and (slot is None or name.endswith(format(8192+slot,'x'))) and (runid is None or name.startswith(PREFIX+runid)):names.add(name)
    # Covers crashes between NAT-table creation and namespace registration.
    r=subprocess.run([lab.NFT,'-j','list','tables'],capture_output=True)
    if r.returncode==0:
        for item in json.loads(r.stdout).get('nftables',[]):
            t=item.get('table',{});name=t.get('name','')
            if t.get('family')=='ip' and OWN.fullmatch(name) and (slot is None or name.endswith(format(8192+slot,'x'))) and (runid is None or name.startswith(PREFIX+runid)):names.add(name)
    for name in names:
        cg=pathlib.Path('/sys/fs/cgroup')/name
        with contextlib.suppress(OSError):(cg/'cgroup.kill').write_text('1')
        for args in [[lab.NFT,'delete','table','ip',name],[lab.IP,'link','del',name+'h'],[lab.IP,'netns','del',name]]:subprocess.run(args,capture_output=True,timeout=5)
        with contextlib.suppress(OSError):cg.rmdir()
        for p in pathlib.Path('/dev/shm').glob(name+'-*'):
            if p.is_dir() and not p.is_symlink():shutil.rmtree(p)

def main(core_only=False):
    global OWN
    if PREFIX not in ('crw','csm','csp','csa','ctw','cst','cpt','cat','crr','crt'):raise SystemExit('invalid resource prefix')
    OWN=re.compile(r'^'+PREFIX+r'[0-9a-f]{6}20(?:0[0-8])$')
    a=argparse.ArgumentParser();a.add_argument('--core');a.add_argument('--client-core',default=CLIENT_CORE);a.add_argument('--slot',type=int);a.add_argument('--cleanup',action='store_true');a.add_argument('--cleanup-slot',type=int);a.add_argument('--self-test',action='store_true');a.add_argument('--run-id');a.add_argument('--cleanup-run');args=a.parse_args()
    if os.geteuid()!=0:raise SystemExit('root supervisor required')
    for value in [args.run_id,args.cleanup_run]:
        if value is not None and not re.fullmatch('[0-9a-f]{6}',value):raise SystemExit('invalid run id')
    # Prevent orphan supervisors after their Go parent is killed.
    parent=os.getppid()
    signal.signal(signal.SIGTERM,lambda *_:sys.exit(143))
    if parent==1:raise SystemExit('parent unavailable')
    def watch_parent():
        while True:
            time.sleep(.25)
            if os.getppid()!=parent:os.kill(os.getpid(),signal.SIGTERM);return
    threading.Thread(target=watch_parent,daemon=True).start()
    if args.cleanup_slot is not None and not 1<=args.cleanup_slot<=8:raise SystemExit('invalid slot')
    if args.cleanup or args.cleanup_slot is not None:cleanup(args.cleanup_slot,args.cleanup_run);return
    if not args.core or lab.digest(pathlib.Path(args.core).read_bytes())!=CORE_SHA:raise SystemExit('pinned core mismatch')
    if lab.digest(pathlib.Path(args.client_core).read_bytes())!=CLIENT_CORE_SHA:raise SystemExit('pinned client core mismatch')
    if args.self_test:
        runid=uuid.uuid4().hex[:6]
        try:
            with lab.Sandbox(args.core,runid,8192,prefix=PREFIX) as box:
                for host in ['127.0.0.1','169.254.169.254',box.gateway,'1.1.1.1']:
                    r=box.net(['/usr/bin/python3','-c','import socket,sys;s=socket.socket();s.settimeout(.3);print(s.connect_ex((sys.argv[1],443)))',host],timeout=3)
                    if r.stdout.strip()==b'0':raise RuntimeError('egress_control_failed')
                box.policy({'public':{'ips':['1.1.1.1','1.0.0.1'],'port':443}})
                good=False
                for host in ['1.1.1.1','1.0.0.1']:
                    try:
                        r=box.net(['/usr/bin/curl','--noproxy','*','--max-time','10','--silent','--output','/dev/null','--write-out','%{http_code}','https://'+host+'/cdn-cgi/trace'],timeout=12)
                        if r.stdout.strip()==b'200':good=True;break
                    except subprocess.SubprocessError:pass
                if not good:raise RuntimeError('positive_control_failed')
                box.policy({});p=box.launch(b'{"inbounds":[],"outbounds":[{"protocol":"blackhole","settings":{}}],"routing":{}}',True);p.wait(timeout=10)
                if p.returncode!=0:raise RuntimeError('core_launch_failed')
                if validate_client(box,b'{"inbounds":[],"outbounds":[{"protocol":"blackhole","settings":{}}],"routing":{}}',args.client_core) is not None:raise RuntimeError('client_core_launch_failed')
                box.stop();marker=box.work/'host-only.pem';marker.write_text('synthetic host canary')
                denied={'inbounds':[],'outbounds':[{'protocol':'trojan','settings':{'address':'1.1.1.1','port':443,'password':'synthetic'},'streamSettings':{'security':'tls','tlsSettings':{'certificates':[{'certificateFile':str(marker),'usage':'verify'}]}}}],'routing':{}}
                p=box.launch(json.dumps(denied).encode(),True);p.wait(timeout=10)
                if p.returncode==0 or b'no such file' not in box.logs():raise RuntimeError('filesystem_control_failed')
                box.stop()
        finally:cleanup(0,runid)
        return
    if args.slot is None or not 1<=args.slot<=8:raise SystemExit('invalid slot')
    request=json.load(sys.stdin);plan=request['plan'];timeout=request['timeout']
    if not isinstance(timeout,int) or not 1<=timeout<=120 or not 1<=len(plan['candidates'])<=12:raise SystemExit('invalid request')
    runid=args.run_id or uuid.uuid4().hex[:6]
    rows=[];best=None
    try:
        with lab.Sandbox(args.core,runid,8192+args.slot,prefix=PREFIX) as box:
            for candidate in plan['candidates']:
                client_failure={'code':'not_run'}
                try:
                    data=base64.b64decode(candidate['config'],validate=True)
                    if len(data)>2*1024*1024 or lab.digest(data)!=candidate['sha256']:raise ValueError('invalid candidate')
                    client_failure=validate_client(box,data,args.client_core)
                    if client_failure is None:
                        for variant in app.variants(data):
                            client_failure=validate_client(box,variant,args.client_core)
                            if client_failure is not None:
                                client_failure["code"]="app_transform_core_rejected"
                                break
                    if client_failure is not None:
                        r=client_failure
                    elif core_only:
                        box.policy({})
                        process=box.launch(data,True);process.wait(timeout=15)
                        passed=process.returncode==0
                        r={'core':'passed' if passed else 'failed','connection':'not_run','code':'smart_core_validated' if passed else 'smart_core_rejected','runtime_sha256':lab.digest(data),'endpoints_passed':0}
                        box.stop()
                    else:
                        # Both official cores must build the exact output. Real traffic
                        # runs on the core actually bundled with the target Android app.
                        latest_failure=validate_client(box,data,args.core)
                        if latest_failure is not None:
                            r=latest_failure
                        else:
                            r=run_target(box,candidate,timeout,args.client_core)
                except Exception:r={'core':'blocked','connection':'not_run','code':'probe_infrastructure_error','runtime_sha256':'','endpoints_passed':0}
                r.setdefault('evidence',{})['bundled_core']='passed' if client_failure is None else 'failed'
                r['evidence']['target_core']=r['core']
                r['evidence']['app_import_contract']='passed' if client_failure is None else 'failed'
                r['evidence']['android']='not_run'
                r.update(sha256=candidate['sha256'],rule=candidate['rule']);rows.append(r)
                score=attempt_score(r,candidate)
                if best is None or score>best_score:best=r;best_score=score
                if r['connection']=='passed' and r.get('code')=='https_payload_verified':break
        print(json.dumps(best|{'attempts':rows}))
    finally:cleanup(args.slot,runid)

if __name__=='__main__':main()
