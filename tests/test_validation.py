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
