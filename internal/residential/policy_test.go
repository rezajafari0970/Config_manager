package residential

import (
	"crypto/ed25519"
	"encoding/base64"
	"strings"
	"testing"
)

func qualifiedFixture() (Report, Policy, Trust, int64) {
	now := int64(1800000000000)
	p := DefaultPolicy()
	p.Domains = []string{"googleads.g.doubleclick.net"}
	p.Families = []int{4}
	p.Transports = []string{"tcp"}
	p.Repeats = 2
	trust := EmptyTrust()
	trust.Providers = map[string]Provider{"specialist": {Group: "one", Specialist: true}, "independent": {Group: "two"}}
	key := ed25519.NewKeyFromSeed([]byte(strings.Repeat("s", 32)))
	trust.Capabilities["fixture-attestation"] = Capability{Method: "server-flow-ed25519-v1", PublicKey: base64.StdEncoding.EncodeToString(key.Public().(ed25519.PublicKey)), Domains: p.Domains, Transports: p.Transports, Expires: now + 600000, EvidenceSHA256: strings.Repeat("a", 64)}
	r := Report{Version: Version, PolicyHash: p.Digest(), Started: now - 10000, Finished: now, InfrastructureOK: true, Worker: "fixture", InfrastructureEpoch: "fixture-epoch"}
	for _, name := range []string{"raw", "template"} {
		r.Variants = append(r.Variants, Variant{Name: name, Hash: strings.Repeat("b", 64), RuntimeHash: strings.Repeat("c", 64), Core: "passed", Health: "HEALTHY", Paths: []string{"single"}, PathsComplete: true})
		for wave := 1; wave <= 2; wave++ {
			for _, h := range []string{"www.cloudflare.com", "api.ipify.org"} {
				r.Controls = append(r.Controls, Observation{Scope: Scope{Variant: name, Domain: h}, At: now, Wave: wave, TLSVerified: true, OriginVerified: true, Outcome: "passed"})
			}
			r.Observations = append(r.Observations, Observation{Scope: Scope{name, p.Domains[0], 4, "tcp", "single"}, At: now, Wave: wave, ConnectionID: strings.Repeat(string(rune('d'+wave)), 32), Nonce: strings.Repeat("f", 32), IP: "8.8.8.8", Method: "server-flow-ed25519-v1", ConfigHash: strings.Repeat("b", 64), RuntimeHash: strings.Repeat("c", 64), PolicyHash: p.Digest(), Capability: "fixture-attestation", TLSVerified: true, OriginVerified: true, MeaningVerified: true, NonceBound: true, SameConnection: true, Connections: 1, ActualDomainRequest: true, ResponseSHA256: strings.Repeat("e", 64), Outcome: "passed"})
		}
	}
	// Synthetic evidence is policy-fixture data, not a real classification of IPs.
	for _, provider := range []string{"specialist", "independent"} {
		r.IPFacts = append(r.IPFacts, IPFact{IP: "8.8.8.8", Source: provider, At: now, Expires: now + 600000, Class: "Residential", EvidenceSHA256: strings.Repeat("a", 64)})
	}
	for i := range r.Observations {
		r.Observations[i].Attestation = base64.StdEncoding.EncodeToString(ed25519.Sign(key, SigningPayload(r.Observations[i])))
	}
	return r, p, trust, now
}
func TestQualifiedAndAllResidentialControls(t *testing.T) {
	r, p, tr, now := qualifiedFixture()
	for i := range r.Controls {
		r.Controls[i].IP = "8.8.8.8"
	}
	d := Evaluate(r, p, tr, now)
	if !d.Eligible || d.ValidUntil != now-10000+600000 {
		t.Fatalf("%+v", d)
	}
}
func TestEvidenceBoundaries(t *testing.T) {
	cases := []struct {
		name   string
		change func(*Report, *Policy, *Trust)
	}{
		{"untrusted-method", func(r *Report, p *Policy, tr *Trust) { tr.Capabilities = map[string]Capability{} }},
		{"separate-connection", func(r *Report, p *Policy, tr *Trust) { r.Observations[0].SameConnection = false }},
		{"nonce-not-bound", func(r *Report, p *Policy, tr *Trust) { r.Observations[0].NonceBound = false }},
		{"unproven-semantics", func(r *Report, p *Policy, tr *Trust) { r.Observations[0].MeaningVerified = false }},
		{"intel-conflict", func(r *Report, p *Policy, tr *Trust) { r.IPFacts[1].Class = "Datacenter" }},
		{"same-provider-group", func(r *Report, p *Policy, tr *Trust) { tr.Providers["independent"] = Provider{Group: "one"} }},
		{"non-specialist", func(r *Report, p *Policy, tr *Trust) { tr.Providers["specialist"] = Provider{Group: "one"} }},
		{"intel-expired", func(r *Report, p *Policy, tr *Trust) { r.IPFacts[0].Expires = r.Finished - 1 }},
		{"ipv6-missing", func(r *Report, p *Policy, tr *Trust) { p.Families = []int{4, 6}; r.PolicyHash = p.Digest() }},
		{"quic-missing", func(r *Report, p *Policy, tr *Trust) {
			p.Transports = []string{"tcp", "quic"}
			r.PolicyHash = p.Digest()
		}},
		{"unknown-balancer-path", func(r *Report, p *Policy, tr *Trust) { r.Variants[0].PathsComplete = false }},
		{"only-template", func(r *Report, p *Policy, tr *Trust) { r.Variants = r.Variants[1:] }},
		{"one-control", func(r *Report, p *Policy, tr *Trust) { r.Controls = r.Controls[1:] }},
		{"infrastructure-outage", func(r *Report, p *Policy, tr *Trust) { r.InfrastructureOK = false }},
		{"policy-changed", func(r *Report, p *Policy, tr *Trust) { p.Domains = append(p.Domains, "pubads.g.doubleclick.net") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, p, tr, now := qualifiedFixture()
			tc.change(&r, &p, &tr)
			d := Evaluate(r, p, tr, now)
			if d.Eligible || d.CredibleNegative {
				t.Fatalf("uncertainty authorized action: %+v", d)
			}
		})
	}
}
func TestVerifiedNonResidential(t *testing.T) {
	for _, class := range []string{"Mobile", "Datacenter"} {
		r, p, tr, now := qualifiedFixture()
		for i := range r.IPFacts {
			r.IPFacts[i].Class = class
		}
		d := Evaluate(r, p, tr, now)
		if d.Eligible || !d.CredibleNegative {
			t.Fatalf("%s: %+v", class, d)
		}
	}
}
func TestRotatingMixedExitCannotQualify(t *testing.T) {
	r, p, tr, now := qualifiedFixture()
	r.Observations[0].IP = "1.1.1.1"
	r.Observations[0].Attestation = base64.StdEncoding.EncodeToString(ed25519.Sign(ed25519.NewKeyFromSeed([]byte(strings.Repeat("s", 32))), SigningPayload(r.Observations[0])))
	for _, f := range append([]IPFact{}, r.IPFacts...) {
		f.IP = "1.1.1.1"
		r.IPFacts = append(r.IPFacts, f)
	}
	d := Evaluate(r, p, tr, now)
	if !d.Eligible || !d.RotationObserved {
		t.Fatalf("%+v", d)
	}
	r.IPFacts[len(r.IPFacts)-1].Class = "Datacenter"
	d = Evaluate(r, p, tr, now)
	if d.Eligible || d.CredibleNegative {
		t.Fatalf("conflicting intel %+v", d)
	}
}

