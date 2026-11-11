package probe

import (
	"bytes"
	"encoding/base64"
	"net/url"
	"strings"
	"testing"
)

func TestVMessFragmentAndEscapedPayload(t *testing.T) {
	raw := shareFixture(M{"net": "ws", "path": "/same?ed#literal", "port": "80"})
	base := Prepare(raw)
	for _, suffix := range []string{"#label", "#✅%40channel"} {
		input := append(bytes.Clone(raw), []byte(suffix)...)
		before := bytes.Clone(input)
		p := Prepare(input)
		if p.Code != "prepared" || !bytes.Equal(input, before) || !bytes.Equal(withoutShareRemarks(p.Candidates[0].Config), withoutShareRemarks(base.Candidates[0].Config)) || !strings.Contains(p.Candidates[0].Rule, "fragment-label") {
			t.Fatal(p.Code)
		}
	}
	escaped := []byte("vmess://" + strings.ReplaceAll(string(raw[8:]), "=", "%3D") + "#name")
	if p := Prepare(escaped); p.Code != "prepared" {
		t.Fatal(p.Code)
	}
	for _, bad := range []string{"vmess://%%%%#label", "vmess://broken!#label", "vmess://" + string(raw[8:]) + "?unknown=1"} {
		if p := Prepare([]byte(bad)); p.Code == "prepared" {
			t.Fatal("invented payload recovery")
		}
	}
}
func TestVMessKnownExporterFieldsAndPorts(t *testing.T) {
	for _, fields := range []M{{"security": "auto", "scy": "", "skip-cert-verify": true}, {"snib": "", "net": "ws", "type": "---"}, {"ech": "", "pqv": ""}, {"port": " ۴۴۳ "}, {"port": "٤٤٣"}} {
		p := Prepare(shareFixture(fields))
		if p.Code != "prepared" || len(p.Candidates) != 1 {
			t.Fatal(p.Code)
		}
	}
	for _, fields := range []M{{"snib": "server.example"}, {"ech": "active"}, {"pqv": "active"}, {"security": "none", "scy": "auto"}, {"security": "invented"}, {"port": ""}, {"port": "65536"}, {"port": "443.4"}, {"port": "44x"}, {"random": ""}} {
		if p := Prepare(shareFixture(fields)); p.Code == "prepared" {
			t.Fatal("ambiguous input accepted", fields)
		}
	}
	// Base64url unpadded is independently accepted; this never recovers lost bytes.
	b := []byte(`{"add":"192.0.2.1","port":443,"id":"11111111-1111-4111-8111-111111111111","net":"ws"}`)
	if p := Prepare([]byte("vmess://" + base64.RawURLEncoding.EncodeToString(b) + "#label")); p.Code != "prepared" {
		t.Fatal(p.Code)
	}
}
func TestRealityVersionAnnotationBoundaries(t *testing.T) {
	base := "vless://11111111-1111-4111-8111-111111111111@192.0.2.1:443?security=reality&type=tcp&pbk=" + url.QueryEscape(base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32)))
	p := Prepare([]byte(base + "&realityVersion=26.2.6"))
	if p.Code != "prepared" || !strings.Contains(p.Candidates[0].Rule, "subscription-metadata") {
		t.Fatal(p.Code)
	}
	for _, suffix := range []string{"&realityVersion=next", "&realityVersion=26.2.6&realityVersion=26.2.6", "&ty"} {
		if p := Prepare([]byte(base + suffix)); p.Code == "prepared" {
			t.Fatal("ambiguous version/truncation admitted")
		}
	}
}
