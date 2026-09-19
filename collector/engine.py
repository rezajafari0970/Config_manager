import asyncio,time,httpx
from .db import connect
from .parser import extract
from .intelligence import analyze
from .runtime import STATE
from .resources import allowed
from .global_governor import fetch_limit
running=set()
async def fetch_one(row):
 sid=row['id']; running.add(sid); start=time.monotonic(); now=time.time()
 try:
  while STATE['active']>=fetch_limit(): await asyncio.sleep(.15)
  STATE['active']+=1
  async with httpx.AsyncClient(timeout=httpx.Timeout(8,connect=4),follow_redirects=True,headers={'User-Agent':'ConfigManager/2.0'}) as client:
   resp=await client.get(row['url']); resp.raise_for_status()
   if len(resp.content)>4*1024*1024: raise ValueError('RESPONSE_TOO_LARGE: maximum 4 MiB')
   items=extract(resp.text)
  c=connect(); c.execute('DELETE FROM issues WHERE source_id=?',(sid,)); raw_count=len(items); valid=invalid=dups=new=known=0; seen=set()
  for it in items:
   fp=it['fingerprint']; intel=analyze(it['kind'],it['raw'])
   if fp in seen: dups+=1; continue
   seen.add(fp)
   if intel['state']!='valid': c.execute('INSERT INTO intelligence_events(fingerprint,source_id,kind,state,confidence,engine_version,details,seen_at) VALUES(?,?,?,?,?,?,?,?)',(fp,sid,it['kind'],intel['state'],intel['confidence'],intel['version'],';'.join(x['code'] for x in intel['issues']),now))
   if it['issues']:
    for problem in it['issues']:
     c.execute('INSERT INTO issues(source_id,seen_at,kind,code,raw,repairable) VALUES(?,?,?,?,?,0)',(sid,now,it['kind'],problem['code']+': '+problem['message'],it['raw'][:4000]))
     if problem['level']=='warning': c.execute('INSERT INTO warnings(fingerprint,source_id,kind,code,message,raw,first_seen,last_seen,hits) VALUES(?,?,?,?,?,?,?,?,1) ON CONFLICT(fingerprint,source_id,code) DO UPDATE SET message=excluded.message,last_seen=excluded.last_seen,hits=warnings.hits+1',(fp,sid,it['kind'],problem['code'],problem['message'],it['raw'][:16000],now,now))
   if it['hard']:
    invalid+=1; reasons=' | '.join(x['code']+': '+x['message'] for x in it['hard']); c.execute('INSERT INTO quarantine(fingerprint,source_id,kind,raw,reasons,first_seen,last_seen,hits) VALUES(?,?,?,?,?,?,?,1) ON CONFLICT(fingerprint,source_id) DO UPDATE SET reasons=excluded.reasons,last_seen=excluded.last_seen,hits=quarantine.hits+1',(fp,sid,it['kind'],it['raw'][:16000],reasons,now,now)); continue
   valid+=1; exists=c.execute('SELECT id FROM configs WHERE fingerprint=?',(fp,)).fetchone()
   if exists: cid=exists[0]; known+=1; c.execute('UPDATE configs SET last_seen=? WHERE id=?',(now,cid))
   else:
    cur=c.execute('INSERT INTO configs(fingerprint,kind,raw,first_seen,last_seen) VALUES(?,?,?,?,?)',(fp,it['kind'],it['raw'],now,now)); cid=cur.lastrowid; new+=1
   c.execute('INSERT INTO source_configs(source_id,config_id,last_seen) VALUES(?,?,?) ON CONFLICT(source_id,config_id) DO UPDATE SET last_seen=excluded.last_seen',(sid,cid,now))
   c.execute("INSERT INTO test_candidates(fingerprint,kind,raw,origin,stage,created_at,updated_at) VALUES(?,?,?,'source','queued',?,?) ON CONFLICT(fingerprint) DO UPDATE SET raw=excluded.raw,kind=excluded.kind",(fp,it['kind'],it['raw'],now,now))
  lifetime=c.execute('SELECT COUNT(*) FROM source_configs WHERE source_id=?',(sid,)).fetchone()[0]; issues=c.execute('SELECT COUNT(*) FROM issues WHERE source_id=?',(sid,)).fetchone()[0]; ms=int((time.monotonic()-start)*1000); status='ok' if valid else ('warning' if invalid else 'empty')
  c.execute("UPDATE sources SET last_fetch=?,last_ok=?,next_fetch=?,status=?,http_status=?,duration_ms=?,total=?,raw_count=?,valid=?,invalid=?,duplicates=?,unique_count=?,new_count=?,known_count=?,lifetime_seen=?,issue_count=?,consecutive_errors=0,error='' WHERE id=?",(now,now,now+row['interval_sec'],status,resp.status_code,ms,raw_count,raw_count,valid,invalid,dups,valid,new,known,lifetime,issues,sid)); c.commit(); c.close()
 except Exception as e:
  ms=int((time.monotonic()-start)*1000); c=connect(); c.execute("UPDATE sources SET last_fetch=?,next_fetch=?,status='error',duration_ms=?,consecutive_errors=consecutive_errors+1,error=? WHERE id=?",(now,now+row['interval_sec'],ms,(str(e) or repr(e))[:500],sid)); c.commit(); c.close()
 finally: running.discard(sid); STATE['active']=max(0,STATE['active']-1); STATE['completed']+=1
async def scheduler():
 while True:
  STATE['loops']+=1; STATE['last_loop']=time.time(); now=STATE['last_loop']; c=connect(); rows=c.execute('SELECT * FROM sources WHERE enabled=1 AND (next_fetch IS NULL OR next_fetch<=?)',(now,)).fetchall(); c.close()
  for r in rows:
   if r['id'] not in running: asyncio.create_task(fetch_one(r))
  await asyncio.sleep(.25)
