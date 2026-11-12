package compat

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"reflect"
	"testing"
)

const uploadFixture = `{"inbounds":[{"tag":"socks","listen":"127.0.0.1","port":10808,"protocol":"socks","settings":{"udp":true}}],"outbounds":[{"tag":"proxy","protocol":"vless","settings":{"address":"127.0.0.1","port":443,"id":"11111111-1111-4111-8111-111111111111","encryption":"none"},"streamSettings":{"network":"tcp","security":"none","tcpSettings":{"header":{"type":"http","request":{"path":["/"],"headers":{"Host":["fixture.test"]}}}},"sockopt":{"dialerProxy":"transport"}}},{"tag":"transport","protocol":"freedom","settings":{"redirect":"192.0.2.1:443"}}],"routing":{"rules":[{"type":"field","network":"tcp,udp","outboundTag":"proxy"}]}}`

func uploadObject(t *testing.T, b []byte) M {
	t.Helper()
	var m M
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}
func TestUploadAlternativePreservesGraphAndOriginal(t *testing.T) {
	raw := []byte(uploadFixture)
	before := bytes.Clone(raw)
	got, changes, err := UploadAlternative(raw)
	if err != nil || len(changes) != 1 {
		t.Fatal(changes, err)
	}
	a, b := uploadObject(t, raw), uploadObject(t, got)
	ob := b["outbounds"].([]any)[0].(M)
	mux := ob["mux"].(M)
	if mux["enabled"] != true || mux["concurrency"] != float64(1) || mux["xudpConcurrency"] != float64(-1) || mux["xudpProxyUDP443"] != "skip" {
		t.Fatal(mux)
	}
	delete(ob, "mux")
	if !reflect.DeepEqual(a, b) || !bytes.Equal(raw, before) {
		t.Fatal("server/graph/raw changed")
	}
	again, c, err := UploadAlternative(got)
	if err != nil || len(c) != 0 || !bytes.Equal(again, got) {
		t.Fatal("active mux overwritten")
	}
}
func TestUploadAlternativeNarrowScope(t *testing.T) {
	changes := []func(M){
		func(o M) { o["streamSettings"].(M)["security"] = true },
		func(o M) { o["settings"].(M)["encryption"] = true },
		func(o M) { o["streamSettings"].(M)["sockopt"].(M)["dialerProxy"] = "unknown-chain" },
		func(o M) { o["protocol"] = "vmess" },
		func(o M) { o["streamSettings"].(M)["security"] = "tls" },
		func(o M) { o["streamSettings"].(M)["security"] = "reality" },
		func(o M) { o["streamSettings"].(M)["network"] = "ws" },
		func(o M) { o["streamSettings"].(M)["method"] = "xhttp" },
		func(o M) { o["streamSettings"].(M)["finalmask"] = M{} },
		func(o M) { o["settings"].(M)["flow"] = "xtls-rprx-vision" },
		func(o M) { o["settings"].(M)["encryption"] = "mlkem768x25519plus.native.0rtt.placeholder" },
		func(o M) { o["settings"].(M)["packetEncoding"] = "xudp" },
		func(o M) { o["mux"] = M{"enabled": true, "concurrency": 8} },
		func(o M) { o["proxySettings"] = M{"tag": "chain"} },
	}
	for i, f := range changes {
		m := uploadObject(t, []byte(uploadFixture))
		f(m["outbounds"].([]any)[0].(M))
		raw, _ := json.Marshal(m)
		b, c, e := UploadAlternative(raw)
		if e != nil || len(c) != 0 || !bytes.Equal(b, raw) {
			t.Fatalf("case%d altered", i)
		}
	}
}
func TestUploadAlternativeCoreValidation(t *testing.T) {
	core := os.Getenv("XRAY_TEST_BINARY")
	if core == "" {
		t.Skip("pinned core required")
	}
	b, _, err := UploadAlternative([]byte(uploadFixture))
	if err != nil {
		t.Fatal(err)
	}
	file := t.TempDir() + "/profile.json"
	if err = os.WriteFile(file, b, 0600); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(core, "run", "-test", "-c", file).CombinedOutput()
	if err != nil {
		t.Fatalf("%v %s", err, out)
	}
}
