#!/usr/bin/env python3
"""Service-aware project Smart measurements, private snapshots and owned runtime.
No production subscription is changed. No raw config is repaired in this layer.
"""
import argparse, concurrent.futures, hashlib, json, os, pathlib, signal, subprocess, sys, time
from benchmarks.core import Session, encoded, digest, free_port
from servicequality.catalog import SERVICES, UNMEASURED, VERSION as DRIVERS
from servicequality.policy import VERSION, assess, choose
from servicequality.probes import measure, save
from servicequality.profile import adapt

def load(path):return json.loads(pathlib.Path(path).read_text())
def corpus(path):
    rows=load(path)['rows']
    if not 1<=len(rows)<=100:raise ValueError('corpus_limit')
    hashes=[digest(encoded(r['config'])) for r in rows]
    if len(set(hashes))!=len(hashes):raise ValueError('duplicate_members')
    return rows
def context(a):return {'core_sha256':digest(pathlib.Path(a.core).read_bytes()),'network_label':a.network,'network_epoch':a.epoch}
def history(directory):
    found=[]
    for p in sorted(pathlib.Path(directory).glob('*/receipt.json')):
        try:found.append(load(p))
        except (OSError,ValueError):pass
    return found
def decisions(rows,receipts,ctx,previous=None,now=None,bootstrap=None):
    now=time.time() if now is None else now;start=time.perf_counter_ns()
    result={};scores={};scope=digest(encoded(ctx))
    for name in SERVICES:
        scores[name]={}
        for i,r in enumerate(rows):
            tag='proxy-'+str(i+1).zfill(max(2,len(str(len(rows)))))
            h=digest(encoded(r['config']))
            scores[name][tag]=assess(receipts,name,now,dict(ctx,config_sha256=h))
            if name=='default' and bootstrap is not None and tag not in bootstrap:
                scores[name][tag]={'eligible':False,'reason':'not_independent_ip_bootstrap'}
        result[name]=choose(scores[name],(previous or {}).get(name),now,scope)
    return {'version':VERSION,'context':ctx,'time':now,'decisions':result,'scores':scores,
            'decision_ms':(time.perf_counter_ns()-start)/1e6,'unmeasured':UNMEASURED}
def compose(rows,compiler):
    import base64
    # Mapping is only defined for one entry per Simple input; balanced/custom
    # inputs with multiple entries need a separate identity-preserving driver.
    for row in rows:
        config=row['config'];catch=[r for r in config.get('routing',{}).get('rules',[]) if set(r)=={'type','network','outboundTag'} and r['network']=='tcp,udp']
        if len(catch)!=1:raise ValueError('requires_single_entry_simple_input')
    p=subprocess.run([compiler,'--smart'],input=encoded([r['config'] for r in rows]),capture_output=True,check=True,timeout=30)
    plan=json.loads(p.stdout)
    if plan.get('code')!='prepared':raise ValueError('project_composition_rejected')
    c=plan['candidates'][0];body=base64.b64decode(c['config'],validate=True)
    if digest(body)!=c['sha256']:raise ValueError('compiler_hash_mismatch')
    return json.loads(body)
def apply(core,port,service,target):
    # Only the caller's owned runtime is managed. Never accept a remote API host.
    cmd=[core,'api','bo','-s=127.0.0.1:'+str(port),'-t=1','-b','sq-'+service,target]
    subprocess.run(cmd,capture_output=True,check=True,timeout=3)
    reply=subprocess.run([core,'api','bi','--json','-s=127.0.0.1:'+str(port),'-t=1','sq-'+service],capture_output=True,check=True,timeout=3)
    if json.loads(reply.stdout).get('balancer',{}).get('override',{}).get('target')!=target:raise RuntimeError('override_readback_mismatch')

