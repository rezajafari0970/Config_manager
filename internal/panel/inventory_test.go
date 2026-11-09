package panel

import (
	"context"
	"github.com/rezajafari0970/Config_manager/internal/collect"
	"testing"
	"time"
)

func TestReceiptDistinguishesResponseDuplicatesFromExisting(t *testing.T) {
	for _, staged := range []bool{false, true} {
		t.Run(map[bool]string{false: "direct", true: "staged"}[staged], func(t *testing.T) {
			s := testStore(t)
			ctx := context.Background()
			id, _ := s.Save(ctx, "", 0, testSource())
			r := acquisitionResult(t, s, "vless://one@vpn.example:443#A", "vless://one@vpn.example:443#B", "vless://two@vpn.example:443#C")
			for round := 0; round < 2; round++ {
				j := claimOne(t, s, id)
				copy := r
				if staged {
					for _, item := range copy.Items {
						if _, e := s.Stage(ctx, j, item); e != nil {
							t.Fatal(e)
						}
					}
					copy.Items = nil
					copy.Staged = true
				}
				if e := s.Complete(ctx, j, copy, time.Now()); e != nil {
					t.Fatal(e)
				}
				got, _ := s.Get(ctx, id)
				wantNew := 2
				if round == 1 {
					wantNew = 0
				}
				if got.ReceiptUnique != 2 || got.ReceiptNew != wantNew || got.ConfigCount != 2 || got.ReceivedCount != 3 || got.ReceiptAt == 0 {
					t.Fatalf("bad receipt %+v", got)
				}
			}
			before, _ := s.Get(ctx, id)
			if e := s.Complete(ctx, claimOne(t, s, id), collect.Result{State: "unchanged", HTTPStatus: 304}, time.Now()); e != nil {
				t.Fatal(e)
			}
			after, _ := s.Get(ctx, id)
			if after.ReceiptAt != before.ReceiptAt || after.ReceiptUnique != 2 || after.ReceivedCount != 3 {
				t.Fatal("304 rewrote body receipt", after)
			}
		})
	}
}

func TestExpiryQueuesAffectedSourcesBeforeEveryCascade(t *testing.T) {
	for _, path := range []string{"gc", "completion", "settings"} {
		t.Run(path, func(t *testing.T) {
			s := testStore(t)
			ctx := context.Background()
			in := testSource()
			in.Min = 2
			in.Max = 89
			in.Unit = "minutes"
			id, _ := s.Save(ctx, "", 0, in)
			r := acquisitionResult(t, s, "vless://one@vpn.example:443", "vless://two@vpn.example:443")
			s.Complete(ctx, claimOne(t, s, id), r, time.Now())
			other := in
			other.URL = "https://example.net/other"
			other.Enabled = false
			disabled, _ := s.Save(ctx, "", 0, other)
			future := time.Now().Add(time.Hour).UnixNano()
			s.db.Exec("UPDATE sources SET next_run=?", future)
			s.db.Exec("UPDATE artifacts SET expires=? WHERE id=?", time.Now().Add(-time.Second).UnixMilli(), r.Items[0].Identity)
			var e error
			switch path {
			case "gc":
				e = s.Expire(ctx, time.Now())
			case "settings":
				v, _ := s.Settings(ctx)
				e = s.SaveSettings(ctx, v)
			case "completion":
				third := in
				third.URL = "https://example.net/third"
				tid, _ := s.Save(ctx, "", 0, third)
				e = s.Complete(ctx, claimOne(t, s, tid), acquisitionResult(t, s, "vless://three@vpn.example:443"), time.Now())
			}
			if e != nil {
				t.Fatal(e)
			}
			got, _ := s.Get(ctx, id)
			if got.RefreshReason != "retention" || got.NextRunAt.After(time.Now()) || got.ConfigCount != 1 || got.Min != 2 || got.Max != 89 {
				t.Fatalf("expiry not recovered %+v", got)
			}
			off, _ := s.Get(ctx, disabled)
			if off.NextRunAt.UnixNano() != future || off.Enabled {
				t.Fatal("disabled source changed")
			}
			jobs, e := s.Claim(ctx, time.Now(), 20, time.Minute)
			if e != nil {
				t.Fatal(e)
			}
			found := false
			for _, j := range jobs {
				if j.ID == id {
					found = true
					if j.ETag != "" || j.Modified != "" {
						t.Fatal("partial expiry sent validators")
					}
					if e = s.Complete(ctx, j, r, time.Now()); e != nil {
						t.Fatal(e)
					}
				}
			}
			if !found {
				t.Fatal("not immediately claimed")
			}
			got, _ = s.Get(ctx, id)
			if got.ConfigCount != 2 || got.RefreshReason != "" {
				t.Fatal("inventory not replenished", got)
			}
		})
	}
}

