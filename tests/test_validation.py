from collector.parser import extract
import json,base64
def enc(o): return base64.urlsafe_b64encode(json.dumps(o).encode()).decode().rstrip('=')
def test_nonstandard_vless_preserved(): assert not extract('vless://bad@example.com:443')[0]['hard']
def test_good_vless(): assert not extract('vless://11111111-1111-1111-1111-111111111111@example.com:443')[0]['hard']
def test_bad_vmess(): assert extract('vmess://'+enc({'add':'a','port':443,'id':'bad'}))[0]['hard']
def test_custom_json_preserved():
 raw='{"hello":"world","unknown":[1,2],"dns":{"hosts":[]}}'; x=extract(raw)[0]; assert x['kind']=='json-custom' and x['raw']==raw and not x['hard']
def test_xray_preserved():
 raw='{"outbounds":[{"protocol":"freedom","settings":{}}],"dns":{"hosts":[]}}'; x=extract(raw)[0]; assert x['kind']=='json-xray' and x['raw']==raw and not x['hard']
def test_xray_bad_outbound(): assert extract('{"outbounds":[{}]}')[0]['hard']
def test_custom_array_preserved():
 raw='[{"type":"private-app-format","x":1}]'; x=extract(raw)[0]; assert x['kind']=='json-custom' and x['raw']==raw and not x['hard']
def test_xray_stats_array_is_hard_invalid(): assert any(x['code']=='XRAY_STATS_NOT_OBJECT' for x in extract('{"outbounds":[{"protocol":"freedom"}],"stats":[]}')[0]['issues'])
def test_xray_settings_array_is_hard_invalid(): assert any(x['code']=='XRAY_SETTINGS_NOT_OBJECT' for x in extract('{"outbounds":[{"protocol":"freedom","settings":[]}]}')[0]['issues'])
def test_xray_duplicate_tags_invalid(): assert any(x['code']=='XRAY_DUPLICATE_OUTBOUND_TAG' for x in extract('{"outbounds":[{"tag":"x","protocol":"freedom"},{"tag":"x","protocol":"blackhole"}]}')[0]['hard'])
def test_unknown_xray_fields_preserved():
 raw='{"outbounds":[{"protocol":"freedom","futureField":{"a":1}}],"vendor":{"x":true}}'; x=extract(raw)[0]; assert x['raw']==raw and not x['hard']
def test_missing_port(): assert extract('vless://11111111-1111-1111-1111-111111111111@example.com')[0]['hard']
def test_json_unknown_shape_never_mutated():
 raw='{"version":9,"servers":[{"mystery":true}],"extra":null}'; x=extract(raw)[0]; assert x['kind']=='json-custom' and x['raw']==raw and not x['hard']
def test_xray_empty_outbounds_hard(): assert extract('{"outbounds":[]}')[0]['hard']
def test_hard_invalid_never_changes_raw():
 raw='vless://bad@example.com:443?x=1#keep-me'; x=extract(raw)[0]; assert x['raw']==raw and not x['hard']
def test_safe_xray_repair_keeps_original_immutable():
 from collector.repair import repair
 raw='{"outbounds":[{"protocol":"freedom","settings":[]}],"stats":[]}'
 x=repair('json-xray',raw); assert x and raw.endswith('[]}') and '"settings":{}' in x['raw'] and '"stats":{}' in x['raw']
def test_custom_json_never_auto_repaired():
 from collector.repair import repair
 assert repair('json-custom','{"settings":[],"stats":[]}') is None
def test_singbox_detect_preserve():
 raw='{"log":{"level":"info"},"outbounds":[{"type":"direct","tag":"direct"}],"route":{"rules":[]}}'; x=extract(raw)[0]; assert x['kind']=='json-singbox' and x['raw']==raw and not x['hard']
