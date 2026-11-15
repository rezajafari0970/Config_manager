package panel

import (
	"context"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/egress"
	"strings"
	"testing"
)

func partialProfileReport(j egress.Job) egress.Report {
	r := syntheticEgress(j)
	for i := range r.Samples {
		s := &r.Samples[i]
		if s.ActualAds && !strings.HasSuffix(s.Domain, "doubleclick.net") {
			s.Source = egress.Hash([]byte("1.1.1.1"))
			s.Geo.Country = "IE"
			for k := range s.Geo.Providers {
				s.Geo.Providers[k].Country = "IE"
			}
		}
	}
	return r
}
func TestPartialProfileCountsAndFourExactExports(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	raw := sourceURI(231)
	newExportSource(t, s, "partial-profile", raw, raw)
	j, err := s.ClaimEgress(ctx)
	if err != nil || j == nil {
		t.Fatal(err)
	}
	if err = s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: partialProfileReport(*j)}); err != nil {
		t.Fatal(err)
	}
	counts, err := egressProfileSummary(ctx, s.db)
	if err != nil {
		t.Fatal(err)
	}
	if counts["partial_tested_ads"] != 1 || counts["complete_tested_ads"] != 0 || counts["published"] != 1 || counts["profiles"].(map[string]int)["doubleclick.v1"] != 1 {
		t.Fatal(counts)
	}
	app, err := NewServer(s, "http://127.0.0.1:18191")
	if err != nil {
		t.Fatal(err)
	}
	asset := exportRequest(t, app.Handler(), "GET", app.origin+"/assets/egress-profiles.js")
	if asset.Code != 200 || !strings.Contains(asset.Body.String(), "export function profileDetails") {
		t.Fatal("profile UI module unavailable", asset.Code)
	}
	base := app.origin + "/egress-subscription/" + s.digest("egress-export-v2", "all") + "/"
	var template []byte
	for _, v := range egress.Prepare(j.Raw).Variants {
		if v.Name == "template" {
			template = v.Config
		}
	}
	for _, f := range []string{"configs.json", "configs.jsonl", "raw.txt", "next.json"} {
		rr := exportRequest(t, app.Handler(), "GET", base+f)
		if rr.Code != 200 || rr.Header().Get("X-Config-Count") != "1" || rr.Header().Get("X-Residential-Definition") != "verified-ads-profile-split" {
			t.Fatal(f, rr.Code, rr.Body.String())
		}
		if f == "raw.txt" {
			if strings.TrimSpace(rr.Body.String()) != strings.TrimSpace(string(j.Raw)) {
				t.Fatal("raw changed")
			}
			continue
		}
		b := rr.Body.Bytes()
		if f == "configs.json" {
			var values []json.RawMessage
			if json.Unmarshal(b, &values) != nil || len(values) != 1 {
				t.Fatal("array")
			}
			b = values[0]
		}
		if egress.Hash([]byte(strings.TrimSpace(string(b)))) != egress.Hash(template) {
			t.Fatal(f, "exact template changed")
		}
	}
	// A country observed only on a shared Ads path cannot select this profile.
	if _, err = s.db.Exec(`UPDATE egress_country_policy SET allowed='["IE"]',revision=revision+1`); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"configs.json", "configs.jsonl", "raw.txt", "next.json"} {
		if rr := exportRequest(t, app.Handler(), "GET", base+f); rr.Code != 503 {
			t.Fatal(f, rr.Code)
		}
	}
}

func TestProfileCountryHistorySurvivesUnknownButNotProfileChange(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "country-profile", sourceURI(232))
	j, err := s.ClaimEgress(ctx)
	if err != nil || j == nil {
		t.Fatal(err)
	}
	r := partialProfileReport(*j)
	p := egress.Prepare(j.Raw)
	now := egress.Now()
	d := egress.Evaluate(r, p, j.Hash, now)
	save := func(r egress.Report, d egress.Decision) {
		t.Helper()
		tx, err := s.db.Begin()
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		if err = saveEgressCountries(ctx, tx, j.Hash, r, d, now); err != nil {
			t.Fatal(err)
		}
		if err = tx.Commit(); err != nil {
			t.Fatal(err)
		}
	}
	save(r, d)
	if _, err = s.db.Exec(`INSERT INTO egress_countries VALUES(?,?,?,?)`, j.Hash, "TW", now, now); err != nil {
		t.Fatal(err)
	}
	save(egress.Report{}, egress.Decision{})
	save(egress.Report{}, d) // GeoIP outage with the route still qualified.
	save(r, d)
	var n int
	if err = s.db.QueryRow("SELECT count(*) FROM egress_countries WHERE hash=? AND country='TW'", j.Hash).Scan(&n); err != nil || n != 1 {
		t.Fatal("unknown erased rolling history", err, n)
	}
	// Both complete profiles now qualify; previous profile scope must not leak.
	r = syntheticEgress(*j)
	d = egress.Evaluate(r, p, j.Hash, now)
	save(r, d)
	if err = s.db.QueryRow("SELECT count(*) FROM egress_countries WHERE hash=? AND country='TW'", j.Hash).Scan(&n); err != nil || n != 0 {
		t.Fatal("profile change reused former countries", err, n)
	}
}