func TestReceiptDoesNotExtendTTLAndDeadlinePreventsLongGap(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	in := testSource()
	in.Min = 3
	in.Max = 3
	in.Unit = "hours"
	id, _ := s.Save(ctx, "", 0, in)
	r := acquisitionResult(t, s, "vless://one@vpn.example:443")
	s.Complete(ctx, claimOne(t, s, id), r, time.Now())
	var expiry int64
	s.db.QueryRow("SELECT expires FROM artifacts WHERE id=?", r.Items[0].Identity).Scan(&expiry)
	for _, result := range []collect.Result{r, {State: "unchanged", HTTPStatus: 304}} {
		if e := s.Complete(ctx, claimOne(t, s, id), result, time.Now()); e != nil {
			t.Fatal(e)
		}
		got, _ := s.Get(ctx, id)
		if got.NextRunAt.UnixMilli() > expiry {
			t.Fatal("scheduled after retention", got.NextRunAt, expiry)
		}
	}
	var after int64
	s.db.QueryRow("SELECT expires FROM artifacts WHERE id=?", r.Items[0].Identity).Scan(&after)
	if after != expiry {
		t.Fatal("TTL extended")
	}
}

func TestPartial304SchedulesUnconditionalBoundedRetry(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	id, _ := s.Save(ctx, "", 0, testSource())
	r := acquisitionResult(t, s, "vless://one@vpn.example:443", "vless://two@vpn.example:443")
	s.Complete(ctx, claimOne(t, s, id), r, time.Now())
	j := claimOne(t, s, id)
	s.db.Exec("DELETE FROM artifacts WHERE id=?", r.Items[0].Identity)
	if e := s.Complete(ctx, j, collect.Result{State: "unchanged", HTTPStatus: 304}, time.Now()); e != nil {
		t.Fatal(e)
	}
	got, _ := s.Get(ctx, id)
	if got.ErrorCode != "cache_missing" || got.RefreshReason != "missing_inventory" || got.NextRunAt.After(time.Now().Add(6*time.Second)) {
		t.Fatal(got)
	}
	j = claimOne(t, s, id)
	if j.ETag != "" || j.Modified != "" {
		t.Fatal("conditional retry")
	}
}