def test_singbox_missing_type_hard(): assert extract('{"outbounds":[{"tag":"x","type":"direct"},{"tag":"y"}]}')[0]['hard']
def test_singbox_duplicate_tag_hard(): assert extract('{"outbounds":[{"type":"direct","tag":"x"},{"type":"block","tag":"x"}]}')[0]['hard']
def test_singbox_unknown_fields_preserved():
 raw='{"outbounds":[{"type":"direct","tag":"d","future":{"x":1}}],"experimental":{"cache_file":{"enabled":true}}}'; x=extract(raw)[0]; assert x['raw']==raw and not x['hard']
def test_hy2_fixture():
 x=extract('hy2://auth@example.com:443?sni=example.com#x')[0]; assert x['kind']=='hy2' and not x['hard']
def test_hy_fixture():
 x=extract('hysteria://auth@example.com:443?sni=example.com#x')[0]; assert x['kind']=='hy' and not x['hard']
def test_wireguard_fixture():
 x=extract('wireguard://key@example.com:51820#x')[0]; assert x['kind']=='wireguard' and not x['hard']
def test_custom_json_fixture_preserved():
 raw='{"vendor":"custom","servers":[{"host":"example.com","port":443}],"opaque":{"a":[1,2]}}'; x=extract(raw)[0]; assert x['kind']=='json-custom' and x['raw']==raw
def test_intelligence_valid_state():
 from collector.intelligence import analyze
 assert analyze('vless','vless://11111111-1111-1111-1111-111111111111@example.com:443')['state']=='valid'
def test_intelligence_warning_state():
 from collector.intelligence import analyze
 assert analyze('vless','vless://custom-id@example.com:443')['state']=='compatible-nonstandard'
def test_intelligence_repairable_xray():
 from collector.intelligence import analyze
 assert analyze('json-xray','{"outbounds":[{"protocol":"freedom","settings":[]}]}')['state']=='repairable'
def test_rulepack_repairs_only_known_safe_codes():
 from collector.rules import policy
 assert policy('XRAY_SETTINGS_NOT_OBJECT')['action']=='repair' and policy('UNKNOWN')['action']=='observe'
def test_regression_gate_repair_is_valid():
 from collector.repair import repair
 from collector.validator import validate,hard_errors
 raw='{"outbounds":[{"protocol":"freedom","settings":[]}],"stats":[]}'
 x=repair('json-xray',raw); assert x and not hard_errors(validate('json-xray',x['raw']))
def test_xray_dns_hosts_array_safe_repair():
 from collector.repair import repair
 raw='{"dns":{"hosts":[]},"outbounds":[{"protocol":"freedom","settings":{}}]}'
 x=extract(raw)[0]; assert any(i['code']=='XRAY_DNS_HOSTS_NOT_OBJECT' for i in x['issues']); r=repair('json-xray',raw); assert '"hosts":{}' in r['raw']
def test_xray_http_inbound_settings_array_repair():
 from collector.repair import repair
 raw='{"inbounds":[{"protocol":"http","settings":[]}],"outbounds":[{"protocol":"freedom","settings":{}}]}'
 r=repair('json-xray',raw); assert r and '"settings":{}' in r['raw']
def test_xray_stream_array_objects_repair():
 from collector.repair import repair
 raw='{"outbounds":[{"protocol":"vless","settings":{},"streamSettings":{"sockopt":[],"realitySettings":[]}}]}'
 r=repair('json-xray',raw); assert r and '"sockopt":{}' in r['raw']
def test_xray_finalmask_zero_warns_but_not_repaired():
 from collector.repair import repair
 raw='{"outbounds":[{"protocol":"vless","settings":{},"streamSettings":{"finalmask":{"tcp":[{"type":"fragment","settings":{"packets":"tlshello","lengths":["0","104","1"],"delays":["0"],"maxSplit":"0"}}]}}}]}'
 x=extract(raw)[0]; assert any(i['code']=='XRAY_FINALMASK_ZERO' for i in x['issues']); assert repair('json-xray',raw) is None
def test_xray_deep_transport_shape_warning():
 raw='{"outbounds":[{"protocol":"vless","settings":{},"streamSettings":{"network":"ws","wsSettings":[]}}]}'
 x=extract(raw)[0]; assert any(i['code']=='XRAY_TRANSPORT_SETTINGS_SHAPE' for i in x['issues']) and not x['hard']
