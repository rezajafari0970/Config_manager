#!/usr/bin/env python3
"""Read-only measurement from the durable one-time reset cohort.
Unknown/unsupported, retries, aliases and archive rows cannot inflate settlement.
"""
import argparse,collections,hashlib,json,pathlib,sqlite3,time

def snapshot(db,run=None):
 db.execute('BEGIN')
 try:
  row=db.execute('SELECT id,started FROM egress_runs WHERE id=?' if run else 'SELECT id,started FROM egress_runs ORDER BY id DESC LIMIT 1',(run,) if run else ()).fetchone()
  if row is None:raise ValueError('no_reset_cohort')
  run,started=row
  members=db.execute('SELECT hash,status,completed_at,state,settled_at,settled_state FROM egress_run_members WHERE run=?',(run,)).fetchall()
  missing=db.execute('SELECT count(DISTINCT m.raw_hash) FROM memberships m WHERE NOT EXISTS(SELECT 1 FROM egress_jobs j WHERE j.hash=m.raw_hash)').fetchone()[0]
 finally:db.commit()
 completed=sorted(m[2] for m in members if m[1]=='completed')
 settled=sorted(m[4] for m in members if m[1]=='completed' and m[4]>0)
 result=dict(mode='read_only_reset_cohort',run=run,started=started,observed_at=int(time.time()*1000),total=len(members),completed=len(completed),settled=len(settled),pending=sum(m[1]=='pending' for m in members),excluded=sum(m[1]=='excluded' for m in members),first_states=dict(collections.Counter(m[3] for m in members if m[1]=='completed')),settled_states=dict(collections.Counter(m[5] for m in members if m[4]>0)),cohort_sha256=hashlib.sha256('\n'.join(sorted(m[0] for m in members)).encode()).hexdigest(),unregistered_current=missing,unit='collector_identity')
 for count,deadline in [(1000,600),(2000,900)]:
  for label,values in [('assessed',completed),('definitive',settled)]:
   clock=(values[count-1]-started)/1000 if len(values)>=count else None
   result[f'{label}_{count}_seconds']=clock
   result[f'{label}_at_{deadline}s']=sum(v<=started+deadline*1000 for v in values)
   result[f'{label}_{count}_target_met']=clock is not None and clock<=deadline
  result[f'{count}_cohort_available']=len(members)>=count
 result['scope']='Current unique reset cohort only; first assessment may be UNKNOWN. Definitive requires verified positive, route negative, confirmed unsafe retirement or confirmed Iran. No ad-render, future-uptime or unmeasured 2000-input claim.'
 return result

def observe(database,seconds,output,run=None):
 db=sqlite3.connect('file:'+str(pathlib.Path(database).resolve())+'?mode=ro',uri=True,timeout=5)
 db.execute('PRAGMA query_only=ON');begin=time.monotonic();timeline=[];out=pathlib.Path(output)
 while True:
  receipt=snapshot(db,run);run=receipt['run'];elapsed=time.monotonic()-begin
  if not timeline or elapsed-timeline[-1]['observer_seconds']>=30:
   progress={k:receipt[k] for k in ['total','completed','settled','pending','excluded']};progress.update(observer_seconds=round(elapsed,3),since_reset_seconds=round((receipt['observed_at']-receipt['started'])/1000,3));timeline.append(progress);print(json.dumps(progress),flush=True)
  receipt['timeline']=timeline;out.write_text(json.dumps(receipt,indent=2))
  if elapsed>=seconds:break
  time.sleep(min(2,seconds-elapsed))
 db.close();print(json.dumps({k:v for k,v in receipt.items() if k!='timeline'}),flush=True)

if __name__=='__main__':
 p=argparse.ArgumentParser();p.add_argument('--panel-config',default='/etc/config-repair/panel.json');p.add_argument('--seconds',type=int,default=960);p.add_argument('--run',type=int);p.add_argument('--output',required=True);a=p.parse_args()
 if not 10<=a.seconds<=3600:raise SystemExit('invalid measurement bounds')
 cfg=json.loads(pathlib.Path(a.panel_config).read_text());observe(cfg['database'],a.seconds,a.output,a.run)
