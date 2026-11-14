package egress

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// All IPs/classes in this fixture are synthetic policy inputs, NOT live findings.
func fixture() (Report, Plan, int64) {
	now := Now()
	p := Prepare([]byte("vless://00000000-0000-4000-8000-000000000001@192.0.2.1:443?security=tls&sni=example.com"))
	r := Report{Version: Version, Hash: "fixture", Started: now - 1000, Finished: now, Core: "xray-26.11.9", CoreSHA: Cores["xray-26.11.9"], Infrastructure: true, Waves: 3}
	for _, v := range p.Variants {
		r.Checks = append(r.Checks, Check{v.Name, v.Hash, v.Hash, true, "ok"})
		for wave := 1; wave <= 3; wave++ {
			for _, d := range append(append([]string{}, Domains...), ControlDomains...) {
				ip := "1.1.1.1"
				ads := false
				for _, a := range Domains {
					if a == d {
						ads = true
					}
				}
				if ads {
					ip = "8.8.8.8"
				}
				n := Hash([]byte(fmt.Sprint(v.Name, wave, d)))[:32]
				r.Samples = append(r.Samples, Sample{RequestBytes: 200, ResponseBytes: 1000, ResponseComplete: true, Variant: v.Name, Wave: wave, Domain: d, At: now, Source: Hash([]byte(ip)), Family: 4, Transport: "tcp", TLS: true, Echo: true, SameStream: true, ActualAds: ads, Nonce: n, Connection: n, Certificate: strings.Repeat("c", 64), Response: strings.Repeat("d", 64)})
			}
		}
	}
	return r, p, now
}
func TestFiniteObservationPolicy(t *testing.T) {
	r, p, now := fixture()
	r.Waves = 2
	keep := []Sample{}
	for _, x := range r.Samples {
		if x.Wave <= 2 {
			keep = append(keep, x)
		}
	}
	r.Samples = keep
	if d := Evaluate(r, p, "fixture", now); !d.Eligible || d.Route != "DIFFERENT" || d.Reason != "repeated_ads_exclusive_observation" {
		t.Fatal(d)
	}
	cases := []struct {
		name   string
		change func(*Report, *Plan)
	}{
		{"tls", func(r *Report, p *Plan) { r.Samples[0].TLS = false }},
		{"reconnected", func(r *Report, p *Plan) { r.Samples[0].SameStream = false }},
		{"nonce", func(r *Report, p *Plan) { r.Samples[0].Nonce = "wrong" }},
		{"control_is_not_ads", func(r *Report, p *Plan) { r.Samples[0].ActualAds = false }},
		{"wrong_variant", func(r *Report, p *Plan) { r.Checks[1].Hash = "wrong" }},
		{"rewritten_raw", func(r *Report, p *Plan) { r.Checks[0].RuntimeHash = Hash([]byte("wrong")) }},
		{"partial", func(r *Report, p *Plan) { r.Samples = r.Samples[1:] }},
		{"multipath", func(r *Report, p *Plan) { p.Complete = false }},
		{"fake_core", func(r *Report, p *Plan) { r.CoreSHA = "wrong" }},
		{"mixed_family", func(r *Report, p *Plan) { r.Samples[0].Family = 6 }},
		{"plaintext_source", func(r *Report, p *Plan) { r.Samples[0].Source = "8.8.8.8" }},
		{"replayed_control", func(r *Report, p *Plan) {
			r.Samples[0].Connection = r.Samples[1].Connection
			r.Samples[0].Nonce = r.Samples[1].Nonce
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var copy Report
			b, _ := json.Marshal(r)
			json.Unmarshal(b, &copy)
			pp := p
			pp.Variants = append([]Variant{}, p.Variants...)
			c.change(&copy, &pp)
			if d := Evaluate(copy, pp, "fixture", now); d.Eligible {
				t.Fatal("false positive", d)
			}
		})
	}
}
func TestPreparationPreservesProtocolAndSecrets(t *testing.T) {
	raw := []byte("vless://00000000-0000-4000-8000-000000000001@192.0.2.1:443?security=tls&sni=example.com")
	p := Prepare(raw)
	if len(p.Variants) != 2 || !p.Complete {
		t.Fatal(p.Code)
	}
	for _, v := range p.Variants {
		if !json.Valid(v.Config) || v.Hash != Hash(v.Config) {
			t.Fatal(v.Name)
		}
	}
	p = Prepare([]byte(`{"outbounds":[{"protocol":"freedom"}]}`))
	if p.Complete {
		t.Fatal("direct marked complete")
	}
}

