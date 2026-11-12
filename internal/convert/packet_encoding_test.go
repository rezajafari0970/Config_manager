package convert

import (
	"bytes"
	"strings"
	"testing"
)

func TestExplicitNativeUDPKeepsExactFinalBytesAndRaw(t *testing.T) {
	for _, suffix := range []string{
		"security=none&type=tcp&headerType=http&host=example.com",
		"security=tls&type=ws&sni=example.com&host=example.com&path=%2Fws",
		"security=reality&type=tcp&sni=example.com&pbk=AgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgI",
	} {
		base := "vless://11111111-1111-4111-8111-111111111111@192.0.2.1:443?" + suffix
		raw := []byte(base + "&packetEncoding=none")
		original := bytes.Clone(raw)
		before, after := Prepare([]byte(base)), Prepare(raw)
		if before.Code != "prepared" || after.Code != "prepared" || len(after.Candidates) != len(before.Candidates) {
			t.Fatal(before.Code, after.Code)
		}
		for i, c := range after.Candidates {
			if !bytes.Equal(c.Config, before.Candidates[i].Config) || c.SHA256 != before.Candidates[i].SHA256 {
				t.Fatal("native UDP changed credentials, transport or final bytes")
			}
			if !strings.Contains(c.Rule, "vless-packet-encoding.v1.native-udp") || !c.RequiresConnection {
				t.Fatal("missing rule or independent connection gate")
			}
			if bytes.Contains(c.Config, []byte("__input_rule")) || bytes.Contains(c.Config, []byte("packetEncoding")) {
				t.Fatal("share metadata leaked into core config")
			}
		}
		if !bytes.Equal(raw, original) {
			t.Fatal("raw was mutated")
		}
	}
	for _, q := range []string{"packetEncoding=", "packetEncoding=unknown", "packetEncoding=packetaddr", "packetEncoding=none&packetEncoding=xudp"} {
		if p := Prepare([]byte("vless://synthetic@192.0.2.1:443?security=tls&" + q)); len(p.Candidates) != 0 {
			t.Fatal("ambiguous or unsupported encoding accepted", q)
		}
	}
}
