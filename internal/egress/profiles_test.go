package egress

import (
	"strings"
	"testing"
)

func TestNamedProfilePartialCoverage(t *testing.T) {
	for _, remainder := range []string{"shared", "unobservable"} {
		t.Run(remainder, func(t *testing.T) {
			r, p, now := singleProfileFixture()
			for i := range r.Samples {
				s := &r.Samples[i]
				if s.ActualAds && !strings.HasSuffix(s.Domain, "doubleclick.net") && remainder == "unobservable" {
					s.Echo = false
					s.ResponseComplete = false
				}
			}
			d := Evaluate(r, p, "fixture", now)
			if !d.Eligible || d.AdsCoverage != "PARTIAL" || len(d.QualifiedProfiles) != 1 || d.QualifiedProfiles[0] != "doubleclick.v1" || len(d.QualifiedDomains) != 3 || AdsHealthForDecision(r, p, d, now) != "healthy" {
				t.Fatalf("%+v", d)
			}
		})
	}
}

func TestCannotCombineProfilesAcrossVariantsOrIncompleteMembers(t *testing.T) {
	for _, mode := range []string{"different_profiles", "one_member_missing", "replayed_copies", "health_wrong_waves"} {
		t.Run(mode, func(t *testing.T) {
			r, p, now := singleProfileFixture()
			for i := range r.Samples {
				s := &r.Samples[i]
				switch mode {
				case "different_profiles":
					if s.Variant == "template" && s.ActualAds {
						if strings.HasSuffix(s.Domain, "doubleclick.net") {
							s.Source = Hash([]byte("1.1.1.1"))
						} else {
							s.Source = Hash([]byte("8.8.8.8"))
						}
					}
				case "one_member_missing":
					if s.Domain == Domains[0] {
						s.Echo = false
					}
				case "health_wrong_waves":
					if s.Domain == Domains[0] && s.Wave == 1 {
						s.ResponseComplete = false
					}
					if !s.ActualAds && s.Wave == 3 {
						s.Echo = false
					}
				}
			}
			if mode == "replayed_copies" {
				originals := append([]Sample(nil), r.Samples...)
				for _, s := range originals {
					if s.Domain == Domains[0] {
						r.Samples = append(r.Samples, s)
					}
				}
			}
			d := Evaluate(r, p, "fixture", now)
			if d.Eligible || d.Negative || d.State != "UNKNOWN" {
				t.Fatal(d)
			}
		})
	}
}

func profileGeo(s *Sample, country string, now int64) {
	s.Geo = CountryFact{Version: CountryVersion, Status: "located", Country: country, At: now, Providers: []CountryProvider{
		{Name: "one", Lineage: "one", Kind: "database", Dataset: strings.Repeat("a", 64), ReleasedAt: now - 86400000, Country: country},
		{Name: "two", Lineage: "two", Kind: "database", Dataset: strings.Repeat("b", 64), ReleasedAt: now - 86400000, Country: country},
	}}
}
func TestProfileCountryScopeAndIndependentIranVeto(t *testing.T) {
	for _, mode := range []string{"shared_other_country", "shared_country_missing", "shared_iran", "shared_iran_minority"} {
		t.Run(mode, func(t *testing.T) {
			r, p, now := singleProfileFixture()
			for i := range r.Samples {
				s := &r.Samples[i]
				if !s.ActualAds {
					continue
				}
				profileGeo(s, "TW", now)
				if !strings.HasSuffix(s.Domain, "doubleclick.net") {
					profileGeo(s, "IE", now)
					switch mode {
					case "shared_country_missing":
						s.Geo = CountryFact{}
					case "shared_iran":
						profileGeo(s, "IR", now)
					case "shared_iran_minority":
						s.Geo.Providers = append(s.Geo.Providers, CountryProvider{Name: "three", Lineage: "three", Kind: "database", Dataset: strings.Repeat("c", 64), ReleasedAt: now - 86400000, Country: "IR"})
					}
				}
			}
			d := Evaluate(r, p, "fixture", now)
			g := Countries(r, d, now)
			if mode == "shared_iran" {
				if d.Eligible || Iran(r, p, "fixture", now) == nil {
					t.Fatal("nonqualifying Ads hid Iran", d)
				}
				return
			}
			if mode == "shared_iran_minority" {
				if g.Complete || g.Status != "country_iran_conflict" {
					t.Fatal(g)
				}
				return
			}
			if !g.Complete || len(g.Countries) != 1 || g.Countries[0] != "TW" || g.ProfileKey != "doubleclick.v1" {
				t.Fatal(g)
			}
		})
	}
}

