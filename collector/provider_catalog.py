import json,urllib.request,pathlib,time
URL='https://raw.githubusercontent.com/librespeed/speedtest/master/server-list.json';BUILTIN=[{'name':'Cloudflare','download':'https://speed.cloudflare.com/__down?bytes=65536','upload':'https://speed.cloudflare.com/__up','base':'https://speed.cloudflare.com','source':'builtin'},{'name':'HTTPBin','download':'https://httpbin.org/bytes/65536','upload':'https://httpbin.org/post','base':'https://httpbin.org','source':'builtin'},{'name':'Postman Echo','download':'https://postman-echo.com/bytes/65536','upload':'https://postman-echo.com/post','base':'https://postman-echo.com','source':'builtin'}];CACHE=pathlib.Path('/root/Config_manager/data/librespeed_servers.json')
def norm(base,path):
 if base.startswith('//'):base='https:'+base
 return base.rstrip('/')+'/'+path.lstrip('/')
def refresh():
 a=json.load(urllib.request.urlopen(URL,timeout=10));out=[]
 for x in a:
  try:out.append({'name':x['name'],'download':norm(x['server'],x['dlURL']),'upload':norm(x['server'],x['ulURL']),'base':norm(x['server'],'').rstrip('/'),'source':'librespeed'})
  except:pass
 seen={x['base'] for x in out};out=BUILTIN+[x for x in out if x['base'] not in {b['base'] for b in BUILTIN}];CACHE.write_text(json.dumps({'updated':time.time(),'items':out}));return out
def load():
 try:
  d=json.loads(CACHE.read_text());return d['items'] if time.time()-d['updated']<86400 else refresh()
 except:return BUILTIN+refresh()