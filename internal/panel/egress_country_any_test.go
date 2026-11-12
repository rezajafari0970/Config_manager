package panel

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/rezajafari0970/Config_manager/internal/egress"
)

func TestEgressAnyCountrySharedOutputsAndMigration(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "three-country", sourceURI(226))
	j, err := s.ClaimEgress(ctx)
	if err != nil || j == nil {
		t.Fatal(err)
	}
	r := syntheticEgress(*j)
	countryByDomain := map[string]string{egress.Domains[0]: "TW", egress.Domains[1]: "MX", egress.Domains[2]: "CY"}
	for i := range r.Samples {
		sample := &r.Samples[i]
		if country := countryByDomain[sample.Domain]; sample.ActualAds && country != "" {
			sample.Geo.Country = country
			sample.Source = egress.Hash([]byte("fixture-exit-" + country))
			for k := range sample.Geo.Providers {
				sample.Geo.Providers[k].Country = country
			}
		}
	}
	if err = s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: r}); err != nil {
		t.Fatal(err)
	}
	app, err := NewServer(s, "http://127.0.0.1:18191")
	if err != nil {
		t.Fatal(err)
	}
	module := egressModule{app}
	base := app.origin + "/egress-subscription/" + s.digest("egress-export-v2", "all") + "/"
	check := func(published int) {
		t.Helper()
		info, err := countryState(ctx, s.db)
		if err != nil {
			t.Fatal(err)
		}
		if info["published"] != published || info["shared"] != 1 || info["known"] != 1 || info["match_mode"] != "any" {
			t.Fatal(info)
		}
		if !reflect.DeepEqual(info["counts"], map[string]int{"TW": 1, "MX": 1, "CY": 1}) {
			t.Fatal(info)
		}
		countries, err := entryCountries(ctx, s.db, j.Hash)
		if err != nil || !reflect.DeepEqual(countries, []string{"CY", "MX", "TW"}) {
			t.Fatal(countries, err)
		}
		var retained int
		if err = s.db.QueryRow("SELECT count(*) FROM egress_jobs WHERE hash=? AND ever_accepted=1 AND state!='DELETED'", j.Hash).Scan(&retained); err != nil || retained != 1 {
			t.Fatal("filter removed stored config", retained, err)
		}
		for _, file := range []string{"configs.json", "configs.jsonl", "next.json", "raw.txt"} {
			for _, method := range []string{"GET", "HEAD"} {
				rr := exportRequest(t, app.Handler(), method, base+file)
				want := 200
				if published == 0 {
					want = 503
				}
				if rr.Code != want || published == 1 && rr.Header().Get("X-Config-Count") != "1" {
					t.Fatalf("%s %s: %d %s", method, file, rr.Code, rr.Body.String())
				}
				if method == "GET" && want == 200 && file == "configs.json" {
					var entries []json.RawMessage
					if err := json.Unmarshal(rr.Body.Bytes(), &entries); err != nil || len(entries) != 1 {
						t.Fatal("shared member duplicated", err, len(entries))
					}
				}
			}
		}
	}
	for _, tc := range []struct {
		countries []string
		published int
	}{
		{[]string{"TW", "CY"}, 1}, {[]string{"TW"}, 1}, {[]string{"CY"}, 1},
		{[]string{"MX"}, 1}, {[]string{"AU"}, 0}, {[]string{}, 0},
		{[]string{"TW", "MX", "CY"}, 1}, {[]string{"TW", "CY"}, 1},
	} {
		info, err := countryState(ctx, s.db)
		if err != nil {
			t.Fatal(err)
		}
		policy := info["policy"].(countryPolicy)
		payload, _ := json.Marshal(countryPolicy{Revision: policy.Revision, Countries: tc.countries})
		rr := httptest.NewRecorder()
		module.countries(rr, httptest.NewRequest("POST", "/api/egress/countries", strings.NewReader(string(payload))))
		var saved struct {
			Published int
			Revision  int64
		}
		if rr.Code != 200 || json.Unmarshal(rr.Body.Bytes(), &saved) != nil || saved.Published != tc.published || saved.Revision != policy.Revision+1 {
			t.Fatal("incorrect transactional save count", rr.Code, rr.Body.String())
		}
		check(tc.published)
	}
	before, err := countryState(ctx, s.db)
	if err != nil {
		t.Fatal(err)
	}
	// Recreate the deployed schema28 subset view, then migrate without refetching,
	// reclassifying, deleting evidence or changing the user's selection/revision.
	_, err = s.db.Exec("DROP VIEW egress_export_eligible; CREATE VIEW egress_export_eligible AS SELECT e.* FROM egress_geo_eligible e WHERE NOT EXISTS(SELECT 1 FROM egress_countries c WHERE c.hash=e.hash AND c.last_seen>CAST(unixepoch('subsec')*1000 AS INTEGER)-86400000 AND c.country NOT IN(SELECT value FROM json_each((SELECT allowed FROM egress_country_policy WHERE id=1)))); PRAGMA user_version=28;")
	if err != nil {
		t.Fatal(err)
	}
	var oldCount int
	if err = s.db.QueryRow("SELECT count(*) FROM egress_export_eligible").Scan(&oldCount); err != nil || oldCount != 0 {
		t.Fatal(oldCount, err)
	}
	if err = s.migrateV29(); err != nil {
		t.Fatal(err)
	}
	if err = s.migrateV29(); err != nil {
		t.Fatal("migration not idempotent", err)
	}
	after, err := countryState(ctx, s.db)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("migration changed policy or evidence", before, after, err)
	}
	var version int
	if err = s.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || version != 29 {
		t.Fatal(version, err)
	}
	check(1)
	// Selected observations outside the24h window cannot match a fresh unselected MX.
	if _, err = s.db.Exec("UPDATE egress_countries SET last_seen=1 WHERE country IN ('TW','CY')"); err != nil {
		t.Fatal(err)
	}
	info, err := countryState(ctx, s.db)
	if err != nil || info["published"] != 0 {
		t.Fatal("expired selected country matched", info, err)
	}
}
