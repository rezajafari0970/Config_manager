import json,base64
from collector.validator import validate
def enc(o):return 'vmess://'+base64.urlsafe_b64encode(json.dumps(o).encode()).decode().rstrip('=')
B={'v':'2','ps':'x','add':'example.com','port':'443','id':'00000000-0000-0000-0000-000000000000','aid':'0','scy':'auto','net':'ws','type':'none','host':'x.example','path':'/','tls':'tls','sni':'x.example'}
cases={
'good':B,
'bad_port':{**B,'port':'70000'},
'bad_uuid':{**B,'id':'bad'},
'unknown_net':{**B,'net':'future'},
'grpc_missing':{**B,'net':'grpc','path':''},
'bad_path':{**B,'path':[]},
'bad_aid':{**B,'aid':'x'},
'unknown_tls':{**B,'tls':'reality'}}
for n,o in cases.items():print(n,[x['code'] for x in validate('vmess',enc(o))])