import asyncio,time,statistics
import httpx
BASE='http://127.0.0.1:4040'
async def hit(client,path):
 t=time.perf_counter()
 try:
  r=await client.get(BASE+path); return r.status_code,(time.perf_counter()-t)*1000
 except Exception:return 0,(time.perf_counter()-t)*1000
async def main():
 async with httpx.AsyncClient(timeout=4) as c:
  tasks=[]
  for _ in range(120): tasks += [asyncio.create_task(hit(c,'/api/stats')),asyncio.create_task(hit(c,'/health/ready'))]
  a=await asyncio.gather(*tasks); ok=sum(x[0]==200 for x in a); ms=[x[1] for x in a]; print('requests',len(a),'ok',ok,'failed',len(a)-ok,'p50_ms',round(statistics.median(ms),2),'p95_ms',round(sorted(ms)[int(len(ms)*.95)-1],2),'max_ms',round(max(ms),2))
asyncio.run(main())
