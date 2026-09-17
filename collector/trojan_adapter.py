import json,urllib.parse
def parse(raw):
 p=urllib.parse.urlsplit(raw);q=urllib.parse.parse_qs(p.query,keep_blank_values=True);g=lambda k,d='':q.get(k,[d])[0];return p,g,urllib.parse.unquote(p.username or '')
def xray(raw):
 p,g,pw=parse(raw);net=(g('type','tcp') or 'tcp').lower();sec=(g('security','tls') or 'tls').lower();st={'network':net,'security':sec}
 key={'ws':'wsSettings','grpc':'grpcSettings','httpupgrade':'httpupgradeSettings','xhttp':'xhttpSettings'}.get(net)
 if key:
  st[key]={}
  if g('path'):st[key]['path']=g('path')
  if net=='grpc' and g('serviceName'):st[key]['serviceName']=g('serviceName')
 if sec=='tls':st['tlsSettings']={'serverName':g('sni') or p.hostname}
 return json.dumps({'outbounds':[{'protocol':'trojan','settings':{'servers':[{'address':p.hostname,'port':p.port,'password':pw}]},'streamSettings':st}]})
def singbox(raw):
 p,g,pw=parse(raw);net=(g('type','tcp') or 'tcp').lower();x={'type':'trojan','tag':'proxy','server':p.hostname,'server_port':p.port,'password':pw}
 if (g('security','tls') or 'tls').lower()=='tls':x['tls']={'enabled':True,'server_name':g('sni') or p.hostname}
 if net in ('ws','grpc','httpupgrade'):
  x['transport']={'type':net}
  if g('path'):x['transport']['path' if net!='grpc' else 'service_name']=g('path')
  if net=='grpc' and g('serviceName'):x['transport']['service_name']=g('serviceName')
 return json.dumps({'outbounds':[x]})