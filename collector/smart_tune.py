import json,pathlib,time
from .db import connect
from .health_settings import load,save
from .provider_pool import active
from .resource_scheduler import snapshot
from .server_telemetry import snapshot as server_snapshot
from .system_pressure import snapshot as pressure_snapshot
P=pathlib.Path('/root/Config_manager/data/smart_tune.json')
def pctl(a,p):
 if not a:return 0
 a=sorted(a);return a[min(len(a)-1,int((len(a)-1)*p))]
def step():
 c=connect();now=time.time();tele=server_snapshot();pressure=pressure_snapshot();rows=c.execute('SELECT upload_ok,download_ok,started_at,finished_at FROM health_attempts WHERE finished_at>=?',(now-180,)).fetchall();c.close();dur=[(r['finished_at']-r['started_at'])*1000 for r in rows];cfg=load();sys=snapshot();providers=len({x.get('base') for x in active()});
 if providers<cfg['min_active_providers']:
  state={'target_tpm':200,'actual_tpm':round(len(rows)/3,1),'providers':providers,'paused_for_provider_quorum':True,'settings':cfg,'telemetry':tele,'pressure':pressure,'updated':now};P.write_text(json.dumps(state));return state
 p50=pctl(dur,.5);p90=pctl(dur,.9);fail=sum(not(r['upload_ok'] and r['download_ok']) for r in rows)/max(1,len(rows));retry=max(10,min(45,int(max(10,p90/1000*3))))
 up=16384 if sys['load_ratio']>.7 else (32768 if sys['load_ratio']>.4 else 49152);down=32 if sys['load_ratio']>.7 else (64 if sys['load_ratio']>.4 else 96);quorum=max(2,min(4,providers//3 or 2));conc=cfg['max_concurrency'];target_tpm=200;tw=pressure['sockets'].get('TCP_tw',0);cpu_psi=pressure['psi']['cpu'].get('avg10',0);io_psi=pressure['psi']['io'].get('avg10',0);actual=len(rows)/3;conc=max(2,conc-1) if (sys['load_ratio']>.90 or tele['ram_free_ratio']<.18 or tele['disk_free_ratio']<.10 or cpu_psi>25 or io_psi>10 or tw>4000) else (min(16,conc+1) if actual<target_tpm and sys['load_ratio']<.88 and sys['mem_free_ratio']>.25 and tele['disk_free_ratio']>.15 else conc);new=save({'retry_seconds':retry,'upload_bytes':up,'download_kb':down,'min_active_providers':quorum,'max_concurrency':conc});state={'target_tpm':200,'actual_tpm':round(len(rows)/3,1),'samples':len(rows),'p50_ms':round(p50),'p90_ms':round(p90),'failure_rate':round(fail,3),'providers':providers,'load_ratio':round(sys['load_ratio'],3),'settings':new,'telemetry':tele,'pressure':pressure,'updated':now};P.write_text(json.dumps(state));return state
if __name__=='__main__':print(json.dumps(step()))