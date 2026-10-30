package probe

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestAdditionalProtocolDrivers(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32))
	pub := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{2}, 32))
	cases := []string{"socks5://user:secret@192.0.2.1:1080", "socks://192.0.2.1:1080", "socks://" + base64.RawStdEncoding.EncodeToString([]byte("user:password")) + "@192.0.2.1:1080", "wireguard://" + key + "@192.0.2.1:51820?publickey=" + pub + "&address=10.0.0.2%2F32&reserved=1,2,3", "[Interface]\nPrivateKey = " + key + "\nAddress = 10.0.0.2/32\n[Peer]\nPublicKey = " + pub + "\nEndpoint = [2001:db8::1]:51820\nAllowedIPs = 0.0.0.0/0\n[Peer]\nPublicKey = " + key + "\nEndpoint = 192.0.2.2:51820\nAllowedIPs = 10.0.0.0/8"}
	for _, raw := range cases {
		p := Prepare([]byte(raw))
		if p.Code != "prepared" || len(p.Candidates) == 0 {
			t.Fatal(p.Kind, p.Code)
		}
		if !json.Valid(p.Candidates[0].Config) {
			t.Fatal("invalid candidate")
		}
		if p.AndroidImport != "not_run" {
			t.Fatal("invented Android evidence")
		}
	}
}
func TestUnsupportedAndAmbiguousVariantsStayExplicit(t *testing.T) {
	for raw, code := range map[string]string{"hysteria://secret@192.0.2.1:443": "hysteria1_unsupported_by_target_core", "socks5://secret@192.0.2.1:1080": "socks_auth_invalid", "wireguard://key@192.0.2.1:10?publickey=x&wnoise=1": "wireguard_extension_unsupported", "[Interface]\nPrivateKey=x\nPrivateKey=y": "wireguard_duplicate_key"} {
		p := Prepare([]byte(raw))
		if p.Code != code || len(p.Candidates) > 0 {
			t.Fatal(raw, p.Code)
		}
	}
}
func TestCustomEnvelope(t *testing.T) {
	raw := `{"inbounds":[],"outbounds":[{"protocol":"blackhole","settings":{}}],"routing":{}}`
	p := Prepare([]byte("custom://" + base64.StdEncoding.EncodeToString([]byte(raw))))
	if p.Code != "prepared" || len(p.Candidates) != 1 || string(p.Candidates[0].Config) != raw {
		t.Fatal("custom envelope did not preserve JSON", p.Code)
	}
}

func TestWireguardDNSAndHysteriaHoppingPreserved(t *testing.T) {
	p := Prepare([]byte("[Interface]\nPrivateKey=synthetic\nAddress=10.0.0.2/32\nDNS=1.1.1.1,8.8.8.8\n[Peer]\nPublicKey=synthetic\nEndpoint=192.0.2.1:51820"))
	if p.Code != "prepared" || !bytes.Contains(p.Candidates[0].Config, []byte(`"servers":["1.1.1.1","8.8.8.8"]`)) || bytes.Contains(p.Candidates[0].Config, []byte("__dns")) {
		t.Fatal("DNS lost", p.Code)
	}
	p = Prepare([]byte("hy2://synthetic@example.com:443?mport=443,8443-8444&pinSHA256=synthetic"))
	if p.Code != "prepared" || !bytes.Contains(p.Candidates[0].Config, []byte(`"udpHop":{"interval":"30","ports":"443,8443-8444"}`)) || !bytes.Contains(p.Candidates[0].Config, []byte(`"pinnedPeerCertSha256":"synthetic"`)) {
		t.Fatal("hopping or certificate pin lost", p.Code)
	}
}
