import json,os,time
from .db import connect
from .singbox_core import check
STATE='/root/Config_manager/data/singbox_audit_state.json'
def load():
 try:return json.load(open(STATE))
 except:return {'offset':0,'batch':20,'round':0,'total':0,'last':{},'updated':0}
def save(s):
 t=STATE+'.tmp';open(t,'w').write(json.dumps(s));os.replace(t,STATE)
def step():
 s=load();c=connect();rows=c.execute("select id,raw from configs where kind='json-singbox' order by id").fetchall();c.close();total=len(rows);s['total']=total
 if not total:s.update(offset=0,last={'tested':0,'ok':0,'failed':0,'idle':True},updated=time.time());save(s);return s
 if s['offset']>=total:s['offset']=0;s['round']+=1
 part=rows[s['offset']:s['offset']+s['batch']];ok=0;failed=0;errors={}
 for _,raw in part:
  r=check(raw);ok+=int(r['ok']);failed+=int(not r['ok']);k=(r['error'][:120] if r['error'] else 'OK');errors[k]=errors.get(k,0)+int(not r['ok'])
 s['offset']+=len(part);s['last']={'tested':len(part),'ok':ok,'failed':failed,'errors':errors};s['updated']=time.time();save(s);return s
if __name__=='__main__':print(json.dumps(step()))