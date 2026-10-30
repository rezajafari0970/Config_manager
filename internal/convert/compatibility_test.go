package convert

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/repair"
	"strings"
	"testing"
)

func TestURICompatibilityOptionsAndExactCredential(t *testing.T) {
	raw := []byte("vless://two words here@192.0.2.1:443?security=tls&sni=example.com&fp=unsafe&cs=TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256&type=ws&host=example.com&path=%2Fws&packetEncoding=xudp&fm=%7B%22tcp%22%3A%5B%5D%7D\n")
	original := bytes.Clone(raw)
	p := Prepare(raw)
	if p.Code != "prepared" || len(p.Candidates) != 1 {
		t.Fatal(p.Code)
	}
	c := p.Candidates[0]
	m, _ := object(c.Config)
	ob := m["outbounds"].([]any)[0].(M)
	user := ob["settings"].(M)["vnext"].([]any)[0].(M)["users"].([]any)[0].(M)
	stream := ob["streamSettings"].(M)
	tls := stream["tlsSettings"].(M)
	mux := ob["mux"].(M)
	if user["id"] != "two words here" || tls["fingerprint"] != "unsafe" || tls["cipherSuites"] != "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256" || tls["allowInsecure"] != false {
		t.Fatal("credentials/TLS modified")
	}
	if mux["concurrency"] != json.Number("-1") || mux["xudpConcurrency"] != json.Number("1") || mux["xudpProxyUDP443"] != "allow" || stream["finalmask"] == nil {
		t.Fatal("explicit UDP/mask lost")
	}
	if !strings.HasPrefix(c.Rule, Version+"."+repair.RulesVersion) || !bytes.Equal(raw, original) {
		t.Fatal("provenance")
	}
	again, e := Apply(c.Config)
	if e != nil || !bytes.Equal(again, c.Config) {
		t.Fatal("idempotence", e)
	}
}
func TestURIOptionsRejectIncompatibleOrAmbiguousValues(t *testing.T) {
	base := "vless://synthetic@192.0.2.1:443?"
	for query, code := range map[string]string{
		"security=tls&packetEncoding=packetaddr":               "unsupported_packet_encoding",
		"security=tls&packetEncoding=xudp&packetEncoding=xudp": "ambiguous_query_parameter",
		"security=none&cs=TLS_AES_128_GCM_SHA256":              "cipher_suites_require_tls",
		"security=tls&cs=a&cs=b":                               "ambiguous_query_parameter",
		"security=tls&newOption=secret":                        "unknown_query_parameter",
	} {
		p := Prepare([]byte(base + query))
		if p.Code != code || len(p.Candidates) > 0 {
			t.Fatal(query, p.Code)
		}
	}
	p := Prepare([]byte("trojan://synthetic@192.0.2.1:443?packetEncoding=xudp"))
	if p.Code != "unsupported_packet_encoding" {
		t.Fatal(p.Code)
	}
}
func TestRealityCredentialsDiagnosedWithoutDowngrade(t *testing.T) {
	key := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{2}, 32))
	base := "vless://synthetic@192.0.2.1:443?security=reality&"
	for query, code := range map[string]string{
		"sni=example.com":                        "missing_reality_public_key",
		"pbk=invalid":                            "invalid_reality_public_key",
		"pbk=" + key + "&sid=a":                  "invalid_reality_short_id",
		"pbk=" + key + "&sid=123456789012345678": "invalid_reality_short_id",
	} {
		p := Prepare([]byte(base + query))
		if p.Code != code || len(p.Candidates) > 0 {
			t.Fatal(p.Code, code)
		}
	}
	for _, field := range []string{"publicKey", "password"} {
		m := M{"outbounds": []any{M{"protocol": "vless", "settings": M{"address": "192.0.2.1", "port": 443, "id": "synthetic", "encryption": "none"}, "streamSettings": M{"security": "reality", "realitySettings": M{field: key, "shortId": ""}}}}}
		p := Prepare(encode(t, m))
		if p.Code != "prepared" {
			t.Fatal(field, p.Code)
		}
	}
	pool := balancedFixture("random")
	member := pool["outbounds"].([]any)[1].(M)
	member["streamSettings"] = M{"security": "reality", "realitySettings": M{"serverName": "example.com"}}
	p := Prepare(encode(t, pool))
	if p.Code != "missing_reality_public_key" || len(p.Candidates) > 0 {
		t.Fatal("pool missing key", p.Code)
	}
	// Diagnose nested download settings too.
	ob := M{"streamSettings": M{"network": "xhttp", "xhttpSettings": M{"extra": M{"downloadSettings": M{"security": "reality", "realitySettings": M{}}}}}}
	if err := validateReality(ob); err == nil || err.Error() != "missing_reality_public_key" {
		t.Fatal("nested download", err)
	}
}
