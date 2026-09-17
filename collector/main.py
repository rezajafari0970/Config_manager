import asyncio,time
from fastapi import FastAPI,Request,HTTPException
from fastapi.responses import HTMLResponse
from fastapi.staticfiles import StaticFiles
from .db import init,connect
from .engine import scheduler,fetch_one
app=FastAPI(title='Config Manager'); app.mount('/static',StaticFiles(directory='collector/static'),name='static')
@app.on_event('startup')
async def start(): init()
@app.get('/',response_class=HTMLResponse)
def home(): return open('collector/templates/index.html',encoding='utf8').read()
_stats_cache={'at':0.0,'data':None}
@app.get('/api/stats')
def stats():
 now=time.time()
 if _stats_cache['data'] is not None and now-_stats_cache['at']<0.75: return _stats_cache['data']
 c=connect(); sources=[dict(x) for x in c.execute('SELECT * FROM sources ORDER BY id DESC')]; total=c.execute('SELECT COUNT(*) FROM configs').fetchone()[0]; kinds={r[0]:r[1] for r in c.execute('SELECT kind,COUNT(*) FROM configs GROUP BY kind')}; c.close(); data={'sources':sources,'configs':total,'kinds':kinds,'now':now}; _stats_cache.update(at=now,data=data); return data
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
from fastapi.responses import PlainTextResponse
@app.get('/api/configs/raw',response_class=PlainTextResponse)
def configs_raw():
 c=connect(); rows=c.execute('SELECT raw FROM configs ORDER BY id').fetchall(); c.close(); return '\n'.join(x[0] for x in rows)
@app.get('/api/quarantine/raw',response_class=PlainTextResponse)
def quarantine_raw():
 c=connect(); rows=c.execute('SELECT raw FROM quarantine ORDER BY id').fetchall(); c.close(); return '\n'.join(x[0] for x in rows)
@app.get('/api/warnings')
def warnings(limit:int=150):
 c=connect(); rows=[dict(x) for x in c.execute('''SELECT w.id,w.kind,w.code,w.message,w.raw,w.first_seen,w.last_seen,w.hits,s.url source_url FROM warnings w LEFT JOIN sources s ON s.id=w.source_id ORDER BY w.last_seen DESC LIMIT ?''',(max(1,min(limit,500)),))]; total=c.execute('SELECT COUNT(*) FROM warnings').fetchone()[0]; by_code={x[0]:x[1] for x in c.execute('SELECT code,COUNT(*) FROM warnings GROUP BY code ORDER BY 2 DESC')}; c.close(); return {'items':rows,'count':total,'by_code':by_code}
@app.delete('/api/warnings')
def clear_warnings():
 c=connect(); n=c.execute('SELECT COUNT(*) FROM warnings').fetchone()[0]; c.execute('DELETE FROM warnings'); c.commit(); c.close(); return {'ok':True,'deleted':n}
@app.get('/api/repairs')
def repairs(limit:int=100):
 c=connect(); rows=[dict(x) for x in c.execute('SELECT id,kind,changes,created_at,raw_original,raw_repaired FROM repairs ORDER BY created_at DESC LIMIT ?',(max(1,min(limit,500)),))]; total=c.execute('SELECT COUNT(*) FROM repairs').fetchone()[0]; c.close(); return {'items':rows,'count':total}
@app.get('/api/configs/repaired',response_class=PlainTextResponse)
def repaired_configs():
 c=connect(); rows=c.execute('''SELECT COALESCE(r.raw_repaired,c.raw) FROM configs c LEFT JOIN repairs r ON r.fingerprint=c.fingerprint ORDER BY c.id''').fetchall(); c.close(); return '\n'.join(x[0] for x in rows)
@app.get('/api/intelligence')
def intelligence():
 import json
 from .intelligence import learn_snapshot
 c=connect(); snap=learn_snapshot(c); states={r[0]:r[1] for r in c.execute('SELECT state,COUNT(*) FROM intelligence_events WHERE seen_at>? GROUP BY state',(time.time()-3600,))}; c.close(); return {**snap,'recent_states':states}
@app.get('/api/intelligence/rules')
def intelligence_rules():
 from .rules import RULEPACK_VERSION,RULES
 return {'version':RULEPACK_VERSION,'rules':RULES}
@app.post('/api/intelligence/discover')
def intelligence_discover():
 from .learning import discover
 return discover()
@app.get('/api/intelligence/proposals')
def intelligence_proposals():
 from .learning import list_proposals
 return {'items':list_proposals()}
@app.post('/api/intelligence/audit')
def intelligence_audit():
 from .auto_audit import run
 return run()
