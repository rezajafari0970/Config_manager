package smart

import (
	"bytes"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/convert"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLibV40RealPingDefault(t *testing.T) {
	if os.Getenv("XRAY_TRAFFIC_TEST") != "1" {
		t.Skip("explicit isolated traffic test")
	}
	helper := os.Getenv("XRAY_LIBPING_BINARY")
	if helper == "" {
		t.Fatal("build scripts/testdata/libping-fixture.go against pinned Xray first")
	}
	helper, _ = filepath.Abs(helper)
	plan := convert.Prepare([]byte("vless://11111111-1111-4111-8111-111111111111@127.0.0.1:24443?security=none"))
	if len(plan.Candidates) != 1 {
		t.Fatal(plan.Code)
	}
	b, err := Compose([]Member{{Body: plan.Candidates[0].Config}})
	if err != nil {
		t.Fatal(err)
	}
	var config any
	_ = json.Unmarshal(b, &config)
	request, _ := json.Marshal(map[string]any{"helper": helper, "config": config})
	cmd := exec.Command("python3", "../../scripts/smart-libping-test.py")
	cmd.Stdin = bytes.NewReader(request)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("libping: %v\n%s", err, out)
	}
	var receipt struct{ Passed bool }
	if json.Unmarshal(out, &receipt) != nil || !receipt.Passed {
		t.Fatal(string(out))
	}
	t.Log(string(out))
}
