package smart

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/rezajafari0970/Config_manager/internal/convert"
	"github.com/rezajafari0970/Config_manager/internal/probe"
	"github.com/rezajafari0970/Config_manager/internal/repair"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func member(t *testing.T, n int, source string) Member {
	t.Helper()
	raw := fmt.Sprintf("vless://synthetic-%d@192.0.2.%d:443?security=tls&sni=example.com", n, n)
	p := convert.Prepare([]byte(raw))
	if len(p.Candidates) == 0 {
		t.Fatal(p.Code)
	}
	c := p.Candidates[0]
	return Member{ID: fmt.Sprint(n), Source: source, Hash: c.SHA256, RawHash: repair.Digest([]byte(raw)), Body: c.Config}
}
func TestSourceCoverageDedupAndLeaderCycle(t *testing.T) {
	pool := []Member{}
	for i := 1; i <= 30; i++ {
		source := "big"
		if i == 29 {
			source = "small"
		}
		if i == 30 {
			source = "medium"
		}
		pool = append(pool, member(t, i, source))
	}
	for _, count := range []int{1, 2, 3, 10, 50} {
		ledger := Ledger{Sources: map[string]Usage{}, Members: map[string]Usage{}}
		covered := map[string]bool{}
		heads := map[string]bool{}
		for i, head := range pool {
			v, e := Select(pool, head.ID, count, ledger)
			if e != nil {
				t.Fatal(e)
			}
			if v[0].ID != head.ID || heads[v[0].ID] {
				t.Fatal("head repeated")
			}
			heads[v[0].ID] = true
			ids := map[string]bool{}
			sources := map[string]bool{}
			for _, m := range v {
				if ids[m.ID] {
					t.Fatal("duplicate")
				}
				ids[m.ID] = true
				sources[m.Source] = true
				covered[m.Source] = true
				u := ledger.Members[Key(m)]
				u.Uses++
				u.Last = int64(i)
				ledger.Members[Key(m)] = u
				u = ledger.Sources[m.Source]
				u.Uses++
				u.Last = int64(i)
				ledger.Sources[m.Source] = u
			}
			if len(v) != min(count, len(pool)) {
				t.Fatal("wrong size")
			}
			if count >= 3 && len(sources) != 3 {
				t.Fatal("source quota lost")
			}
		}
		if len(covered) != 3 || len(heads) != 30 {
			t.Fatal("incomplete coverage")
		}
	}
	shared := pool[0]
	shared.Source = "duplicate"
	pool = append(pool, shared)
	v, e := Select(pool, "1", 100, Ledger{})
	if e != nil || len(v) != 30 {
		t.Fatal("shared-source duplicate")
	}
}
func TestComposeGraphCredentialsAndPinnedCore(t *testing.T) {
	balanced := `{"outbounds":[{"tag":"a","protocol":"vless","settings":{"address":"192.0.2.100","port":443,"id":"synthetic-a","encryption":"none"},"streamSettings":{"security":"tls","tlsSettings":{"serverName":"example.com"}}},{"tag":"b","protocol":"trojan","settings":{"address":"192.0.2.101","port":443,"password":"synthetic-secret"},"streamSettings":{"security":"tls","tlsSettings":{"serverName":"example.com"}}}],"routing":{"balancers":[{"tag":"pool","selector":["a","b"],"strategy":{"type":"leastLoad","settings":{"expected":1}}}],"rules":[{"type":"field","network":"tcp,udp","balancerTag":"pool"}]},"burstObservatory":{"subjectSelector":["a","b"],"pingConfig":{"destination":"https://www.gstatic.com/generate_204","interval":"20s","sampling":3,"timeout":"4s"}}}`
	vm := base64.RawStdEncoding.EncodeToString([]byte(`{"add":"192.0.2.102","port":"443","id":"synthetic-vm","net":"ws","tls":"tls","sni":"example.com"}`))
	raws := []string{"vmess://" + vm, "ss://" + base64.RawStdEncoding.EncodeToString([]byte("chacha20-ietf-poly1305:synthetic")) + "@192.0.2.103:443", "hy2://synthetic@192.0.2.104:443?sni=example.com", "socks5://synthetic:secret@192.0.2.105:1080", balanced}
	members := []Member{member(t, 1, "A")}
	for i, raw := range raws {
		p := convert.Prepare([]byte(raw))
		if len(p.Candidates) == 0 {
			t.Fatal(p.Code)
		}
		members = append(members, Member{ID: fmt.Sprint(i + 2), Source: "B", Body: p.Candidates[0].Config})
	}
	before := make([][]byte, len(members))
	for i := range members {
		before[i] = bytes.Clone(members[i].Body)
	}
	b, e := Compose(members)
	if e != nil {
		t.Fatal(e)
	}
	var m M
	json.Unmarshal(b, &m)
	out := m["outbounds"].([]any)
	tags := map[string]bool{}
	for i, v := range out {
		tag := v.(M)["tag"].(string)
		if tags[tag] {
			t.Fatal("tag collision")
		}
		tags[tag] = true
		if i < len(members) && tag != fmt.Sprintf("proxy-%02d", i+1) {
			t.Fatal("entry order", i, tag)
		}
	}
	for i := range members {
		if !bytes.Equal(before[i], members[i].Body) {
			t.Fatal("member mutated")
		}
	}
	if !bytes.Contains(b, []byte("synthetic-secret")) || m["remarks"] != "@shenvpn-smart-parallel-v6b" {
		t.Fatal("lost credentials/envelope")
	}
	if bytes.Contains(b, []byte(`"dialerProxy":"proxy"`)) || bytes.Contains(b, []byte(`"balancerTag":"pool"`)) {
		t.Fatal("unrenamed references")
	}
	for _, count := range []int{1, 2, 5, 7, 10, 15, 50, 100} {
		group := []Member{}
		for i := 1; i <= count; i++ {
			group = append(group, member(t, i, "A"))
		}
		data, e := Compose(group)
		if e != nil {
			t.Fatal(e)
		}
		validateCore(t, data)
	}
	validateCore(t, b)
}
func validateCore(t *testing.T, b []byte) {
	t.Helper()
	core := os.Getenv("XRAY_TEST_BINARY")
	if core == "" {
		return
	}
	binary, e := os.ReadFile(core)
	if e != nil || (repair.Digest(binary) != probe.CoreSHA256 && repair.Digest(binary) != "cc4b84d3069569f297a61f04b914f2bb37e9d202fd7b84cb50def1fc30fd40e1") {
		t.Fatal("wrong pinned core")
	}
	path := filepath.Join(t.TempDir(), "smart.json")
	os.WriteFile(path, b, 0600)
	out, e := exec.Command(core, "run", "-test", "-c", path).CombinedOutput()
	if e != nil {
		t.Fatalf("pinned smart: %s", out)
	}
}
func TestComposeDoesNotInventMissingDependency(t *testing.T) {
	m := member(t, 1, "A")
	m.Body = []byte(strings.Replace(string(m.Body), `"sockopt":{`, `"sockopt":{"dialerProxy":"missing",`, 1))
	if _, e := Compose([]Member{m}); e == nil {
		t.Fatal("missing dependency accepted")
	}
}