def test_xray_deep_unknown_network_preserved():
 raw='{"outbounds":[{"protocol":"vless","settings":{},"streamSettings":{"network":"future-net","futureSettings":{"x":1}}}]}'
 x=extract(raw)[0]; assert any(i['code']=='XRAY_UNKNOWN_NETWORK' for i in x['issues']) and x['raw']==raw
def test_xray_deep_routing_rules_shape():
 raw='{"routing":{"rules":{}},"outbounds":[{"protocol":"freedom","settings":{}}]}'
 assert any(i['code']=='XRAY_ROUTING_RULES_SHAPE' for i in extract(raw)[0]['issues'])
def test_xray_flattened_vless_preserved():
 raw='{"outbounds":[{"protocol":"vless","settings":{"address":"a.example","port":443,"id":"custom","encryption":"none"}}]}'
 x=extract(raw)[0]; assert not any(i['code']=='XRAY_VNEXT_MISSING' for i in x['issues'])
def test_xray_protocol_missing_vnext_warns():
 raw='{"outbounds":[{"protocol":"vless","settings":{}}]}'
 assert any(i['code']=='XRAY_VNEXT_MISSING' for i in extract(raw)[0]['issues'])
def test_xray_system_dns_shapes_warn():
 raw='{"dns":{"servers":"1.1.1.1","hosts":[]},"outbounds":[{"protocol":"freedom","settings":{}}]}'
 codes={i['code'] for i in extract(raw)[0]['issues']}; assert 'XRAY_DNS_SERVERS_SHAPE' in codes and 'XRAY_DNS_HOSTS_SHAPE' in codes
def test_xray_system_inbound_duplicate_tag_warn():
 raw='{"inbounds":[{"tag":"x","protocol":"socks"},{"tag":"x","protocol":"http"}],"outbounds":[{"protocol":"freedom","settings":{}}]}'
 assert any(i['code']=='XRAY_DUPLICATE_INBOUND_TAG' for i in extract(raw)[0]['issues'])
def test_xray_system_policy_api_shape_warn():
 raw='{"policy":[],"api":[],"outbounds":[{"protocol":"freedom","settings":{}}]}'
 codes={i['code'] for i in extract(raw)[0]['issues']}; assert {'XRAY_POLICY_SHAPE','XRAY_API_SHAPE'}<=codes
def test_xray_refs_missing_outbound_warns():
 raw='{"routing":{"rules":[{"outboundTag":"missing"}]},"outbounds":[{"tag":"direct","protocol":"freedom","settings":{}}]}'
 assert any(i['code']=='XRAY_ROUTE_OUTBOUND_REF' for i in extract(raw)[0]['issues'])
def test_xray_refs_proxy_missing_warns():
 raw='{"outbounds":[{"tag":"proxy","protocol":"freedom","settings":{},"proxySettings":{"tag":"gone"}}]}'
 assert any(i['code']=='XRAY_PROXY_REF' for i in extract(raw)[0]['issues'])
def test_xray_refs_virtual_dns_tags_preserved():
 raw='{"routing":{"rules":[{"inboundTag":["dns-proxy","dns-direct"],"outboundTag":"api"}]},"outbounds":[{"tag":"direct","protocol":"freedom","settings":{}}]}'
 codes={i['code'] for i in extract(raw)[0]['issues']}; assert 'XRAY_ROUTE_INBOUND_REF' not in codes and 'XRAY_ROUTE_OUTBOUND_REF' not in codes
def test_xray_mutation_core_shapes_detected():
 from collector.validator import validate
 import json
 cases=['{"outbounds":{}}','{"outbounds":[]}','{"outbounds":[{}]}','{"outbounds":[{"protocol":"freedom","settings":[]}]}']
 assert all(validate('json-xray',x) for x in cases)
def test_xray_mutation_broken_reference_detected():
 from collector.validator import validate
 raw='{"routing":{"rules":[{"outboundTag":"gone"}]},"outbounds":[{"tag":"direct","protocol":"freedom","settings":{}}]}'
 assert any(i['code']=='XRAY_ROUTE_OUTBOUND_REF' for i in validate('json-xray',raw))
