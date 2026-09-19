import json,time,pathlib,threading
P=pathlib.Path('/root/Config_manager/data/provider_scores.json');L=threading.Lock()
def load():
 try:return json.loads(P.read_text())
 except:return {}
def record(base,ok,ms,side=None):
 with L:
  d=load();x=d.get(base,{'ewma_ms':1000,'ok':0,'fail':0});x['ewma_ms']=round(x['ewma_ms']*.7+ms*.3,1);x['ok' if ok else 'fail']+=1
  if side:
   y=x.setdefault(side,{'ewma_ms':1000,'ok':0,'fail':0});y['ewma_ms']=round(y['ewma_ms']*.7+ms*.3,1);y['ok' if ok else 'fail']+=1;y['updated']=time.time()
  x['updated']=time.time();d[base]=x;P.write_text(json.dumps(d))
def rank(items,side=None):
 d=load()
 def score(x):
  s=d.get(x.get('base'),{});z=s.get(side,s) if side else s;n=z.get('ok',0)+z.get('fail',0);rate=(z.get('ok',0)+2)/(n+4);return z.get('ewma_ms',800)+(1-rate)*1200+min(800,z.get('fail',0)*8)
 return sorted(items,key=score)