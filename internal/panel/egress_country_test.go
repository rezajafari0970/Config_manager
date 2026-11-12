package panel

import (
	"context"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/egress"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEgressCountriesAllOutputsAndPolicy(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "countries", sourceURI(221), sourceURI(222), sourceURI(223))
	jobs := []egress.Job{}
	for i := 0; i < 3; i++ {
		j, e := s.ClaimEgress(ctx)
		if e != nil || j == nil {
			t.Fatal(e)
		}
		r := syntheticEgress(*j)
		for k := range r.Samples {
			sample := &r.Samples[k]
			if sample.ActualAds && (i == 2 || i == 1 && sample.Domain == egress.Domains[0]) {
				sample.Geo.Country = "MO"
				for k := range sample.Geo.Providers {
					sample.Geo.Providers[k].Country = "MO"
				}
				sample.Source = egress.Hash([]byte("second-fixture-source"))
			}
		}
		if e = s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: r}); e != nil {
			t.Fatal(e)
		}
		jobs = append(jobs, *j)
	}
	app, e := NewServer(s, "http://127.0.0.1:18191")
	if e != nil {
		t.Fatal(e)
	}
	m := egressModule{app}
	base := app.origin + "/egress-subscription/" + s.digest("egress-export-v2", "all") + "/"
	check := func(n string, code int) {
		t.Helper()
		for _, f := range []string{"configs.json", "configs.jsonl", "raw.txt", "next.json"} {
			for _, method := range []string{"GET", "HEAD"} {
				rr := exportRequest(t, app.Handler(), method, base+f)
				want := n
				if f == "next.json" && code == 200 {
					want = "1"
				}
				if rr.Code != code || code == 200 && rr.Header().Get("X-Config-Count") != want {
					t.Fatalf("%s %s: %d %s", method, f, rr.Code, rr.Body.String())
				}
			}
		}
	}
	check("3", 200)
	info, e := countryState(ctx, s.db)
	if e != nil {
		t.Fatal(e)
	}
	counts := info["counts"].(map[string]int)
	if counts["TR"] != 2 || counts["MO"] != 2 || info["shared"] != 1 {
		t.Fatal(info)
	}
	save := func(revision int64, countries []string, want int) {
		t.Helper()
		b, _ := json.Marshal(countryPolicy{revision, countries})
		rr := httptest.NewRecorder()
		m.countries(rr, httptest.NewRequest("POST", "/api/egress/countries", strings.NewReader(string(b))))
		if rr.Code != want {
			t.Fatal(rr.Code, rr.Body.String())
		}
	}
	save(1, []string{"TR"}, 200)
	check("2", 200) // Either observed country is enough; shared member appears once.
	save(1, []string{"MO"}, 409)
	save(2, []string{"ZZ"}, 422)
	check("2", 200)
	save(2, []string{}, 200)
	check("0", 503)
	save(3, []string{"MO", "TR"}, 200)
	check("3", 200)
	// Harmless renewal does not invalidate generation; country policy does.
	_, before, e := egressSnapshot(ctx, s.db)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec("UPDATE egress_jobs SET valid_until=valid_until+1000"); e != nil {
		t.Fatal(e)
	}
	_, after, e := egressSnapshot(ctx, s.db)
	if e != nil || before != after {
		t.Fatal("renewal changed publication", e)
	}
	if _, e = s.db.Exec("UPDATE egress_geo_results SET valid_until=1 WHERE hash=?", jobs[0].Hash); e != nil {
		t.Fatal(e)
	}
	check("2", 200)
	if _, e = s.db.Exec("UPDATE egress_geo_results SET complete=0"); e != nil {
		t.Fatal(e)
	}
	check("0", 503)
}

func TestEgressCountryFactsCannotQualifyMissingConflictingOrStale(t *testing.T) {
	s := testStore(t)
	newExportSource(t, s, "fixture", sourceURI(224))
	j, e := s.ClaimEgress(context.Background())
	if e != nil || j == nil {
		t.Fatal(e)
	}
	for _, mode := range []string{"missing", "stale", "disagree", "source-conflict", "future", "wrong-domain"} {
		r := syntheticEgress(*j)
		switch mode {
		case "missing":
			r.Samples[0].Geo = egress.CountryFact{}
		case "stale":
			r.Samples[0].Geo.Providers[0].ReleasedAt = 1
		case "disagree":
			r.Samples[0].Geo.Providers[0].Country = "MO"
		case "source-conflict":
			r.Samples[0].Geo.Country = "MO"
			r.Samples[0].Geo.Providers[0].Country = "MO"
		case "future":
			r.Samples[0].Geo.At = r.Finished + 10000
		case "wrong-domain":
			for i := range r.Samples {
				r.Samples[i].Domain = "example.com"
			}
		}
		d := egress.Evaluate(r, egress.Prepare(j.Raw), j.Hash, egress.Now())
		if g := egress.Countries(r, d, egress.Now()); g.Complete {
			t.Fatal(mode, g)
		}
	}
}

func TestEgressFirstDeepRetrySurvivesArchivedRetention(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "archive", sourceURI(225))
	j, e := s.ClaimEgress(ctx)
	if e != nil || j == nil {
		t.Fatal(e)
	}
	r := syntheticEgress(*j)
	r.Samples = nil
	if e = s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: r}); e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec("UPDATE egress_jobs SET first_seen=1,next_run=0 WHERE hash=?", j.Hash); e != nil {
		t.Fatal(e)
	}
	if e = s.Expire(ctx, time.Now()); e != nil {
		t.Fatal(e)
	}
	next, e := s.ClaimEgress(ctx)
	if e != nil || next == nil || !next.Deep {
		t.Fatal("deep retry lost", next, e)
	}
	if e = s.Expire(ctx, time.Now()); e != nil {
		t.Fatal(e)
	}
	r = syntheticEgress(*next)
	r.Samples = nil
	if e = s.CompleteEgress(ctx, egress.Completion{Job: *next, Report: r}); e != nil {
		t.Fatal(e)
	}
	if e = s.Expire(ctx, time.Now()); e != nil {
		t.Fatal(e)
	}
	var state string
	if e = s.db.QueryRow("SELECT state FROM egress_jobs WHERE hash=?", j.Hash).Scan(&state); e != nil || state == "DELETED" {
		t.Fatal("durable investigation removed by ordinary retention", state, e)
	}
}
