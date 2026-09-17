import threading,time,json,pathlib
P=pathlib.Path('/root/Config_manager/data/stage_profile.json');L=threading.Lock();S={'startup_wait_ms':0,'startup_exec_ms':0,'probe_wait_ms':0,'probe_exec_ms':0,'startup_n':0,'probe_n':0}
def add(k,ms):
 with L:S[k]+=ms;S[k.replace('_ms','_n') if k.endswith('exec_ms') else k]=S.get(k.replace('_ms','_n'),S.get(k,0));P.write_text(json.dumps({**S,'updated':time.time()}))
def snap():
 with L:return dict(S)