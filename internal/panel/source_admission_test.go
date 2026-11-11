package panel

import (
	"context"
	"errors"
	"fmt"
	"github.com/rezajafari0970/Config_manager/internal/collect"
	"testing"
	"time"
)

func cardinalityFixture(t *testing.T, s *Store, n int) collect.Result {
	raws := make([]string, n)
	for i := range raws {
		raws[i] = fmt.Sprintf("vless://00000000-0000-4000-8000-%012d@vpn.example:443?security=tls", i)
	}
	return acquisitionResult(t, s, raws...)
}
func TestSourceCardinalityTransientDrop(t *testing.T) {
	for _, staged := range []bool{false, true} {
		t.Run(fmt.Sprint(staged), func(t *testing.T) {
			s := testStore(t)
			ctx := context.Background()
			id, e := s.Save(ctx, "", 0, testSource())
			if e != nil {
				t.Fatal(e)
			}
			now := time.Now()
			large := cardinalityFixture(t, s, 148)
			small := cardinalityFixture(t, s, 2)
			small.Extracted = 6
			small.ETag = "small"
			apply := func(r collect.Result, at time.Time) {
				t.Helper()
				j := claimOne(t, s, id)
				if staged {
					for _, item := range r.Items {
						if _, e := s.Stage(ctx, j, item); e != nil {
							t.Fatal(e)
						}
					}
					r.Items = nil
					r.Staged = true
				}
				if e := s.Complete(ctx, j, r, at); e != nil {
					t.Fatal(e)
				}
			}
			apply(large, now)
			var expires int64
			s.db.QueryRow("SELECT expires FROM artifacts WHERE id=?", large.Items[50].Identity).Scan(&expires)
			apply(small, now.Add(time.Second))
			got, e := s.Get(ctx, id)
			if e != nil {
				t.Fatal(e)
			}
			if got.ConfigCount != 148 || got.ReceiptUnique != 2 || got.ReceiptNew != -1 || got.ReceivedCount != 6 || got.ErrorCode != "source_count_drop_pending" || got.NextRunAt.After(now.Add(16*time.Second)) {
				t.Fatal(got)
			}
			var hash string
			if e = s.db.QueryRow("SELECT hash FROM responses WHERE source_id=?", id).Scan(&hash); e != nil || hash != small.Raw.Hash {
				t.Fatal("raw not preserved", hash, e)
			}
			var after int64
			s.db.QueryRow("SELECT expires FROM artifacts WHERE id=?", large.Items[50].Identity).Scan(&after)
			if after != expires {
				t.Fatal("TTL extended")
			}
			j := claimOne(t, s, id)
			if j.ETag != "" || j.Modified != "" || !j.Revalidate {
				t.Fatal("conditional retry", j.ETag, j.Revalidate)
			}
			if e = s.Complete(ctx, j, large, now.Add(16*time.Second)); e != nil {
				t.Fatal(e)
			}
			got, _ = s.Get(ctx, id)
			if got.ConfigCount != 148 || got.ErrorCode != "" || got.RefreshReason != "" {
				t.Fatal(got)
			}
			var attempts int
			s.db.QueryRow("SELECT pending_attempts FROM source_admission WHERE source_id=?", id).Scan(&attempts)
			if attempts != 0 {
				t.Fatal(attempts)
			}
		})
	}
}
func TestSourceCardinalityPersistentDropAndStaleLease(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	id, _ := s.Save(ctx, "", 0, testSource())
	now := time.Now()
	large := cardinalityFixture(t, s, 64)
	small := cardinalityFixture(t, s, 2)
	if e := s.Complete(ctx, claimOne(t, s, id), large, now); e != nil {
		t.Fatal(e)
	}
	for round := 1; round <= 3; round++ {
		j := claimOne(t, s, id)
		if e := s.Complete(ctx, j, small, now.Add(time.Duration(round)*15*time.Second)); e != nil {
			t.Fatal(e)
		}
		if e := s.Complete(ctx, j, small, now.Add(time.Duration(round)*15*time.Second)); !errors.Is(e, ErrConflict) {
			t.Fatal("stale lease accepted", e)
		}
		got, _ := s.Get(ctx, id)
		want := 64
		if round == 3 {
			want = 2
		}
		if got.ConfigCount != want {
			t.Fatal(round, got)
		}
	}
	var decision string
	s.db.QueryRow("SELECT decision FROM source_admission WHERE source_id=?", id).Scan(&decision)
	if decision != "drop_repeated" {
		t.Fatal(decision)
	}
}
func TestSourceCardinalityExpiryAndDeleteWin(t *testing.T) {
	for _, deleted := range []bool{false, true} {
		t.Run(fmt.Sprint(deleted), func(t *testing.T) {
			s := testStore(t)
			ctx := context.Background()
			id, _ := s.Save(ctx, "", 0, testSource())
			now := time.Now()
			large := cardinalityFixture(t, s, 64)
			small := cardinalityFixture(t, s, 2)
			if e := s.Complete(ctx, claimOne(t, s, id), large, now); e != nil {
				t.Fatal(e)
			}
			if deleted {
				if _, e := s.ManageConfigs(ctx, ConfigBatch{Action: "delete", All: true}); e != nil {
					t.Fatal(e)
				}
			} else {
				if _, e := s.db.Exec("UPDATE artifacts SET expires=?", now.Add(-time.Second).UnixMilli()); e != nil {
					t.Fatal(e)
				}
			}
			if e := s.Complete(ctx, claimOne(t, s, id), small, now.Add(time.Second)); e != nil {
				t.Fatal(e)
			}
			got, _ := s.Get(ctx, id)
			if got.ErrorCode == "source_count_drop_pending" {
				t.Fatal("expired/deleted inventory preserved", got)
			}
			if deleted && got.ConfigCount != 0 {
				t.Fatal("manual deletion resurrected", got)
			}
		})
	}
}
func TestSourceCardinalityMigrationPreservesState(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	id, _ := s.Save(ctx, "", 0, testSource())
	large := cardinalityFixture(t, s, 64)
	if e := s.Complete(ctx, claimOne(t, s, id), large, time.Now()); e != nil {
		t.Fatal(e)
	}
	settings, _ := s.Settings(ctx)
	if _, e := s.db.Exec("DROP TABLE source_admission; PRAGMA user_version=21"); e != nil {
		t.Fatal(e)
	}
	if e := s.migrateV22(); e != nil {
		t.Fatal(e)
	}
	if e := s.migrateV22(); e != nil {
		t.Fatal(e)
	}
	got, _ := s.Get(ctx, id)
	after, _ := s.Settings(ctx)
	if got.ConfigCount != 64 || fmt.Sprint(settings) != fmt.Sprint(after) {
		t.Fatal("migration altered state")
	}
	var v int
	s.db.QueryRow("PRAGMA user_version").Scan(&v)
	if v != 22 {
		t.Fatal(v)
	}
	if e := s.Complete(ctx, claimOne(t, s, id), cardinalityFixture(t, s, 2), time.Now()); e != nil {
		t.Fatal(e)
	}
	got, _ = s.Get(ctx, id)
	if got.ConfigCount != 64 || got.ErrorCode != "source_count_drop_pending" {
		t.Fatal("upgrade missed baseline", got)
	}
}

