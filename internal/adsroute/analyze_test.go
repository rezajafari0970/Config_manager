package adsroute

import (
	"fmt"
	"testing"

	"github.com/rezajafari0970/Config_manager/internal/egress"
)

// Synthetic documentation addresses and synthetic source IDs only.
func fixture() (egress.Report, egress.Plan, int64) {
	now := egress.Now()
	p := egress.Prepare([]byte("vless://00000000-0000-4000-8000-000000000001@192.0.2.1:443?security=tls&sni=example.com"))
	r := egress.Report{Version: egress.Version, Hash: "fixture", Started: now - 1000, Finished: now, Core: "xray-26.5.9", CoreSHA: egress.Cores["xray-26.5.9"], Waves: 3}
	for _, v := range p.Variants {
		r.Checks = append(r.Checks, egress.Check{Name: v.Name, Hash: v.Hash, RuntimeHash: v.Hash, Core: true})
		for wave := 1; wave <= 3; wave++ {
			for _, target := range Targets() {
				nonce := egress.Hash([]byte(fmt.Sprint(v.Name, target.Domain, wave)))[:32]
				source := "baseline"
				if target.Group == "doubleclick" {
					source = fmt.Sprint("rotating-ads", v.Name, target.Domain, wave)
				}
				r.Samples = append(r.Samples, egress.Sample{Domain: target.Domain, Variant: v.Name, Wave: wave, At: now, Source: egress.Hash([]byte(source)), Family: 4, Transport: "tcp", TLS: true, SameStream: true, Echo: true, ActualAds: target.Role == "ads", Nonce: nonce, Connection: nonce, Certificate: egress.Hash([]byte("certificate")), Response: egress.Hash([]byte("response")), RequestBytes: 200, ResponseBytes: 800, ResponseComplete: true})
			}
		}
	}
	return r, p, now
}

func TestPartialSplitSurvivesUnrelatedAdsOverlap(t *testing.T) {
	r, p, now := fixture()
	admitted := egress.Evaluate(r, p, r.Hash, now)
	if admitted.State != "RESIDENTIAL" || len(admitted.QualifiedProfiles) != 1 || admitted.QualifiedProfiles[0] != "doubleclick.v1" {
		t.Fatal("authorized profile was not admitted", admitted)
	}
	d := Analyze(r, p, r.Hash, now)
	if d.State != "PARTIAL_ADS_SPLIT" || len(d.SplitDomains) != 3 || len(d.SharedDomains) != 3 || d.PublicationReady {
		t.Fatal(d)
	}
	for _, domain := range d.Domains {
		if domain.Group == "doubleclick" && (domain.State != "SEPARATE" || domain.Variants[0].SourceCount != 3) {
			t.Fatal(domain)
		}
	}
}

func TestWholeGoogleCannotMasqueradeAsAdsOnly(t *testing.T) {
	r, p, now := fixture()
	for i := range r.Samples {
		s := &r.Samples[i]
		if s.ActualAds || (s.Domain == "www.google.com" || s.Domain == "ipv4.google.com") {
			s.Source = egress.Hash([]byte("whole-google-route"))
		}
	}
	d := Analyze(r, p, r.Hash, now)
	if d.State != "SHARED_OBSERVED" || len(d.SplitDomains) != 0 {
		t.Fatal(d)
	}
}

