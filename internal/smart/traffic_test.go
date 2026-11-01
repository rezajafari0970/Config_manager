package smart

import (
	"bytes"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/convert"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

func TestSmartObservedFailoverTCPAndUDP(t *testing.T) {
	if os.Getenv("XRAY_TRAFFIC_TEST") != "1" {
		t.Skip("explicit isolated traffic test")
	}
	core, e := filepath.Abs(os.Getenv("XRAY_TEST_BINARY"))
	if e != nil || core == "" {
		t.Fatal("pinned core required")
	}
	members := []Member{}
	for i, port := range []int{24446, 24443} {
		uri := "vless://11111111-1111-4111-8111-111111111111@127.0.0.1:" + strconv.Itoa(port) + "?security=tls&sni=localhost&flow=xtls-rprx-vision-udp443"
		plan := convert.Prepare([]byte(uri))
		if len(plan.Candidates) == 0 {
			t.Fatal(plan.Code)
		}
		members = append(members, Member{ID: strconv.Itoa(i), Body: plan.Candidates[0].Config})
	}
	body, e := Compose(members)
	if e != nil {
		t.Fatal(e)
	}
	var config any
	json.Unmarshal(body, &config)
	request, _ := json.Marshal(map[string]any{"core": core, "config": config})
	cmd := exec.Command("python3", "../../scripts/smart-traffic-test.py")
	cmd.Stdin = bytes.NewReader(request)
	out, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatalf("smart forwarding: %v\n%s", e, out)
	}
	var receipt struct{ Passed bool }
	if json.Unmarshal(out, &receipt) != nil || !receipt.Passed {
		t.Fatal(string(out))
	}
	t.Log(string(out))
}
