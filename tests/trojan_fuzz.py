from collector.validator import validate
B='trojan://pass@example.com:443'
cases={'good':B+'?type=ws&security=tls&sni=x&path=%2F','missing_pass':'trojan://@example.com:443?security=tls&sni=x','bad_port':'trojan://pass@example.com:70000','unknown_type':B+'?type=future&security=tls&sni=x','grpc_missing':B+'?type=grpc&security=tls&sni=x','sni_missing':B+'?type=tcp&security=tls','dup':B+'?type=ws&type=grpc&sni=x','bad_percent':B+'?type=ws&sni=x&path=%ZZ'}
for n,r in cases.items():print(n,[x['code'] for x in validate('trojan',r)])