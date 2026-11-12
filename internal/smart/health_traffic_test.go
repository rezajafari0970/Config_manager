package smart

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPayloadObserverRejectsTruncationAndRecovers(t *testing.T) {
	if os.Getenv("XRAY_TRAFFIC_TEST") != "1" {
		t.Skip("explicit isolated traffic test")
	}
	core, e := filepath.Abs(os.Getenv("XRAY_TEST_BINARY"))
	if e != nil {
		t.Fatal(e)
	}
	members := []Member{}
	for _, port := range []int{24449, 24450} {
		body, _ := json.Marshal(M{"outbounds": []any{M{"tag": "proxy", "protocol": "vless", "settings": M{"address": "127.0.0.1", "port": port, "id": "11111111-1111-4111-8111-111111111111", "encryption": "none"}}},
			"routing": M{"rules": []any{M{"type": "field", "network": "tcp,udp", "outboundTag": "proxy"}}}})
		members = append(members, Member{Body: body})
	}
	candidate, e := Compose(members)
	if e != nil {
		t.Fatal(e)
	}
	old, _ := object(candidate)
	old["burstObservatory"].(M)["pingConfig"] = M{"destination": "https://www.gstatic.com/generate_204", "interval": "15s", "sampling": 3, "timeout": "5s", "httpMethod": "GET"}
	for _, v := range old["routing"].(M)["balancers"].([]any) {
		b := v.(M)
		if b["fallbackTag"] == "block" {
			b["strategy"].(M)["settings"] = M{"expected": 1, "maxRTT": "5s", "tolerance": 0.5}
		}
	}
	baseline, e := encodeEnvelope(old)
	if e != nil {
		t.Fatal(e)
	}
	dir := t.TempDir()
	before := filepath.Join(dir, "baseline.json")
	after := filepath.Join(dir, "candidate.json")
	receipt := filepath.Join(dir, "receipt.json")
	if e = os.WriteFile(before, baseline, 0600); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(after, candidate, 0600); e != nil {
		t.Fatal(e)
	}
	cmd := exec.Command("python3", "../../scripts/smart-payload-observer-test.py", "--core", core, "--baseline", before, "--candidate", after, "--slot", "8212", "--output", receipt)
	out, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatalf("payload observation: %v\n%s", e, out)
	}
	b, e := os.ReadFile(receipt)
	if e != nil {
		t.Fatal(e)
	}
	var result struct{ Passed bool }
	if json.Unmarshal(b, &result) != nil || !result.Passed {
		t.Fatal(string(b))
	}
	if bytes.Equal(candidate, baseline) {
		t.Fatal("missing baseline distinction")
	}
	t.Log(string(out))
}
