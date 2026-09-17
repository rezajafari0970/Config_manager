import json,time,pathlib,threading
P=pathlib.Path('/root/Config_manager/data/provider_scores.json');L=threading.Lock()
def load():
 try:return json.loads(P.read_text())
 except:return {}
def record(base,ok,ms):
 with L:
  d=load();x=d.get(base,{'ewma_ms':1000,'ok':0,'fail':0});x['ewma_ms']=round(x['ewma_ms']*.7+ms*.3,1);x['ok' if ok else 'fail']+=1;x['updated']=time.time();d[base]=x;P.write_text(json.dumps(d))
def rank(items):
 d=load()
 def score(x):
  s=d.get(x.get('base'),{});return s.get('ewma_ms',800)+(s.get('fail',0)/(s.get('ok',0)+1))*500
 return sorted(items,key=score)