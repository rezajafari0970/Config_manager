#!/usr/bin/env python3
"""Compile reproducible paired Smart pools; never publishes or mutates inputs."""
import argparse,base64,collections,hashlib,importlib.util,json,os,pathlib,subprocess
from benchmarks.core import encoded,digest
from benchmarks.profiles import apply,PROFILES
spec=importlib.util.spec_from_file_location('families',pathlib.Path(__file__).with_name('template-audit.py'));family=importlib.util.module_from_spec(spec);spec.loader.exec_module(family)
def build(inputs,compiler,priority=None):
    ranking={x['config_sha256']:i for i,x in enumerate(sorted((priority or {}).get('results',[]),key=lambda x:(not x.get('transfers',{}).get('both_passed',False),x.get('connection')!='passed',x.get('first_verified_https_ms') or 1e9)))}
    seen=set();groups=collections.defaultdict(list)
    for c in inputs:
        key=digest(encoded(c))
        if key in seen:continue
        seen.add(key);groups[family.family(c)].append(c)
    for items in groups.values():items.sort(key=lambda c:(ranking.get(digest(encoded(c)),999999),digest(encoded(c))))
    pool=[]
    while any(groups.values()):
        for f in sorted(groups):
            if groups[f]:pool.append(groups[f].pop(0))
    rows=[]
    for size in [5,7,10,15,50,100]:
        if len(pool)<size:continue
        p=subprocess.run([compiler,'--smart'],input=encoded(pool[:size]),capture_output=True,timeout=30,check=True);plan=json.loads(p.stdout)
        if plan['code']!='prepared':raise ValueError('matrix_compose_'+plan['code'])
        config=json.loads(base64.b64decode(plan['candidates'][0]['config'],validate=True))
        members=[digest(encoded(c)) for c in pool[:size]]
        # Same members/order for both variants. Alternating order reduces fixed order bias.
        names=list(PROFILES)
        if size in [7,15,100]:names.reverse()
        for name in names:
            c=apply(config,name);rows.append({'config':c,'family':'smart','label':str(size)+'/'+name,'member_count':size,'member_sha256':members,'profile':name})
    return {'version':'smart-matrix.v1','rows':rows,'pool_size':len(pool),'scope':'single sweep; repeat for confidence, no automatic promotion'}
if __name__=='__main__':
    p=argparse.ArgumentParser();p.add_argument('--input',required=True);p.add_argument('--compiler',required=True);p.add_argument('--priority-report');p.add_argument('--output',required=True);a=p.parse_args();os.umask(0o077)
    inputs=json.loads(pathlib.Path(a.input).read_text());priority=json.loads(pathlib.Path(a.priority_report).read_text()) if a.priority_report else None
    result=build(inputs,a.compiler,priority);pathlib.Path(a.output).write_text(json.dumps(result));print(json.dumps({'rows':len(result['rows']),'pool_size':result['pool_size']}))
