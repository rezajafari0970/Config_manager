package smart

import (
	"bytes"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/compat"
	"github.com/rezajafari0970/Config_manager/internal/convert"
	"testing"
)

func relayMember(t *testing.T, host string) Member {
	t.Helper()
	p := convert.Prepare([]byte("vless://11111111-1111-4111-8111-111111111111@" + host + ":443?security=none&type=tcp&headerType=http&host=fast.com"))
	if len(p.Candidates) != 1 {
		t.Fatal(p.Code)
	}
	return Member{Body: p.Candidates[0].Config}
}
func TestSmartRedirectsPreserveEndpointsAndCarrier(t *testing.T) {
	a, b := relayMember(t, "203.0.113.81"), relayMember(t, "203.0.113.82")
	for _, ads := range []bool{false, true} {
		var body []byte
		var e error
		if ads {
			body, e = ComposeAds([]Member{a}, b)
		} else {
			body, e = Compose([]Member{a, b})
		}
		if e != nil {
			t.Fatal(e)
		}
		previous := -1
		for _, key := range []string{"remarks", "log", "dns", "inbounds", "outbounds", "routing", "burstObservatory", "policy", "stats", "metrics"} {
			pos := bytes.Index(body, []byte(`"`+key+`":`))
			if pos <= previous {
				t.Fatal("relay changed root field order", key)
			}
			previous = pos
		}
		var root M
		json.Unmarshal(body, &root)

		if len(root["inbounds"].([]any)) != 1 {
			t.Fatal("local listener remains")
		}
		count := 0
		carrier := false
		targets := map[string]bool{}
		for _, v := range root["outbounds"].([]any) {
			m := v.(M)
			settings, _ := m["settings"].(M)
			redirect, _ := settings["redirect"].(string)
			if m["protocol"] == "freedom" && redirect != "" {
				if targets[redirect] {
					t.Fatal("duplicate redirect")
				}
				targets[redirect] = true
				count++
				s, _ := m["streamSettings"].(M)
				o, _ := s["sockopt"].(M)
				if o["dialerProxy"] == "main-chain-dispatch" {
					carrier = true
				}
			}
		}
		if count != 2 || ads && !carrier {
			t.Fatal(count, carrier)
		}
		if _, e = compat.Logical(root); e != nil {
			t.Fatal("unrecognized relay graph", e)
		}
	}
}
