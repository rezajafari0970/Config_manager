package smart

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// Independent contract fixture for the explicitly revised responsive envelope.
const expectedEnvelope = `{"remarks":"@shenvpn-smart-parallel-v6b","log":{"loglevel":"none","access":"none","error":"none","dnsLog":false},"dns":{"tag":"dns-module","queryStrategy":"UseIPv4","servers":[{"address":"https://8.8.8.8/dns-query","domains":[],"skipFallback":true,"tag":"dns-bootstrap","timeoutMs":2500},{"address":"https://1.1.1.1/dns-query","domains":[],"skipFallback":true,"tag":"dns-bootstrap","timeoutMs":2500},{"address":"https://1.1.1.1/dns-query","skipFallback":false,"timeoutMs":2500},{"address":"https://8.8.8.8/dns-query","skipFallback":false,"timeoutMs":2500}],"disableFallbackIfMatch":true,"disableCache":false,"serveStale":true,"serveExpiredTTL":60,"enableParallelQuery":true},"inbounds":[{"tag":"socks","listen":"127.0.0.1","port":10808,"protocol":"socks","settings":{"auth":"noauth","udp":true},"sniffing":{"enabled":true,"destOverride":["http","tls","quic"],"routeOnly":true}}],"outbounds":[],"routing":{"domainStrategy":"AsIs","balancers":[{"tag":"smart-parallel-balancer","selector":["proxy-"],"strategy":{"type":"leastLoad","settings":{"expected":3,"maxRTT":"2s","tolerance":0.15}},"fallbackTag":"smart-recovery"},{"tag":"bootstrap-ip-balancer","selector":[],"strategy":{"type":"leastLoad","settings":{"expected":3,"maxRTT":"2s","tolerance":0.15}},"fallbackTag":"bootstrap-recovery"},{"tag":"smart-recovery-balancer","selector":["proxy-"],"strategy":{"type":"leastLoad","settings":{"expected":1,"maxRTT":"10s","tolerance":0.15}},"fallbackTag":"block"},{"tag":"bootstrap-recovery-balancer","selector":[],"strategy":{"type":"leastLoad","settings":{"expected":1,"maxRTT":"10s","tolerance":0.15}},"fallbackTag":"block"}],"rules":[{"type":"field","inboundTag":["bootstrap-recovery-egress"],"network":"tcp,udp","balancerTag":"bootstrap-recovery-balancer"},{"type":"field","inboundTag":["smart-recovery-egress"],"network":"tcp,udp","balancerTag":"smart-recovery-balancer"},{"type":"field","inboundTag":["socks","tun"],"port":"53","network":"tcp,udp","outboundTag":"smart-dns-out"},{"type":"field","inboundTag":["dns-bootstrap"],"balancerTag":"bootstrap-ip-balancer"},{"type":"field","inboundTag":["dns-module","smart-dns-egress"],"balancerTag":"smart-parallel-balancer"},{"type":"field","network":"tcp,udp","balancerTag":"smart-parallel-balancer"}]},"burstObservatory":{"subjectSelector":["proxy-"],"pingConfig":{"destination":"https://speed.cloudflare.com/__down?bytes=65536","interval":"15s","sampling":2,"timeout":"10s","httpMethod":"GET"}},"policy":{"levels":{"0":{"handshake":15,"connIdle":600,"uplinkOnly":2,"downlinkOnly":5}},"system":{"statsInboundUplink":true,"statsInboundDownlink":true,"statsOutboundUplink":true,"statsOutboundDownlink":true}},"stats":{},"metrics":{"tag":"debug-metrics","listen":"127.0.0.1:11111"}}`

