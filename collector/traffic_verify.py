import pathlib,time
def io_bytes(pid):
 try:
  d={}
  for line in pathlib.Path(f'/proc/{pid}/io').read_text().splitlines():
   k,v=line.split(':',1);d[k]=int(v.strip())
  return d.get('rchar',0)+d.get('wchar',0)
 except:return 0
def snapshot(handle):return {'pid':handle['process'].pid,'io':io_bytes(handle['process'].pid),'at':time.time()}
def verified(before,after):return after['pid']==before['pid'] and after['io']>before['io']