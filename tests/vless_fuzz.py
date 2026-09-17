from collector.validator import validate
U='00000000-0000-0000-0000-000000000000'
cases={
'ipv6':f'vless://{U}@[2001:db8::1]:443?type=tcp&security=none#x',
'bad_port':f'vless://{U}@example.com:70000?type=tcp',
'bad_id':'vless://custom-id@example.com:443?type=tcp',
'unknown_transport':f'vless://{U}@example.com:443?type=future',
'reality_missing':f'vless://{U}@example.com:443?type=tcp&security=reality',
'reality_full':f'vless://{U}@example.com:443?type=tcp&security=reality&sni=x.example&pbk=abc&sid=01&spx=%2F&fp=chrome',
'grpc':f'vless://{U}@example.com:443?type=grpc&serviceName=svc&mode=multi',
'ws':f'vless://{U}@example.com:443?type=ws&path=%2Fa%3Fb%3D1&host=x.example',
'xhttp':f'vless://{U}@example.com:443?type=xhttp&path=%2Fx&mode=auto',
'duplicate_type':f'vless://{U}@example.com:443?type=ws&type=grpc&path=%2F',
'bad_percent':f'vless://{U}@example.com:443?type=ws&path=%ZZ'}
for n,r in cases.items():print(n,[x['code'] for x in validate('vless',r)])