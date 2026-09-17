import asyncio,time,httpx
from .db import connect
from .parser import extract
running=set()
async def fetch_one(row):
 sid=row['id']; running.add(sid); start=time.monotonic(); now=time.time()
 try:
  timeout=httpx.Timeout(8.0,connect=4.0)
  async with httpx.AsyncClient(timeout=timeout,follow_redirects=True,headers={'User-Agent':'ConfigManager/1.0'}) as client: r=await client.get(row['url']); r.raise_for_status(); items=extract(r.text)
  c=connect(); before=c.execute('SELECT COUNT(*) FROM source_configs WHERE source_id=?',(sid,)).fetchone()[0]; valid=len(items)
  for fp,kind,raw in items:
   c.execute('INSERT INTO configs(fingerprint,kind,raw,first_seen,last_seen) VALUES(?,?,?,?,?) ON CONFLICT(fingerprint) DO UPDATE SET last_seen=excluded.last_seen',(fp,kind,raw,now,now)); cid=c.execute('SELECT id FROM configs WHERE fingerprint=?',(fp,)).fetchone()[0]; c.execute('INSERT INTO source_configs(source_id,config_id,last_seen) VALUES(?,?,?) ON CONFLICT(source_id,config_id) DO UPDATE SET last_seen=excluded.last_seen',(sid,cid,now))
  uniq=c.execute('SELECT COUNT(*) FROM source_configs WHERE source_id=?',(sid,)).fetchone()[0]; dup=max(0,valid-(uniq-before)); ms=int((time.monotonic()-start)*1000)
  c.execute("UPDATE sources SET last_fetch=?,last_ok=?,next_fetch=?,status='ok',http_status=?,duration_ms=?,total=?,valid=?,invalid=0,duplicates=?,unique_count=?,error='' WHERE id=?",(now,now,now+row['interval_sec'],r.status_code,ms,valid,valid,dup,uniq,sid)); c.commit(); c.close()
 except Exception as e:
  c=connect(); c.execute("UPDATE sources SET last_fetch=?,next_fetch=?,status='error',error=? WHERE id=?",(now,now+row['interval_sec'],str(e)[:300],sid)); c.commit(); c.close()
 finally: running.discard(sid)
async def scheduler():
 while True:
  now=time.time(); c=connect(); rows=c.execute('SELECT * FROM sources WHERE enabled=1 AND (next_fetch IS NULL OR next_fetch<=?)',(now,)).fetchall(); c.close()
  for r in rows:
   if r['id'] not in running: asyncio.create_task(fetch_one(r))
  await asyncio.sleep(.5)
