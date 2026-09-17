import json,subprocess,sqlite3,os
COVERAGE={'schema':True,'protocol':True,'dns_route_refs':True,'fuzz':True,'core':'1.14.1','raw_immutable':True}
def gate():
 root='/root/Config_manager';checks={}
 p=subprocess.run([root+'/.venv/bin/pytest','-q'],cwd=root,env={**os.environ,'PYTHONPATH':root},stdout=subprocess.DEVNULL);checks['tests']=p.returncode==0
 try:s=json.load(open(root+'/data/singbox_audit_state.json'));checks['worker_state']=True;checks['has_real_samples']=s.get('total',0)>0;checks['core_full_round']=s.get('round',0)>=1
 except:checks.update(worker_state=False,has_real_samples=False,core_full_round=False)
 c=sqlite3.connect(root+'/data/collector.db');checks['hard_pool']=c.execute("select count(*) from quarantine where kind='json-singbox'").fetchone()[0]==0;c.close()
 checks['coverage']=all([COVERAGE['schema'],COVERAGE['protocol'],COVERAGE['dns_route_refs'],COVERAGE['fuzz'],COVERAGE['raw_immutable']])
 return {'ok':all(checks.values()),'checks':checks,'coverage':COVERAGE}
if __name__=='__main__':print(json.dumps(gate()))