package smart

import (
	"bytes"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/convert"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

func TestEndpointBootstrapIsScopedAndPreservesOutbounds(t *testing.T) {
	m := member(t, 1, "A")
	root, _ := object(m.Body)
	nodes := root["outbounds"].([]any)
	ob := nodes[0].(M)
	ob["settings"] = M{"vnext": []any{M{"address": "Proxy.Example.test", "port": json.Number("443"), "users": []any{M{"id": "synthetic", "encryption": "none"}}}}}
	ob["streamSettings"].(M)["xhttpSettings"] = M{"extra": M{"downloadSettings": M{"address": "download.example.test"}}}
	root["dns"] = M{"servers": []any{M{"address": "localhost", "domains": []any{"domain:instagram.com"}}}}
	m.Body, _ = json.Marshal(root)
	b, e := Compose([]Member{m})
	if e != nil {
		t.Fatal(e)
	}
	got, _ := object(b)
	dns := got["dns"].(M)
	servers := dns["servers"].([]any)
	if dns["disableFallbackIfMatch"] != true || len(servers) != 5 {
		t.Fatal("missing scoped bootstrap")
	}
	want := []any{"full:download.example.test", "full:proxy.example.test"}
	for _, v := range servers[:2] {
		s := v.(M)
		if s["skipFallback"] != true || !reflect.DeepEqual(s["domains"], want) {
			t.Fatal("bootstrap escaped exact endpoint scope", s)
		}
	}
	if bytes.Contains(b, []byte("instagram.com")) || bytes.Contains(b, []byte("localhost")) {
		t.Fatal("source DNS globals copied")
	}
	// No endpoint or TLS rewriting to avoid a DNS failure.
	copied := got["outbounds"].([]any)[0].(M)
	if !reflect.DeepEqual(copied["settings"], ob["settings"]) {
		t.Fatal("endpoint altered")
	}
}

func TestSmartResponsiveDNSAndLatency(t *testing.T) {
	if os.Getenv("XRAY_TRAFFIC_TEST") != "1" {
		t.Skip("explicit isolated traffic test")
	}
	core, e := filepath.Abs(os.Getenv("XRAY_TEST_BINARY"))
	if e != nil {
		t.Fatal(e)
	}
	members := []Member{}
	for i, port := range []int{24446, 24443} {
		p := convert.Prepare([]byte("vless://11111111-1111-4111-8111-111111111111@127.0.0.1:" + strconv.Itoa(port) + "?security=none&type=tcp"))
		if len(p.Candidates) == 0 {
			t.Fatal(p.Code)
		}
		members = append(members, Member{ID: strconv.Itoa(i), Body: p.Candidates[0].Config})
	}
	body, e := Compose(members)
	if e != nil {
		t.Fatal(e)
	}
	var config any
	json.Unmarshal(body, &config)
	request, _ := json.Marshal(map[string]any{"core": core, "config": config})
	cmd := exec.Command("python3", "../../scripts/smart-responsive-test.py")
	cmd.Stdin = bytes.NewReader(request)
	out, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatalf("responsive comparison: %v\n%s", e, out)
	}
	var receipt struct{ Passed bool }
	if json.Unmarshal(out, &receipt) != nil || !receipt.Passed {
		t.Fatal(string(out))
	}
	t.Log(string(out))
}
