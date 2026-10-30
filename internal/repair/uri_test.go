package repair

import (
	"bytes"
	"strings"
	"testing"
)

func TestURIUserinfoRepairPreservesExactCredentialAndOriginal(t *testing.T) {
	for _, protocol := range []string{"vless", "trojan", "ss", "hy2", "hysteria2", "socks", "socks5"} {
		raw := []byte(protocol + "://user+name%25:two words @192.0.2.1:443?security=tls&sni=example.com#name\n")
		before := bytes.Clone(raw)
		d := NewRegistry().Propose(raw, true, protocol, "invalid")
		expected := strings.ReplaceAll(string(raw), "two words ", "two%20words%20")
		if d.Status != "repaired" || string(d.Candidate) != expected || !bytes.Equal(before, raw) || len(d.Patches) != 1 {
			t.Fatal(protocol, d.Status)
		}
		again := NewRegistry().Propose(d.Candidate, true, protocol, "valid")
		if again.Status != "unchanged" || len(again.Candidate) != 0 {
			t.Fatal("not idempotent")
		}
	}
}
func TestURIRepairRejectsAmbiguityAndUnrelatedDamage(t *testing.T) {
	for _, raw := range []string{
		"vless://a b@c@192.0.2.1:443",
		"vless://a b%ZZ@192.0.2.1:443",
		"vless://a b@bad host:443",
		"vless://a\t b@192.0.2.1:443",
		"vless://a b@192.0.2.1",
		"vless://a b/other@192.0.2.1:443",
	} {
		d := NewRegistry().Propose([]byte(raw), true, "vless", "invalid")
		if d.Status != "blocked" || len(d.Candidate) > 0 {
			t.Fatal("unsafe repair", d.Status)
		}
	}
}
