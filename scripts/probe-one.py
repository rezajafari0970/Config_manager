#!/usr/bin/env python3
"""Trusted one-job adapter. Payload only on stdin; receipts never contain secrets."""
import argparse, threading, time, signal, base64, contextlib, importlib.util, json, os, pathlib, re, shutil, subprocess, sys, uuid

HERE=pathlib.Path(__file__).resolve().parent
spec=importlib.util.spec_from_file_location('probe_sandbox',HERE/'live-probe.py')
lab=importlib.util.module_from_spec(spec);spec.loader.exec_module(lab)
CORE_SHA='64d46afb80adea1bf97a0d467e83f4a9ac1ebd0995891e84bca3f1a1d1affb1d'
OWN=re.compile(r'^crw[0-9a-f]{6}20(?:0[0-8])$')

def cleanup(slot=None,runid=None):
    names=set()
    for folder in ['/run/netns','/sys/fs/cgroup','/dev/shm']:
        for p in pathlib.Path(folder).glob('crw*'):
            name=p.name.split('-')[0]
            if OWN.fullmatch(name) and (slot is None or name.endswith(format(8192+slot,'x'))) and (runid is None or name.startswith('crw'+runid)):names.add(name)
    # Covers crashes between NAT-table creation and namespace registration.
    r=subprocess.run([lab.NFT,'-j','list','tables'],capture_output=True)
    if r.returncode==0:
        for item in json.loads(r.stdout).get('nftables',[]):
            t=item.get('table',{});name=t.get('name','')
            if t.get('family')=='ip' and OWN.fullmatch(name) and (slot is None or name.endswith(format(8192+slot,'x'))) and (runid is None or name.startswith('crw'+runid)):names.add(name)
    for name in names:
        cg=pathlib.Path('/sys/fs/cgroup')/name
        with contextlib.suppress(OSError):(cg/'cgroup.kill').write_text('1')
        for args in [[lab.NFT,'delete','table','ip',name],[lab.IP,'link','del',name+'h'],[lab.IP,'netns','del',name]]:subprocess.run(args,capture_output=True,timeout=5)
        with contextlib.suppress(OSError):cg.rmdir()
        for p in pathlib.Path('/dev/shm').glob(name+'-*'):
            if p.is_dir() and not p.is_symlink():shutil.rmtree(p)

def main():
    a=argparse.ArgumentParser();a.add_argument('--core');a.add_argument('--slot',type=int);a.add_argument('--cleanup',action='store_true');a.add_argument('--cleanup-slot',type=int);a.add_argument('--self-test',action='store_true');a.add_argument('--run-id');a.add_argument('--cleanup-run');args=a.parse_args()
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
    if args.self_test:
        runid=uuid.uuid4().hex[:6]
        try:
            with lab.Sandbox(args.core,runid,8192,prefix='crw') as box:
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
    rows=[];best=None;rank={'not_run':0,'failed':1,'partial':2,'passed':3}
    try:
        with lab.Sandbox(args.core,runid,8192+args.slot,prefix='crw') as box:
            for candidate in plan['candidates']:
                try:r=lab.test_candidate(box,candidate,timeout)
                except Exception:r={'core':'blocked','connection':'not_run','code':'probe_infrastructure_error','runtime_sha256':'','endpoints_passed':0}
                r.update(sha256=candidate['sha256'],rule=candidate['rule']);rows.append(r)
                eligible=r['core']=='passed' and (not candidate.get('requires_connection') or r['connection']=='passed')
                score=(eligible,rank[r['connection']],r['core']=='passed')
                if best is None or score>best_score:best=r;best_score=score
                if r['connection']=='passed':break
        print(json.dumps(best|{'attempts':rows}))
    finally:cleanup(args.slot,runid)

if __name__=='__main__':main()
