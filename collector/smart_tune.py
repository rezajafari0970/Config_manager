import json,pathlib,time
from .db import connect
from .health_settings import load,save
from .provider_pool import active
from .resource_scheduler import snapshot
P=pathlib.Path('/root/Config_manager/data/smart_tune.json')
def pctl(a,p):
 if not a:return 0
 a=sorted(a);return a[min(len(a)-1,int((len(a)-1)*p))]
def step():
 c=connect();now=time.time();rows=c.execute('SELECT upload_ok,download_ok,started_at,finished_at FROM health_attempts WHERE finished_at>=?',(now-180,)).fetchall();c.close();dur=[(r['finished_at']-r['started_at'])*1000 for r in rows];cfg=load();sys=snapshot();providers=len({x.get('base') for x in active()});p50=pctl(dur,.5);p90=pctl(dur,.9);fail=sum(not(r['upload_ok'] and r['download_ok']) for r in rows)/max(1,len(rows));retry=max(10,min(45,int(max(10,p90/1000*3))))
 up=16384 if sys['load_ratio']>.7 else (32768 if sys['load_ratio']>.4 else 49152);down=32 if sys['load_ratio']>.7 else (64 if sys['load_ratio']>.4 else 96);quorum=max(2,min(4,providers//3 or 2));conc=cfg['max_concurrency'];conc=max(2,conc-1) if sys['load_ratio']>.8 else (min(16,conc+1) if sys['load_ratio']<.55 and sys['mem_free_ratio']>.35 else conc);new=save({'retry_seconds':retry,'upload_bytes':up,'download_kb':down,'min_active_providers':quorum,'max_concurrency':conc});state={'samples':len(rows),'p50_ms':round(p50),'p90_ms':round(p90),'failure_rate':round(fail,3),'providers':providers,'load_ratio':round(sys['load_ratio'],3),'settings':new,'updated':now};P.write_text(json.dumps(state));return state
if __name__=='__main__':print(json.dumps(step()))