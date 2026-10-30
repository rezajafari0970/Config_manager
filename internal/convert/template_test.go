package convert

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/probe"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func fixtures() map[string]string {
	key := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32))
	pub := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{2}, 32))
	jsonConfig := `{"outbounds":[{"tag":"origin","protocol":"trojan","settings":{"address":"192.0.2.1","port":443,"password":"synthetic"},"streamSettings":{"security":"tls","tlsSettings":{"serverName":"example.com"}}}]}`
	vm := `{"add":"192.0.2.1","port":"443","id":"11111111-1111-4111-8111-111111111111","net":"ws","host":"example.com","path":"/a","tls":"tls"}`
	return map[string]string{
		"vless-userinfo-space": "vless://two words here@192.0.2.1:443?security=tls&sni=example.com",
		"vless-cipher-suites":  "vless://synthetic@192.0.2.1:443?security=tls&sni=example.com&fp=unsafe&cs=TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
		"vless-explicit-xudp":  "vless://synthetic@192.0.2.1:443?security=tls&sni=example.com&packetEncoding=xudp",
		"vless-tls":            "vless://11111111-1111-4111-8111-111111111111@192.0.2.1:443?security=tls&sni=example.com",
		"vless-ws":             "vless://11111111-1111-4111-8111-111111111111@192.0.2.1:443?security=tls&type=ws&host=example.com&path=%2Fws",
		"vless-reality":        "vless://11111111-1111-4111-8111-111111111111@192.0.2.1:443?security=reality&sni=example.com&fp=chrome&pbk=" + base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{2}, 32)) + "&sid=ab&flow=xtls-rprx-vision",
		"vless-xhttp":          "vless://11111111-1111-4111-8111-111111111111@192.0.2.1:443?security=tls&type=xhttp&host=example.com&path=%2F&mode=auto",
		"vless-grpc":           "vless://11111111-1111-4111-8111-111111111111@192.0.2.1:443?security=tls&type=grpc&sni=example.com&serviceName=svc",
		"vless-httpupgrade":    "vless://11111111-1111-4111-8111-111111111111@192.0.2.1:443?security=tls&type=httpupgrade&host=example.com&path=%2F",
		"vless-http-header":    "vless://11111111-1111-4111-8111-111111111111@192.0.2.1:443?security=tls&headerType=http&host=example.com&path=%2F",
		"vmess":                "vmess://" + base64.RawStdEncoding.EncodeToString([]byte(vm)),
		"trojan":               "trojan://synthetic%3Asecret@192.0.2.1:443?security=tls&sni=example.com",
		"ss":                   "ss://" + base64.RawStdEncoding.EncodeToString([]byte("aes-128-gcm:synthetic")) + "@192.0.2.1:443",
		"hy2":                  "hy2://synthetic@192.0.2.1:443?sni=example.com",
		"hysteria2":            "hysteria2://synthetic@192.0.2.1:443?sni=example.com&obfs=salamander&obfs-password=test&mport=443,8443-8444",
		"socks5":               "socks5://user:synthetic@192.0.2.1:1080",
		"wireguard":            "wireguard://" + key + "@192.0.2.1:51820?publickey=" + pub + "&address=10.0.0.2%2F32&reserved=1,2,3",
		"wireguard-ini":        "[Interface]\nPrivateKey=" + key + "\nAddress=10.0.0.2/32\n[Peer]\nPublicKey=" + pub + "\nEndpoint=192.0.2.1:51820\nAllowedIPs=0.0.0.0/0",
		"json":                 jsonConfig,
		"custom-json":          "custom://" + base64.StdEncoding.EncodeToString([]byte(jsonConfig)),
		"json-envelope":        "json://" + base64.StdEncoding.EncodeToString([]byte(jsonConfig)),
		"domain-bootstrap":     "trojan://synthetic@proxy.example:443?security=tls&sni=example.com",
	}
}
func TestTemplateProtocolMatrix(t *testing.T) {
	for name, raw := range fixtures() {
		t.Run(name, func(t *testing.T) {
			before := []byte(raw)
			p := Prepare(before)
			if len(p.Candidates) == 0 {
				t.Fatal(p.Code)
			}
			c := p.Candidates[0]
			var m M
			if json.Unmarshal(c.Config, &m) != nil || bytes.ContainsAny(c.Config, "\r\n") {
				t.Fatal("not compact JSON")
			}
			if m["remarks"] != "@shenvpn" || m["dns"].(M)["tag"] != "dns-module" || m["inbounds"].([]any)[0].(M)["tag"] != "socks" {
				t.Fatal("template fields missing")
			}
			if len(m["outbounds"].([]any)) != 4 || m["outbounds"].([]any)[0].(M)["tag"] != "proxy" {
				t.Fatal("outbound envelope")
			}
			if string(before) != raw || p.AndroidImport != "not_run" {
				t.Fatal("original/evidence changed")
			}
			b, e := Apply(c.Config)
			if e != nil || !bytes.Equal(b, c.Config) {
				t.Fatal("not idempotent", e)
			}
		})
	}
}
func TestTemplatePinnedCoreMatrix(t *testing.T) {
	core := os.Getenv("XRAY_TEST_BINARY")
	if core == "" {
		t.Skip("explicit pinned binary required")
	}
	b, e := os.ReadFile(core)
	if e != nil {
		t.Fatal(e)
	}
	h := sha256.Sum256(b)
	if hex.EncodeToString(h[:]) != probe.CoreSHA256 {
		t.Fatal("wrong core")
	}
	for name, raw := range fixtures() {
		t.Run(name, func(t *testing.T) {
			p := Prepare([]byte(raw))
			if len(p.Candidates) == 0 {
				t.Fatal(p.Code)
			}
			file := filepath.Join(t.TempDir(), "config.json")
			if e := os.WriteFile(file, p.Candidates[0].Config, 0600); e != nil {
				t.Fatal(e)
			}
			out, e := exec.Command(core, "run", "-test", "-c", file).CombinedOutput()
			if e != nil {
				t.Fatalf("pinned core: %s", out)
			}
		})
	}
}
func TestTemplatePreservesCredentialsTransportAndNumbers(t *testing.T) {
	raw := []byte(`{"log":{"loglevel":"none"},"outbounds":[{"protocol":"trojan","settings":{"address":"192.0.2.1","port":443,"password":"a:b@&"},"streamSettings":{"security":"tls","tlsSettings":{"serverName":"s.example","pinnedPeerCertSha256":"PIN"},"network":"ws","wsSettings":{"path":"/x?ed=2048","headers":{"Host":"h.example"}},"extra":{"n":9007199254740993}}}]}`)
	p := Prepare(raw)
	if len(p.Candidates) < 1 {
		t.Fatal(p.Code)
	}
	b := p.Candidates[0].Config
	for _, token := range []string{`"password":"a:b@\u0026"`, `"serverName":"s.example"`, `"pinnedPeerCertSha256":"PIN"`, `"path":"/x?ed=2048"`, `9007199254740993`} {
		if !bytes.Contains(b, []byte(token)) {
			t.Fatal("lost value", token)
		}
	}
}
func TestTemplateRejectsUnsupportedAmbiguityAndChains(t *testing.T) {
	cases := map[string]string{
		"hysteria://secret@192.0.2.1:443": "hysteria1_unsupported_by_target_core",
		`{"outbounds":[{"tag":"a","protocol":"socks","settings":{}},{"tag":"b","protocol":"socks","settings":{}}]}`: "ambiguous_proxy_selection",
		`{"outbounds":[{"tag":"proxy","protocol":"socks","settings":{},"proxySettings":{"tag":"missing"}}]}`:        "missing_proxy_dependency",
		`{"outbounds":[{"tag":"proxy","protocol":"socks","settings":{},"proxySettings":{"tag":"proxy"}}]}`:          "cyclic_proxy_chain",
	}
	for raw, code := range cases {
		p := Prepare([]byte(raw))
		if len(p.Candidates) != 0 || p.Code != code {
			t.Fatal(p.Code, code)
		}
	}
	raw := []byte(`{"outbounds":[{"tag":"proxy","protocol":"socks","settings":{"address":"192.0.2.1","port":1080},"streamSettings":{"sockopt":{"dialerProxy":"exit"}}},{"tag":"exit","protocol":"trojan","settings":{"address":"192.0.2.2","port":443,"password":"secret"}}]}`)
	p := Prepare(raw)
	if len(p.Candidates) < 1 || !bytes.Contains(p.Candidates[0].Config, []byte(`"dialerProxy":"chain-1"`)) {
		t.Fatal("chain lost", p.Code)
	}
}
func TestTemplateRepairHypothesisStillNeedsConnection(t *testing.T) {
	p := Prepare([]byte("trojan://secret@192.0.2.1:443?security=none&sni=example.com"))
	if len(p.Candidates) != 2 || !p.Candidates[1].RequiresConnection {
		t.Fatal("missing connection gate")
	}
	raw := []byte(`{"stats":[],"outbounds":[{"protocol":"trojan","settings":{"address":"192.0.2.1","port":443,"password":"synthetic"}}],}`)
	p = Prepare(raw)
	if len(p.Candidates) == 0 {
		t.Fatal(p.Code)
	}
}
func FuzzTemplate(f *testing.F) {
	f.Add([]byte(fixtures()["json"]))
	f.Add([]byte(fixtures()["vless-ws"]))
	f.Fuzz(func(t *testing.T, b []byte) {
		before := bytes.Clone(b)
		p := Prepare(b)
		if !bytes.Equal(b, before) {
			t.Fatal("input mutated")
		}
		for _, c := range p.Candidates {
			if !json.Valid(c.Config) || bytes.ContainsAny(c.Config, "\r\n") {
				t.Fatal("bad JSONL")
			}
		}
	})
}

func TestCustomTrojanTLSAndVLESSDefaults(t *testing.T) {
	p := Prepare([]byte(`{"outbounds":[{"protocol":"trojan","settings":{"address":"192.0.2.1","port":443,"password":"secret"}}]}`))
	if len(p.Candidates) != 2 || !p.Candidates[1].RequiresConnection {
		t.Fatal("JSON TLS hypothesis missing", p.Code)
	}
	p = Prepare([]byte(`{"outbounds":[{"protocol":"vless","settings":{"vnext":[{"address":"192.0.2.1","port":443,"users":[{"id":"11111111-1111-4111-8111-111111111111"}]}]}}]}`))
	if len(p.Candidates) != 1 || !bytes.Contains(p.Candidates[0].Config, []byte(`"encryption":"none"`)) {
		t.Fatal("missing VLESS default", p.Code)
	}
}
