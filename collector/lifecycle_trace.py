import json,time,pathlib,threading
P=pathlib.Path('/root/Config_manager/data/lifecycle_trace.jsonl');L=threading.Lock()
def event(fp,name,**extra):
 row={'ts':time.time(),'fp':fp,'event':name};row.update(extra)
 with L:
  with P.open('a') as f:f.write(json.dumps(row,separators=(',',':'))+'\n')