def main():
    p=argparse.ArgumentParser();p.add_argument('action',choices=['measure','plan','compose','export','run'])
    p.add_argument('--corpus',required=True);p.add_argument('--core',required=True);p.add_argument('--network',required=True);p.add_argument('--epoch',required=True)
    p.add_argument('--output',required=True);p.add_argument('--receipts');p.add_argument('--compiler');p.add_argument('--base')
    p.add_argument('--rounds',type=int,default=2);p.add_argument('--duration',type=int,default=600);p.add_argument('--interval',type=int,default=120)
    p.add_argument('--network-token-file',help='External network monitor epoch file; a change stops this owned runtime')
    p.add_argument('--max-mib',type=int,default=64,help='HTTP body budget per bounded run; at most one bounded probe overshoot')
    p.add_argument('--resume',action='store_true',help='Reuse only completed matching measurement receipts, retaining their original ages')
    a=p.parse_args();os.umask(0o077)
    if a.network.lower() in ('unknown','unverified','unspecified') or not a.epoch:raise SystemExit('explicit_network_scope_required')
    if not 2<=a.rounds<=5 or not 30<=a.duration<=3600 or not 30<=a.interval<=900:raise SystemExit('resource_bounds')
    if not 16<=a.max_mib<=512:raise SystemExit('traffic_budget_bounds')
    if a.resume and a.action!='measure':raise SystemExit('resume_only_measurement')
    rows=corpus(a.corpus);ctx=context(a);out=pathlib.Path(a.output)
    if a.resume:
        if load(out/'context.json')!=ctx:raise SystemExit('resume_context_mismatch')
    else:out.mkdir(parents=True,exist_ok=False,mode=0o700)
    token=pathlib.Path(a.network_token_file).read_bytes() if a.network_token_file else None
    save(out/'context.json',ctx)
    if a.action=='measure':
        used=0
        for n in range(a.rounds):
            for i,r in enumerate(rows):
                case=out/('round-%d-member-%03d'%(n,i));saved=case/'receipt.json'
                if a.resume and saved.exists():
                    prior=load(saved)
                    if any(prior.get(k)!=v for k,v in ctx.items()) or prior.get('version')!=VERSION or prior.get('drivers')!=DRIVERS or prior.get('config_sha256')!=digest(encoded(r['config'])):raise SystemExit('resume_receipt_mismatch')
                    if prior.get('finished'):
                        used+=prior.get('observed_http_bytes',0);print(json.dumps({'round':n,'member':i,'historical_reused':True,'finished':prior['finished']}),flush=True);continue
                if a.resume and case.exists():case.rename(out/(case.name+'.interrupted-'+str(time.time_ns())))
                if used>=a.max_mib*1048576:
                    save(out/'budget-stop.json',{'observed_http_bytes':used,'unmeasured_members_remain':True});return
                receipt=measure(a.core,r['config'],case,ctx)
                used+=receipt['observed_http_bytes']
                print(json.dumps({'round':n,'member':i,'runtime':receipt['runtime'],'payload':receipt.get('payload',{}).get('complete')}),flush=True)
        return
    receipts=history(a.receipts) if a.receipts else []
    report=decisions(rows,receipts,ctx);save(out/'decisions.json',report)
    if a.action=='plan':return
    if a.base:
        manifest=load(a.base)
        if manifest.get('corpus_sha256')!=digest(encoded(rows)):raise ValueError('base_corpus_mismatch')
        base=manifest['config']
    else:base=compose(rows,a.compiler)
    save(out/'base.json',base)
    save(out/'manifest.json',{'corpus_sha256':digest(encoded(rows)),'config':base})
    bootstrap=set(next(b['selector'] for b in base['routing']['balancers'] if b['tag']=='bootstrap-ip-balancer'))
    report=decisions(rows,receipts,ctx,bootstrap=bootstrap);save(out/'decisions.json',report)
    if a.action=='compose':return
    if a.action=='export':
        choices={k:v['target'] for k,v in report['decisions'].items()}
        c=adapt(base,choices=choices);save(out/'candidate.json',c)
        save(out/'status.json',{'runtime':'requires_final_exact_config_test','apk':'not_run','automatic_reassessment':False})
        return
    controlled=adapt(base,api_port=free_port())
    # Session's parent-death guard owns only this isolated core. No orphaned
    # stale balancer overrides survive a controller crash or normal exit.
    class Stop(Exception):pass
    def stop(*_):raise Stop()
    signal.signal(signal.SIGTERM,stop);signal.signal(signal.SIGINT,stop)
    start=time.monotonic();state={};applied={};all_receipts=list(receipts);jobs=iter(range(len(rows)));cycle=0;next_wave=0;used=0
    pool=concurrent.futures.ThreadPoolExecutor(max_workers=1)
    try:
        with Session(a.core,controlled,out/'runtime') as s:
            api=s.port_map[controlled['inbounds'][-1]['port']]
            save(out/'runtime-addresses.json',{'socks':s.port,'api':api,'pid':s.process.pid})
            future=None
            while time.monotonic()-start<a.duration:
                if token is not None and pathlib.Path(a.network_token_file).read_bytes()!=token:raise RuntimeError('network_changed_reassessment_required')
                if s.process.poll() is not None:raise RuntimeError('owned_core_exited')
                if future and future.done():
                    fresh=future.result();all_receipts.append(fresh);used+=fresh['observed_http_bytes'];future=None
                if future is None and time.monotonic()>=next_wave and used<a.max_mib*1048576:
                    i=next(jobs,None)
                    if i is None:cycle+=1;jobs=iter(range(len(rows)));next_wave=time.monotonic()+a.interval
                    else:future=pool.submit(measure,a.core,rows[i]['config'],out/('cycle-%d-member-%03d'%(cycle,i)),ctx)
                report=decisions(rows,all_receipts,ctx,state,bootstrap=bootstrap);state=report['decisions']
                for name,d in state.items():
                    if applied.get(name)!=d['target']:
                        before=time.monotonic();apply(a.core,api,name,d['target']);applied[name]=d['target']
                        with (out/'switches.jsonl').open('a') as events:events.write(json.dumps({'at':time.time(),'service':name,'target':d['target'],'reason':d['reason'],'api_verified_ms':(time.monotonic()-before)*1000})+'\n')
                save(out/'decisions.json',report);time.sleep(1)
    except Stop:pass
    finally:
        save(out/'stopped.json',{'time':time.time(),'owned_runtime_stopped':True})
        pool.shutdown(wait=True,cancel_futures=True)
if __name__=='__main__':main()
