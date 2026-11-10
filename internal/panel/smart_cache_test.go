package panel

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSmartSafeCacheSurvivesGrowthAndFailedReplacement(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "CacheA", sourceURI(1))
	finishSmart(t, s)
	rows, e := querySmart(ctx, s.db, true, 0)
	if e != nil || len(rows) != 1 || !rows[0].Available {
		t.Fatal("initial cache", e, rows)
	}
	old := rows[0]
	original, e := s.smartBytes(ctx, old)
	if e != nil {
		t.Fatal(e)
	}
	newExportSource(t, s, "CacheB", sourceURI(2))
	// A newly eligible source is an improvement opportunity, not revocation of
	// still-fresh, source-owned proof for the existing cache.
	got, e := s.smartBytes(ctx, old)
	if e != nil || !bytes.Equal(got, original) {
		t.Fatal("growth removed safe cache", e)
	}
	var replaced bool
	for n := 0; n < 4; n++ {
		j, e := s.ClaimSmart(ctx)
		if e != nil || j == nil {
			t.Fatal("replacement claim", e)
		}
		if j.Hash == old.ID {
			got, e = s.smartBytes(ctx, old)
			if e != nil || !bytes.Equal(got, original) {
				t.Fatal("claim destroyed cache", e)
			}
			r := smartResult(j)
			r.Core = "failed"
			r.Code = "synthetic_replacement_failure"
			r.Candidate = nil
			if e = s.CompleteSmart(ctx, r); e != nil {
				t.Fatal(e)
			}
			got, e = s.smartBytes(ctx, old)
			if e != nil || !bytes.Equal(got, original) {
				t.Fatal("failed replacement destroyed cache", e)
			}
			app, _ := NewServer(s, "http://127.0.0.1:18191")
			response := exportRequest(t, app.Handler(), "GET", app.origin+"/api/smart")
			var summary struct {
				Stats map[string]int `json:"stats"`
			}
			if json.Unmarshal(response.Body.Bytes(), &summary) != nil || summary.Stats["ready"]+summary.Stats["blocked"]+summary.Stats["warming"] != summary.Stats["total"] {
				t.Fatal("overlapping readiness counts", response.Body.String())
			}
			replaced = true
			break
		}
		if e = s.CompleteSmart(ctx, smartResult(j)); e != nil {
			t.Fatal(e)
		}
	}
	if !replaced {
		t.Fatal("old cache not scheduled for improvement")
	}
	// Retained caches must still obey immediate proof revocation.
	if _, e = s.db.Exec("UPDATE conversion_results SET connection='failed'"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.smartBytes(ctx, old); e == nil {
		t.Fatal("invalid proof was retained")
	}
}

func TestSmartCacheMigrationAndReopenPreserveValidatedBytes(t *testing.T) {
	s, _ := smartFixture(t)
	finishSmart(t, s)
	ctx := context.Background()
	before, e := querySmart(ctx, s.db, true, 0)
	if e != nil {
		t.Fatal(e)
	}
	first, _, cycle, e := s.smartNext(ctx)
	if e != nil || first == nil {
		t.Fatal(e)
	}
	var remaining int
	if e = s.db.QueryRow("SELECT count(*) FROM smart_rotation_queue").Scan(&remaining); e != nil {
		t.Fatal(e)
	}
	dropSplashForMigration(t, s)
	if _, e = s.db.Exec("DROP VIEW smart_refresh_needed;DROP VIEW smart_ready;DROP VIEW smart_cache_owned;DROP TABLE smart_cache_members;DROP TABLE smart_cache;PRAGMA user_version=10"); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(filepath.Dir(s.vault.Directory()), "panel.db")
	for n := 0; n < 2; n++ {
		other, e := Open(path, s.key)
		if e != nil {
			t.Fatal(e)
		}
		rows, e := querySmart(ctx, other.db, true, 0)
		if e != nil || len(rows) != len(before) {
			t.Fatal("cache migration", e)
		}
		for i, row := range rows {
			if !row.Available || row.Hash != before[i].Hash {
				t.Fatal("canonical cache replaced during migration")
			}
		}
		var after, gotCycle int64
		if e = other.db.QueryRow("SELECT cycle,(SELECT count(*) FROM smart_rotation_queue) FROM smart_rotation_state").Scan(&gotCycle, &after); e != nil {
			t.Fatal(e)
		}
		if gotCycle != cycle || after != int64(remaining) {
			t.Fatal("migration reset rotation")
		}
		other.Close()
	}
}

func TestSmartCorruptCacheRebuildAndCanceledReadIsolation(t *testing.T) {
	s, _ := smartFixture(t)
	finishSmart(t, s)
	ctx := context.Background()
	rows, e := querySmart(ctx, s.db, true, 0)
	if e != nil {
		t.Fatal(e)
	}
	old := rows[0]
	if e = os.WriteFile(filepath.Join(s.vault.Directory(), old.Hash+".blob"), []byte("corrupt"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = s.smartBytes(ctx, old); e == nil {
		t.Fatal("corrupt cache accepted")
	}
	if used, e := s.Referenced(ctx, old.Hash); e != nil || !used {
		t.Fatal("pending head lost blob reference", e)
	}
	finishSmart(t, s)
	rows, e = querySmart(ctx, s.db, true, 0)
	if e != nil {
		t.Fatal(e)
	}
	for _, row := range rows {
		if !row.Available {
			t.Fatal("cache not rebuilt")
		}
		if _, e = s.smartBytes(ctx, row); e != nil {
			t.Fatal(e)
		}
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	s.retrySmartBlob(canceled, rows[0])
	if _, e = s.smartBytes(ctx, rows[0]); e != nil {
		t.Fatal("canceled request revoked cache", e)
	}
	// Fresh proof still cannot extend fixed artifact/cache TTL.
	if _, e = s.db.Exec("UPDATE smart_cache SET expires=? WHERE id=?", time.Now().Add(-time.Second).UnixMilli(), rows[0].ID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.smartBytes(ctx, rows[0]); e == nil {
		t.Fatal("expired cache survived")
	}
}
