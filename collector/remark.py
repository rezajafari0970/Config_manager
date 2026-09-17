import json,urllib.parse,base64
def apply(kind,raw,remark):
 if kind in ('vless','trojan','ss','hy','hy2','wireguard'):
  base=raw.split('#',1)[0];return base+'#'+urllib.parse.quote(remark,safe='')
 if kind=='vmess':
  try:
   b=raw.split('://',1)[1];o=json.loads(base64.urlsafe_b64decode(b+'==='));o['ps']=remark;return 'vmess://'+base64.urlsafe_b64encode(json.dumps(o,separators=(',',':')).encode()).decode().rstrip('=')
  except:return raw
 if kind.startswith('json-'):
  try:o=json.loads(raw);o['remarks']=remark;return json.dumps(o,separators=(',',':'),ensure_ascii=False)
  except:return raw
 return raw