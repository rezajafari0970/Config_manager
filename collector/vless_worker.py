import json,os,time,tempfile,subprocess
from .db import connect
from .vless_adapter import xray,singbox
from .singbox_core import check
STATE='/root/Config_manager/data/vless_core_state.json'
def load():
 try:return json.load(open(STATE))
 except:return {'offset':0,'batch':10,'round':0,'total':0,'last':{},'updated':0}
def save(s):
 t=STATE+'.tmp';open(t,'w').write(json.dumps(s));os.replace(t,STATE)
def xcheck(raw):
 fn=None
 try:
  f=tempfile.NamedTemporaryFile('w',delete=False,suffix='.json');f.write(xray(raw));f.close();fn=f.name
  p=subprocess.run(['/usr/bin/timeout','2s','/usr/local/bin/xray','run','-test','-c',fn],stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True,timeout=2.5);return p.returncode==0,' '.join(p.stdout.split())[-240:]
 except Exception as e:return False,type(e).__name__
 finally:
  if fn:
   try:os.unlink(fn)
   except:pass
def step():
 s=load();c=connect();rows=c.execute("select id,raw from configs where kind='vless' order by id").fetchall();c.close();s['total']=len(rows)
 if not rows:s.update(offset=0,last={'tested':0},updated=time.time());save(s);return s
 if s['offset']>=len(rows):s['offset']=0;s['round']+=1
 part=rows[s['offset']:s['offset']+s['batch']];C={'both_ok':0,'xray_only':0,'singbox_only':0,'both_fail':0};errors={}
 for r in part:
  xo,xe=xcheck(r['raw']);q=check(singbox(r['raw']));so=q['ok'];k='both_ok' if xo and so else ('xray_only' if xo else ('singbox_only' if so else 'both_fail'));C[k]+=1
  if k!='both_ok':errors[str(r['id'])]={'class':k,'xray':xe,'singbox':q['error'][-240:]}
 s['offset']+=len(part);s['last']={'tested':len(part),**C,'errors':errors};s['updated']=time.time();save(s);return s
if __name__=='__main__':print(json.dumps(step()))