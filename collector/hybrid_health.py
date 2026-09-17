import time,json,pathlib,concurrent.futures
from .shared_batch import run as shared_run
from .health_worker import step as legacy_step
from .health_settings import load
from .system_pressure import snapshot as pressure
P=pathlib.Path('/root/Config_manager/data/health_feedback.json')
def tune():
 try:d=json.loads(P.read_text());return int(d.get('shared_batch_size',8)),int(d.get('shared_lanes',1)),int(d.get('max_concurrency',4))
 except:return 8,1,4
def cycle():
 batch,lanes,conc=tune();psi=pressure()['psi']['cpu'].get('avg10',0);lanes=1 if psi>45 else min(lanes,2);legacy=max(1,min(2,conc//4));jobs=[]
 with concurrent.futures.ThreadPoolExecutor(max_workers=lanes+legacy) as e:
  for _ in range(lanes):jobs.append(e.submit(shared_run,batch))
  for _ in range(legacy):jobs.append(e.submit(legacy_step))
  return [x.result() for x in jobs]
def main():
 while True:
  if not load()['enabled']:time.sleep(1);continue
  try:r=cycle();busy=any(not x.get('idle') for x in r);time.sleep(.03 if busy else .2)
  except Exception:time.sleep(.3)
if __name__=='__main__':main()