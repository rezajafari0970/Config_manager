import os,time,shutil,socket,json,pathlib
P=pathlib.Path('/root/Config_manager/data/server_telemetry.json');LAST={}
def net():
 global LAST
 d={}
 for n in os.listdir('/sys/class/net'):
  if n=='lo':continue
  try:d[n]=[int(open(f'/sys/class/net/{n}/statistics/rx_bytes').read()),int(open(f'/sys/class/net/{n}/statistics/tx_bytes').read())]
  except:pass
 now=time.time();rate=0
 if LAST:
  dt=max(.1,now-LAST['t']);rate=sum(max(0,v[0]-LAST['d'].get(k,v)[0])+max(0,v[1]-LAST['d'].get(k,v)[1]) for k,v in d.items())/dt
 LAST={'t':now,'d':d};return rate
def snapshot():
 mem={};
 for l in open('/proc/meminfo'):
  k,v=l.split(':',1);mem[k]=int(v.split()[0])
 disk=shutil.disk_usage('/');load=os.getloadavg()[0]/max(1,os.cpu_count() or 1);s={'cpu_load_ratio':load,'ram_free_ratio':mem.get('MemAvailable',0)/max(1,mem.get('MemTotal',1)),'disk_free_ratio':disk.free/disk.total,'disk_free_gb':round(disk.free/2**30,1),'net_bytes_sec':round(net()),'uptime_sec':float(open('/proc/uptime').read().split()[0]),'tcp_established':sum(1 for l in open('/proc/net/tcp') if ' 01 ' in l),'fd_used':len(os.listdir('/proc/self/fd')),'ts':time.time()};P.write_text(json.dumps(s));return s