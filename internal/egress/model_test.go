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
	r := Report{Version: Version, Hash: "fixture", Started: now - 1000, Finished: now, Core: "xray-26.11.9", CoreSHA: Cores["xray-26.11.9"], Infrastructure: true}
	for _, v := range p.Variants {
		r.Checks = append(r.Checks, Check{v.Name, v.Hash, v.Hash, true, "ok"})
		for wave := 1; wave <= 3; wave++ {
			for _, d := range append(append([]string{}, Domains...), "www.cloudflare.com", "api.ipify.org") {
				ip := "1.1.1.1"
				ads := d != "www.cloudflare.com" && d != "api.ipify.org"
				if ads {
					ip = "8.8.8.8"
				}
				n := Hash([]byte(fmt.Sprint(v.Name, wave, d)))[:32]
				r.Samples = append(r.Samples, Sample{Variant: v.Name, Wave: wave, Domain: d, At: now, IP: ip, Family: 4, Transport: "tcp", TLS: true, Echo: true, SameStream: true, ActualAds: ads, Nonce: n, Connection: n, Certificate: strings.Repeat("c", 64), Response: strings.Repeat("d", 64)})
			}
		}
	}
	r.Facts = []Fact{{IP: "8.8.8.8", Provider: "proxycheck-v3", Class: "Residential", At: now, Expires: now + 100000, Digest: strings.Repeat("f", 64)}}
	return r, p, now
}
func TestFiniteObservationPolicy(t *testing.T) {
	r, p, now := fixture()
	if d := Evaluate(r, p, "fixture", now); !d.Eligible || d.Route != "DIFFERENT" {
		t.Fatal(d)
	}
	cases := []struct {
		name   string
		change func(*Report, *Plan)
	}{{"tls", func(r *Report, p *Plan) { r.Samples[0].TLS = false }}, {"reconnected", func(r *Report, p *Plan) { r.Samples[0].SameStream = false }}, {"nonce", func(r *Report, p *Plan) { r.Samples[0].Nonce = "wrong" }}, {"public_ip_cannot_be_ads", func(r *Report, p *Plan) { r.Samples[0].ActualAds = false }}, {"expired_provider", func(r *Report, p *Plan) { r.Facts[0].Expires = now - 1 }}, {"asn_is_not_class", func(r *Report, p *Plan) { r.Facts[0].Class = ""; r.Facts[0].ASN = "AS4609" }}, {"wrong_variant", func(r *Report, p *Plan) { r.Checks[1].Hash = "wrong" }}, {"partial_coverage", func(r *Report, p *Plan) { r.Samples = r.Samples[1:] }}, {"multipath", func(r *Report, p *Plan) { p.Complete = false }}, {"fake_core", func(r *Report, p *Plan) { r.CoreSHA = "wrong" }}, {"wrong_ip_class", func(r *Report, p *Plan) { r.Facts[0].IP = "1.1.1.1" }}, {"mixed_family", func(r *Report, p *Plan) { r.Samples[0].Family = 6 }}}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, p, now := fixture()
			c.change(&r, &p)
			if d := Evaluate(r, p, "fixture", now); d.Eligible {
				t.Fatal("false positive", d)
			}
		})
	}
	r.Facts[0].Class = "Hosting"
	if d := Evaluate(r, p, "fixture", now); d.Eligible || !d.Negative {
		t.Fatal(d)
	}
	r.Facts = nil
	if d := Evaluate(r, p, "fixture", now); d.Negative {
		t.Fatal("missing classifier caused purge")
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
