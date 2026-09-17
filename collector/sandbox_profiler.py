import threading,time,json,pathlib
P=pathlib.Path('/root/Config_manager/data/sandbox_profile.json');L=threading.Lock();S={'port_ms':0,'build_ms':0,'write_ms':0,'spawn_ms':0,'ready_wait_ms':0,'ok':0,'fail':0}
def add(k,v):
 with L:S[k]=S.get(k,0)+v;P.write_text(json.dumps({**S,'updated':time.time()}))