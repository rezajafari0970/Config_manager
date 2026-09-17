import subprocess,re
def snapshot(pid,ports):
 p=subprocess.run(['/usr/bin/ss','-tinp'],stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,text=True,timeout=2);out={x:{'sent':0,'received':0} for x in ports};lines=p.stdout.splitlines()
 for i,l in enumerate(lines):
  if f'pid={pid},' not in l:continue
  for port in ports:
   if f':{port} ' not in l:continue
   info=lines[i+1] if i+1<len(lines) else '';a=re.search(r'bytes_sent:(\d+)',info);b=re.search(r'bytes_received:(\d+)',info);out[port]['sent']+=int(a.group(1)) if a else 0;out[port]['received']+=int(b.group(1)) if b else 0
 return out
def verified(before,after,port):
 b=before.get(port,{});a=after.get(port,{});return a.get('sent',0)>b.get('sent',0) and a.get('received',0)>b.get('received',0)