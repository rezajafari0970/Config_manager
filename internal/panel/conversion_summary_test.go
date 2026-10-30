package panel

import (
	"context"
	"github.com/rezajafari0970/Config_manager/internal/convert"
	"github.com/rezajafari0970/Config_manager/internal/repair"
	"testing"
	"time"
)

func TestSourceConversionSummaryMatchesScopedCanonicalOutput(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	rawA := sourceURI(1) + "-A"
	rawB := sourceURI(1) + "-B"
	a, _ := newExportSource(t, s, "SummaryA", rawA)
	b, _ := newExportSource(t, s, "SummaryB", rawB)
	sources := []Source{{ID: a}, {ID: b}}
	exec := func(q string, args ...any) {
		t.Helper()
		if _, e := s.db.Exec(q, args...); e != nil {
			t.Fatal(e)
		}
	}
	check := func(wantA, wantB SourceConversionSummary) {
		t.Helper()
		got, e := s.SourceConversionSummaries(ctx, sources)
		if e != nil {
			t.Fatal(e)
		}
		if got[a] != wantA || got[b] != wantB {
			t.Fatalf("summaries %+v %+v, want %+v %+v", got[a], got[b], wantA, wantB)
		}
	}
	check(SourceConversionSummary{Total: 1, Ready: 1, Connected: 1}, SourceConversionSummary{Total: 1, Ready: 1, Connected: 1})
	// A can use its own other representation; B cannot borrow that proof.
	exec("UPDATE conversion_results SET core='failed',candidate_hash='' WHERE hash=?", repair.Digest([]byte(rawB)))
	check(SourceConversionSummary{Total: 1, Ready: 1, Connected: 1}, SourceConversionSummary{Total: 1, Blocked: 1})
	exec("UPDATE conversion_results SET rule=? WHERE hash=?", convert.Version+"."+repair.RulesVersion, repair.Digest([]byte(rawA)))
	check(SourceConversionSummary{Total: 1, Ready: 1, Connected: 1, Repaired: 1}, SourceConversionSummary{Total: 1, Blocked: 1})
	// Expired connection evidence is not a fresh pass, but structural output remains ready.
	exec("UPDATE conversion_results SET valid_until=0 WHERE hash=?", repair.Digest([]byte(rawA)))
	check(SourceConversionSummary{Total: 1, Ready: 1, Repaired: 1}, SourceConversionSummary{Total: 1, Blocked: 1})
	// Old rules need a new result; no old proof is presented as ready.
	exec("UPDATE conversion_results SET version='old'")
	check(SourceConversionSummary{Total: 1, Pending: 1}, SourceConversionSummary{Total: 1, Pending: 1})
	// Disabled source and expired artifacts disappear using Output's same filters.
	exec("UPDATE sources SET enabled=0 WHERE id=?", a)
	check(SourceConversionSummary{}, SourceConversionSummary{Total: 1, Pending: 1})
	exec("UPDATE artifacts SET expires=?", time.Now().Add(-time.Minute).UnixMilli())
	check(SourceConversionSummary{}, SourceConversionSummary{})
	got, e := s.SourceConversionSummaries(ctx, nil)
	if e != nil || len(got) != 0 {
		t.Fatal("empty page", e)
	}
}
