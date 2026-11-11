package convert

import (
	"bytes"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/compat"
	"testing"
)

func TestLegacyRelaySharedConversionIsIdempotent(t *testing.T) {
	raw := []byte("vless://11111111-1111-4111-8111-111111111111@203.0.113.81:443?security=none&type=tcp&headerType=http&host=fast.com")
	plan := Prepare(raw)
	if len(plan.Candidates) != 1 {
		t.Fatal(plan.Code)
	}
	c := plan.Candidates[0]
	if !c.RequiresConnection {
		t.Fatal("unverified publication")
	}
	var root M
	json.Unmarshal(c.Config, &root)
	if len(root["inbounds"].([]any)) != 1 {
		t.Fatal("unexpected local relay")
	}
	again, e := Apply(c.Config)
	if e != nil || !bytes.Equal(again, c.Config) {
		t.Fatal("not idempotent", e)
	}
	ch, e := compat.Logical(root)
	if e != nil || len(ch) != 1 {
		t.Fatal(e)
	}
	ob := root["outbounds"].([]any)[0].(M)
	node := ob["settings"].(M)["vnext"].([]any)[0].(M)
	if node["address"] != "203.0.113.81" {
		t.Fatal("server changed")
	}
}
