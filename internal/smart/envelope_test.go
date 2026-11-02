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
const expectedEnvelope = `{"remarks":"@shenvpn-smart-parallel-v6b","log":{"loglevel":"none","access":"none","error":"none","dnsLog":false},"dns":{"tag":"dns-module","queryStrategy":"UseIPv4","servers":[{"address":"https://1.1.1.1/dns-query","skipFallback":false},{"address":"https://8.8.8.8/dns-query","skipFallback":false},{"address":"https://9.9.9.9/dns-query","skipFallback":false}],"disableFallbackIfMatch":true},"inbounds":[{"tag":"socks","listen":"127.0.0.1","port":10808,"protocol":"socks","settings":{"auth":"noauth","udp":true},"sniffing":{"enabled":true,"destOverride":["http","tls","quic"],"routeOnly":true}}],"outbounds":[],"routing":{"domainStrategy":"AsIs","balancers":[{"tag":"smart-parallel-balancer","selector":["proxy-"],"strategy":{"type":"leastLoad","settings":{"expected":3,"maxRTT":"2s","tolerance":0.15}},"fallbackTag":"block"}],"rules":[{"type":"field","inboundTag":["socks","tun"],"port":"53","network":"tcp,udp","outboundTag":"smart-dns-out"},{"type":"field","inboundTag":["dns-module"],"balancerTag":"smart-parallel-balancer"},{"type":"field","network":"tcp,udp","balancerTag":"smart-parallel-balancer"}]},"burstObservatory":{"subjectSelector":["proxy-"],"pingConfig":{"destination":"https://www.gstatic.com/generate_204","interval":"10s","sampling":1,"timeout":"3s","httpMethod":"GET"}},"policy":{"levels":{"0":{"handshake":15,"connIdle":600,"uplinkOnly":2,"downlinkOnly":5}}},"stats":{}}`

func TestExactOrderedEnvelopeAndAllOutboundValues(t *testing.T) {
	members := []Member{}
	originals := [][]any{}
	total := 0
	for i := 1; i <= 10; i++ {
		m := member(t, i, "A")
		root, _ := object(m.Body)
		nodes := root["outbounds"].([]any)
		// Unknown fields, large numbers and explicitly absent keepalive must survive.
		nodes[0].(M)["extension"] = M{"large": json.Number("9007199254740993"), "array": []any{"z", "a"}}
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
	stripped := bytes.Replace(got, raw["outbounds"], []byte("[]"), 1)
	if string(stripped) != expectedEnvelope {
		t.Fatal("fixed envelope values or field order changed", string(stripped))
	}
	parsed, _ := object(got)
	out := parsed["outbounds"].([]any)
	if len(out) != total+4 {
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
	if len(out) != 7 || out[0].(M)["tag"] != "proxy-01" || out[1].(M)["tag"] != "proxy-02" || out[2].(M)["protocol"] != "loopback" {
		t.Fatal("pool members or helper lost")
	}
	var raw map[string]json.RawMessage
	json.Unmarshal(b, &raw)
	if string(bytes.Replace(b, raw["outbounds"], []byte("[]"), 1)) != expectedEnvelope {
		t.Fatal("source routing leaked")
	}
	validateCore(t, b)
}
