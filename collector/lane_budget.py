import threading,time
from .db import connect
L=threading.Lock();A={'fast':0,'normal':0,'slow':0};CACHE={'at':0,'limits':{'fast':3,'normal':3,'slow':2}}
def limits(total=8):
 now=time.time()
 if now-CACHE['at']<2:return CACHE['limits']
 c=connect();rows=dict(c.execute("select coalesce(lane,'fast'),count(*) from test_candidates where stage in ('queued','retry_wait') group by coalesce(lane,'fast')").fetchall());c.close();n=max(1,sum(rows.values()));raw={k:max(1,round(total*rows.get(k,0)/n)) for k in A};raw['fast']=max(2,raw['fast']);raw['slow']=min(max(1,raw['slow']),max(2,total//2))
 while sum(raw.values())>total:
  k=max(raw,key=lambda x:(raw[x],x!='fast'));raw[k]-=1
 while sum(raw.values())<total:raw[max(rows,key=rows.get) if rows else 'fast']+=1
 CACHE.update(at=now,limits=raw);return raw
def enter(lane):
 lane=lane if lane in A else 'fast'
 while True:
  with L:
   if A[lane]<limits()[lane]:A[lane]+=1;return lane
  time.sleep(.005)
def leave(lane):
 with L:A[lane]=max(0,A[lane]-1)
def snapshot():
 with L:return {'active':dict(A),'limits':dict(limits())}