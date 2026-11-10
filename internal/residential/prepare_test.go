package residential

import (
	"bytes"
	"testing"
)

func TestOriginalJSONWhitespaceRemainsExactAndMultiPathPartial(t *testing.T) {
	raw := []byte("{\n  \"outbounds\": [{\"tag\":\"p\",\"protocol\":\"socks\",\"settings\":{\"servers\":[{\"address\":\"192.0.2.1\",\"port\":1080}]}}]\n}\n")
	plan := NewCoreRegistry().Prepare("xray-26.7.28", raw)
	if len(plan.Variants) != 2 || !bytes.Equal(plan.Variants[0].Config, raw) || plan.Variants[0].SHA256 != Digest(raw) {
		t.Fatalf("raw representation changed: %s", plan.Code)
	}
	for _, raw := range [][]byte{[]byte(`{"outbounds":[{"tag":"a","protocol":"socks"},{"tag":"b","protocol":"socks"}],"routing":{"balancers":[{"tag":"pool","selector":["a","b"]}]}}`), []byte(`{"outbounds":[{"tag":"a","protocol":"socks"}],"routing":{"rules":[{"domain":["geosite:google"],"outboundTag":"a"}]}}`)} {
		if _, complete := executionPaths(raw); complete {
			t.Fatal("unresolved path coverage accepted")
		}
	}
}

func TestRawImportUsesOnlyRecordedStructuralRepair(t *testing.T) {
	raw := []byte(`{"outbounds":[{"tag":"p","protocol":"socks","settings":{"servers":[{"address":"192.0.2.1","port":1080}]}}],"inbounds":[],"routing":{},"stats":[]}`)
	before := append([]byte{}, raw...)
	p := NewCoreRegistry().Prepare("xray-26.7.28", raw)
	if len(p.Variants) != 2 || p.Variants[0].ImportRule == "original-json" || !bytes.Contains(p.Variants[0].Config, []byte(`"stats":{}`)) {
		t.Fatal("safe structural repair discarded", p.Code)
	}
	if !bytes.Equal(raw, before) {
		t.Fatal("original changed")
	}
	invalid := bytes.Replace(raw, []byte(`"stats":[]`), []byte(`"stats":[1]`), 1)
	if p = NewCoreRegistry().Prepare("xray-26.7.28", invalid); len(p.Variants) != 0 {
		t.Fatal("ambiguous nonempty value repaired")
	}
}

func TestDirectDefaultIsNotCompleteProxyCoverage(t *testing.T) {
	for _, tc := range []struct {
		rules    string
		complete bool
	}{
		{`[]`, false},
		{`[{"type":"field","network":"tcp","outboundTag":"proxy"}]`, false},
		{`[{"type":"field","network":"tcp,udp","outboundTag":"proxy"}]`, true},
	} {
		raw := []byte(`{"outbounds":[{"tag":"direct","protocol":"freedom"},{"tag":"proxy","protocol":"vless"}],"routing":{"rules":` + tc.rules + `}}`)
		_, complete := executionPaths(raw)
		if complete != tc.complete {
			t.Fatal(tc.rules, complete)
		}
	}
}
