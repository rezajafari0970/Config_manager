import threading,json,pathlib,time
P=pathlib.Path('/root/Config_manager/data/stage_tuning.json');L=threading.Lock();active={'startup':0,'probe':0}
def limits():
 try:d=json.loads(P.read_text());return int(d.get('startup',1)),int(d.get('probe',6))
 except:return 1,6
def enter(stage):
 while True:
  with L:
   lim=limits()[0 if stage=='startup' else 1]
   if active[stage]<lim:active[stage]+=1;return
  time.sleep(.005)
def leave(stage):
 with L:active[stage]=max(0,active[stage]-1)
def startup_enter():enter('startup')
def startup_leave():leave('startup')
def probe_enter():enter('probe')
def probe_leave():leave('probe')