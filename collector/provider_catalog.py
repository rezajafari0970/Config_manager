import json,urllib.request,pathlib,time
URL='https://raw.githubusercontent.com/librespeed/speedtest/master/server-list.json';CACHE=pathlib.Path('/root/Config_manager/data/librespeed_servers.json')
def norm(base,path):
 if base.startswith('//'):base='https:'+base
 return base.rstrip('/')+'/'+path.lstrip('/')
def refresh():
 a=json.load(urllib.request.urlopen(URL,timeout=10));out=[]
 for x in a:
  try:out.append({'name':x['name'],'download':norm(x['server'],x['dlURL']),'upload':norm(x['server'],x['ulURL']),'base':norm(x['server'],'').rstrip('/'),'source':'librespeed'})
  except:pass
 CACHE.write_text(json.dumps({'updated':time.time(),'items':out}));return out
def load():
 try:
  d=json.loads(CACHE.read_text());return d['items'] if time.time()-d['updated']<86400 else refresh()
 except:return refresh()