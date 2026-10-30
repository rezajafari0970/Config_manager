package convert

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/probe"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func balancedFixture(strategy string) M {
	return M{"outbounds": []any{
		M{"tag": "node-a", "protocol": "vless", "settings": M{"address": "192.0.2.1", "port": 443, "id": "11111111-1111-4111-8111-111111111111", "encryption": "none"}},
		M{"tag": "node-b", "protocol": "trojan", "settings": M{"address": "192.0.2.2", "port": 443, "password": "synthetic"}, "streamSettings": M{"security": "tls", "tlsSettings": M{"serverName": "example.com"}}},
		M{"tag": "direct", "protocol": "freedom"}, M{"tag": "block", "protocol": "blackhole"}},
		"routing": M{"balancers": []any{M{"tag": "pool", "selector": []any{"node-"}, "strategy": M{"type": strategy}}},
			"rules": []any{M{"type": "field", "network": "udp", "port": "443", "outboundTag": "block"}, M{"type": "field", "port": "0-65535", "balancerTag": "pool"}}}}
}
func encode(t *testing.T, value any) []byte {
	t.Helper()
	b, e := json.Marshal(value)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestBalancedTemplatePreservesPoolAndFallback(t *testing.T) {
	m := balancedFixture("leastPing")
	m["observatory"] = M{"subjectSelector": []any{"node-"}, "probeUrl": "https://www.gstatic.com/generate_204", "probeInterval": "15s", "enableConcurrency": true}
	bal := m["routing"].(M)["balancers"].([]any)[0].(M)
	bal["fallbackTag"] = "node-b"
	raw := encode(t, m)
	p := Prepare(raw)
	if len(p.Candidates) == 0 {
		t.Fatal(p.Code)
	}
	out, e := object(p.Candidates[0].Config)
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(encode(t, out["observatory"]), encode(t, m["observatory"])) || !bytes.Equal(encode(t, out["routing"].(M)["balancers"]), encode(t, m["routing"].(M)["balancers"])) {
		t.Fatal("observer/balancer changed")
	}
	obs := out["outbounds"].([]any)
	if obs[0].(M)["tag"] != "node-a" || obs[1].(M)["tag"] != "node-b" || obs[2].(M)["protocol"] != "loopback" {
		t.Fatal("default/pool/adapter lost")
	}
	if obs[1].(M)["settings"].(M)["password"] != "synthetic" {
		t.Fatal("credentials changed")
	}
	again, e := Apply(p.Candidates[0].Config)
	if e != nil || !bytes.Equal(again, p.Candidates[0].Config) {
		t.Fatal("not idempotent", e)
	}
	if !bytes.Equal(raw, encode(t, m)) {
		t.Fatal("original changed")
	}
}
func TestBalancedSelectorCollisionChainsAndRoutingPriority(t *testing.T) {
	m := balancedFixture("random")
	obs := m["outbounds"].([]any)
	obs[0].(M)["tag"] = "proxy"
	obs[1].(M)["tag"] = "proxy-backup"
	m["routing"].(M)["balancers"].([]any)[0].(M)["selector"] = []any{"proxy"}
	p := Prepare(encode(t, m))
	if len(p.Candidates) == 0 {
		t.Fatal(p.Code)
	}
	out, _ := object(p.Candidates[0].Config)
	count := 0
	for _, v := range out["outbounds"].([]any) {
		ob := v.(M)
		if matches(ob["tag"].(string), []string{"proxy"}) {
			count++
			if !isProxy(ob) {
				t.Fatal("helper entered pool")
			}
		}
	}
	if count != 2 {
		t.Fatal(count)
	}
	// Explicit default route wins over an unrelated outbound named proxy.
	m["routing"] = M{"rules": []any{M{"type": "field", "network": "tcp,udp", "outboundTag": "proxy-backup"}}}
	p = Prepare(encode(t, m))
	if len(p.Candidates) == 0 {
		t.Fatal(p.Code)
	}
	out, _ = object(p.Candidates[0].Config)
	if out["outbounds"].([]any)[0].(M)["protocol"] != "trojan" {
		t.Fatal("routing ignored")
	}
}
func TestBalancedTemplateRefusesAmbiguousOrUnsafeGraph(t *testing.T) {
	for name, mutate := range map[string]func(M){
		"empty":              func(m M) { m["routing"].(M)["balancers"].([]any)[0].(M)["selector"] = []any{"missing"} },
		"direct-pool":        func(m M) { m["routing"].(M)["balancers"].([]any)[0].(M)["selector"] = []any{"node-", "direct"} },
		"direct-fallback":    func(m M) { m["routing"].(M)["balancers"].([]any)[0].(M)["fallbackTag"] = "direct" },
		"missing-fallback":   func(m M) { m["routing"].(M)["balancers"].([]any)[0].(M)["fallbackTag"] = "missing" },
		"cycle":              func(m M) { m["outbounds"].([]any)[0].(M)["proxySettings"] = M{"tag": "node-a"} },
		"missing-dependency": func(m M) { m["outbounds"].([]any)[0].(M)["proxySettings"] = M{"tag": "missing"} },
		"selective-only": func(m M) {
			m["routing"].(M)["rules"] = []any{M{"type": "field", "domain": []any{"example.com"}, "balancerTag": "pool"}}
		},
		"empty-prefix": func(m M) { m["routing"].(M)["balancers"].([]any)[0].(M)["selector"] = []any{""} },
	} {
		t.Run(name, func(t *testing.T) {
			m := balancedFixture("random")
			mutate(m)
			p := Prepare(encode(t, m))
			if len(p.Candidates) > 0 {
				t.Fatal("ambiguous graph accepted")
			}
		})
	}
}
func TestBalancedTemplatePinnedCore(t *testing.T) {
	core := os.Getenv("XRAY_TEST_BINARY")
	if core == "" {
		t.Skip("pinned core required")
	}
	b, e := os.ReadFile(core)
	if e != nil {
		t.Fatal(e)
	}
	h := sha256.Sum256(b)
	if hex.EncodeToString(h[:]) != probe.CoreSHA256 {
		t.Fatal("wrong core")
	}
	for _, strategy := range []string{"random", "roundRobin", "leastPing", "leastLoad"} {
		t.Run(strategy, func(t *testing.T) {
			m := balancedFixture(strategy)
			if strategy == "leastPing" {
				m["observatory"] = M{"subjectSelector": []any{"node-"}, "probeUrl": "https://www.gstatic.com/generate_204", "probeInterval": "15s", "enableConcurrency": true}
			}
			if strategy == "leastLoad" {
				m["burstObservatory"] = M{"subjectSelector": []any{"node-"}, "pingConfig": M{"destination": "https://www.gstatic.com/generate_204", "interval": "5m", "sampling": 2, "timeout": "30s", "httpMethod": "HEAD"}}
			}
			p := Prepare(encode(t, m))
			if len(p.Candidates) == 0 {
				t.Fatal(p.Code)
			}
			file := filepath.Join(t.TempDir(), "synthetic.json")
			os.WriteFile(file, p.Candidates[0].Config, 0600)
			if out, e := exec.Command(core, "run", "-test", "-c", file).CombinedOutput(); e != nil {
				t.Fatalf("pinned core: %s", out)
			}
			again, e := Apply(p.Candidates[0].Config)
			if e != nil || !bytes.Equal(again, p.Candidates[0].Config) {
				t.Fatal("not idempotent", e)
			}
		})
	}
}
