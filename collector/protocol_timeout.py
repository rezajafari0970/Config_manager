import sqlite3,time,statistics,pathlib,json
DB='/root/Config_manager/data/collector.db';P=pathlib.Path('/root/Config_manager/data/protocol_timeouts.json')
def transport(raw):
 s=raw.lower()
 for x in ('grpc','ws','xhttp','httpupgrade','tcp'):
  if 'type='+x in s or '"network":"'+x+'"' in s:return x
 return 'other'
def learn():
 c=sqlite3.connect(DB);c.row_factory=sqlite3.Row;rows=c.execute('select t.kind,t.raw,p.total_ms from profile_samples p join test_candidates t on t.fingerprint=p.fingerprint where p.ts>? and p.total_ms>0',(time.time()-3600,)).fetchall();c.close();d={}
 for r in rows:d.setdefault(r['kind']+':'+transport(r['raw']),[]).append(r['total_ms'])
 out={}
 for k,v in d.items():
  if len(v)>=8:v.sort();p90=v[int(.9*(len(v)-1))];out[k]={'samples':len(v),'timeout_ms':max(900,min(4500,int(p90*1.25)))}
 P.write_text(json.dumps({'updated':time.time(),'items':out}));return out
def get(kind,raw,default=4000):
 try:d=json.loads(P.read_text())['items'];return d.get(kind+':'+transport(raw),{}).get('timeout_ms',default)
 except:return default