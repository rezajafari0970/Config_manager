import pathlib
def sock():
 d={}
 for f in ('/proc/net/sockstat','/proc/net/sockstat6'):
  for l in open(f):
   p=l.replace(':','').split();name=p[0]
   for i in range(1,len(p)-1,2):
    try:d[name+'_'+p[i]]=int(p[i+1])
    except:pass
 return d
def io():
 p=pathlib.Path('/proc/pressure/io');m=pathlib.Path('/proc/pressure/memory');c=pathlib.Path('/proc/pressure/cpu')
 def psi(x):
  try:return {z.split('=')[0]:float(z.split('=')[1]) for z in x.read_text().splitlines()[0].split()[1:]}
  except:return {}
 return {'io':psi(p),'memory':psi(m),'cpu':psi(c)}
def snapshot():return {'sockets':sock(),'psi':io()}