package residential

import "testing"

func TestControlAliasesKeepProviderIndependenceAndNoFalseNegative(t *testing.T) {
	if controlGroup("1.1.1.1") != controlGroup("www.cloudflare.com") || controlGroup("api.ipify.org") != controlGroup("api64.ipify.org") || controlGroup("1.1.1.1") == controlGroup("api.ipify.org") {
		t.Fatal("incorrect control independence")
	}
	r, p, _, _ := qualifiedFixture()
	for i := range r.Controls {
		r.Controls[i].Outcome = "failed"
		r.Controls[i].TLSVerified = false
	}
	if !repeatedControlFailure(r, "raw", p) {
		t.Fatal("expected repeated failure")
	}
	r.Controls = append(r.Controls, Observation{Scope: Scope{Variant: "raw", Domain: "1.1.1.1"}, Wave: 1, Outcome: "passed"})
	if repeatedControlFailure(r, "raw", p) {
		t.Fatal("working family hidden by failed other family")
	}
}

func TestRouteDiagnosticsAreNotClassification(t *testing.T) {
	r, p, tr, now := qualifiedFixture()
	for i := range r.Controls {
		r.Controls[i].IP = "1.1.1.1"
		r.Controls[i].Family = 4
		r.Controls[i].Transport = "tcp"
		if r.Controls[i].Domain == "api.ipify.org" {
			r.Controls[i].Domain = "api64.ipify.org"
		}
	}
	for i := range r.Observations {
		r.Observations[i].MeaningVerified = false
	}
	for _, source := range []string{"metadata-ipinfo", "metadata-ipwhois"} {
		for _, ip := range []string{"8.8.8.8", "1.1.1.1"} {
			f := r.IPFacts[0]
			f.Source = source
			f.IP = ip
			f.Class = "Unknown"
			f.ASN = 1
			f.Country = "AA"
			if ip == "1.1.1.1" {
				f.ASN = 2
				f.Country = "BB"
			}
			r.IPFacts = append(r.IPFacts, f)
		}
	}
	// No specialist metadata needed to show geography, but nothing may qualify.
	tr.Providers = map[string]Provider{}
	d := Evaluate(r, p, tr, now)
	if d.Eligible || d.CredibleNegative || len(d.Diagnostics) != 2 {
		t.Fatalf("%+v", d)
	}
	for _, x := range d.Diagnostics {
		if x.Status != "CANDIDATE_ONLY" || x.Different != 2 || x.ASNChanges != 2 || x.CountryChanges != 2 {
			t.Fatalf("%+v", x)
		}
	}
	for i := range r.Controls {
		if r.Controls[i].Domain == "api64.ipify.org" {
			r.Controls[i].IP = "2606:4700:4700::1111"
			r.Controls[i].Family = 6
		}
	}
	for _, x := range RouteDiagnostics(r, p, tr, now) {
		if x.Compared != 0 {
			t.Fatal("cross-family comparison", x)
		}
	}
}

func TestRouteMetadataConflictExpiryAndIndependentPublishers(t *testing.T) {
	r, p, tr, now := qualifiedFixture()
	f := r.IPFacts[0]
	f.Source = "metadata-ipinfo"
	f.Country = "AA"
	f.ASN = 1
	g := f
	g.Source = "metadata-ipwhois"
	country, asn, _ := corroboratedMetadata(f.IP, []IPFact{f, g}, p, tr, now)
	if country != "AA" || asn != 1 {
		t.Fatal(country, asn)
	}
	for _, change := range []func(*IPFact){func(g *IPFact) { g.Country = "BB" }, func(g *IPFact) { g.ASN = 2 }, func(g *IPFact) { g.Expires = now }, func(g *IPFact) { g.Source = f.Source }} {
		bad := g
		change(&bad)
		country, asn, _ = corroboratedMetadata(f.IP, []IPFact{f, bad}, p, tr, now)
		if country != "" || asn != 0 {
			t.Fatal("unsupported corroboration", country, asn)
		}
	}
}
