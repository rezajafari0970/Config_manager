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
