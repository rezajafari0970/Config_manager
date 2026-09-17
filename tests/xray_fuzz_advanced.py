import copy,json,sqlite3,tempfile,subprocess,os
from collector.validator import validate
c=sqlite3.connect('data/collector.db');raw=c.execute("select raw from configs where kind='json-xray' and raw like '%\"vnext\"%' limit 1").fetchone()[0];base=json.loads(raw)
def core(o):
 f=tempfile.NamedTemporaryFile('w',suffix='.json',delete=False);json.dump(o,f);f.close()
 try:p=subprocess.run(['/usr/bin/timeout','2s','/usr/local/bin/xray','run','-test','-c',f.name],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,timeout=2.5);return p.returncode==0
 finally:os.unlink(f.name)
def setmut(name):
 x=copy.deepcopy(base);o=x['outbounds'][0];s=o['settings'];v=s['vnext'][0]
 if name=='address_empty':v['address']=''
 elif name=='port_zero':v['port']=0
 elif name=='port_high':v['port']=70000
 elif name=='users_empty':v['users']=[]
 elif name=='uuid_bad':v['users'][0]['id']='not-a-uuid'
 elif name=='routing_missing':x['routing']={'rules':[{'outboundTag':'gone'}]}
 elif name=='tls_array':o.setdefault('streamSettings',{})['security']='tls';o['streamSettings']['tlsSettings']=[]
 elif name=='ws_array':o.setdefault('streamSettings',{})['network']='ws';o['streamSettings']['wsSettings']=[]
 return x
names=['address_empty','port_zero','port_high','users_empty','uuid_bad','routing_missing','tls_array','ws_array']
for n in names:
 x=setmut(n);issues=validate('json-xray',json.dumps(x));print(n,'validator',bool(issues),[i['code'] for i in issues],'core_ok',core(x))