func TestMixedSharingCannotBecomeCachedNegative(t *testing.T) {
	r, p, now := fixture()
	for i := range r.Samples {
		s := &r.Samples[i]
		if s.ActualAds && s.Wave <= 2 {
			s.Source = Hash([]byte("1.1.1.1"))
		}
	}
	d := Evaluate(r, p, r.Hash, now)
	if d.Eligible || d.Negative || d.State != "UNKNOWN" {
		t.Fatal(d)
	}
}

func TestUnpairedDifferentObservationPreventsTerminalNegative(t *testing.T) {
	r, p, now := fixture()
	for i := range r.Samples {
		s := &r.Samples[i]
		if s.ActualAds && s.Wave <= 2 {
			s.Source = Hash([]byte("1.1.1.1"))
		}
		if !s.ActualAds && s.Wave == 3 {
			s.Echo = false
		}
	}
	if d := Evaluate(r, p, r.Hash, now); d.Eligible || d.Negative || d.State != "UNKNOWN" {
		t.Fatal(d)
	}
}

func TestSelectedAttemptIranSurvivesUnrelatedResourcePressure(t *testing.T) {
	r, p, now := singleProfileFixture()
	for i := range r.Samples {
		if r.Samples[i].ActualAds {
			profileGeo(&r.Samples[i], "IR", now)
		}
	}
	part := r
	part.Samples = nil
	part.SelectedSamples = true
	r.Evidence = []Report{part}
	r.ResourcePressure = true
	if Iran(r, p, r.Hash, now) == nil {
		t.Fatal("unrelated pressure erased authenticated selected-attempt Iran")
	}
	r.Evidence[0].ResourcePressure = true
	if Iran(r, p, r.Hash, now) != nil {
		t.Fatal("pressured attempt qualified")
	}
	r.Evidence[0].ResourcePressure = false
	r.Evidence[0].CoreSHA = "invalid"
	if Iran(r, p, r.Hash, now) != nil {
		t.Fatal("unbound sample reference qualified")
	}
}

func TestEarlierAttemptMinorityIranWithdrawsAllPublication(t *testing.T) {
	r, p, now := fixture()
	for i := range r.Samples {
		if r.Samples[i].ActualAds {
			profileGeo(&r.Samples[i], "TW", now)
		}
	}
	part := r
	s := r.Samples[0]
	s.Nonce = Hash([]byte("earlier-independent-connection"))[:32]
	s.Connection = s.Nonce
	s.Geo.Providers = append(append([]CountryProvider(nil), s.Geo.Providers...), CountryProvider{Name: "three", Lineage: "three", Kind: "database", Dataset: strings.Repeat("c", 64), ReleasedAt: now - 86400000, Country: "IR"})
	part.Samples = []Sample{s}
	r.Evidence = []Report{part}
	d := Evaluate(r, p, r.Hash, now)
	g := Countries(r, d, now)
	if d.Eligible || g.Complete || g.Status != "country_iran_conflict" || Iran(r, p, r.Hash, now) != nil {
		t.Fatal(d, g)
	}
	r.Evidence[0].CoreSHA = "unbound"
	d = Evaluate(r, p, r.Hash, now)
	if !Countries(r, d, now).Complete {
		t.Fatal("unbound attempt blocked countries")
	}
}
func TestAttemptReplayCannotQualify(t *testing.T) {
	r, p, now := singleProfileFixture()
	part := r
	part.Samples = nil
	for _, s := range r.Samples {
		if s.Domain == Domains[0] {
			part.Samples = append(part.Samples, s)
		}
	}
	r.Evidence = []Report{part}
	if d := Evaluate(r, p, r.Hash, now); d.Eligible {
		t.Fatal("replayed attempt qualified", d)
	}
}
func TestLocatedProfileSurvivesOtherProfileCountryFailure(t *testing.T) {
	r, p, now := fixture()
	for i := range r.Samples {
		s := &r.Samples[i]
		if s.ActualAds && strings.HasSuffix(s.Domain, "doubleclick.net") {
			profileGeo(s, "TW", now)
		}
	}
	d := Evaluate(r, p, r.Hash, now)
	g := Countries(r, d, now)
	if !d.Eligible || len(d.QualifiedProfiles) != 2 || !g.Complete || g.ProfileKey != "doubleclick.v1" || len(g.Countries) != 1 || g.Countries[0] != "TW" {
		t.Fatal(d, g)
	}
}
func TestPolicyRegistryCannotBeChangedThroughReturnedSlices(t *testing.T) {
	before := PolicyDigest()
	copy := Profiles()
	copy[0].Domains[0] = "untrusted.example"
	if PolicyDigest() != before || len(before) != 64 {
		t.Fatal("mutable policy")
	}
}
