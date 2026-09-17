import json,os,time
from .db import connect
from .ss_classifier import classify
from .ss_recovery import assess
STATE='/root/Config_manager/data/ss_recovery_state.json'
def load():
 try:return json.load(open(STATE))
 except:return {'offset':0,'batch':10,'round':0,'total':0,'last':{},'updated':0}
def save(s):
 t=STATE+'.tmp';open(t,'w').write(json.dumps(s));os.replace(t,STATE)
def step():
 s=load();c=connect();rows=[r for r in c.execute("select id,raw from configs where kind='ss' order by id").fetchall() if classify(r['raw'])['class']=='vless-like-mislabeled'];c.close();s['total']=len(rows)
 if s['offset']>=len(rows) and rows:s['offset']=0;s['round']+=1
 part=rows[s['offset']:s['offset']+s['batch']];ok=0;fail=0;ids=[]
 for r in part:
  a=assess(r['raw']);ok+=int(a['recoverable']);fail+=int(not a['recoverable']);ids.append({'id':r['id'],'recoverable':a['recoverable']})
 s['offset']+=len(part);s['last']={'tested':len(part),'recoverable':ok,'failed':fail,'items':ids};s['updated']=time.time();save(s);return s
if __name__=='__main__':print(json.dumps(step()))