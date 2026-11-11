package compat

import (
	"bytes"
	"encoding/json"
	"testing"
)

const sample = `{"inbounds":[{"tag":"socks","listen":"127.0.0.1","port":10808,"protocol":"socks","settings":{"auth":"noauth","udp":true}}],"outbounds":[{"tag":"proxy","protocol":"vless","settings":{"address":"203.0.113.81","port":24443,"id":"11111111-1111-4111-8111-111111111111","encryption":"none"},"streamSettings":{"network":"tcp","security":"none","tcpSettings":{"header":{"type":"http","request":{"version":"1.1","method":"GET","path":["/"],"headers":{"Host":["fast.com"]}}}}}},{"tag":"block","protocol":"blackhole","settings":{}}],"routing":{"rules":[{"type":"field","network":"tcp,udp","outboundTag":"proxy"}]}}`

func root(t *testing.T) M {
	t.Helper()
	var m M
	if e := json.Unmarshal([]byte(sample), &m); e != nil {
		t.Fatal(e)
	}
	return m
}
func clone(t *testing.T, m M) M {
	t.Helper()
	b, _ := json.Marshal(m)
	var x M
	json.Unmarshal(b, &x)
	return x
}
func TestRelayRoundTripAndIdempotence(t *testing.T) {
	m := root(t)
	original := clone(t, m)
	changes, e := Wrap(m)
	if e != nil || len(changes) != 1 {
		t.Fatal(changes, e)
	}
	first, _ := json.Marshal(m)
	if _, e = Wrap(m); e != nil {
		t.Fatal(e)
	}
	second, _ := json.Marshal(m)
	if !bytes.Equal(first, second) {
		t.Fatal("not idempotent")
	}
	if _, e = Logical(m); e != nil {
		t.Fatal(e)
	}
	a, _ := json.Marshal(original)
	b, _ := json.Marshal(m)
	if !bytes.Equal(a, b) {
		t.Fatalf("logical roundtrip changed semantics: %s", b)
	}
}
func TestRelayCollisionAndCarrier(t *testing.T) {
	m := root(t)
	obs := m["outbounds"].([]any)
	ob := obs[0].(M)
	ob["streamSettings"].(M)["sockopt"] = M{"dialerProxy": "carrier", "tcpKeepAliveIdle": 45}
	ob["sendThrough"] = "0.0.0.0"
	obs = append(obs, M{"tag": "compat-transport-1", "protocol": "blackhole", "settings": M{}})
	m["outbounds"] = obs
	m["inbounds"] = append(m["inbounds"].([]any), M{"tag": "compat-relay-1", "port": 20000, "listen": "127.0.0.1", "protocol": "socks"})
	if _, e := Wrap(m); e != nil {
		t.Fatal(e)
	}
	if _, ok := ob["streamSettings"].(M)["sockopt"]; ok {
		t.Fatal("remote carrier on local leg")
	}
	e, _ := getEndpoint(ob)
	if e.port == 20000 || e.host != "127.0.0.1" {
		t.Fatal("port collision")
	}
	leg := m["outbounds"].([]any)[3].(M)
	if socksOf(leg)["dialerProxy"] != "carrier" || leg["tag"] == "compat-transport-1" {
		t.Fatal(leg)
	}
}
func TestNoEncryptionDowngrade(t *testing.T) {
	for _, security := range []string{"tls", "reality"} {
		m := root(t)
		ob := m["outbounds"].([]any)[0].(M)
		ob["streamSettings"].(M)["security"] = security
		a, _ := json.Marshal(m)
		ch, e := Wrap(m)
		b, _ := json.Marshal(m)
		if e != nil || len(ch) != 0 || !bytes.Equal(a, b) {
			t.Fatal(security, e)
		}
	}
	m := root(t)
	m["outbounds"].([]any)[0].(M)["settings"].(M)["encryption"] = "mlkem768x25519plus.native.0rtt.synthetic"
	ch, e := Wrap(m)
	if e != nil || len(ch) != 0 {
		t.Fatal(ch, e)
	}
}
func TestUnsupportedFamiliesRemainExplicit(t *testing.T) {
	for _, network := range []string{"ws", "xhttp", "grpc", "httpupgrade", "kcp", "quic"} {
		m := root(t)
		m["outbounds"].([]any)[0].(M)["streamSettings"].(M)["network"] = network
		if _, e := Wrap(m); e == nil {
			t.Fatal(network)
		}
	}
}
func TestRelayRejectsUnboundedGraphs(t *testing.T) {
	mutations := []func(M){
		func(m M) { m["routing"].(M)["rules"].([]any)[0].(M)["port"] = "1-65535" },
		func(m M) { r := m["routing"].(M); r["rules"] = r["rules"].([]any)[1:] },
		func(m M) { m["inbounds"].([]any)[1].(M)["settings"].(M)["followRedirect"] = true },
		func(m M) { m["routing"].(M)["rules"].([]any)[1].(M)["outboundTag"] = "proxy" },
		func(m M) {
			m["routing"].(M)["rules"] = append([]any{M{"type": "field", "network": "tcp,udp", "outboundTag": "proxy"}}, m["routing"].(M)["rules"].([]any)...)
		},
		func(m M) { m["outbounds"].([]any)[2].(M)["settings"] = M{"redirect": "1.1.1.1:443"} },
	}
	for i, mutate := range mutations {
		m := root(t)
		Wrap(m)
		mutate(m)
		if _, e := Logical(m); e == nil {
			t.Fatal("accepted unsafe graph", i)
		}
	}
}

func TestRelayPreservesLargeIntegerExtensions(t *testing.T) {
	body := bytes.Replace([]byte(sample), []byte(`"outbounds":`), []byte(`"opaque":9007199254740993,"outbounds":`), 1)
	result, _, err := ApplyJSON(body)
	if err != nil || !bytes.Contains(result, []byte(`"opaque":9007199254740993`)) {
		t.Fatal("integer precision changed", err)
	}
}

func TestRelayRejectsTrailingJSON(t *testing.T) {
	if _, _, err := ApplyJSON([]byte(sample + ` {}`)); err == nil {
		t.Fatal("accepted trailing document")
	}
}
