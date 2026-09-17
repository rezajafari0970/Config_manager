import asyncio,time
from fastapi import FastAPI,Request,HTTPException
from fastapi.responses import HTMLResponse
from fastapi.staticfiles import StaticFiles
from .db import init,connect
from .engine import scheduler,fetch_one
app=FastAPI(title='Config Manager'); app.mount('/static',StaticFiles(directory='collector/static'),name='static')
@app.on_event('startup')
async def start(): init(); asyncio.create_task(scheduler())
@app.get('/',response_class=HTMLResponse)
def home(): return open('collector/templates/index.html',encoding='utf8').read()
@app.get('/api/stats')
def stats():
 c=connect(); sources=[dict(x) for x in c.execute('SELECT * FROM sources ORDER BY id DESC')]; total=c.execute('SELECT COUNT(*) FROM configs').fetchone()[0]; kinds={r[0]:r[1] for r in c.execute('SELECT kind,COUNT(*) FROM configs GROUP BY kind')}; c.close(); issues=c.execute('SELECT COUNT(*) FROM issues').fetchone()[0] if False else 0; return {'sources':sources,'configs':total,'kinds':kinds,'now':time.time()}
@app.post('/api/sources')
async def add(req:Request):
 d=await req.json(); interval=max(2,min(86400,int(d.get('interval',10)))); urls=[x.strip() for x in d.get('urls','').splitlines() if x.strip()]; c=connect(); added=0
 for u in dict.fromkeys(urls):
  if not u.startswith(('http://','https://')): continue
  cur=c.execute('INSERT OR IGNORE INTO sources(url,interval_sec,next_fetch) VALUES(?,?,?)',(u,interval,time.time())); added+=cur.rowcount
 c.commit(); c.close(); return {'ok':True,'added':added,'received':len(urls)}
@app.post('/api/sources/{sid}/fetch')
async def fetch(sid:int):
 c=connect(); r=c.execute('SELECT * FROM sources WHERE id=?',(sid,)).fetchone(); c.close()
 if not r: raise HTTPException(404)
 asyncio.create_task(fetch_one(r)); return {'ok':True}
@app.delete('/api/sources/{sid}')
def delete(sid:int):
 c=connect(); c.execute('DELETE FROM source_configs WHERE source_id=?',(sid,)); c.execute('DELETE FROM sources WHERE id=?',(sid,)); c.commit(); c.close(); return {'ok':True}
@app.get('/health')
def health(): return {'ok':True}
@app.post('/api/sources/delete-selected')
async def delete_selected(req:Request):
 d=await req.json(); ids=[int(x) for x in d.get('ids',[]) if str(x).isdigit()]
 if not ids: return {'ok':True,'deleted':0}
 q=','.join('?'*len(ids)); c=connect(); c.execute(f'DELETE FROM issues WHERE source_id IN ({q})',ids); c.execute(f'DELETE FROM source_configs WHERE source_id IN ({q})',ids); cur=c.execute(f'DELETE FROM sources WHERE id IN ({q})',ids); c.commit(); n=cur.rowcount; c.close(); return {'ok':True,'deleted':n}
@app.delete('/api/sources')
def delete_all_sources():
 c=connect(); n=c.execute('SELECT COUNT(*) FROM sources').fetchone()[0]; c.execute('DELETE FROM issues'); c.execute('DELETE FROM source_configs'); c.execute('DELETE FROM sources'); c.commit(); c.close(); return {'ok':True,'deleted':n}
@app.delete('/api/configs')
def delete_all_configs():
 c=connect(); n=c.execute('SELECT COUNT(*) FROM configs').fetchone()[0]; c.execute('DELETE FROM source_configs'); c.execute('DELETE FROM configs'); c.execute('UPDATE sources SET lifetime_seen=0,unique_count=0,new_count=0,known_count=0'); c.commit(); c.close(); return {'ok':True,'deleted':n}
@app.get('/api/review')
def review():
 from .runtime import review_reason
 c=connect(); rows=[dict(x) for x in c.execute("SELECT * FROM sources WHERE status IN ('error','empty','warning') ORDER BY CASE status WHEN 'error' THEN 1 WHEN 'warning' THEN 2 ELSE 3 END,id DESC")];
 for x in rows: x['reason']=review_reason(x)
 c.close(); return {'items':rows,'count':len(rows)}
@app.get('/health/live')
def live(): return {'ok':True}
@app.get('/health/ready')
def ready():
 from .runtime import STATE
 age=time.time()-STATE['last_loop'] if STATE['last_loop'] else 999
 return {'ok':age<5,'scheduler_age_sec':round(age,3),'active_fetches':STATE['active'],'completed_fetches':STATE['completed']}
@app.get('/api/system')
def system_state():
 from .resources import sample
 from .runtime import STATE
 r=sample(); return {**r,'active_fetches':STATE['active'],'completed_fetches':STATE['completed'],'uptime_sec':round(time.time()-STATE['started'])}
@app.get('/api/quarantine')
def quarantine(limit:int=100):
 c=connect(); rows=[dict(x) for x in c.execute('''SELECT q.id,q.kind,q.reasons,q.first_seen,q.last_seen,q.hits,q.raw,s.url source_url FROM quarantine q LEFT JOIN sources s ON s.id=q.source_id ORDER BY q.last_seen DESC LIMIT ?''',(max(1,min(limit,500)),))]; total=c.execute('SELECT COUNT(*) FROM quarantine').fetchone()[0]; c.close(); return {'items':rows,'count':total}
@app.delete('/api/quarantine')
def clear_quarantine():
 c=connect(); n=c.execute('SELECT COUNT(*) FROM quarantine').fetchone()[0]; c.execute('DELETE FROM quarantine'); c.commit(); c.close(); return {'ok':True,'deleted':n}