def test_xray_advanced_mutations_detected():
 from collector.validator import validate
 cases=[
 '{"outbounds":[{"protocol":"vless","settings":{"vnext":[{"address":"","port":443,"users":[{"id":"x"}]}]}}]}',
 '{"outbounds":[{"protocol":"vless","settings":{"vnext":[{"address":"a","port":70000,"users":[]}]}}]}',
 '{"outbounds":[{"protocol":"vless","settings":{},"streamSettings":{"network":"ws","wsSettings":[]}}]}' ]
 assert all(validate('json-xray',x) for x in cases)
def test_singbox_deep_server_and_port():
 raw='{"outbounds":[{"type":"vless","tag":"p","server":"","server_port":70000}]}'
 codes={i['code'] for i in extract(raw)[0]['issues']}; assert {'SINGBOX_SERVER_MISSING','SINGBOX_PORT_INVALID'}<=codes
def test_singbox_deep_tls_transport_shapes():
 raw='{"outbounds":[{"type":"vless","server":"a","server_port":443,"tls":[],"transport":[]}]}'
 codes={i['code'] for i in extract(raw)[0]['issues']}; assert {'SINGBOX_TLS_SHAPE','SINGBOX_TRANSPORT_SHAPE'}<=codes
def test_singbox_deep_route_dns_shapes():
 raw='{"outbounds":[{"type":"direct","tag":"d"}],"route":[],"dns":[]}'
 codes={i['code'] for i in extract(raw)[0]['issues']}; assert {'SINGBOX_ROUTE_SHAPE','SINGBOX_DNS_SHAPE'}<=codes
def test_singbox_protocol_credentials():
 cases=[
 '{"outbounds":[{"type":"vless","server":"a","server_port":443,"uuid":"bad"}]}',
 '{"outbounds":[{"type":"trojan","server":"a","server_port":443}]}',
 '{"outbounds":[{"type":"shadowsocks","server":"a","server_port":443,"password":"x"}]}',
 '{"outbounds":[{"type":"hysteria2","server":"a","server_port":443}]}',
 '{"outbounds":[{"type":"tuic","server":"a","server_port":443,"uuid":"x"}]}' ]
 assert all(extract(x)[0]['issues'] for x in cases)
def test_singbox_transport_type_missing_warns():
 raw='{"outbounds":[{"type":"vless","server":"a","server_port":443,"uuid":"00000000-0000-0000-0000-000000000000","transport":{}}]}'
 assert any(i['code']=='SINGBOX_TRANSPORT_TYPE_MISSING' for i in extract(raw)[0]['issues'])
def test_singbox_refs_detour_and_route():
 raw='{"inbounds":[{"type":"mixed","tag":"in"}],"outbounds":[{"type":"direct","tag":"d","detour":"gone"}],"route":{"rules":[{"inbound":["missing"],"outbound":"gone"}]}}'
 codes={i['code'] for i in extract(raw)[0]['issues']}; assert {'SINGBOX_DETOUR_REF','SINGBOX_ROUTE_OUTBOUND_REF','SINGBOX_ROUTE_INBOUND_REF'}<=codes
def test_singbox_dns_reference():
 raw='{"outbounds":[{"type":"direct","tag":"d"}],"dns":{"servers":[{"tag":"cf","address":"1.1.1.1"}],"rules":[{"server":"gone"}]}}'
 assert any(i['code']=='SINGBOX_DNS_SERVER_REF' for i in extract(raw)[0]['issues'])
def test_singbox_refs_valid_links_clean():
 raw='{"inbounds":[{"type":"mixed","tag":"in"}],"outbounds":[{"type":"direct","tag":"d"}],"route":{"rules":[{"inbound":["in"],"outbound":"d"}]},"dns":{"servers":[{"tag":"cf","address":"1.1.1.1"}],"rules":[{"server":"cf"}]}}'
 codes={i['code'] for i in extract(raw)[0]['issues']}; assert not any(x.startswith('SINGBOX_ROUTE_') or x=='SINGBOX_DNS_SERVER_REF' for x in codes)
