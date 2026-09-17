import json
from .ss_deep import parse
def xray(raw):
 m,p,h,port=parse(raw)
 return json.dumps({'outbounds':[{'protocol':'shadowsocks','settings':{'servers':[{'address':h,'port':port,'method':m,'password':p}]}}]})
def singbox(raw):
 m,p,h,port=parse(raw)
 return json.dumps({'outbounds':[{'type':'shadowsocks','tag':'proxy','server':h,'server_port':port,'method':m,'password':p}]})