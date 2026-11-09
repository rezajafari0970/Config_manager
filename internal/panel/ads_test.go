package panel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/rezajafari0970/Config_manager/internal/smart"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func finishAds(t *testing.T, p *SmartPipeline) {
	t.Helper()
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		j, e := p.ClaimSmart(ctx)
		if e != nil {
			t.Fatal(e)
		}
		if j == nil {
			return
		}
		if j.Issue != "" {
			t.Fatal(j.Issue)
		}
		r := smartResult(j)
		r.Version = smart.AdsVersion
		r.Rule = smart.AdsVersion
		if e = p.CompleteSmart(ctx, r); e != nil {
			t.Fatal(e)
		}
	}
	t.Fatal("ads did not quiesce")
}

func TestAdsRolesRoundRestartAndHTTPIsolation(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	normal, _ := newExportSource(t, s, "Main", sourceURI(1), sourceURI(2), sourceURI(3))
	splash, _ := newExportSource(t, s, "Splash", sourceURI(11), sourceURI(12), sourceURI(13))
	markSplash(t, s, splash, true)
	p := s.AdsSmart()
	v, _ := p.SmartSettings(ctx)
	v.Count = 2
	if e := p.SaveSmartSettings(ctx, v); e != nil {
		t.Fatal(e)
	}
	finishSmart(t, s)
	finishPipeline(t, s.SplashSmart())
	finishAds(t, p)
	rows, e := querySmart(ctx, p.reader(), true, 0)
	if e != nil || len(rows) != 3 {
		t.Fatal(e, rows)
	}
	for _, row := range rows {
		if !row.Available || row.Count != 3 {
			t.Fatal(row)
		}
		var invalid int
		e = s.db.QueryRow(`SELECT count(*) FROM ads_smart_cache_members WHERE head=? AND ((position=0 AND source!=?) OR(position>0 AND source!=?))`, row.ID, splash, normal).Scan(&invalid)
		if e != nil || invalid != 0 {
			t.Fatal("role contamination", e, invalid)
		}
	}
	app, e := NewServer(s, "http://127.0.0.1:18191")
	if e != nil {
		t.Fatal(e)
	}
	h := app.Handler()
	for _, path := range []string{"/smart-ads", "/api/smart-ads", "/api/smart-ads/settings"} {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest("GET", app.origin+path, nil))
		if rr.Code != 200 {
			t.Fatal(path, rr.Code, rr.Body.String())
		}
	}
	for _, path := range []string{"/smart-ads/claim", "/smart-ads/complete"} {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest("POST", app.origin+path, nil))
		if rr.Code == 200 {
			t.Fatal("worker API exposed")
		}
	}
	exportRequest(t, h, "HEAD", app.pipelineURL(p, true))
	seen := map[string]bool{}
	for i := 0; i < 3; i++ {
		var selected *SmartEntry
		var body []byte
		var cycle int64
		if i == 1 {
			path := ""
			s.db.QueryRow("SELECT file FROM pragma_database_list WHERE name='main'").Scan(&path)
			other, e := Open(filepath.Clean(path), s.key)
			if e != nil {
				t.Fatal(e)
			}
			selected, body, cycle, e = other.AdsSmart().smartNext(ctx)
			other.Close()
			if e != nil {
				t.Fatal(e)
			}
		} else {
			selected, body, cycle, e = p.smartNext(ctx)
		}
		if e != nil || selected == nil || cycle != 1 || seen[selected.ID] || !json.Valid(body) {
			t.Fatal("round", e, cycle)
		}
		seen[selected.ID] = true
	}
	for _, scope := range []string{"smart", "splash_smart"} {
		var cycle int
		s.db.QueryRow("SELECT cycle FROM " + scope + "_rotation_state").Scan(&cycle)
		if cycle != 0 {
			t.Fatal("consumed other round")
		}
	}
	rr := exportRequest(t, h, "GET", app.pipelineURL(p, false))
	if rr.Code != 200 || len(bytes.Split(bytes.TrimSpace(rr.Body.Bytes()), []byte{'\n'})) != 3 {
		t.Fatal(rr.Code, rr.Body.String())
	}
	markSplash(t, s, splash, false)
	for _, row := range rows {
		if _, e = p.smartBytes(ctx, row); !errors.Is(e, ErrNotFound) {
			t.Fatal("reclassified ads survived", e)
		}
	}
}

func TestAdsFreshProofAndLateCompletion(t *testing.T) {
	for _, cause := range []string{"ads-proof", "main-proof", "disabled", "classification", "settings"} {
		t.Run(cause, func(t *testing.T) {
			s := testStore(t)
			ctx := context.Background()
			a, _ := newExportSource(t, s, "Main", sourceURI(1))
			b, _ := newExportSource(t, s, "Splash", sourceURI(2))
			markSplash(t, s, b, true)
			p := s.AdsSmart()
			j, e := p.ClaimSmart(ctx)
			if e != nil || j == nil || j.Issue != "" {
				t.Fatal(e, j)
			}
			switch cause {
			case "ads-proof", "main-proof":
				id := b
				if cause == "main-proof" {
					id = a
				}
				_, e = s.db.Exec("UPDATE conversion_results SET connection='failed' WHERE hash IN(SELECT raw_hash FROM memberships WHERE source_id=?)", id)
			case "disabled":
				_, e = s.db.Exec("UPDATE sources SET enabled=0 WHERE id=?", b)
			case "classification":
				markSplash(t, s, b, false)
			case "settings":
				v, _ := p.SmartSettings(ctx)
				v.Count++
				e = p.SaveSmartSettings(ctx, v)
			}
			if e != nil {
				t.Fatal(e)
			}
			if e = p.RenewSmart(ctx, *j); !errors.Is(e, ErrConflict) {
				t.Fatal("renew accepted", e)
			}
			r := smartResult(j)
			r.Version = smart.AdsVersion
			r.Rule = smart.AdsVersion
			if e = p.CompleteSmart(ctx, r); !errors.Is(e, ErrConflict) {
				t.Fatal("stale completion", e)
			}
		})
	}
}

func TestAdsCacheProtectedFromBlobGC(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "Main", sourceURI(1))
	a, _ := newExportSource(t, s, "Splash", sourceURI(2))
	markSplash(t, s, a, true)
	p := s.AdsSmart()
	finishAds(t, p)
	rows, e := querySmart(ctx, p.reader(), true, 0)
	if e != nil || len(rows) != 1 {
		t.Fatal(e)
	}
	used, e := s.Referenced(ctx, rows[0].Hash)
	if e != nil || !used {
		t.Fatal("ads cache unprotected", e)
	}
}

// Historical fixtures must remove newer dependent views before rebuilding old DDL.
func dropAdsForMigration(t *testing.T, s *Store) {
	t.Helper()
	rows, e := s.db.Query("SELECT name FROM sqlite_master WHERE type='view' AND name GLOB 'ads_smart_*'")
	if e != nil {
		t.Fatal(e)
	}
	var names []string
	for rows.Next() {
		var n string
		rows.Scan(&n)
		names = append(names, n)
	}
	rows.Close()
	for _, n := range names {
		if _, e = s.db.Exec("DROP VIEW " + n); e != nil {
			t.Fatal(e)
		}
	}
	for _, n := range []string{"cache_members", "members", "rotation_queue", "cache", "usage", "source_usage", "worker", "settings", "rotation_state", "heads"} {
		if _, e = s.db.Exec("DROP TABLE IF EXISTS ads_smart_" + n); e != nil {
			t.Fatal(e)
		}
	}
}
