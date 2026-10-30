package probe

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/repair"
	"net/url"
	"testing"
)

func TestImmutableAndExactJSON(t *testing.T) {
	raw := []byte("{\n \"inbounds\":[], \"outbounds\":[{\"protocol\":\"blackhole\",\"settings\":{}}], \"routing\":{}\n}")
	saved := bytes.Clone(raw)
	p := Prepare(raw)
	if p.Code != "prepared" || len(p.Candidates) != 1 || !bytes.Equal(p.Candidates[0].Config, raw) || !bytes.Equal(saved, raw) {
		t.Fatal(p.Code)
	}
	if p.Candidates[0].SHA256 != repair.Digest(raw) || p.AndroidImport != "not_run" {
		t.Fatal("evidence conflated")
	}
	b, _ := json.Marshal(p)
	var round Plan
	if json.Unmarshal(b, &round) != nil || !bytes.Equal(round.Candidates[0].Config, raw) {
		t.Fatal("transport changed bytes")
	}
}
func TestTrojanTLSIsUnverifiedHypothesis(t *testing.T) {
	raw := []byte("trojan://synthetic%3Apass@example.com:500?security=none&type=tcp&sni=example.com#test")
	p := Prepare(raw)
	if p.Code != "prepared" || len(p.Candidates) != 2 {
		t.Fatal(p.Code)
	}
	if p.Candidates[0].RequiresConnection || !p.Candidates[1].RequiresConnection {
		t.Fatal("missing evidence gate")
	}
	var a, b M
	json.Unmarshal(p.Candidates[0].Config, &a)
	json.Unmarshal(p.Candidates[1].Config, &b)
	aob := a["outbounds"].([]any)[0].(map[string]any)
	bob := b["outbounds"].([]any)[0].(map[string]any)
	if aob["settings"].(map[string]any)["password"] != "synthetic:pass" {
		t.Fatal("auth changed")
	}
	if aob["streamSettings"].(map[string]any)["security"] != "none" || bob["streamSettings"].(map[string]any)["security"] != "tls" {
		t.Fatal("not separate candidates")
	}
	if bob["streamSettings"].(map[string]any)["tlsSettings"].(map[string]any)["allowInsecure"] != false {
		t.Fatal("TLS weakened")
	}
}
func TestAmbiguityDoesNotGetGuessed(t *testing.T) {
	for _, s := range []string{
		"vless://id@example.com:443?security=tls&security=none", "vless://id@example.com:443?futureKey=value",
		"ss://not-a-valid-auth@example.com:443", "vless://id@example.com:99999?security=tls",
		`{"outbounds":[],"outbounds":[],"inbounds":[],"routing":{}}`,
	} {
		p := Prepare([]byte(s))
		if len(p.Candidates) != 0 {
			t.Fatalf("ambiguous input accepted: %s", p.Code)
		}
	}
}
func TestJSONStructuralRepairSeparately(t *testing.T) {
	raw := []byte(`{"stats":[],"outbounds":[{"protocol":"freedom","settings":[]}],"dns":{"hosts":[]}}`)
	p := Prepare(raw)
	if len(p.Candidates) != 1 || bytes.Equal(raw, p.Candidates[0].Config) {
		t.Fatal(p.Code)
	}
	q := Prepare(p.Candidates[0].Config)
	if !bytes.Equal(q.Candidates[0].Config, p.Candidates[0].Config) {
		t.Fatal("not idempotent")
	}
}
func TestVMessGRPCMapsServiceName(t *testing.T) {
	raw := `{"v":"2","add":"example.com","port":"443","id":"11111111-1111-4111-8111-111111111111","net":"grpc","type":"multi","path":"service","host":"authority.example","tls":"tls","scy":"auto"}`
	p := Prepare([]byte("vmess://" + base64.RawStdEncoding.EncodeToString([]byte(raw))))
	if p.Code != "prepared" || !bytes.Contains(p.Candidates[0].Config, []byte(`"serviceName":"service"`)) {
		t.Fatal(p.Code)
	}
}
func TestUnknownVMessFieldsAreNotLost(t *testing.T) {
	raw := `{"add":"example.com","port":"443","id":"id","futureSecret":"preserve-me"}`
	p := Prepare([]byte("vmess://" + base64.StdEncoding.EncodeToString([]byte(raw))))
	if p.Code != "unknown_vmess_field" || len(p.Candidates) > 0 {
		t.Fatal(p.Code)
	}
}
func TestRemovedInsecureGetsStricterCandidateOnly(t *testing.T) {
	p := Prepare([]byte("trojan://synthetic@example.com:443?security=tls&allowInsecure=1"))
	if len(p.Candidates) != 2 || !p.Candidates[1].RequiresConnection {
		t.Fatal("missing strict TLS alternative")
	}
	if !bytes.Contains(p.Candidates[0].Config, []byte(`"allowInsecure":true`)) || !bytes.Contains(p.Candidates[1].Config, []byte(`"allowInsecure":false`)) {
		t.Fatal("original or candidate policy wrong")
	}
}
func TestEncodedValuesRemainUnambiguous(t *testing.T) {
	cases := []string{
		"vless://id@example.com:443?type=xhttp&extra=" + url.QueryEscape(`{"a":1,"a":2}`),
		"vless://id@example.com:443?fm=" + url.QueryEscape(`{"a":1,"a":2}`),
		"trojan://%FF@example.com:443?security=tls",
		"trojan://password@example.com:443?security=tls&sni=%FF",
	}
	for _, raw := range cases {
		p := Prepare([]byte(raw))
		if len(p.Candidates) > 0 {
			t.Fatal("ambiguous/lossy conversion")
		}
	}
	p := Prepare([]byte("vless://id@example.com:443?type=xhttp&extra=" + url.QueryEscape(`{"unknown":9007199254740993}`)))
	if len(p.Candidates) != 1 || !bytes.Contains(p.Candidates[0].Config, []byte("9007199254740993")) {
		t.Fatal("large number lost")
	}
}
func FuzzPrepare(f *testing.F) {
	f.Add([]byte("trojan://pass@example.com:443?security=tls"))
	f.Add([]byte(`{"outbounds":[]}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		before := bytes.Clone(raw)
		p := Prepare(raw)
		if !bytes.Equal(before, raw) {
			t.Fatal("original mutated")
		}
		for _, c := range p.Candidates {
			if c.SHA256 != repair.Digest(c.Config) || !json.Valid(c.Config) {
				t.Fatal("bad candidate")
			}
		}
	})
}

func TestXHTTPPaddingCompatibilityAlias(t *testing.T) {
	base := "vless://11111111-1111-4111-8111-111111111111@192.0.2.1:443?type=xhttp&security=none"
	for _, query := range []string{"&x_padding_bytes=100-1000", "&x_padding_bytes=100-1000&extra=%7B%22xPaddingBytes%22%3A%22100-1000%22%7D", "&x_padding_bytes=100&extra=%7B%22xPaddingBytes%22%3A100%7D"} {
		p := Prepare([]byte(base + query))
		if len(p.Candidates) != 1 || !bytes.Contains(p.Candidates[0].Config, []byte("xPaddingBytes")) {
			t.Fatal("known alias lost", p.Code)
		}
	}
	for _, query := range []string{"&x_padding_bytes=100-1000&extra=%7B%22xPaddingBytes%22%3A%22200-300%22%7D", "&x_padding_bytes=0", "&x_padding_bytes=1000-100", "&x_padding_bytes=2147483648", "&x_padding_bytes=", "&x_padding_bytes=100&x_padding_bytes=200", "&unknown_future_option=1"} {
		p := Prepare([]byte(base + query))
		if len(p.Candidates) > 0 {
			t.Fatal("ambiguous alias accepted")
		}
	}
	p := Prepare([]byte("vless://11111111-1111-4111-8111-111111111111@192.0.2.1:443?type=tcp&x_padding_bytes=100"))
	if len(p.Candidates) > 0 {
		t.Fatal("alias ignored on wrong transport")
	}
}
