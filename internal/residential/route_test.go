package residential

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
)

func routeFixture() (Report, Policy, Trust, int64) {
	r, p, tr, now := qualifiedFixture()
	p.Repeats = 3
	r.PolicyHash = p.Digest()
	r.IPFacts = nil
	r.Controls = nil
	for _, v := range r.Variants {
		for _, o := range append([]Observation{}, r.Observations...) {
			if o.Variant == v.Name && o.Wave == 2 {
				o.Wave = 3
				r.Observations = append(r.Observations, o)
			}
		}
		for wave := 1; wave <= 3; wave++ {
			for i, h := range []string{"www.cloudflare.com", "api.ipify.org"} {
				r.Controls = append(r.Controls, Observation{Scope: Scope{v.Name, h, 4, "tcp", v.Paths[0]}, At: now, Wave: wave, IP: "1.1.1.1", ConnectionID: fmt.Sprintf("control-%s-%d-%d-unique", v.Name, wave, i), TLSVerified: true, OriginVerified: true, SameConnection: true, Connections: 1, ActualDomainRequest: true, ResponseSHA256: strings.Repeat("a", 64), Outcome: "passed"})
			}
		}
	}
	for i := range r.Observations {
		o := &r.Observations[i]
		o.PolicyHash = p.Digest()
		o.ConnectionID = fmt.Sprintf("ads-%s-%d-unique-connection", o.Variant, o.Wave)
		o.Nonce = o.ConnectionID
		signRoute(o)
	}
	return r, p, tr, now
}
func signRoute(o *Observation) {
	o.Attestation = base64.StdEncoding.EncodeToString(ed25519.Sign(ed25519.NewKeyFromSeed([]byte(strings.Repeat("s", 32))), SigningPayload(*o)))
}

func TestAdsDifferenceIsNotResidentialClassification(t *testing.T) {
	r, p, tr, now := routeFixture()
	d := Evaluate(r, p, tr, now)
	if !d.Eligible || d.State != "ADS_DIFFERENT" || d.Network != "Unknown" || d.CredibleNegative || len(d.AdsRoute.Different) != 2 {
		t.Fatalf("%+v", d)
	}
	for _, source := range []string{"specialist", "independent"} {
		r.IPFacts = append(r.IPFacts, IPFact{IP: "8.8.8.8", Source: source, At: now, Expires: now + 600000, Class: "Datacenter", EvidenceSHA256: strings.Repeat("a", 64)})
	}
	d = Evaluate(r, p, tr, now)
	if !d.Eligible || d.State != "ADS_DIFFERENT" || d.Network != "Datacenter" || d.CredibleNegative {
		t.Fatalf("separate axes %+v", d)
	}
}
func TestAdsDifferenceCannotUseRotationCandidatesOrMissingControls(t *testing.T) {
	cases := map[string]func(*Report){
		"control-rotation": func(r *Report) {
			for i := range r.Controls {
				if r.Controls[i].Wave == 2 {
					r.Controls[i].IP = "9.9.9.9"
				}
			}
		},
		"cross-wave-overlap": func(r *Report) {
			for i := range r.Observations {
				if r.Observations[i].Wave == 2 {
					r.Observations[i].IP = "1.1.1.1"
					signRoute(&r.Observations[i])
				}
			}
		},
		"only-one-independent-control": func(r *Report) {
			for i := range r.Controls {
				r.Controls[i].Domain = "www.cloudflare.com"
			}
		},
		"candidate-not-attributed": func(r *Report) {
			for i := range r.Observations {
				r.Observations[i].Attestation = ""
			}
		},
		"separate-connection": func(r *Report) {
			for i := range r.Observations {
				r.Observations[i].SameConnection = false
				signRoute(&r.Observations[i])
			}
		},
		"family-mismatch": func(r *Report) {
			for i := range r.Controls {
				r.Controls[i].IP = "2606:4700:4700::1111"
				r.Controls[i].Family = 6
			}
		},
		"control-path-mismatch": func(r *Report) {
			for i := range r.Controls {
				r.Controls[i].Path = "other"
			}
		},
		"replayed-nonce": func(r *Report) {
			for i := range r.Observations {
				r.Observations[i].Nonce = "same-ads-nonce-123456789"
				signRoute(&r.Observations[i])
			}
		},
		"replayed-control": func(r *Report) {
			for i := range r.Controls {
				r.Controls[i].ConnectionID = "same-control-connection-12345"
			}
		},
		"missing-wave":          func(r *Report) { r.Controls = r.Controls[2:] },
		"unverified-balancer":   func(r *Report) { r.Variants[0].PathsComplete = false },
		"infrastructure-outage": func(r *Report) { r.InfrastructureOK = false },
		"unhealthy":             func(r *Report) { r.Variants[0].Health = "UNHEALTHY" },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			r, p, tr, now := routeFixture()
			change(&r)
			d := Evaluate(r, p, tr, now)
			if d.Eligible || d.AdsRoute.State == "DIFFERENT" || d.CredibleNegative {
				t.Fatalf("%+v", d)
			}
		})
	}
}
func TestAdsLossRequiresSamePreviouslyAcceptedScopes(t *testing.T) {
	r, p, tr, now := routeFixture()
	before := Evaluate(r, p, tr, now)
	for i := range r.Observations {
		r.Observations[i].IP = "1.1.1.1"
		signRoute(&r.Observations[i])
	}
	after := Evaluate(r, p, tr, now)
	if after.Eligible || after.AdsRoute.State != "SAME" || !LostQualification(before, after) {
		t.Fatalf("%+v", after)
	}
	after.AdsRoute.Same = after.AdsRoute.Same[:1]
	if LostQualification(before, after) {
		t.Fatal("missing prior scope voted deletion")
	}
	if LostQualification(before, Decision{Health: "HEALTHY", CredibleNegative: true, Network: "Datacenter"}) {
		t.Fatal("network class alone revoked different route")
	}
}

func TestAdsComparisonCanonicalIPv6AndMappedFamily(t *testing.T) {
	for _, same := range []bool{true, false} {
		r, p, tr, now := routeFixture()
		p.Families = []int{6}
		r.PolicyHash = p.Digest()
		for i := range r.Observations {
			o := &r.Observations[i]
			o.Family = 6
			o.IP = "2606:4700::1111"
			o.PolicyHash = p.Digest()
			signRoute(o)
		}
		for i := range r.Controls {
			o := &r.Controls[i]
			o.Family = 6
			if same {
				o.IP = "2606:4700:0:0:0:0:0:1111"
			} else if i%2 == 0 {
				o.IP = "2001:4860:4860::8888"
			} else {
				o.IP = "2001:4860:4860:0:0:0:0:8888"
			}
		}
		d := Evaluate(r, p, tr, now)
		if same && (d.Eligible || d.AdsRoute.State != "SAME") || !same && (!d.Eligible || d.AdsRoute.State != "DIFFERENT") {
			t.Fatal(same, d)
		}
	}
	if _, ok := publicIP("::ffff:8.8.8.8"); ok {
		t.Fatal("mapped IPv4 cannot masquerade as IPv6 evidence")
	}
}
