package panel

import (
	"context"
	"github.com/rezajafari0970/Config_manager/internal/repair"
	"testing"
	"time"
)

func TestSmartRequiresFreshOwnedConnectionForEveryInput(t *testing.T) {
	for _, scenario := range []string{"failed", "partial", "not_run", "expired"} {
		t.Run(scenario, func(t *testing.T) {
			s, _ := smartFixture(t)
			ctx := context.Background()
			finishSmart(t, s)
			rows, e := querySmart(ctx, s.db, true, 0)
			if e != nil || len(rows) != 7 {
				t.Fatal("fixture", e)
			}
			// Source B has only one member, so every ready profile includes this raw.
			hash := repair.Digest([]byte(sourceURI(5)))
			connection, expires := scenario, time.Now().Add(time.Hour).UnixMilli()
			if scenario == "expired" {
				connection = "passed"
				expires = time.Now().Add(-time.Second).UnixMilli()
			}
			_, e = s.db.Exec("UPDATE conversion_results SET connection=?,valid_until=?,requires_connection=0 WHERE hash=?", connection, expires, hash)
			if e != nil {
				t.Fatal(e)
			}
			var n int
			s.db.QueryRow("SELECT count(*) FROM smart_inputs WHERE raw_hash=?", hash).Scan(&n)
			if n != 0 {
				t.Fatal("core-only input admitted", scenario)
			}
			for _, r := range rows {
				if _, e = s.smartBytes(ctx, r); e == nil {
					t.Fatal("old cache survived loss of connection evidence")
				}
			}
			// Normal exports retain their existing independent core-valid contract.
			normal, e := s.Conversions(ctx, 0, true)
			if e != nil || len(normal) != 7 {
				t.Fatal("normal output changed", e, len(normal))
			}
			finishSmart(t, s)
			inventory, e := smartInventory(ctx, s.db)
			if e != nil || len(inventory) != 6 {
				t.Fatal("quality pool", e, len(inventory))
			}
			for _, m := range inventory {
				if m.RawHash == hash {
					t.Fatal("excluded member selected")
				}
			}
			// Fresh successful revalidation makes the input eligible automatically.
			_, e = s.db.Exec("UPDATE conversion_results SET connection='passed',valid_until=? WHERE hash=?", time.Now().Add(time.Hour).UnixMilli(), hash)
			if e != nil {
				t.Fatal(e)
			}
			finishSmart(t, s)
			inventory, e = smartInventory(ctx, s.db)
			if e != nil || len(inventory) != 7 {
				t.Fatal("recovered input not restored", e)
			}
		})
	}
}
func TestSmartConnectionDeadlineBoundsCacheAndCannotBorrowProof(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	rawA, rawB := sourceURI(1)+"-A", sourceURI(1)+"-B"
	a, _ := newExportSource(t, s, "QualityA", rawA)
	b, _ := newExportSource(t, s, "QualityB", rawB)
	deadline := time.Now().Add(5 * time.Minute).UnixMilli()
	_, e := s.db.Exec("UPDATE conversion_results SET connection='passed',valid_until=? WHERE hash=?", deadline, repair.Digest([]byte(rawA)))
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.db.Exec("UPDATE conversion_results SET connection='failed',valid_until=0,requires_connection=0 WHERE hash=?", repair.Digest([]byte(rawB)))
	if e != nil {
		t.Fatal(e)
	}
	inventory, e := smartInventory(ctx, s.db)
	if e != nil || len(inventory) != 1 || inventory[0].Source != a || inventory[0].Source == b || inventory[0].Expires != deadline {
		t.Fatal("proof borrowed or deadline wrong", e, inventory)
	}
	j, e := s.ClaimSmart(ctx)
	if e != nil || j == nil {
		t.Fatal(e)
	}
	var expiry int64
	s.db.QueryRow("SELECT expires FROM smart_heads WHERE id=?", j.Hash).Scan(&expiry)
	if expiry > deadline {
		t.Fatal("cache outlives connection proof")
	}
	_, e = s.db.Exec("UPDATE conversion_results SET connection='failed',valid_until=0 WHERE hash=?", inventory[0].RawHash)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.CompleteSmart(ctx, smartResult(j)); e == nil {
		t.Fatal("stale worker resurrected unhealthy member")
	}
	j, e = s.ClaimSmart(ctx)
	if e != nil || j != nil {
		t.Fatal("unhealthy-only source selected", e)
	}
}
