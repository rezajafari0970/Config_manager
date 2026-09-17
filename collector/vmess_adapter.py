import json
from .vmess_deep import decode
def xray(raw):
 o=decode(raw);net=str(o.get('net') or 'tcp').lower();st={'network':net,'security':'tls' if str(o.get('tls','')).lower()=='tls' else 'none'}
 key={'ws':'wsSettings','grpc':'grpcSettings','h2':'httpSettings','http':'httpSettings','kcp':'kcpSettings'}.get(net)
 if key:
  st[key]={}
  if o.get('path'):st[key]['path']=o['path']
  if net=='grpc' and o.get('path'):st[key]['serviceName']=o['path']
 if st['security']=='tls':st['tlsSettings']={'serverName':o.get('sni') or o.get('host') or ''}
 u={'id':o.get('id'),'alterId':int(o.get('aid') or 0),'security':o.get('scy') or 'auto'}
 return json.dumps({'outbounds':[{'protocol':'vmess','settings':{'vnext':[{'address':o.get('add'),'port':int(o.get('port')),'users':[u]}]},'streamSettings':st}]})
def singbox(raw):
 o=decode(raw);net=str(o.get('net') or 'tcp').lower();x={'type':'vmess','tag':'proxy','server':o.get('add'),'server_port':int(o.get('port')),'uuid':o.get('id'),'security':o.get('scy') or 'auto','alter_id':int(o.get('aid') or 0)}
 if str(o.get('tls','')).lower()=='tls':x['tls']={'enabled':True,'server_name':o.get('sni') or o.get('host') or ''}
 if net in ('ws','grpc'):
  x['transport']={'type':net}
  if o.get('path'):x['transport']['path' if net=='ws' else 'service_name']=o['path']
 return json.dumps({'outbounds':[x]})