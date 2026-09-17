import json,pathlib,statistics,time
P=pathlib.Path('/root/Config_manager/data/canary_samples.jsonl');O=pathlib.Path('/root/Config_manager/data/canary_analysis.json')
def transport(raw):
 s=raw.lower()
 for x in ('grpc','ws','xhttp','httpupgrade','tcp'):
  if 'type='+x in s or '"network":"'+x+'"' in s:return x
 return 'other'
def add(group,kind,raw,ms,ok):
 with P.open('a') as f:f.write(json.dumps({'ts':time.time(),'g':group,'k':kind,'t':transport(raw),'ms':ms,'ok':bool(ok)})+'\n')
def analyze():
 if not P.exists():return {}
 rows=[json.loads(x) for x in P.read_text().splitlines()[-2000:]];out={}
 for g in ('baseline','canary'):
  for key in sorted({r['k']+':'+r['t'] for r in rows if r['g']==g}):
   v=[r for r in rows if r['g']==g and r['k']+':'+r['t']==key];a=sorted(r['ms'] for r in v);out[g+':'+key]={'n':len(v),'p50':round(statistics.median(a),1),'p90':round(a[min(len(a)-1,int(.9*len(a)))],1),'success':round(sum(r['ok'] for r in v)/len(v),3)}
 O.write_text(json.dumps({'updated':time.time(),'groups':out}));return out