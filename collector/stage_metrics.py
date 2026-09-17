import threading,time,json,pathlib
P=pathlib.Path('/root/Config_manager/data/stage_metrics.json');L=threading.Lock();S={'startup':0,'probe':0,'db':0,'active':0,'peak_active':0,'completed':0,'updated':0}
def enter(stage):
 with L:S[stage]+=1;S['active']+=1;S['peak_active']=max(S['peak_active'],S['active']);flush()
def leave(stage):
 with L:S[stage]=max(0,S[stage]-1);S['active']=max(0,S['active']-1);S['completed']+=1;flush()
def flush():S['updated']=time.time();P.write_text(json.dumps(S))
def snapshot():
 with L:return dict(S)