@app.get('/api/intelligence/audit')
def intelligence_audit_state():
 from .auto_audit import STATE
 return STATE
@app.get('/api/xray-audit')
def xray_audit_state():
 from .xray_auditor import STATE
 return STATE
@app.post('/api/xray-audit')
def xray_audit(limit:int=0,offset:int=0):
 from .xray_auditor import run
 return run(max(0,min(limit,5000)),max(0,offset))
@app.get('/api/xray-worker')
def xray_worker_state():
 from .xray_worker import load
 return load()
@app.get('/api/xray-freeze-gate')
def xray_freeze_gate():
 from .xray_freeze import gate
 return gate()
@app.get('/api/singbox-worker')
def singbox_worker_state():
 from .singbox_worker import load
 return load()
@app.get('/api/singbox-freeze-gate')
def singbox_freeze_gate():
 from .singbox_freeze import gate
 return gate()
@app.get('/api/vless-core-worker')
def vless_core_worker_state():
 from .vless_worker import load
 return load()
@app.get('/api/vmess-core-worker')
def vmess_core_worker_state():
 from .vmess_worker import load
 return load()
@app.get('/api/trojan-core-worker')
def trojan_core_worker_state():
 from .trojan_worker import load
 return load()
@app.get('/api/ss-recovery-worker')
def ss_recovery_worker_state():
 from .ss_recovery_worker import load
 return load()
@app.get('/api/ss-recovery-candidates')
def ss_recovery_candidates(limit:int=50):
 from .ss_classifier import classify
 from .ss_recovery import candidate
 c=connect();rows=c.execute("select id,raw from configs where kind='ss' order by id desc").fetchall();c.close();out=[]
 for r in rows:
  x=classify(r['raw'])
  if x['class']=='vless-like-mislabeled':out.append({'id':r['id'],'class':x['class'],'confidence':x['confidence'],'original':r['raw'],'candidate':candidate(r['raw'])})
  if len(out)>=min(limit,200):break
 return {'items':out,'count':len(out)}
@app.get('/api/recovery-registry')
def recovery_registry(limit:int=50):
 c=connect();rows=c.execute('SELECT id,original_kind,candidate_kind,class,confidence,validator_ok,xray_ok,singbox_ok,recoverable,first_seen,last_seen,hits FROM recovery_registry ORDER BY last_seen DESC LIMIT ?',(min(limit,200),)).fetchall();total=c.execute('SELECT COUNT(*) FROM recovery_registry').fetchone()[0];c.close();return {'total':total,'items':[dict(r) for r in rows]}
@app.get('/api/recovered-candidates')
def recovered_candidates(limit:int=50):
 c=connect();rows=c.execute('SELECT id,kind,status,source,first_seen,last_seen FROM recovered_candidates ORDER BY last_seen DESC LIMIT ?',(min(limit,200),)).fetchall();total=c.execute('SELECT COUNT(*) FROM recovered_candidates').fetchone()[0];c.close();return {'total':total,'health_definition':'NOT_HEALTHY_UNTIL_UPLOAD_AND_DOWNLOAD_PASS','items':[dict(r) for r in rows]}
@app.get('/api/health-dashboard')
def health_dashboard():
 c=connect();st={r['stage']:r['n'] for r in c.execute('SELECT stage,COUNT(*) n FROM test_candidates GROUP BY stage')};res={r['result']:r['n'] for r in c.execute('SELECT result,COUNT(*) n FROM health_attempts GROUP BY result')};healthy=c.execute('SELECT COUNT(*) FROM healthy_configs').fetchone()[0];countries=[dict(r) for r in c.execute("SELECT country_flag,country_name,COUNT(*) count FROM healthy_configs GROUP BY country_code,country_name,country_flag ORDER BY count DESC")];cdn=[dict(r) for r in c.execute("SELECT cdn_state,COUNT(*) count FROM healthy_configs GROUP BY cdn_state")];items=[dict(r) for r in c.execute('SELECT h.remark,h.kind,h.country_name,h.asn,h.network_org,h.cdn_state,h.egress_ip,h.last_healthy,h.next_check,n.provider,n.hosting,n.cdn,n.cdn_provider,n.facility,n.facility_confidence FROM healthy_configs h LEFT JOIN network_intelligence n ON n.fingerprint=h.fingerprint ORDER BY h.last_healthy DESC LIMIT 50')];c.close();return {'stages':st,'results':res,'healthy':healthy,'countries':countries,'cdn':cdn,'items':items}
@app.get('/api/health-settings')
def get_health_settings():
 from .health_settings import load
 return load()
