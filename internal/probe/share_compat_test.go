package probe

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

func shareFixture(extra M) []byte {
	m := M{"v": "2", "ps": "fixture", "add": "192.0.2.1", "port": "443", "id": "11111111-1111-4111-8111-111111111111", "aid": "0", "scy": "auto", "net": "tcp", "type": "none", "tls": ""}
	for k, v := range extra {
		m[k] = v
	}
	b, _ := json.Marshal(m)
	return []byte("vmess://" + base64.StdEncoding.EncodeToString(b))
}
func TestVMessShareCompatibilityPreservesWireAndRaw(t *testing.T) {
	original := shareFixture(nil)
	base := Prepare(original)
	cases := []M{{"name": "label", "test_name": "measurement"}, {"skip-cert-verify": true}, {"batch": "", "echConfigList": "", "echServerKeys": "", "fakehost_domain": "", "interval": "", "length": "", "mux": "", "muxConcurrency": "", "packets": "", "xhttp_extra": ""}}
	for _, extra := range cases {
		raw := shareFixture(extra)
		copyRaw := bytes.Clone(raw)
		p := Prepare(raw)
		if p.Code != "prepared" || len(p.Candidates) != 1 {
			t.Fatalf("unexpected plan %s", p.Code)
		}
		if !bytes.Equal(raw, copyRaw) || !bytes.Equal(withoutShareRemarks(p.Candidates[0].Config), withoutShareRemarks(base.Candidates[0].Config)) {
			t.Fatal("raw or wire semantics changed")
		}
		if !strings.Contains(p.Candidates[0].Rule, shareCompatibilityVersion) || bytes.Contains(p.Candidates[0].Config, []byte("__input_rule")) {
			t.Fatal("provenance missing or leaked")
		}
	}
}
func TestVMessTLSAliasUsesStrictCandidateAndRejectsConflicts(t *testing.T) {
	p := Prepare(shareFixture(M{"tls": "tls", "skip-cert-verify": true}))
	if p.Code != "prepared" || len(p.Candidates) != 1 {
		t.Fatal(p.Code, len(p.Candidates))
	}
	c := p.Candidates[0]
	if !c.RequiresConnection || !strings.Contains(c.Rule, "strict-tls-hypothesis") || !strings.Contains(c.Rule, shareCompatibilityVersion+".tls-verification-alias") || bytes.Contains(c.Config, []byte(`"allowInsecure":true`)) {
		t.Fatal("insecure candidate bypass")
	}
	for _, extra := range []M{{"skip-cert-verify": "yes"}, {"skip-cert-verify": M{}}, {"mux": "1"}, {"echConfigList": "not-empty"}, {"mux": false}, {"name": M{}}, {"new-unknown-option": ""}} {
		p := Prepare(shareFixture(extra))
		if p.Code == "prepared" || len(p.Candidates) > 0 {
			t.Fatal("ambiguous/active extension admitted")
		}
	}
}
func TestShareAnnotationsAreClosedAndVersioned(t *testing.T) {
	raw := "vless://11111111-1111-4111-8111-111111111111@192.0.2.1:443?security=tls&type=ws&sni=example.com"
	base := Prepare([]byte(raw))
	for _, suffix := range []string{"&x-sub=https%3A%2F%2Fexample.com%2Fsub", "&Telegram=channel"} {
		p := Prepare([]byte(raw + suffix))
		if p.Code != "prepared" || !bytes.Equal(withoutShareRemarks(p.Candidates[0].Config), withoutShareRemarks(base.Candidates[0].Config)) || !strings.Contains(p.Candidates[0].Rule, "subscription-metadata") {
			t.Fatal("annotation changed connection", p.Code)
		}
	}
	for _, suffix := range []string{"&x-sub=one&x-sub=two", "&Telegram=bad%0Avalue", "&new-option=1", "&realityVersion=2"} {
		if p := Prepare([]byte(raw + suffix)); p.Code == "prepared" {
			t.Fatal("unknown or ambiguous annotation accepted")
		}
	}
	p := Prepare([]byte(strings.Replace(raw, "type=ws", "type=tcp%23labelvless%3A%2F%2Fother", 1)))
	if p.Code != "concatenated_config_uri" {
		t.Fatal(p.Code)
	}
}
func TestWireguardOfficialDNSAndFinalMaskURIFields(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32))
	q := url.Values{"publickey": {base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{2}, 32))}, "address": {"10.0.0.2/32"}, "dns": {"1.1.1.1,8.8.8.8"}, "fm": {`{"udp":[{"type":"salamander","settings":{"password":"synthetic"}}]}`}}
	raw := "wireguard://" + url.User(key).String() + "@192.0.2.1:51820?"
	p := Prepare([]byte(raw + q.Encode()))
	if p.Code != "prepared" || !bytes.Contains(p.Candidates[0].Config, []byte(`"servers":["1.1.1.1","8.8.8.8"]`)) || !bytes.Contains(p.Candidates[0].Config, []byte(`"finalmask"`)) {
		t.Fatal("official fields lost", p.Code)
	}
	q.Set("finalmask", `{"udp":[]}`)
	if p := Prepare([]byte(raw + q.Encode())); p.Code != "conflicting_finalmask" {
		t.Fatal(p.Code)
	}
	q.Del("finalmask")
	q.Set("wnoise", "quic")
	if p := Prepare([]byte(raw + q.Encode())); p.Code != "wireguard_extension_unsupported" {
		t.Fatal("legacy obfuscation silently removed", p.Code)
	}
}

func withoutShareRemarks(b []byte) []byte {
	var m M
	_ = json.Unmarshal(b, &m)
	delete(m, "remarks")
	out, _ := json.Marshal(m)
	return out
}

func TestShareClosedFieldsAndStrictConflict(t *testing.T) {
	for _, extra := range []M{{"net type": ""}, {"type host": ""}} {
		if p := Prepare(shareFixture(extra)); p.Code != "unknown_vmess_field" {
			t.Fatal(p.Code)
		}
	}
	p := Prepare(shareFixture(M{"tls": "tls", "skip-cert-verify": true, "insecure": "0"}))
	if p.Code != "prepared" || len(p.Candidates) != 1 || !p.Candidates[0].RequiresConnection || !strings.Contains(p.Candidates[0].Rule, "tls-verification-strict-conflict") || bytes.Contains(p.Candidates[0].Config, []byte(`"allowInsecure":true`)) || bytes.Contains(p.Candidates[0].Config, []byte("__require_connection")) {
		t.Fatal("conflicting hint weakened verification", p.Code)
	}
	raw := "wireguard://synthetic@192.0.2.1:51820?publickey=synthetic&fm=" + url.QueryEscape(`{"udp":[],"tcp":[]}`) + "&finalmask=" + url.QueryEscape(`{ "tcp": [], "udp": [] }`)
	if p := Prepare([]byte(raw)); p.Code != "prepared" {
		t.Fatal("equivalent aliases rejected", p.Code)
	}
	if p := Prepare([]byte(raw + "&dns%20fm=value")); p.Code != "wireguard_extension_unsupported" {
		t.Fatal("unknown combined field accepted", p.Code)
	}
}