def test_singbox_fuzz_key_mutations_detected():
 from collector.validator import validate
 cases=[
 '{"outbounds":[{"type":"vless","server":"","server_port":443,"uuid":"bad"}]}',
 '{"outbounds":[{"type":"vless","server":"a","server_port":70000,"uuid":"00000000-0000-0000-0000-000000000000"}]}',
 '{"outbounds":[{"type":"direct","tag":"d","detour":"gone"}]}' ]
 assert all(validate('json-singbox',x) for x in cases)
def test_singbox_core_checker_valid_and_invalid():
 import json
 from collector.singbox_core import check
 assert check(json.dumps({'outbounds':[{'type':'direct','tag':'direct'}],'route':{'final':'direct'}}))['ok']
 assert not check(json.dumps({'outbounds':[{'type':'vless','server':'a','server_port':70000,'uuid':'bad'}]}))['ok']
def test_vless_deep_reality_and_grpc_warnings():
 from collector.validator import validate
 raw='vless://00000000-0000-0000-0000-000000000000@example.com:443?type=grpc&security=reality'
 codes={i['code'] for i in validate('vless',raw)}; assert {'VLESS_GRPC_SERVICE_MISSING','VLESS_SNI_MISSING','VLESS_REALITY_KEY_MISSING'}<=codes
def test_vless_unknown_transport_preserved():
 from collector.validator import validate
 raw='vless://00000000-0000-0000-0000-000000000000@example.com:443?type=futuretransport&security=none#x'
 z=validate('vless',raw); assert any(i['code']=='VLESS_UNKNOWN_TRANSPORT' for i in z)
def test_vless_raw_query_preserved():
 raw='vless://00000000-0000-0000-0000-000000000000@example.com:443?type=ws&path=%2Fa%3Fb%3D1&host=x.example&extra=future#name'
 assert extract(raw)[0]['raw']==raw
def test_vless_ipv6_and_reality_full_clean():
 from collector.validator import validate
 u='00000000-0000-0000-0000-000000000000'
 assert not validate('vless',f'vless://{u}@[2001:db8::1]:443?type=tcp&security=none')
 assert not validate('vless',f'vless://{u}@example.com:443?type=tcp&security=reality&sni=x&pbk=abc&sid=01&spx=%2F&fp=chrome')
def test_vless_duplicate_query_and_bad_percent_warn():
 from collector.validator import validate
 u='00000000-0000-0000-0000-000000000000'
 a=validate('vless',f'vless://{u}@example.com:443?type=ws&type=grpc');b=validate('vless',f'vless://{u}@example.com:443?type=ws&path=%ZZ')
 assert any(x['code']=='VLESS_DUPLICATE_QUERY' for x in a) and any(x['code']=='VLESS_BAD_PERCENT_ENCODING' for x in b)
def test_vless_validation_adapters_do_not_mutate_raw():
 from collector.vless_adapter import xray,singbox
 raw='vless://00000000-0000-0000-0000-000000000000@example.com:443?type=ws&security=tls&sni=x&path=%2Fa#n'; before=raw
 assert 'outbounds' in xray(raw) and 'outbounds' in singbox(raw) and raw==before
def test_vless_reality_singbox_adapter_has_utls():
 import json
 from collector.vless_adapter import singbox
 raw='vless://00000000-0000-0000-0000-000000000000@example.com:443?type=tcp&security=reality&sni=x&pbk=abc&fp=chrome'
 o=json.loads(singbox(raw))['outbounds'][0]; assert o['tls']['utls']['enabled'] and o['tls']['reality']['enabled']
def test_vless_reality_missing_key_is_hard():
 from collector.validator import validate,hard_errors
 raw='vless://00000000-0000-0000-0000-000000000000@example.com:443?type=xhttp&security=reality&sni='
 assert any(i['code']=='VLESS_REALITY_KEY_MISSING' for i in hard_errors(validate('vless',raw)))