func TestDNSDialerIsNotSecondEgress(t *testing.T) {
	raw := []byte(`{"outbounds":[{"tag":"proxy","protocol":"vless","settings":{"vnext":[{"address":"8.8.8.8","port":443,"users":[{"id":"00000000-0000-4000-8000-000000000001","encryption":"none"}]}]},"streamSettings":{"security":"tls","tlsSettings":{"serverName":"example.com"}}},{"tag":"dns-out","protocol":"dns","streamSettings":{"sockopt":{"dialerProxy":"proxy"}}},{"tag":"direct","protocol":"freedom"}],"routing":{"rules":[{"outboundTag":"dns-out","port":"53"},{"outboundTag":"proxy","network":"tcp,udp"}]}}`)
	before := string(raw)
	if !singlePath(raw) || string(raw) != before {
		t.Fatal("safe DNS dependency rejected or input changed")
	}
	if singlePath([]byte(strings.Replace(before, `"outboundTag":"proxy"`, `"outboundTag":"direct"`, 1))) {
		t.Fatal("direct destination rule accepted")
	}
	if singlePath([]byte(strings.Replace(before, `"dialerProxy":"proxy"`, `"dialerProxy":"missing"`, 1))) {
		t.Fatal("unknown DNS path accepted")
	}
	if singlePath([]byte(strings.Replace(before, `"security":"tls"`, `"sockopt":{"dialerProxy":"dns-out"},"security":"tls"`, 1))) {
		t.Fatal("cycle accepted")
	}
}
func TestOverlapNeverQualifiesAsDedicatedResidential(t *testing.T) {
	r, p, now := fixture()
	for i := range r.Samples {
		r.Samples[i].Source = Hash([]byte("8.8.8.8"))
	}
	d := Evaluate(r, p, "fixture", now)
	if d.Eligible || d.State != "NO_SPLIT" || !d.Negative {
		t.Fatal(d)
	}
	r, p, now = fixture()
	d = Evaluate(r, p, "fixture", now)
	if !d.Eligible || d.State != "RESIDENTIAL" || d.Route != "DIFFERENT" || len(d.Sources) != 1 {
		t.Fatal(d)
	}
}
func TestPartialEchoRetainsEvidenceWithoutQualifying(t *testing.T) {
	r, p, now := fixture()
	stride := len(Domains) + len(ControlDomains)
	r.Samples = append(r.Samples[1:stride], r.Samples[stride+1:]...)
	d := Evaluate(r, p, "fixture", now)
	if d.Eligible || d.Route != "PARTIAL_DIFFERENCE" || len(d.Sources) == 0 {
		t.Fatal(d)
	}
}

func TestTwoPairedRepeatsPerDomainAndVariantQualify(t *testing.T) {
	r, p, now := fixture()
	r.Samples = r.Samples[1:]
	d := Evaluate(r, p, "fixture", now)
	if !d.Eligible || d.State != "RESIDENTIAL" || d.Route != "DIFFERENT" || d.Missing != 0 {
		t.Fatal(d)
	}
}

func TestResolverBypassIsOutsideNamedHTTPSScope(t *testing.T) {
	raw := `{"outbounds":[{"tag":"proxy","protocol":"vless"},{"tag":"direct","protocol":"freedom"}],"routing":{"rules":[RULE]}}`
	for _, rule := range []string{`{"type":"field","port":"53","outboundTag":"direct"}`, `{"type":"field","ip":["1.1.1.1","104.16.248.249"],"outboundTag":"direct"}`} {
		if !singlePath([]byte(strings.Replace(raw, "RULE", rule, 1))) {
			t.Fatal("resolver-only rule rejected", rule)
		}
	}
	for _, rule := range []string{`{"port":"443","outboundTag":"direct"}`, `{"ip":["1.1.1.0/24"],"outboundTag":"direct"}`, `{"ip":["1.1.1.1","8.7.6.5"],"outboundTag":"direct"}`, `{"domain":["googleads.g.doubleclick.net"],"outboundTag":"direct"}`, `{"ip":["1.1.1.1"],"unknown":"unsafe","outboundTag":"direct"}`} {
		if singlePath([]byte(strings.Replace(raw, "RULE", rule, 1))) {
			t.Fatal("unsafe direct rule accepted", rule)
		}
	}
}
func TestDriverFailureIsActionableAndNeverNegative(t *testing.T) {
	r, p, now := fixture()
	r.Checks = nil
	r.Code = "direct_route_not_qualified"
	d := Evaluate(r, p, "fixture", now)
	if d.Reason != r.Code || d.Eligible || d.Negative {
		t.Fatal(d)
	}
}

func TestRotatingAdsPoolAndConflictingEvidence(t *testing.T) {
	r, p, now := fixture()
	r.Samples[1].Source = Hash([]byte("9.9.9.9"))
	if d := Evaluate(r, p, "fixture", now); !d.Eligible {
		t.Fatal("rotating split rejected", d)
	}
	r.Samples[1].Source = Hash([]byte("1.1.1.1"))
	if d := Evaluate(r, p, "fixture", now); d.Eligible || d.Negative || d.Reason != "mixed_route_observations" {
		t.Fatal("mixed routes", d)
	}
	r, p, now = fixture()
	r.Samples[len(Domains)].Source = Hash([]byte("2.2.2.2"))
	if d := Evaluate(r, p, "fixture", now); !d.Eligible || d.Negative {
		t.Fatal("heterogeneous controls rejected", d)
	}
	r, p, now = fixture()
	r.Samples[1].Source = Hash([]byte("1.1.1.1"))
	r.Samples[len(Domains)].TLS = false
	r.Samples[len(Domains)+1].TLS = false
	if d := Evaluate(r, p, "fixture", now); d.Eligible {
		t.Fatal("unpaired overlap ignored", d)
	}
}

func TestRawOnlySharedRouteIsARejectionNeverAcceptance(t *testing.T) {
	r, p, now := fixture()
	r.Code = "raw_shared_egress"
	r.Checks = r.Checks[:1]
	if d := Evaluate(r, p, "fixture", now); d.Eligible || d.Negative {
		t.Fatal("missing template promoted", d)
	}
	for i := range r.Samples {
		r.Samples[i].Source = Hash([]byte("1.1.1.1"))
	}
	if d := Evaluate(r, p, "fixture", now); d.Eligible || !d.Negative || d.State != "NO_SPLIT" {
		t.Fatal(d)
	}
	for i := range r.Samples {
		if r.Samples[i].Domain == Domains[0] {
			r.Samples[i].TLS = false
		}
	}
	if d := Evaluate(r, p, "fixture", now); d.Eligible || d.Negative || d.State == "NO_SPLIT" {
		t.Fatal("incomplete raw rejected", d)
	}
}
