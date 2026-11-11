package convert

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestBracketedIPv6SharedTemplateRepair(t *testing.T) {
	raw := []byte(`{"remarks":"fixture","inbounds":[],"routing":{},"outbounds":[{"tag":"proxy","protocol":"vless","settings":{"vnext":[{"address":"[2001:db8::7]","port":8443,"users":[{"id":"11111111-1111-4111-8111-111111111111","encryption":"none"}]}]},"streamSettings":{"network":"tcp","tcpSettings":{"header":{"type":"http","request":{"headers":{"Host":["unchanged.example"]}}}}}}]}`)
	before := bytes.Clone(raw)
	p := Prepare(raw)
	if p.Code != "prepared" || len(p.Candidates) != 1 || !bytes.Equal(raw, before) {
		t.Fatal(p.Code)
	}
	c := p.Candidates[0]
	if !strings.Contains(c.Rule, "endpoint-ipv6-literal.v1") || !bytes.Contains(c.Config, []byte(`"[2001:db8::7]:8443"`)) || !bytes.Contains(c.Config, []byte("unchanged.example")) {
		t.Fatal("endpoint or provenance lost")
	}
	if bytes.Contains(c.Config, []byte("full:[2001:db8::7]")) {
		t.Fatal("IP used as DNS hostname")
	}
	twice, e := Apply(c.Config)
	if e != nil || !bytes.Equal(twice, c.Config) {
		t.Fatal("not idempotent", e)
	}
	var m M
	json.Unmarshal(c.Config, &m)
	for _, v := range m["outbounds"].([]any) {
		ob := v.(M)
		if ob["protocol"] == "vless" {
			s := ob["streamSettings"].(M)
			if s["security"] == "tls" || s["security"] == "reality" {
				t.Fatal("invented encryption")
			}
		}
	}
}
