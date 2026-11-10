package residential

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNativeDriverSelectsProtocolsWithoutChangingWireIdentity(t *testing.T) {
	for _, raw := range []string{
		"tuic://00000000-0000-4000-8000-000000000001:synthetic@192.0.2.10:443?sni=example.com&alpn=h3&congestion_control=bbr",
		"hysteria://192.0.2.10:443?auth=synthetic&upmbps=10&downmbps=20&peer=example.com",
	} {
		p := NewCoreRegistry().Select([]byte(raw))
		if p.Driver != "sing-box" || p.CoreSHA256 != SingBoxSHA256 || len(p.Variants) != 2 {
			t.Fatalf("%+v", p)
		}
		for _, v := range p.Variants {
			var m map[string]any
			if json.Unmarshal(v.Config, &m) != nil {
				t.Fatal("JSON")
			}
			o := m["outbounds"].([]any)[0].(map[string]any)
			if o["server"] != "192.0.2.10" || o["server_port"] != float64(443) || o["tls"].(map[string]any)["server_name"] != "example.com" {
				t.Fatal("wire values changed")
			}
		}
	}
}

func TestNativeCorePlanFixture(t *testing.T) {
	dir := os.Getenv("NATIVE_TEST_PLAN_DIR")
	if dir == "" {
		t.Skip("explicit isolated native core fixture")
	}
	inputs := []string{
		"tuic://00000000-0000-4000-8000-000000000001:synthetic@8.8.8.8:24443?sni=example.com&alpn=h3&congestion_control=bbr",
		"hysteria://8.8.8.8:24443?auth=synthetic&upmbps=10&downmbps=20&peer=example.com",
		`{"log":{"disabled":true},"inbounds":[{"type":"socks","tag":"socks","listen":"127.0.0.1","listen_port":10808}],"outbounds":[{"type":"hysteria2","tag":"proxy","server":"8.8.8.8","server_port":24443,"password":"synthetic","tls":{"enabled":true,"server_name":"example.com"}}],"route":{"final":"proxy"}}`,
	}
	for i, raw := range inputs {
		p := NewCoreRegistry().Select([]byte(raw))
		if p.Driver != "sing-box" || len(p.Variants) != 2 {
			t.Fatal(p.Code)
		}
		b, _ := json.Marshal(p)
		if e := os.WriteFile(filepath.Join(dir, []string{"tuic.json", "hysteria.json", "hysteria2.json"}[i]), b, 0600); e != nil {
			t.Fatal(e)
		}
	}
}
func TestNativeDriverNeverSilentlyDropsExtensionsOrWeakensTLS(t *testing.T) {
	base := "tuic://00000000-0000-4000-8000-000000000001:synthetic@192.0.2.10:443?sni=example.com"
	for _, extra := range []string{"&unknown=1", "&insecure=1", "&sni=other.example", "&udp_over_stream=1", "&peer=other.example"} {
		if p := NewCoreRegistry().Select([]byte(base + extra)); len(p.Variants) > 0 {
			t.Fatal("ambiguous accepted", extra)
		}
	}
	p := NewCoreRegistry().Select([]byte("tuic://00000000-0000-4000-8000-000000000001:synthetic@vpn.example:443?sni=example.com"))
	if len(p.Variants) != 2 || p.Variants[0].PathsComplete {
		t.Fatal("unproven DNS bootstrap certified")
	}
}

func TestNativeJSONAmbiguityNeverReceivesExecutionPlan(t *testing.T) {
	for _, raw := range []string{
		`{"outbounds":[],"outbounds":[{"type":"tuic"}]}`,
		`{"outbounds":[{"type":"tuic","tls":{"enabled":true,"enabled":false}}]}`,
		`{"outbounds":[{"type":"tuic"}]} {}`,
	} {
		if p := NewCoreRegistry().Select([]byte(raw)); len(p.Variants) > 0 {
			t.Fatal("ambiguous input planned")
		}
	}
}