@app.post('/api/health-settings')
def set_health_settings(payload:dict):
 from .health_settings import save
 return save(payload)
from fastapi.responses import PlainTextResponse
@app.get('/sub/all',response_class=PlainTextResponse)
def sub_all():
 from .remark import apply
 c=connect();rows=c.execute('SELECT kind,raw,remark FROM healthy_configs WHERE upload_ok=1 AND download_ok=1 ORDER BY country_name,kind,id').fetchall();c.close();return '\n'.join(apply(r['kind'],r['raw'],r['remark'] or '🌐 Unknown') for r in rows)

def _healthy_sub(where='',args=()):
 from .remark import apply
 c=connect();rows=c.execute('SELECT kind,raw,remark FROM healthy_configs WHERE upload_ok=1 AND download_ok=1 '+where+' ORDER BY country_name,kind,id',args).fetchall();c.close();return '\n'.join(apply(r['kind'],r['raw'],r['remark'] or '🌐 Unknown') for r in rows)
@app.get('/sub/country/{code}',response_class=PlainTextResponse)
def sub_country(code:str):return _healthy_sub('AND UPPER(country_code)=?',(code.upper(),))
@app.get('/sub/cdn',response_class=PlainTextResponse)
def sub_cdn():return _healthy_sub("AND cdn_state='cdn'")
@app.get('/sub/non-cdn',response_class=PlainTextResponse)
def sub_noncdn():return _healthy_sub("AND cdn_state!='cdn'")
@app.get('/sub/datacenter/{asn}',response_class=PlainTextResponse)
def sub_datacenter(asn:str):return _healthy_sub('AND asn=?',(asn,))
@app.get('/api/subscription-index')
def subscription_index():
 c=connect();countries=[dict(r) for r in c.execute('SELECT country_code code,country_flag flag,country_name name,COUNT(*) count FROM healthy_configs GROUP BY country_code,country_flag,country_name ORDER BY country_name')];dcs=[dict(r) for r in c.execute("SELECT asn,network_org name,COUNT(*) count FROM healthy_configs WHERE asn IS NOT NULL AND asn!='' GROUP BY asn,network_org ORDER BY count DESC")];cdn=c.execute("SELECT SUM(cdn_state='cdn') cdn,SUM(cdn_state!='cdn') noncdn FROM healthy_configs").fetchone();total=c.execute('SELECT COUNT(*) FROM healthy_configs').fetchone()[0];c.close();return {'total':total,'countries':countries,'datacenters':dcs,'cdn':cdn['cdn'] or 0,'noncdn':cdn['noncdn'] or 0}
@app.get('/api/performance')
def performance():
 from .throughput import sample
 from .resource_scheduler import snapshot
 return {**sample(),**snapshot()}
@app.get('/api/autotune')
def autotune_state():
 import json,pathlib
 try:return json.loads(pathlib.Path('/root/Config_manager/data/autotune.json').read_text())
 except:return {'status':'warming-up'}
@app.get('/api/queue-lanes')
def queue_lanes():
 c=connect();rows=[dict(r) for r in c.execute("SELECT COALESCE(lane,'fast') lane,COUNT(*) count,ROUND(AVG(cost_ms),1) avg_ms FROM test_candidates WHERE stage='queued' GROUP BY COALESCE(lane,'fast')")];retry=c.execute("SELECT COUNT(*) FROM test_candidates WHERE stage='retry_wait'").fetchone()[0];c.close();return {'policy':{'fast':70,'normal':20,'slow':10,'idle_capacity':'borrowed'},'lanes':rows,'retry_wait':retry}
@app.get('/api/smart-tune')
def smart_tune_state():
 import json,pathlib
 try:return json.loads(pathlib.Path('/root/Config_manager/data/smart_tune.json').read_text())
 except:return {'status':'warming-up'}
@app.get('/api/profile')
def profile_summary():
 from .profiler import summary
 return summary()
@app.get('/api/server-telemetry')
def server_telemetry():
 from .server_telemetry import snapshot
 return snapshot()
@app.get('/api/system-pressure')
def system_pressure():
 from .system_pressure import snapshot
 return snapshot()
@app.get('/api/governor')
def governor():
 from .global_governor import snapshot
 return snapshot()
@app.get('/api/governor-feedback')
def governor_feedback():
 import json,pathlib
 try:return json.loads(pathlib.Path('/root/Config_manager/data/governor_feedback.json').read_text())
 except:return {'status':'warming-up'}
@app.get('/api/health-feedback')
def health_feedback():
 import json,pathlib
 try:return json.loads(pathlib.Path('/root/Config_manager/data/health_feedback.json').read_text())
 except:return {'status':'warming-up'}
