import json,pathlib,time,concurrent.futures
from .shared_batch import run
from .health_settings import load
P=pathlib.Path('/root/Config_manager/data/health_feedback.json')
def settings():
 try:
  d=json.loads(P.read_text());return max(4,min(16,int(d.get('shared_batch_size',8)))),max(1,min(3,int(d.get('shared_lanes',1))))
 except:return 8,1
def main():
 while True:
  if not load()['enabled']:time.sleep(2);continue
  batch,lanes=settings()
  with concurrent.futures.ThreadPoolExecutor(max_workers=lanes) as e:rs=list(e.map(lambda _:run(batch),range(lanes)))
  if all(r.get('idle') for r in rs):time.sleep(.25)
  elif any(r.get('error') for r in rs):time.sleep(.5)
if __name__=='__main__':main()