def test_vless_reality_missing_key_never_enters_healthy_extract():
 from collector.validator import hard_errors
 raw='vless://00000000-0000-0000-0000-000000000000@example.com:443?type=tcp&security=reality&sni=x'
 x=extract(raw)[0]; assert hard_errors(x['issues']) and x['hard']
def test_vmess_deep_fuzz_fields():
 import json,base64
 from collector.validator import validate
 def e(o):return 'vmess://'+base64.urlsafe_b64encode(json.dumps(o).encode()).decode().rstrip('=')
 b={'add':'a','port':'443','id':'00000000-0000-0000-0000-000000000000','net':'tcp'}
 assert any(x['code']=='VMESS_UNKNOWN_NETWORK' for x in validate('vmess',e({**b,'net':'future'})))
 assert any(x['code']=='VMESS_ALTERID_INVALID' for x in validate('vmess',e({**b,'aid':'x'})))
def test_vmess_raw_preserved():
 import json,base64
 o={'add':'a','port':'443','id':'00000000-0000-0000-0000-000000000000','net':'ws','extra':'future'};raw='vmess://'+base64.urlsafe_b64encode(json.dumps(o).encode()).decode().rstrip('=')
 assert extract(raw)[0]['raw']==raw
def test_vmess_validation_adapters_preserve_raw():
 from collector.vmess_adapter import xray,singbox
 import json,base64
 o={'add':'a','port':'443','id':'00000000-0000-0000-0000-000000000000','net':'ws','path':'/'};raw='vmess://'+base64.urlsafe_b64encode(json.dumps(o).encode()).decode().rstrip('=');before=raw
 assert 'outbounds' in xray(raw) and 'outbounds' in singbox(raw) and raw==before
def test_trojan_deep_fuzz_fields():
 from collector.validator import validate
 b='trojan://pass@example.com:443'
 assert any(x['code']=='TROJAN_UNKNOWN_TRANSPORT' for x in validate('trojan',b+'?type=future&sni=x'))
 assert any(x['code']=='TROJAN_GRPC_SERVICE_MISSING' for x in validate('trojan',b+'?type=grpc&sni=x'))
 assert any(x['code']=='TROJAN_DUPLICATE_QUERY' for x in validate('trojan',b+'?type=ws&type=grpc&sni=x'))
def test_trojan_raw_preserved():
 raw='trojan://pass@example.com:443?type=ws&security=tls&sni=x&path=%2Fa&extra=future#n';assert extract(raw)[0]['raw']==raw
def test_trojan_validation_adapters_preserve_raw():
 from collector.trojan_adapter import xray,singbox
 raw='trojan://pass@example.com:443?type=ws&security=tls&sni=x&path=%2F#n';before=raw
 assert 'outbounds' in xray(raw) and 'outbounds' in singbox(raw) and raw==before
def test_ss_sip002_and_legacy_preserved():
 import base64
 sip='ss://'+base64.urlsafe_b64encode(b'aes-128-gcm:password').decode().rstrip('=')+'@example.com:443#x'
 legacy='ss://'+base64.urlsafe_b64encode(b'aes-128-gcm:password@example.com:443').decode().rstrip('=')+'#x'
 assert not extract(sip)[0]['hard'] and not extract(legacy)[0]['hard'] and extract(sip)[0]['raw']==sip
def test_ss_unknown_method_warning():
 import base64
 raw='ss://'+base64.urlsafe_b64encode(b'future-cipher:pass').decode().rstrip('=')+'@example.com:443'
 assert any(x['code']=='SS_UNKNOWN_METHOD' for x in extract(raw)[0]['issues'])
def test_ss_opaque_userinfo_is_warning_only():
 raw='ss://18f1b94e-35f1-4c7f-953c-7b4681c52339@example.com:443?encryption=none&type=tcp'
 x=extract(raw)[0];assert not x['hard'] and any(i['code']=='SS_OPAQUE_USERINFO' for i in x['issues'])
