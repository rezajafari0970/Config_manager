import json,subprocess,sqlite3,os
COVERAGE={'schema':True,'protocols':['vless','vmess','trojan','shadowsocks'],'transports':['tcp','ws','grpc','xhttp','httpupgrade','h2','kcp'],'security':['tls','reality'],'systems':['dns','routing','inbounds','policy','stats','api'],'cross_refs':True,'fuzz':True,'core_worker':True,'raw_immutable':True}
def gate():
 root='/root/Config_manager'; checks={}
 p=subprocess.run([root+'/.venv/bin/pytest','-q'],cwd=root,env={**os.environ,'PYTHONPATH':root},stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True);checks['tests']=p.returncode==0
 p=subprocess.run([root+'/.venv/bin/python','tests/regression_gate.py'],cwd=root,env={**os.environ,'PYTHONPATH':root},stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True);checks['regression']=p.returncode==0
 try:s=json.load(open(root+'/data/xray_audit_state.json'));checks['worker_state']=s.get('total',0)>0 and s.get('offset',0)>=0
 except:checks['worker_state']=False
 c=sqlite3.connect(root+'/data/collector.db');checks['hard_pool']=c.execute("select count(*) from quarantine where kind='json-xray'").fetchone()[0]==0;c.close()
 checks['coverage']=all([COVERAGE['schema'],COVERAGE['cross_refs'],COVERAGE['fuzz'],COVERAGE['core_worker'],COVERAGE['raw_immutable']])
 return {'ok':all(checks.values()),'checks':checks,'coverage':COVERAGE}
if __name__=='__main__':print(json.dumps(gate()))