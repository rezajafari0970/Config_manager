#!/usr/bin/env python3
"""Real-device core benchmark. Build/start/traffic/app-functionality stay separate.
Runs beside an existing VPN on allocated loopback ports; never edits it.
"""
import argparse,concurrent.futures,hashlib,importlib.util,json,os,pathlib,subprocess,sys,time,urllib.request
from benchmarks.core import Session,encoded,digest
from benchmarks.http import site,transfers
from benchmarks.quic import check as curl_http3
from benchmarks.quality import BENCHMARK_VERSION
VERSION=BENCHMARK_VERSION
HERE=pathlib.Path(__file__).resolve().parent

def save(path,obj):
    p=pathlib.Path(path);p.parent.mkdir(parents=True,exist_ok=True);tmp=p.with_suffix(p.suffix+'.tmp');tmp.write_text(json.dumps(obj,indent=2));tmp.replace(p)
def transport(port,timeout):
    p=subprocess.run([sys.executable,str(HERE/'transport-checks.py'),'--socks-port',str(port),'--timeout',str(timeout)],capture_output=True,timeout=timeout*3+5)
    try:return json.loads(p.stdout)
    except ValueError:return {'status':'infrastructure_error','exit_code':p.returncode}
def http3(port,python,timeout):
    if not python:return curl_http3(port,timeout)
    p=subprocess.run([python,str(HERE/'http3-check.py'),str(timeout),'--socks-port',str(port)],capture_output=True,timeout=timeout+4)
    try:return json.loads(p.stdout)
    except ValueError:return {'status':'infrastructure_error','exit_code':p.returncode}
def run_one(args,pair):
    index,row=pair;c=row['config'];key=digest(encoded(c));work=pathlib.Path(args.output)/('case-%03d-'%index+key[:12]);result={'benchmark_version':VERSION,'index':index,'family':row.get('family','smart'),'label':row.get('label','baseline'),'member_count':row.get('member_count',1),'profile':row.get('profile','simple'),'config_sha256':key,'core_sha256':args.core_sha,'network_label':args.network_label,'started':int(time.time()),'build':'not_run','startup':'not_run','connection':'not_run','app_functionality':'not_run'}
    if args.resume:
        saved=work/'result.json'
        if saved.exists():
            prior=json.loads(saved.read_text())
            if prior.get('benchmark_version')==VERSION and prior.get('config_sha256')==key and prior.get('core_sha256')==args.core_sha and prior.get('network_label')==args.network_label and prior.get('benchmark_stage')==args.stage and prior.get('observation_seconds')==args.observe_seconds and prior.get('finished'):
                prior['historical_reused']=True
                print(json.dumps({'index':index,'status':'reused_completed_receipt','finished':prior['finished']}),flush=True)
                return prior
            raise ValueError('resume_context_mismatch_use_fresh_output')
        if work.exists():
            archive=work.with_name(work.name+'.interrupted-'+str(time.time_ns()));work.rename(archive)
            result['interrupted_attempt']=archive.name
    elif work.exists():raise ValueError('output_case_exists_use_resume_or_fresh_output')
    result['benchmark_stage']=args.stage;result['observation_seconds']=args.observe_seconds
    try:
        battery=subprocess.run(['termux-battery-status'],capture_output=True,timeout=3)
        result['battery_before']=json.loads(battery.stdout)
        if result['battery_before'].get('temperature',0)>=43:
            result['environment']='thermal_limit';save(work/'result.json',result);return result
    except (OSError,ValueError,subprocess.SubprocessError):result['battery_before']={'status':'unavailable'}
    session=Session(args.core,c,work)
    try:
        with session:
            result.update(build=session.build['status'],startup='passed',startup_ms=session.startup_ms,runtime_sha256=digest(session.runtime_bytes),runtime_substitutions=['loopback listener ports and matching relay endpoints','metrics listener port if present'])
            start=time.monotonic();initial=[]
            while time.monotonic()-start<8:
                check=site(session.port,'google',min(2,args.timeout));initial.append(check)
                if check['status']=='passed':break
                time.sleep(.15)
            result['first_verified_https_ms']=round((time.monotonic()-session.started)*1000,2) if initial[-1]['status']=='passed' else None
            result['startup_attempts']=initial
            if args.observe_seconds:
                end=time.monotonic()+args.observe_seconds;observations=[]
                while time.monotonic()<end:
                    sample={'elapsed_ms':round((time.monotonic()-session.started)*1000,2),'https':site(session.port,'google',min(args.timeout,3))}
                    if session.metrics_port:
                        try:
                            opener=urllib.request.build_opener(urllib.request.ProxyHandler({}))
                            with opener.open('http://127.0.0.1:'+str(session.metrics_port)+'/debug/vars',timeout=1) as r:metrics=json.load(r)
                            nodes=metrics.get('observatory',{});sample['observed_nodes']=len(nodes);sample['alive_nodes']=sum(bool(n.get('alive')) for n in nodes.values());sample['health']=nodes
                        except Exception:sample['metrics']='unavailable'
                    observations.append(sample);time.sleep(max(0,min(4,end-time.monotonic())))
                result['observations']=observations
            names=['google','instagram','telegram']
            if args.stage=='full':names+=['telegram_web','youtube','youtube_image','play']
            with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:checks=list(pool.map(lambda n:site(session.port,n,args.timeout),names))
            result['https']=checks;result['connection']='passed' if all(x['status']=='passed' for x in checks[:3]) else 'partial' if any(x['transport_tls']=='verified' for x in checks) else 'failed'
            result['transfers']=transfers(session.port,work,args.timeout+2)
            if args.stage=='full':
                result['transport']=transport(session.port,min(args.timeout,5));result['http3']=http3(session.port,args.http3_python,min(args.timeout+2,8))
            result['resources']=session.resources()
            if session.process.poll() is not None:result['runtime_failure']='core_exited_during_measurement'
    except Exception as e:
        if hasattr(session,'build'):result['build']=session.build['status']
        result['infrastructure_or_startup_error']={'type':type(e).__name__,'code':str(e) if str(e).startswith(('core_','local_','unsupported_')) else 'benchmark_exception'}
    result['finished']=int(time.time());save(work/'result.json',result)
    print(json.dumps({k:result.get(k) for k in ['index','family','label','build','connection','first_verified_https_ms','infrastructure_or_startup_error']}|{'both_transfers':result.get('transfers',{}).get('both_passed')}),flush=True)
    return result

