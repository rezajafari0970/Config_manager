import time
from .shared_batch import run
from .health_settings import load
from .system_pressure import snapshot as pressure
def size():
 p=pressure();cpu=p['psi']['cpu'].get('avg10',0);return 4 if cpu>35 else (8 if cpu>20 else 12)
def main():
 while True:
  if not load()['enabled']:time.sleep(2);continue
  r=run(size())
  if r.get('idle'):time.sleep(.2)
  elif r.get('error'):time.sleep(.5)
if __name__=='__main__':main()
