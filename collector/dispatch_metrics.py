import threading,time,json,pathlib
P=pathlib.Path('/root/Config_manager/data/dispatch_metrics.json');L=threading.Lock();S={'claim_ms':0,'lane_wait_ms':0,'jobs':0,'active':0,'peak':0}
def add(key,ms=0):
 with L:S[key]=S.get(key,0)+ms;P.write_text(json.dumps({**S,'updated':time.time()}))
def start():
 with L:S['jobs']+=1;S['active']+=1;S['peak']=max(S['peak'],S['active'])
def finish():
 with L:S['active']=max(0,S['active']-1)
def snapshot():
 with L:return dict(S)