func TestSourceBalancerGlobalsCannotOverrideFixedEnvelope(t *testing.T) {
	m := member(t, 1, "A")
	root, _ := object(m.Body)
	first := root["outbounds"].([]any)[0].(M)
	first["tag"] = "node-bandwidth-42"
	root["outbounds"] = []any{first}
	root["routing"] = M{"balancers": []any{M{"tag": "balance", "selector": []string{"node-"}, "strategy": M{"type": "leastLoad", "settings": M{"costs": []any{M{"match": "bandwidth-42", "value": 0}}}}}}, "rules": []any{M{"type": "field", "network": "tcp,udp", "balancerTag": "balance"}}}
	m.Body, _ = json.Marshal(root)
	b, e := Compose([]Member{m})
	if e != nil {
		t.Fatal(e)
	}
	got, _ := object(b)
	want, _ := object([]byte(base))
	got["routing"].(M)["balancers"].([]any)[1].(M)["selector"] = []any{}
	if !reflect.DeepEqual(got["routing"], want["routing"]) || !reflect.DeepEqual(got["burstObservatory"], want["burstObservatory"]) {
		t.Fatal("source globals leaked into template")
	}
	out := got["outbounds"].([]any)
	first["tag"] = "proxy-01"
	applySmartSocketPolicy(first)
	if !reflect.DeepEqual(out[0], first) {
		t.Fatal("flattening changed outbound values")
	}
	validateCore(t, b)
}

func TestSharedCanonicalSourceCoverageDoesNotRepeatMember(t *testing.T) {
	a := Member{ID: "shared", Source: "A", Hash: "one"}
	b := Member{ID: "shared", Source: "B", Hash: "two"}
	c := Member{ID: "third", Source: "C", Hash: "three"}
	got, e := Select([]Member{a, b, c}, "shared", 3, Ledger{})
	if e != nil || len(got) != 2 {
		t.Fatal("shared variants must deduplicate", e, got)
	}
}
