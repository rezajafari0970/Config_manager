package convert

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"testing"
)

func assertTrafficEnvelope(t *testing.T, root M) {
	t.Helper()
	inbound := root["inbounds"].([]any)[0].(M)
	if inbound["settings"].(M)["udp"] != true {
		t.Fatal("SOCKS UDP disabled")
	}
	sniff := inbound["sniffing"].(M)
	if sniff["enabled"] != true || !bytes.Contains(encode(t, sniff["destOverride"]), []byte(`"quic"`)) {
		t.Fatal("QUIC sniffing lost")
	}
	rules := root["routing"].(M)["rules"].([]any)
	last := rules[len(rules)-1].(M)
	if last["network"] != "tcp,udp" || (last["outboundTag"] != "proxy" && last["balancerTag"] == nil) {
		t.Fatal("TCP/UDP default route lost")
	}
	for _, v := range rules {
		r := v.(M)
		if r["outboundTag"] == "block" {
			t.Fatal("traffic blocked")
		}
	}
}

func TestAllTemplatesPermitTCPUDPAndQUICRouting(t *testing.T) {
	for name, raw := range fixtures() {
		t.Run(name, func(t *testing.T) {
			p := Prepare([]byte(raw))
			if len(p.Candidates) == 0 {
				t.Fatal(p.Code)
			}
			root, e := object(p.Candidates[0].Config)
			if e != nil {
				t.Fatal(e)
			}
			assertTrafficEnvelope(t, root)
			ob := root["outbounds"].([]any)[0].(M)
			if _, ok := ob["mux"]; ok && name != "vless-explicit-xudp" {
				t.Fatal("unsolicited multiplexing")
			}
			if name == "vless-reality" && !bytes.Contains(p.Candidates[0].Config, []byte(`"flow":"xtls-rprx-vision-udp443"`)) {
				t.Fatal("Vision still blocks UDP443")
			}
		})
	}
}

func TestTrafficPolicyCoversFlatNestedChainsAndPools(t *testing.T) {
	m := balancedFixture("random")
	nodes := m["outbounds"].([]any)
	nodes[0].(M)["settings"].(M)["flow"] = "xtls-rprx-vision"
	nodes[0].(M)["mux"] = M{"enabled": true, "concurrency": json.Number("-1"), "xudpConcurrency": json.Number("-1")}
	nodes[1].(M)["proxySettings"] = M{"tag": "hop"}
	nodes = append(nodes, M{"tag": "hop", "protocol": "vless", "settings": M{"vnext": []any{M{"address": "192.0.2.3", "port": 443, "users": []any{M{"id": "11111111-1111-4111-8111-111111111111", "flow": "xtls-rprx-vision", "encryption": "none"}}}}}, "mux": M{"enabled": true, "xudpProxyUDP443": "reject"}})
	m["outbounds"] = nodes
	raw := encode(t, m)
	p := Prepare(raw)
	if len(p.Candidates) == 0 {
		t.Fatal(p.Code)
	}
	root, _ := object(p.Candidates[0].Config)
	assertTrafficEnvelope(t, root)
	vision := 0
	for _, v := range root["outbounds"].([]any) {
		ob := v.(M)
		if ob["protocol"] != "vless" {
			continue
		}
		if bytes.Contains(encode(t, ob), []byte(`"flow":"xtls-rprx-vision-udp443"`)) {
			vision++
		}
		if ob["mux"].(M)["xudpProxyUDP443"] != "allow" {
			t.Fatal("mux UDP443 blocked")
		}
	}
	if vision != 2 {
		t.Fatal("pool or chain gate missed")
	}
	if !bytes.Equal(raw, encode(t, m)) {
		t.Fatal("raw input mutated")
	}
	again, e := Apply(p.Candidates[0].Config)
	if e != nil || !bytes.Equal(again, p.Candidates[0].Config) {
		t.Fatal("not idempotent", e)
	}
	for _, policy := range []string{"allow", "skip", "unexpected"} {
		single := M{"outbounds": []any{M{"protocol": "socks", "settings": M{"address": "192.0.2.1", "port": 1080}, "mux": M{"enabled": false, "xudpProxyUDP443": policy}}}}
		candidate, e := Apply(encode(t, single))
		if policy == "unexpected" {
			if e == nil {
				t.Fatal("unknown policy silently discarded")
			}
			continue
		}
		if e != nil || !bytes.Contains(candidate, []byte(`"xudpProxyUDP443":"`+policy+`"`)) {
			t.Fatal("explicit allowed policy changed", e)
		}
	}
}

