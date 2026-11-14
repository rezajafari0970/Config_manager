package egress

import (
	"encoding/json"
	"testing"
)

func TestExclusiveControlsAreIndependentAndGoogleIsRequired(t *testing.T) {
	r, p, now := fixture()
	for i := range r.Samples {
		s := &r.Samples[i]
		if !s.ActualAds {
			s.Source = Hash([]byte(s.Domain))
		}
	}
	if d := Evaluate(r, p, r.Hash, now); !d.Eligible || !d.CoverageComplete {
		t.Fatal(d)
	}
	for i := range r.Samples {
		if r.Samples[i].Domain == "www.google.com" {
			r.Samples[i].Source = Hash([]byte("8.8.8.8"))
		}
	}
	if d := Evaluate(r, p, r.Hash, now); d.Eligible || d.State != "NO_SPLIT" || d.Reason != "non_ads_uses_ads_egress" {
		t.Fatal(d)
	}
}
func TestReachabilityAndDifferentFamilyAreNotRouteProof(t *testing.T) {
	for _, kind := range []string{"reachable", "timeout", "ipv6"} {
		t.Run(kind, func(t *testing.T) {
			r, p, now := fixture()
			for i := range r.Samples {
				s := &r.Samples[i]
				if s.Domain != "t.me" {
					continue
				}
				switch kind {
				case "reachable":
					s.Echo = false
					s.Source = ""
					s.Code = "reachable_no_source"
				case "timeout":
					s.Echo = false
					s.TLS = false
					s.Code = "connection_timeout"
				case "ipv6":
					s.Family = 6
				}
			}
			if d := Evaluate(r, p, r.Hash, now); d.Eligible || d.Negative || d.Reason != "controls_incomplete" {
				t.Fatal(d)
			}
		})
	}
}
func TestOnlyBoundLocalBlockCountsAsBlockedControl(t *testing.T) {
	r, p, now := fixture()
	for i := range p.Variants {
		v := &p.Variants[i]
		var cfg map[string]any
		json.Unmarshal(v.Config, &cfg)
		cfg["outbounds"] = append(cfg["outbounds"].([]any), map[string]any{"tag": "p58-deny", "protocol": "blackhole"})
		cfg["routing"] = map[string]any{"domainStrategy": "AsIs", "rules": []any{map[string]any{"type": "field", "domain": []string{"full:t.me"}, "outboundTag": "p58-deny"}}}
		v.Config, _ = json.Marshal(cfg)
		v.Hash = Hash(v.Config)
		r.Checks[i].Hash = v.Hash
		r.Checks[i].RuntimeHash = v.Hash
	}
	for i := range r.Samples {
		if r.Samples[i].Domain == "t.me" {
			r.Samples[i].Echo = false
			r.Samples[i].TLS = false
			r.Samples[i].Source = ""
			r.Samples[i].Code = "socks_connect"
		}
	}
	if d := Evaluate(r, p, r.Hash, now); !d.Eligible {
		t.Fatal(d)
	}
	for i := range p.Variants {
		v := &p.Variants[i]
		var cfg map[string]any
		json.Unmarshal(v.Config, &cfg)
		cfg["routing"] = map[string]any{}
		v.Config, _ = json.Marshal(cfg)
		v.Hash = Hash(v.Config)
		r.Checks[i].Hash = v.Hash
		r.Checks[i].RuntimeHash = v.Hash
	}
	if d := Evaluate(r, p, r.Hash, now); d.Eligible {
		t.Fatal("timeouts invented a block", d)
	}
}
func TestAdsHealthIsBoundBidirectionalAndNotGenericPayload(t *testing.T) {
	r, p, now := fixture()
	if got := AdsHealth(r, p, now); got != "healthy" {
		t.Fatal(got)
	}
	for i := range r.Samples {
		if r.Samples[i].ActualAds {
			r.Samples[i].RequestBytes = 0
		}
	}
	if got := AdsHealth(r, p, now); got != "unknown" {
		t.Fatal("zero upload qualified", got)
	}
}
func TestEvidenceExpiresFromObservationNotCompletion(t *testing.T) {
	r, p, now := fixture()
	r.Started = now - 200000
	for i := range r.Samples {
		r.Samples[i].At = r.Started
	}
	d := Evaluate(r, p, r.Hash, now)
	if !d.Eligible || d.Until != r.Started+EvidenceTTL {
		t.Fatal(d)
	}
}