def test_ss_adapter_rejects_opaque_for_core_only():
 from collector.ss_deep import parse
 raw='ss://18f1b94e-35f1-4c7f-953c-7b4681c52339@example.com:443?encryption=none&type=tcp'
 try:parse(raw);assert False
 except ValueError:pass
def test_ss_classifier_identifies_vless_like_without_mutation():
 from collector.ss_classifier import classify
 raw='ss://00000000-0000-0000-0000-000000000000@example.com:443?encryption=none&security=none&type=tcp&headerType=http';before=raw
 x=classify(raw);assert x['class']=='vless-like-mislabeled' and x['suggested_scheme']=='vless' and raw==before
def test_ss_recovery_candidate_does_not_mutate_raw():
 from collector.ss_recovery import candidate
 raw='ss://00000000-0000-0000-0000-000000000000@example.com:443?encryption=none&security=none&type=tcp';before=raw
 v=candidate(raw);assert v.startswith('vless://') and raw==before
def test_recovered_pool_never_labels_core_pass_as_healthy():
 from collector.recovered_pool import sync
 x=sync();assert x['status']=='structurally_verified' and 'healthy' not in x['status']
def test_hy2_deep_valid_and_missing_auth():
 from collector.validator import validate,hard_errors
 good='hy2://pass@example.com:443?sni=x.example';bad='hy2://@example.com:443?sni=x.example'
 assert not hard_errors(validate('hy2',good)) and any(x['code'] in ('HY_MISSING_AUTH','HY2_AUTH_MISSING') for x in hard_errors(validate('hy2',bad)))
def test_hy1_query_auth_and_raw_preserved():
 raw='hy://example.com:443?auth=token&peer=x.example&upmbps=10&downmbps=20#n';x=extract(raw)[0]
 assert not x['hard'] and x['raw']==raw
def test_hysteria_bad_port_detected():
 from collector.validator import validate,hard_errors
 assert hard_errors(validate('hy2','hy2://pass@example.com:70000?sni=x'))
def test_wireguard_deep_required_fields():
 from collector.validator import validate,hard_errors
 bad='wireguard://@example.com:51820';assert hard_errors(validate('wireguard',bad))
def test_wireguard_raw_preserved():
 import base64
 k=base64.b64encode(b'x'*32).decode();raw=f'wireguard://{k}@example.com:51820?publickey={k}&address=10.0.0.2%2F32#n';x=extract(raw)[0];assert x['raw']==raw and not x['hard']
def test_health_requires_both_upload_and_download():
 from collector.test_queue import mark_result
 assert not mark_result('__missing__',True,False) and not mark_result('__missing__',False,True)
def test_health_policy_requires_bidirectional_success():
 from collector.health_policy import accepted,initial_action,recheck_action
 assert accepted(True,True) and not accepted(True,False) and not accepted(False,True)
 assert initial_action(1,False,True)=='retry_after_30s' and initial_action(2,True,False)=='delete'
 assert recheck_action(True,True)=='keep' and recheck_action(True,False)=='delete'
def test_health_policy_location_only_after_health():
 from collector.health_policy import POLICY
 assert POLICY['location_after_health_only'] and POLICY['cdn_classification'] and POLICY['healthy_recheck_seconds']==300
def test_sandbox_uses_loopback_ephemeral_port():
 from collector.network_sandbox import free_port
 p=free_port();assert 1024<p<65536
def test_probe_policy_requires_real_bytes_for_download():
 from collector.health_policy import initial_action
 assert initial_action(1,True,False)=='retry_after_30s' and initial_action(2,True,False)=='delete'
def test_health_policy_requires_traffic_plus_both_directions():
 from collector.health_policy import accepted
 assert accepted(True,True) and not accepted(True,False) and not accepted(False,True)
def test_provider_health_schema():
 from collector.provider_health import PROVIDERS
 assert len(PROVIDERS['download'])>=3 and len(PROVIDERS['upload'])>=1
def test_provider_catalog_has_multi_region_duplex_endpoints():
 from collector.provider_catalog import load
 x=load();assert len(x)>=20 and all('download' in a and 'upload' in a for a in x)