func TestSourceCardinalitySuppressionCannotBypassGuardOrExposeRaw(t *testing.T) {
	for _, staged := range []bool{false, true} {
		for _, manual := range []bool{false, true} {
			t.Run(fmt.Sprint(staged, manual), func(t *testing.T) {
				s := testStore(t)
				ctx := context.Background()
				id, _ := s.Save(ctx, "", 0, testSource())
				now := time.Now()
				large := cardinalityFixture(t, s, 64)
				if e := s.Complete(ctx, claimOne(t, s, id), large, now); e != nil {
					t.Fatal(e)
				}
				raws := []string{"vless://clean1@vpn.example:443", "vless://clean2@vpn.example:443"}
				for i := 0; i < 15; i++ {
					raws = append(raws, fmt.Sprintf("vless://tombstone%d@vpn.example:443", i))
				}
				mixed := acquisitionResult(t, s, raws...)
				for _, item := range mixed.Items[2:] {
					var e error
					if manual {
						_, e = s.db.Exec("INSERT INTO input_exclusions VALUES(?,?,?,1)", item.Identity, now.UnixMilli(), item.Hash)
					} else {
						_, e = s.db.Exec("INSERT INTO config_controls VALUES(?,'deleted',?,?)", item.Identity, now.UnixMilli(), now.Add(time.Hour).UnixMilli())
					}
					if e != nil {
						t.Fatal(e)
					}
				}
				j := claimOne(t, s, id)
				if staged {
					for _, item := range mixed.Items {
						if _, e := s.Stage(ctx, j, item); e != nil {
							t.Fatal(e)
						}
					}
					mixed.Items = nil
					mixed.Staged = true
				}
				if e := s.Complete(ctx, j, mixed, now.Add(time.Second)); e != nil {
					t.Fatal(e)
				}
				got, _ := s.Get(ctx, id)
				if got.ConfigCount != 64 || got.ReceiptUnique != 17 || got.SuppressedCount != 15 || got.ErrorCode != "source_count_drop_pending" {
					t.Fatal(got)
				}
				_, e := s.Response(ctx, id)
				if manual && !errors.Is(e, ErrNotFound) {
					t.Fatal("manually revoked raw exposed", e)
				}
			})
		}
	}
}
