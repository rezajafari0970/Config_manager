import threading,time
from .db import connect
L=threading.Lock();A={'fast':0,'normal':0,'slow':0};TOTAL=8;RESERVE={'fast':2,'normal':1,'slow':1}
def queue_counts():
 c=connect();r=dict(c.execute("select coalesce(lane,'fast'),count(*) from test_candidates where stage in ('queued','retry_wait') group by coalesce(lane,'fast')").fetchall());c.close();return r
def enter(lane):
 lane=lane if lane in A else 'fast'
 while True:
  with L:
   used=sum(A.values());q=queue_counts();reserved_other=sum(max(0,RESERVE[k]-A[k]) for k in A if k!=lane and q.get(k,0)>0)
   if used<TOTAL and (A[lane]<RESERVE[lane] or TOTAL-used>reserved_other):A[lane]+=1;return lane
  time.sleep(.003)
def leave(lane):
 with L:A[lane]=max(0,A[lane]-1)
def snapshot():
 with L:return {'active':dict(A),'total':TOTAL,'reserve':dict(RESERVE),'free':TOTAL-sum(A.values())}