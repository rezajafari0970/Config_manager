package panel

import (
	"context"
	"reflect"
	"testing"
)

func TestSplashQueryMigrationPreservesAdoptedPoolsAndCache(t *testing.T) {
	s := testStore(t)
	// Seed source-owned conversions using the current ingestion contract before
	// recreating the preceding pool DDL. Future ingestion cannot run against an
	// intentionally downgraded schema fixture.
	ctx := context.Background()
	newExportSource(t, s, "Normal", sourceURI(1), sourceURI(2), sourceURI(3))
	id, _ := newExportSource(t, s, "Splash", sourceURI(1), sourceURI(11), sourceURI(12))
	dropSplashForMigration(t, s)
	if e := s.migrateV12(); e != nil {
		t.Fatal(e)
	}
	markSplash(t, s, id, true)
	pipelines := []*SmartPipeline{s.Smart(), s.SplashSmart()}
	type snapshot struct {
		entries  []SmartEntry
		bodies   [][]byte
		settings SmartSettings
		round    [2]int
	}
	take := func(p *SmartPipeline) snapshot {
		t.Helper()
		var v snapshot
		var e error
		v.entries, e = querySmart(ctx, p.db, true, 0)
		if e != nil {
			t.Fatal(e)
		}
		v.settings, e = p.SmartSettings(ctx)
		if e != nil {
			t.Fatal(e)
		}
		if e = p.db.QueryRowContext(ctx, "SELECT cycle,(SELECT count(*) FROM smart_rotation_queue) FROM smart_rotation_state").Scan(&v.round[0], &v.round[1]); e != nil {
			t.Fatal(e)
		}
		for _, r := range v.entries {
			b, e := p.smartBytes(ctx, r)
			if e != nil {
				t.Fatal(e)
			}
			v.bodies = append(v.bodies, b)
		}
		return v
	}
	before := make([]snapshot, 2)
	for i, p := range pipelines {
		settings, e := p.SmartSettings(ctx)
		if e != nil {
			t.Fatal(e)
		}
		settings.Count = i + 1
		if e = p.SaveSmartSettings(ctx, settings); e != nil {
			t.Fatal(e)
		}
		finishPipeline(t, p)
		if v, _, _, e := p.smartNext(ctx); e != nil || v == nil {
			t.Fatal(e)
		}
		before[i] = take(p)
	}
	if len(before[0].entries) != 2 || len(before[1].entries) != 3 {
		t.Fatal("incorrect fixture ownership")
	}
	for pass := 0; pass < 2; pass++ {
		if e := s.migrateV13(); e != nil {
			t.Fatal(e)
		}
		for i, p := range pipelines {
			if !reflect.DeepEqual(before[i], take(p)) {
				t.Fatal("migration changed published scope", i)
			}
		}
	}
	var version int
	if e := s.db.QueryRow("PRAGMA user_version").Scan(&version); e != nil || version != 13 {
		t.Fatal(version, e)
	}
	// Retained Splash identity still reserves the raw config without a candidate;
	// an absent candidate cannot poison unrelated ordinary members.
	if _, e := s.db.Exec("DELETE FROM conversion_results WHERE hash IN(SELECT raw_hash FROM memberships WHERE source_id=?)", id); e != nil {
		t.Fatal(e)
	}
	if got := pipelinePool(t, s.Smart()); len(got) != 2 {
		t.Fatal("missing candidate changed unrelated ordinary members", len(got))
	}
	// Classification changes invalidate the reservation in the very next query.
	markSplash(t, s, id, false)
	if got := pipelinePool(t, s.SplashSmart()); len(got) != 0 {
		t.Fatal("stale Splash reservation")
	}
}