func TestInventorySnapshotDoesNotMixGenerations(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	id, _ := s.Save(ctx, "", 0, testSource())
	r := acquisitionResult(t, s, "vless://one@vpn.example:443")
	s.Complete(ctx, claimOne(t, s, id), r, time.Now())
	snap, finish, e := s.readSnapshot(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer finish()
	sources, e := s.List(snap)
	if e != nil || len(sources) != 1 || sources[0].ConfigCount != 1 {
		t.Fatal(sources, e)
	}
	if _, e = s.ManageConfigs(ctx, ConfigBatch{Action: "delete", All: true}); e != nil {
		t.Fatal(e)
	}
	stats, e := s.Stats(snap)
	if e != nil || stats.Configs != 1 {
		t.Fatal("mixed generation", stats, e)
	}
	summary, e := s.SourceConversionSummaries(snap, sources)
	if e != nil || summary[id].Total != 1 {
		t.Fatal("mixed summary", summary, e)
	}
	current, e := s.Stats(ctx)
	if e != nil || current.Configs != 0 {
		t.Fatal("writer did not commit", current, e)
	}
}

func TestLegacyMissingInventoryRecoveredOnceWithoutClearingControls(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	id, _ := s.Save(ctx, "", 0, testSource())
	r := acquisitionResult(t, s, "vless://one@vpn.example:443")
	s.Complete(ctx, claimOne(t, s, id), r, time.Now())
	s.db.Exec("DELETE FROM artifacts")
	s.db.Exec("DELETE FROM responses")
	s.db.Exec("UPDATE sources SET next_run=?", time.Now().Add(time.Hour).UnixNano())
	if e := s.Recover(ctx); e != nil {
		t.Fatal(e)
	}
	got, _ := s.Get(ctx, id)
	if got.RefreshReason != "missing_inventory" || got.NextRunAt.After(time.Now()) {
		t.Fatal(got)
	}
	j := claimOne(t, s, id)
	s.Complete(ctx, j, collect.Result{State: "error", ErrorCode: "http_error", HTTPStatus: 503}, time.Now())
	before, _ := s.Get(ctx, id)
	s.Recover(ctx)
	after, _ := s.Get(ctx, id)
	if after.NextRunAt != before.NextRunAt {
		t.Fatal("restart erased backoff")
	}
}

func TestInventoryV14UpgradePreservesDefinitionsAndInvalidatesLegacyReceipts(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	id, _ := s.Save(ctx, "", 0, testSource())
	r := acquisitionResult(t, s, "vless://one@vpn.example:443")
	s.Complete(ctx, claimOne(t, s, id), r, time.Now())
	before, _ := s.Get(ctx, id)
	// Rollback can retain additive columns; the old binary cannot maintain them.
	if _, e := s.db.Exec("PRAGMA user_version=13"); e != nil {
		t.Fatal(e)
	}
	if e := s.migrateV14(); e != nil {
		t.Fatal(e)
	}
	after, _ := s.Get(ctx, id)
	if after.Revision != before.Revision || after.URL != before.URL || after.ConfigCount != before.ConfigCount || after.NextRunAt != before.NextRunAt || after.ReceiptUnique != -1 || after.ReceiptNew != -1 || after.ReceiptAt != 0 {
		t.Fatal("migration changed ownership or fabricated measurements", after)
	}
	if e := s.migrateV14(); e != nil {
		t.Fatal(e)
	}
	var version int
	s.db.QueryRow("PRAGMA user_version").Scan(&version)
	if version != 14 {
		t.Fatal(version)
	}
}

func TestInventorySnapshotUsesOneExpiryMoment(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	id, _ := s.Save(ctx, "", 0, testSource())
	r := acquisitionResult(t, s, "vless://one@vpn.example:443")
	s.Complete(ctx, claimOne(t, s, id), r, time.Now())
	s.db.Exec("UPDATE artifacts SET expires=?", time.Now().Add(75*time.Millisecond).UnixMilli())
	snap, finish, e := s.readSnapshot(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer finish()
	sources, e := s.List(snap)
	if e != nil || sources[0].ConfigCount != 1 {
		t.Fatal(sources, e)
	}
	time.Sleep(100 * time.Millisecond)
	stats, e := s.Stats(snap)
	if e != nil || stats.Configs != 1 || stats.GivingConfigs != 1 {
		t.Fatal("mixed expiry moments", stats, e)
	}
	conv, e := s.SourceConversionSummaries(snap, sources)
	if e != nil || conv[id].Total != 1 {
		t.Fatal(conv, e)
	}
	current, _ := s.Get(ctx, id)
	if current.ConfigCount != 0 {
		t.Fatal("later request ignored expiry")
	}
}

func TestSuppressedRecoveryReturnsToNormalSchedule(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	in := testSource()
	in.Min = 10
	in.Max = 10
	in.Unit = "minutes"
	id, _ := s.Save(ctx, "", 0, in)
	r := acquisitionResult(t, s, "vless://one@vpn.example:443")
	s.Complete(ctx, claimOne(t, s, id), r, time.Now())
	s.ManageConfigs(ctx, ConfigBatch{Action: "delete", IDs: []string{r.Items[0].Identity}})
	s.db.Exec("UPDATE sources SET refresh_reason='missing_inventory' WHERE id=?", id)
	if e := s.Complete(ctx, claimOne(t, s, id), r, time.Now()); e != nil {
		t.Fatal(e)
	}
	got, _ := s.Get(ctx, id)
	if got.State != "filtered" || got.ConfigCount != 0 || got.RefreshReason != "" || got.NextRunAt.Before(time.Now().Add(9*time.Minute)) {
		t.Fatal("deleted input caused a recovery loop", got)
	}
}
