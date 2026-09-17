import json,urllib.parse
def parse(raw):
 p=urllib.parse.urlsplit(raw);q=urllib.parse.parse_qs(p.query,keep_blank_values=True);g=lambda k,d='':q.get(k,[d])[0]
 return p,g,urllib.parse.unquote(p.username or '')
def xray(raw):
 p,g,u=parse(raw);net=(g('type','tcp') or 'tcp').lower();sec=(g('security','none') or 'none').lower()
 st={'network':net,'security':sec}; key={'ws':'wsSettings','grpc':'grpcSettings','xhttp':'xhttpSettings','splithttp':'xhttpSettings','httpupgrade':'httpupgradeSettings'}.get(net)
 if key:
  st[key]={}
  if g('path'):st[key]['path']=g('path')
  if net=='grpc' and g('serviceName'):st[key]['serviceName']=g('serviceName')
 if sec=='tls':st['tlsSettings']={'serverName':g('sni')}
 if sec=='reality':st['realitySettings']={'serverName':g('sni'),'publicKey':g('pbk') or g('publicKey'),'shortId':g('sid'),'fingerprint':g('fp')}
 return json.dumps({'outbounds':[{'protocol':'vless','settings':{'vnext':[{'address':p.hostname,'port':p.port,'users':[{'id':u,'encryption':g('encryption','none')}]}]},'streamSettings':st}]})
def singbox(raw):
 p,g,u=parse(raw);net=(g('type','tcp') or 'tcp').lower();sec=(g('security','none') or 'none').lower()
 o={'type':'vless','tag':'proxy','server':p.hostname,'server_port':p.port,'uuid':u}
 if sec in ('tls','reality'):
  o['tls']={'enabled':True,'server_name':g('sni')}
  if sec=='reality':
   o['tls']['utls']={'enabled':True,'fingerprint':g('fp','chrome') or 'chrome'}
   o['tls']['reality']={'enabled':True,'public_key':g('pbk') or g('publicKey'),'short_id':g('sid')}
 if net in ('ws','grpc','httpupgrade'):
  o['transport']={'type':net}
  if g('path'):o['transport']['path']=g('path')
  if net=='grpc' and g('serviceName'):o['transport']['service_name']=g('serviceName')
 return json.dumps({'outbounds':[o]})