func TestTrafficPolicyPreservesNullMux(t *testing.T) {
	raw := []byte(`{"outbounds":[{"protocol":"socks","settings":{"address":"192.0.2.1","port":1080},"mux":null}]}`)
	b, e := Apply(raw)
	if e != nil || !bytes.Contains(b, []byte(`"mux":null`)) {
		t.Fatal("null mux must remain disabled", e)
	}
	invalid := bytes.Replace(raw, []byte(`"mux":null`), []byte(`"mux":[]`), 1)
	if _, e = Apply(invalid); e == nil {
		t.Fatal("non-object mux accepted")
	}
}

// Optional root-only integration exercises real SOCKS TCP and UDP packets in
// the existing default-drop namespace/bwrap sandbox, using synthetic TLS peers.
func TestPinnedCoreTrafficForwarding(t *testing.T) {
	if os.Getenv("XRAY_TRAFFIC_TEST") != "1" {
		t.Skip("explicit isolated network test required")
	}
	core := os.Getenv("XRAY_TEST_BINARY")
	if core == "" {
		t.Fatal("pinned core required")
	}
	for _, mode := range []string{"vision", "mux", "xudp"} {
		t.Run(mode, func(t *testing.T) {
			root, _ := object([]byte(envelope))
			flow := "xtls-rprx-vision"
			if mode == "mux" {
				flow += "-udp443"
			}
			ob := M{"tag": "proxy", "protocol": "vless", "settings": M{"address": "127.0.0.1", "port": 24443, "id": "11111111-1111-4111-8111-111111111111", "flow": flow, "encryption": "none"}, "streamSettings": M{"network": "tcp", "security": "tls", "tlsSettings": M{"serverName": "localhost"}}}
			if mode == "mux" {
				ob["mux"] = M{"enabled": true, "concurrency": -1, "xudpConcurrency": -1, "xudpProxyUDP443": "reject"}
			}
			if mode == "xudp" {
				ob["settings"].(M)["flow"] = ""
				ob["mux"] = M{"enabled": true, "concurrency": -1, "xudpConcurrency": 1, "xudpProxyUDP443": "reject"}
			}
			root["outbounds"] = []any{ob, M{"tag": "dns-out", "protocol": "dns"}, M{"tag": "direct", "protocol": "freedom"}, M{"tag": "block", "protocol": "blackhole"}}
			before := encode(t, root)
			after, e := Apply(before)
			if e != nil {
				t.Fatal(e)
			}
			if mode == "xudp" {
				p := Prepare([]byte("vless://11111111-1111-4111-8111-111111111111@127.0.0.1:24443?security=tls&sni=localhost&packetEncoding=xudp"))
				if len(p.Candidates) != 1 {
					t.Fatal(p.Code)
				}
				after = p.Candidates[0].Config
			}
			payload := M{"core": core, "before": json.RawMessage(before), "after": json.RawMessage(after)}
			command := exec.Command("python3", "../../scripts/traffic-policy-test.py")
			command.Stdin = bytes.NewReader(encode(t, payload))
			out, e := command.CombinedOutput()
			if e != nil {
				t.Fatalf("isolated traffic fixture failed: %s", out)
			}
			t.Log(string(out))
		})
	}
}