func TestExactOrderedEnvelopeAndAllOutboundValues(t *testing.T) {
	members := []Member{}
	originals := [][]any{}
	total := 0
	for i := 1; i <= 10; i++ {
		m := member(t, i, "A")
		root, _ := object(m.Body)
		nodes := root["outbounds"].([]any)
		// Unknown fields, numbers and null Mux survive the explicit socket-policy override.
		nodes[0].(M)["extension"] = M{"large": json.Number("9007199254740993"), "array": []any{"z", "a"}, "domains": []any{}, "selector": []any{}}
		nodes[0].(M)["mux"] = nil
		stream := nodes[0].(M)["streamSettings"].(M)
		delete(stream["sockopt"].(M), "tcpKeepAliveIdle")
		m.Body, _ = json.Marshal(root)
		original, _ := object(m.Body)
		originals = append(originals, original["outbounds"].([]any))
		members = append(members, m)
		total += len(nodes)
	}
	got, err := Compose(members)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if json.Unmarshal(got, &raw) != nil || bytes.ContainsAny(got, "\r\n") {
		t.Fatal("not compact JSON")
	}
	stripped := stripDynamicEnvelope(t, got)
	gotEnvelope, _ := object(stripped)
	wantEnvelope, _ := object([]byte(expectedEnvelope))
	if !reflect.DeepEqual(gotEnvelope, wantEnvelope) {
		t.Fatal("fixed envelope values or field order changed", string(stripped))
	}
	parsed, _ := object(got)
	out := parsed["outbounds"].([]any)
	if len(out) != total+5 {
		t.Fatal("outbound omitted or invented", len(out), total)
	}
	byTag := map[string]M{}
	for _, v := range out {
		ob := v.(M)
		tag := ob["tag"].(string)
		if byTag[tag] != nil {
			t.Fatal("duplicate tag")
		}
		byTag[tag] = ob
	}
	for slot, nodes := range originals {
		mapping := map[string]string{}
		for i, v := range nodes {
			old := v.(M)["tag"].(string)
			next := fmt.Sprintf("member-%04d-out-%04d_", slot+1, i)
			if i == 0 {
				next = fmt.Sprintf("proxy-%02d", slot+1)
			}
			mapping[next] = old
		}
		for next, old := range mapping {
			copyBytes, _ := json.Marshal(byTag[next])
			copy, _ := object(copyBytes)
			copy["tag"] = old
			if e := rewriteReferences(copy, mapping, ""); e != nil {
				t.Fatal(e)
			}
			var original M
			for _, v := range nodes {
				if v.(M)["tag"] == old {
					original = v.(M)
				}
			}
			applySmartSocketPolicy(original)
			if !reflect.DeepEqual(copy, original) {
				t.Fatal("copied outbound changed beyond tag references", slot, old)
			}
		}
	}
	if !bytes.Contains(got, []byte(`{"tag":"proxy-01","protocol":"vless","settings":`)) {
		t.Fatal("outbound field order")
	}
	again, err := Compose(members)
	if err != nil || !bytes.Equal(got, again) {
		t.Fatal("serialization not deterministic")
	}
}
func TestFixedEnvelopeRefusesActiveLoopbackAndDependencyCycles(t *testing.T) {
	for _, mode := range []string{"loopback", "cycle"} {
		m := member(t, 1, "A")
		root, _ := object(m.Body)
		nodes := root["outbounds"].([]any)
		first := nodes[0].(M)
		if mode == "loopback" {
			first["protocol"] = "loopback"
			first["settings"] = M{"inboundTag": "old-route"}
		} else {
			first["proxySettings"] = M{"tag": first["tag"]}
		}
		m.Body, _ = json.Marshal(root)
		_, err := Compose([]Member{m})
		if err == nil || (mode == "loopback" && !strings.Contains(err.Error(), "source_routing")) {
			t.Fatal("unsupported dependency silently exported", mode, err)
		}
	}
}
func TestBalancedAllEntriesAndInactiveHelpersPreserved(t *testing.T) {
	m := member(t, 1, "A")
	root, _ := object(m.Body)
	first := root["outbounds"].([]any)[0].(M)
	first["tag"] = "node-a"
	secondBytes, _ := json.Marshal(first)
	second, _ := object(secondBytes)
	second["tag"] = "node-b"
	helper := M{"tag": "old-loop", "protocol": "loopback", "settings": M{"inboundTag": "old-in"}}
	root["outbounds"] = []any{first, second, helper}
	root["routing"] = M{"balancers": []any{M{"tag": "old-pool", "selector": []string{"node-"}}}, "rules": []any{M{"type": "field", "network": "tcp,udp", "balancerTag": "old-pool"}}}
	m.Body, _ = json.Marshal(root)
	b, err := Compose([]Member{m})
	if err != nil {
		t.Fatal(err)
	}
	parsed, _ := object(b)
	out := parsed["outbounds"].([]any)
	if len(out) != 8 || out[0].(M)["tag"] != "proxy-01" || out[1].(M)["tag"] != "proxy-02" || out[2].(M)["protocol"] != "loopback" || out[3].(M)["tag"] != "block" {
		t.Fatal("pool members or helper lost")
	}
	var raw map[string]json.RawMessage
	json.Unmarshal(b, &raw)
	gotEnvelope, _ := object(stripDynamicEnvelope(t, b))
	wantEnvelope, _ := object([]byte(expectedEnvelope))
	if !reflect.DeepEqual(gotEnvelope, wantEnvelope) {
		t.Fatal("source routing leaked")
	}
	validateCore(t, b)
}

func stripDynamicEnvelope(t *testing.T, b []byte) []byte {
	t.Helper()
	var raw map[string]json.RawMessage
	if e := json.Unmarshal(b, &raw); e != nil {
		t.Fatal(e)
	}
	root, _ := object(b)
	selector := root["routing"].(M)["balancers"].([]any)[1].(M)["selector"]
	selected, _ := json.Marshal(selector)
	out := bytes.Replace(b, raw["outbounds"], []byte("[]"), 1)
	return bytes.ReplaceAll(out, append([]byte("\"selector\":"), selected...), []byte("\"selector\":[]"))
}
