import os,time
STATE={'level':'normal','limit':4,'load':0.0,'mem_available':0,'disk_free':0,'updated':0.0}
def sample():
 cpus=max(1,os.cpu_count() or 1); load=os.getloadavg()[0]/cpus
 mem={}
 with open('/proc/meminfo') as f:
  for line in f:
   k,v,*_=line.split(); mem[k.rstrip(':')]=int(v)
 avail=mem.get('MemAvailable',0)*1024; total=mem.get('MemTotal',1)*1024; mem_ratio=avail/total
 st=os.statvfs('/'); disk=st.f_bavail*st.f_frsize; disk_ratio=st.f_bavail/max(1,st.f_blocks)
 if load>.85 or mem_ratio<.12 or disk_ratio<.08: level,limit='critical',1
 elif load>.60 or mem_ratio<.20 or disk_ratio<.15: level,limit='busy',2
 elif load<.30 and mem_ratio>.45 and disk_ratio>.25: level,limit='fast',6
 else: level,limit='normal',4
 STATE.update(level=level,limit=limit,load=round(load,3),mem_available=avail,disk_free=disk,updated=time.time()); return STATE
def allowed(active): return active < sample()['limit']
