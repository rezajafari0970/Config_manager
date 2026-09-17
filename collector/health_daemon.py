import time,json,signal
from .adaptive_health import run_batch
from .health_settings import load
running=True
def stop(*_):
 global running;running=False
signal.signal(signal.SIGTERM,stop);signal.signal(signal.SIGINT,stop)
def main():
 idle=0
 while running:
  cfg=load()
  if not cfg['enabled']:time.sleep(1);continue
  t=time.time()
  try:r=run_batch();busy=any(not x.get('idle') and x.get('state')!='defer' for x in r.get('results',[]));idle=0 if busy else min(idle+1,10)
  except Exception:idle=min(idle+1,10)
  delay=.005 if idle==0 else min(2,.1*(2**idle));time.sleep(max(0,delay-(time.time()-t)*.01))
if __name__=='__main__':main()