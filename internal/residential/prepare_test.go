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
