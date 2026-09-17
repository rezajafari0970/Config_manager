import json,os,time
from .db import connect
from .xray_auditor import run
STATE_FILE='/root/Config_manager/data/xray_audit_state.json'
def load():
 try:return json.load(open(STATE_FILE))
 except:return {'offset':0,'batch':20,'round':0,'last':{},'updated':0}
def save(s):
 t=STATE_FILE+'.tmp';open(t,'w').write(json.dumps(s));os.replace(t,STATE_FILE)
def step():
 s=load();c=connect();total=c.execute("select count(*) from configs where kind='json-xray'").fetchone()[0];c.close()
 if s['offset']>=total:s['offset']=0;s['round']+=1
 r=run(s['batch'],s['offset']);s['last']=r;s['offset']+=r.get('tested',0) or s['batch'];s['total']=total;s['updated']=time.time();save(s);return s
if __name__=='__main__': print(json.dumps(step()))