func TestAttestationCannotBeReplayedAcrossScope(t *testing.T) {
	for _, mode := range []string{"forged", "config", "runtime", "policy", "nonce", "destination", "unsigned", "geo-conflict"} {
		t.Run(mode, func(t *testing.T) {
			r, p, tr, now := qualifiedFixture()
			switch mode {
			case "forged":
				r.Observations[0].Attestation = strings.Repeat("a", 88)
			case "config":
				r.Observations[0].ConfigHash = strings.Repeat("c", 64)
			case "runtime":
				r.Observations[0].RuntimeHash = strings.Repeat("d", 64)
			case "policy":
				r.Observations[0].PolicyHash = strings.Repeat("e", 64)
			case "nonce":
				r.Observations[0].Nonce = "different-nonce-123456789"
			case "destination":
				r.Observations[0].Domain = "pubads.g.doubleclick.net"
			case "unsigned":
				r.Observations[0].Attestation = ""
			case "geo-conflict":
				r.IPFacts[0].Country = "US"
				r.IPFacts[1].Country = "CA"
			}
			d := Evaluate(r, p, tr, now)
			if d.Eligible || d.CredibleNegative {
				t.Fatalf("unverified accepted/rejected: %+v", d)
			}
		})
	}
}

func TestHealthDoesNotEraseIndependentEgressEvidence(t *testing.T) {
	r, p, tr, now := qualifiedFixture()
	r.Controls = nil
	d := Evaluate(r, p, tr, now)
	if d.Eligible || d.Health != "UNKNOWN" || d.Egress != "OBSERVED" || d.Network != "Residential" {
		t.Fatalf("axes conflated: %+v", d)
	}
}
