import threading,time
L=threading.Lock();A={'fast':0,'normal':0,'slow':0};LIMIT={'fast':5,'normal':2,'slow':1}
def enter(lane):
 lane=lane if lane in A else 'fast'
 while True:
  with L:
   if A[lane]<LIMIT[lane]:A[lane]+=1;return lane
  time.sleep(.005)
def leave(lane):
 with L:A[lane]=max(0,A[lane]-1)
def snapshot():
 with L:return {'active':dict(A),'limits':dict(LIMIT)}