def main():
    p=argparse.ArgumentParser();p.add_argument('--corpus',required=True);p.add_argument('--core',required=True);p.add_argument('--core-sha',required=True);p.add_argument('--output',required=True);p.add_argument('--stage',choices=['screen','full'],default='screen');p.add_argument('--workers',type=int,choices=[1,2],default=1);p.add_argument('--timeout',type=float,default=5);p.add_argument('--network-label',default='unverified');p.add_argument('--resume',action='store_true');p.add_argument('--http3-python');p.add_argument('--observe-seconds',type=int,choices=[0,20,55],default=0);a=p.parse_args()
    if not 1<=a.timeout<=15:raise SystemExit('timeout out of bounds')
    os.umask(0o077);core=pathlib.Path(a.core).resolve();assert digest(core.read_bytes())==a.core_sha,'core checksum mismatch';a.core=str(core)
    corpus=json.loads(pathlib.Path(a.corpus).read_text());rows=corpus['rows'];assert 0<len(rows)<=120
    report={'version':VERSION,'started':int(time.time()),'stage':a.stage,'network_label':a.network_label,'core_sha256':a.core_sha,'limitations':['Termux core runtime, not Android VpnService or APK import','HTTP probes are not Feed/Reels/Stories or messaging tests','Listener ports are isolated; exact original bytes separately core-validated','No IP packet-loss or MTU claim from request failures','Battery discharge unmeasured when charging'],'results':[]}
    with concurrent.futures.ThreadPoolExecutor(max_workers=a.workers) as pool:
        for row in pool.map(lambda pair:run_one(a,pair),enumerate(rows)):
            report['results'].append(row);save(pathlib.Path(a.output)/'report.json',report)
    report['finished']=int(time.time());save(pathlib.Path(a.output)/'report.json',report)
if __name__=='__main__':main()
