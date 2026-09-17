import time,json,concurrent.futures,pathlib
from .shared_batch import run
from .system_pressure import snapshot as pressure
from .resource_scheduler import snapshot as resources
P=pathlib.Path('/root/Config_manager/data/shared_benchmark.json')
def trial(lanes,seconds=45,batch=8):
 start=time.time();done=0;errors=0
 def lane():
  nonlocal done,errors
  while time.time()-start<seconds:
   r=run(batch)
   if r.get('count'):done+=r['count']
   elif r.get('error'):errors+=1;time.sleep(.2)
   else:time.sleep(.1)
 with concurrent.futures.ThreadPoolExecutor(max_workers=lanes) as e:list(e.map(lambda _:lane(),range(lanes)))
 sec=time.time()-start;sys=resources();psi=pressure();return {'lanes':lanes,'batch':batch,'done':done,'seconds':round(sec,1),'tpm':round(done/sec*60,1),'errors':errors,'load':round(sys['load_ratio'],2),'ram_free':round(sys['mem_free_ratio'],2),'cpu_psi':psi['psi']['cpu'].get('avg10',0)}
def main():
 results=[]
 for lanes in (1,2,3):
  r=trial(lanes);results.append(r)
  if r['load']>.95 or r['cpu_psi']>60:break
 best=max(results,key=lambda x:x['tpm']);out={'target_tpm':200,'results':results,'best':best,'goal_met':best['tpm']>=200};P.write_text(json.dumps(out));return out
if __name__=='__main__':print(json.dumps(main()))