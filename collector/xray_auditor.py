import os,time,tempfile,subprocess
from .db import connect
from .repair import repair
from .resources import sample
STATE={'running':False,'last_run':0,'tested':0,'ok':0,'failed':0,'categories':{},'duration_ms':0}
def category(s):
 if 'FreedomConfig' in s:return 'FREEDOM_SETTINGS_ARRAY'
 if "LengthMin can't be 0" in s:return 'FRAGMENT_LENGTHMIN_ZERO'
 if 'empty "password"' in s:return 'REALITY_MISSING_REQUIRED_DATA'
 if 'HTTPServerConfig' in s:return 'HTTP_SETTINGS_ARRAY'
 if 'SocketConfig' in s:return 'SOCKOPT_ARRAY'
 if 'REALITYConfig' in s:return 'REALITY_SETTINGS_ARRAY'
 if 'invalid DNS hosts' in s:return 'DNS_HOSTS'
 if 'Config.stats' in s:return 'STATS_ARRAY'
 return 'OTHER'
def run(limit=0,offset=0):
 if STATE['running']:return dict(STATE)
 STATE['running']=True;t=time.monotonic();C={};tested=ok=failed=0
 try:
  c=connect();rows=c.execute("SELECT id,kind,raw FROM configs WHERE kind='json-xray' ORDER BY id").fetchall();c.close()
  if offset: rows=rows[offset:]
  if limit:rows=rows[:limit]
  for row in rows:
   while sample()['level'] in ('busy','critical'):time.sleep(.5)
   x=repair(row['kind'],row['raw']);data=x['raw'] if x else row['raw'];fn=None
   try:
    with tempfile.NamedTemporaryFile('w',suffix='.json',delete=False) as f:f.write(data);fn=f.name
    p=subprocess.run(['/usr/bin/timeout','2s','/usr/local/bin/xray','run','-test','-c',fn],stdout=subprocess.PIPE,stderr=subprocess.STDOUT,text=True,timeout=2.5);tested+=1
    if p.returncode==0:ok+=1
    else:failed+=1;k=category(p.stdout);C[k]=C.get(k,0)+1
   except subprocess.TimeoutExpired:failed+=1;C['TIMEOUT']=C.get('TIMEOUT',0)+1
   finally:
    if fn:
     try:os.unlink(fn)
     except:pass
    time.sleep(.03)
  STATE.update(last_run=time.time(),tested=tested,ok=ok,failed=failed,categories=C,duration_ms=int((time.monotonic()-t)*1000))
 finally:STATE['running']=False
 return dict(STATE)