func TestInsufficientEvidenceNeverBecomesSplit(t *testing.T) {
	cases := map[string]func(*egress.Report, *egress.Plan){
		"tls": func(r *egress.Report, _ *egress.Plan) {
			for i := range r.Samples {
				r.Samples[i].TLS = false
			}
		},
		"cross_family_google": func(r *egress.Report, _ *egress.Plan) {
			for i := range r.Samples {
				if r.Samples[i].Domain == "www.google.com" || r.Samples[i].Domain == "ipv4.google.com" {
					r.Samples[i].Family = 6
				}
			}
		},
		"google_not_observable": func(r *egress.Report, _ *egress.Plan) {
			for i := range r.Samples {
				if r.Samples[i].Domain == "www.google.com" || r.Samples[i].Domain == "ipv4.google.com" {
					r.Samples[i].Echo = false
				}
			}
		},
		"one_wave": func(r *egress.Report, _ *egress.Plan) {
			for i := range r.Samples {
				if r.Samples[i].Wave > 1 {
					r.Samples[i].Echo = false
				}
			}
		},
		"nonce_replay": func(r *egress.Report, _ *egress.Plan) {
			for i := range r.Samples {
				r.Samples[i].Nonce = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
				r.Samples[i].Connection = r.Samples[i].Nonce
			}
		},
		"variant_rewritten":    func(r *egress.Report, _ *egress.Plan) { r.Checks[1].RuntimeHash = egress.Hash([]byte("rewritten")) },
		"config_hash_tampered": func(_ *egress.Report, p *egress.Plan) { p.Variants[0].Config = []byte("{}") },
		"pressure":             func(r *egress.Report, _ *egress.Plan) { r.ResourcePressure = true },
		"missing_template":     func(r *egress.Report, _ *egress.Plan) { r.Checks = r.Checks[:1] },
		"stale":                func(r *egress.Report, _ *egress.Plan) { r.Started -= 600001 },
		"no_bidirectional_exchange": func(r *egress.Report, _ *egress.Plan) {
			for i := range r.Samples {
				r.Samples[i].ResponseComplete = false
			}
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			r, p, now := fixture()
			change(&r, &p)
			if d := Analyze(r, p, r.Hash, now); len(d.SplitDomains) != 0 || d.PublicationReady {
				t.Fatal(d)
			}
		})
	}
}

func TestOneDomainOverlapDoesNotEraseOtherDomains(t *testing.T) {
	r, p, now := fixture()
	for i := range r.Samples {
		if r.Samples[i].Domain == "googleads.g.doubleclick.net" && r.Samples[i].Wave == 1 {
			r.Samples[i].Source = egress.Hash([]byte("baseline"))
		}
	}
	d := Analyze(r, p, r.Hash, now)
	if d.State != "PARTIAL_ADS_SPLIT" || len(d.SplitDomains) != 2 || len(d.Unresolved) != 1 {
		t.Fatal(d)
	}
}

func TestEveryTestedDomainSeparatedStillDoesNotAuthorizePublication(t *testing.T) {
	r, p, now := fixture()
	for i := range r.Samples {
		if r.Samples[i].ActualAds {
			r.Samples[i].Source = egress.Hash([]byte("ads"))
		}
	}
	d := Analyze(r, p, r.Hash, now)
	if d.State != "ADS_SPLIT_OBSERVED" || d.Coverage != "ALL_TESTED" || len(d.SplitDomains) != 6 || d.PublicationReady {
		t.Fatal(d)
	}
}

func TestIPv4HostnameProvidesComparableControlWithoutRewritingDestination(t *testing.T) {
	r, p, now := fixture()
	for i := range r.Samples {
		s := &r.Samples[i]
		if s.Domain == "www.google.com" || s.Domain == "www.cloudflare.com" {
			s.Family = 6
			s.Source = egress.Hash([]byte("baseline-ipv6"))
		}
	}
	d := Analyze(r, p, r.Hash, now)
	if d.State != "PARTIAL_ADS_SPLIT" || len(d.SplitDomains) != 3 {
		t.Fatal(d)
	}
	for i := range r.Samples {
		if r.Samples[i].Domain == "ipv4.google.com" {
			r.Samples[i].Echo = false
		}
	}
	if d = Analyze(r, p, r.Hash, now); len(d.SplitDomains) != 0 {
		t.Fatal("cross-family observations substituted for missing IPv4 proof", d)
	}
}

func TestIranVetoSurvivesMissingTemplateAndDoesNotComeFromTimeout(t *testing.T) {
	r, p, now := fixture()
	s := &r.Samples[0]
	s.Geo = egress.CountryFact{Version: egress.CountryVersion, Status: "located", Country: "IR", At: now,
		Providers: []egress.CountryProvider{
			{Name: "fixture-one", Lineage: "fixture-one", Kind: "database", Dataset: egress.Hash([]byte("one")), ReleasedAt: now - 1000, Country: "IR"},
			{Name: "fixture-two", Lineage: "fixture-two", Kind: "database", Dataset: egress.Hash([]byte("two")), ReleasedAt: now - 1000, Country: "IR"},
		}}
	r.Checks = r.Checks[:1]
	if d := Analyze(r, p, r.Hash, now); !d.IranObserved || d.State != "IR_QUARANTINE" || d.PublicationReady {
		t.Fatal(d)
	}
	s.TLS = false
	s.Echo = false
	s.Code = "connection_timeout"
	if d := Analyze(r, p, r.Hash, now); d.IranObserved || d.State == "IR_QUARANTINE" {
		t.Fatal("unverified country became an Iran signal", d)
	}
}
