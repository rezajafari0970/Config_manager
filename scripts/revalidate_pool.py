import sqlite3,time
from collector.validator import validate,hard_errors
c=sqlite3.connect('data/collector.db');c.row_factory=sqlite3.Row;now=time.time();moved=[]
for r in c.execute("select * from configs where kind='vless'").fetchall():
 h=hard_errors(validate('vless',r['raw']))
 if not h:continue
 reasons=' | '.join(x['code']+': '+x['message'] for x in h)
 for s in c.execute('select source_id from source_configs where config_id=?',(r['id'],)).fetchall():
  c.execute('INSERT INTO quarantine(fingerprint,source_id,kind,raw,reasons,first_seen,last_seen,hits) VALUES(?,?,?,?,?,?,?,1) ON CONFLICT(fingerprint,source_id) DO UPDATE SET reasons=excluded.reasons,last_seen=excluded.last_seen,hits=quarantine.hits+1',(r['fingerprint'],s[0],r['kind'],r['raw'],reasons,now,now))
 c.execute('delete from source_configs where config_id=?',(r['id'],));c.execute('delete from configs where id=?',(r['id'],));moved.append(r['id'])
c.commit();print({'moved':moved,'count':len(moved)})