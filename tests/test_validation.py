from collector.parser import extract
import json,base64
def enc(o): return base64.urlsafe_b64encode(json.dumps(o).encode()).decode().rstrip('=')
def test_bad_vless_rejected(): assert extract('vless://bad@example.com:443')[0]['hard']
def test_good_vless(): assert not extract('vless://11111111-1111-1111-1111-111111111111@example.com:443')[0]['hard']
def test_bad_vmess(): assert extract('vmess://'+enc({'add':'a','port':443,'id':'bad'}))[0]['hard']
def test_custom_json_preserved():
 raw='{"hello":"world","unknown":[1,2],"dns":{"hosts":[]}}'; x=extract(raw)[0]; assert x['kind']=='json-custom' and x['raw']==raw and not x['hard']
def test_xray_preserved():
 raw='{"outbounds":[{"protocol":"freedom","settings":{}}],"dns":{"hosts":[]}}'; x=extract(raw)[0]; assert x['kind']=='json-xray' and x['raw']==raw and not x['hard']
def test_xray_bad_outbound(): assert extract('{"outbounds":[{}]}')[0]['hard']
