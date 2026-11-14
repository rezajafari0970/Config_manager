package egress

import "testing"

func TestSupplementaryReachabilityDoesNotEraseObservedSplit(t *testing.T) {
	r, p, now := fixture()
	for i := range r.Samples {
		s := &r.Samples[i]
		if s.Domain == "t.me" || s.Domain == "www.instagram.com" {
			s.Source = ""
			s.Echo = false
			s.Code = "reachable_no_source"
		}
	}
	d := Evaluate(r, p, r.Hash, now)
	if !d.Eligible || d.State != "RESIDENTIAL" || d.CoverageComplete || d.EvidenceProfile != "ads-stable-controls.v1" {
		t.Fatal(d)
	}
	for _, s := range d.Services {
		if (s.Domain == "t.me" || s.Domain == "www.instagram.com") && (s.Required || s.Status != "ROUTE_UNOBSERVABLE") {
			t.Fatal(s)
		}
	}
}

func TestRotatingControlsAreNotDistinctRouteProof(t *testing.T) {
	for _, domain := range []string{"www.google.com", "www.cloudflare.com", "api.ipify.org"} {
		t.Run(domain, func(t *testing.T) {
			r, p, now := fixture()
			for i := range r.Samples {
				s := &r.Samples[i]
				if s.Domain == domain && s.Wave == 2 {
					s.Source = Hash([]byte("rotating-second-exit"))
				}
			}
			d := Evaluate(r, p, r.Hash, now)
			if d.Eligible || d.Negative || d.State != "UNKNOWN" {
				t.Fatal(d)
			}
		})
	}
}

func TestMixedAdsFamiliesUsePerDestinationWitnesses(t *testing.T) {
	r, p, now := fixture()
	extra := []Sample{}
	for i := range r.Samples {
		s := &r.Samples[i]
		if s.ActualAds && s.Domain != "www.googleadservices.com" {
			s.Family = 6
			s.Source = Hash([]byte("ads-v6"))
		}
		if s.Domain == "api.ipify.org" {
			s.Family = 6
			s.Source = Hash([]byte("control-v6"))
		}
		if s.Domain == "www.google.com" {
			v := *s
			v.Family = 6
			v.Source = Hash([]byte("google-v6"))
			v.Nonce = Hash([]byte(s.Connection + "ipv6"))[:32]
			v.Connection = v.Nonce
			extra = append(extra, v)
		}
	}
	r.Samples = append(r.Samples, extra...)
	if d := Evaluate(r, p, r.Hash, now); !d.Eligible {
		t.Fatal("matched mixed-family proof rejected", d)
	}
	// Removing Google-v6 leaves one v6 control group, not an adequate comparison.
	for i := range r.Samples {
		s := &r.Samples[i]
		if s.Domain == "www.google.com" && s.Family == 6 {
			s.Echo = false
			s.Source = ""
		}
	}
	if d := Evaluate(r, p, r.Hash, now); d.Eligible || d.Negative || d.Reason != "paired_control_coverage_incomplete" {
		t.Fatal(d)
	}
}

func TestRecoveredAdsNeedsControlsInSameWave(t *testing.T) {
	r, p, now := fixture()
	first := []Sample{}
	for _, s := range r.Samples {
		if s.Wave <= 2 {
			first = append(first, s)
		}
	}
	r.Samples = first
	r.Waves = 3
	for i := range r.Samples {
		s := &r.Samples[i]
		if s.Domain == Domains[0] && s.Wave == 2 {
			s.Wave = 3
		}
	}
	if d := Evaluate(r, p, r.Hash, now); d.Eligible || d.Negative {
		t.Fatal("unpaired recovery accepted", d)
	}
	extra := []Sample{}
	for _, s := range r.Samples {
		if !s.ActualAds && s.Wave == 2 {
			s.Wave = 3
			s.Nonce = Hash([]byte(s.Connection + "recovery"))[:32]
			s.Connection = s.Nonce
			extra = append(extra, s)
		}
	}
	r.Samples = append(r.Samples, extra...)
	if d := Evaluate(r, p, r.Hash, now); !d.Eligible {
		t.Fatal("coherent recovery rejected", d)
	}
}

func TestSupplementaryObservedOverlapStillVetoesPublication(t *testing.T) {
	r, p, now := fixture()
	for i := range r.Samples {
		s := &r.Samples[i]
		if s.Domain == "t.me" {
			s.Source = Hash([]byte("8.8.8.8"))
		}
	}
	if d := Evaluate(r, p, r.Hash, now); d.Eligible || d.State != "NO_SPLIT" {
		t.Fatal(d)
	}
}

func TestVariantNamesCannotAliasEvidenceBuckets(t *testing.T) {
	r, p, now := fixture()
	p.Variants[1] = p.Variants[0]
	r.Checks[1] = r.Checks[0]
	if d := Evaluate(r, p, r.Hash, now); d.Eligible || d.Negative || d.Reason != "variant_binding_invalid" {
		t.Fatal(d)
	}
}
