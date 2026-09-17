import json,os,time,tempfile,subprocess
from .db import connect
from .ss_adapter import xray,singbox
from .ss_deep import parse
from .singbox_core import check
STATE='/root/Config_manager/data/ss_core_state.json'
def load():
 try:return json.load(open(STATE))
 except:return {'offset':0,'batch':10,'round':0,'total':0,'last':{},'updated':0}
def save(s):
 t=STATE+'.tmp';open(t,'w').write(json.dumps(s));os.replace(t,STATE)
def xcheck(raw):
 f=tempfile.NamedTemporaryFile('w',delete=False,suffix='.json');f.write(xray(raw));f.close()
 try:p=subprocess.run(['/usr/bin/timeout','2s','/usr/local/bin/xray','run','-test','-c',f.name],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,text=True,timeout=2.5);return p.returncode==0
 except:return False
 finally:os.unlink(f.name)
def step():
 s=load();c=connect();rows=c.execute("select id,raw from configs where kind='ss' order by id").fetchall();c.close();s['total']=len(rows)
 if s['offset']>=len(rows) and rows:s['offset']=0;s['round']+=1
 part=rows[s['offset']:s['offset']+s['batch']];C={'both_ok':0,'xray_only':0,'singbox_only':0,'both_fail':0,'opaque':0}
 for r in part:
  try:parse(r['raw'])
  except:C['opaque']+=1;continue
  xo=xcheck(r['raw']);so=check(singbox(r['raw']))['ok'];k='both_ok' if xo and so else ('xray_only' if xo else ('singbox_only' if so else 'both_fail'));C[k]+=1
 s['offset']+=len(part);s['last']={'tested':len(part),**C};s['updated']=time.time();save(s);return s
if __name__=='__main__':print(json.dumps(step()))