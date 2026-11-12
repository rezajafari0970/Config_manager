package panel

import (
	"context"
	"github.com/rezajafari0970/Config_manager/internal/egress"
	"strings"
	"testing"
	"time"
)

func TestEgressExpiredLeaseCannotWaitThroughWriter(t *testing.T) {
	for _, operation := range []string{"renew", "complete"} {
		t.Run(operation, func(t *testing.T) {
			s := testStore(t)
			ctx := context.Background()
			newExportSource(t, s, "writer", sourceURI(245))
			j, e := s.ClaimEgress(ctx)
			if e != nil || j == nil {
				t.Fatal(e)
			}
			report := syntheticEgress(*j)
			expiry := egress.Now() + 300
			if _, e = s.db.Exec("UPDATE egress_jobs SET lease_until=? WHERE hash=?", expiry, j.Hash); e != nil {
				t.Fatal(e)
			}
			before, e := egressMetricState(ctx, s.db)
			if e != nil {
				t.Fatal(e)
			}
			var generation int64
			s.db.QueryRow("SELECT generation FROM egress_settings").Scan(&generation)
			waitCount := s.db.Stats().WaitCount
			tx, e := s.db.BeginTx(ctx, nil)
			if e != nil {
				t.Fatal(e)
			}
			if _, e = tx.Exec("UPDATE egress_settings SET generation=generation WHERE id=1"); e != nil {
				t.Fatal(e)
			}
			done := make(chan error, 1)
			go func() {
				if operation == "renew" {
					done <- s.RenewEgress(ctx, *j)
				} else {
					done <- s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: report})
				}
			}()
			deadline := time.Now().Add(2 * time.Second)
			for s.db.Stats().WaitCount == waitCount && time.Now().Before(deadline) {
				time.Sleep(time.Millisecond)
			}
			waiting := s.db.Stats().WaitCount > waitCount
			for egress.Now() <= expiry+30 {
				time.Sleep(time.Millisecond)
			}
			if e = tx.Commit(); e != nil {
				t.Fatal(e)
			}
			if !waiting {
				t.Fatal("operation did not wait for writer")
			}
			select {
			case e = <-done:
				if e != ErrConflict {
					t.Fatal(operation, e)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("blocked completion")
			}
			after, e := egressMetricState(ctx, s.db)
			if e != nil || before != after {
				t.Fatal(before, after, e)
			}
			var health, events int
			var g, lease int64
			s.db.QueryRow("SELECT count(*) FROM egress_health").Scan(&health)
			s.db.QueryRow("SELECT count(*) FROM egress_events").Scan(&events)
			s.db.QueryRow("SELECT generation FROM egress_settings").Scan(&g)
			s.db.QueryRow("SELECT lease_until FROM egress_jobs WHERE hash=?", j.Hash).Scan(&lease)
			if health != 0 || events != 0 || g != generation || lease != expiry {
				t.Fatal("expired writer persisted effects", health, events, g, generation, lease, expiry)
			}
		})
	}
}

func TestEgressFollowupLaneProgressWithInitialBacklog(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "fair", sourceURI(246), sourceURI(247), sourceURI(248), sourceURI(249))
	a, _ := s.ClaimEgress(ctx)
	b, _ := s.ClaimEgress(ctx)
	if _, e := s.db.Exec("UPDATE egress_jobs SET tested=1,state='UNKNOWN',next_run=0,lease_until=1 WHERE hash IN (?,?)", a.Hash, b.Hash); e != nil {
		t.Fatal(e)
	}
	seen := map[string]bool{}
	for round := 0; round < 2; round++ {
		// Initial lane continuously has fresh work available.
		newExportSource(t, s, []string{"more-a", "more-b"}[round], sourceURI(250+round))
		initial, e := s.ClaimEgressLane(ctx, "initial")
		if e != nil || initial == nil || initial.Hash == a.Hash || initial.Hash == b.Hash {
			t.Fatal(initial, e)
		}
		follow, e := s.ClaimEgressLane(ctx, "followup")
		if e != nil || follow == nil {
			t.Fatal(e)
		}
		seen[follow.Hash] = true
	}
	if !seen[a.Hash] || !seen[b.Hash] {
		t.Fatal("expired follow-up leases starved", seen)
	}
}

func TestEgressUpgradeBackfillsStagingBeforeRecovery(t *testing.T) {
	for _, expire := range []bool{false, true} {
		t.Run(map[bool]string{false: "recover", true: "expire"}[expire], func(t *testing.T) {
			s := testStore(t)
			ctx := context.Background()
			id, e := s.Save(ctx, "", 0, testSource())
			if e != nil {
				t.Fatal(e)
			}
			items := acquisitionResult(t, s, sourceURI(253), sourceURI(254)).Items
			j := claimOne(t, s, id)
			// A pre-v28 store had no durable staging trigger.
			if _, e = s.db.Exec("DROP TRIGGER egress_stage"); e != nil {
				t.Fatal(e)
			}
			if n, e := s.StageBatch(ctx, j, items); e != nil || n != 2 {
				t.Fatal(n, e)
			}
			if _, e = s.db.Exec("INSERT INTO input_exclusions VALUES(?,?,?,1)", items[1].Identity, egress.Now(), items[1].Hash); e != nil {
				t.Fatal(e)
			}
			var view string
			s.db.QueryRow("SELECT sql FROM sqlite_master WHERE name='egress_eligible'").Scan(&view)
			cut := strings.Index(view, " AND EXISTS(SELECT 1 FROM egress_health")
			if cut < 0 {
				t.Fatal("health clause missing")
			}
			if _, e = s.db.Exec("DROP VIEW egress_eligible;" + view[:cut] + ";DROP TABLE egress_health;DROP TABLE egress_lifecycle;PRAGMA user_version=27"); e != nil {
				t.Fatal(e)
			}
			if e = s.migrateV28(); e != nil {
				t.Fatal(e)
			}
			if expire {
				e = s.Expire(ctx, time.Now().Add(24*time.Hour))
			} else {
				e = s.Recover(ctx)
			}
			if e != nil {
				t.Fatal(e)
			}
			var staged, jobs, owners int
			s.db.QueryRow("SELECT count(*) FROM staged").Scan(&staged)
			s.db.QueryRow("SELECT count(*) FROM egress_jobs").Scan(&jobs)
			s.db.QueryRow("SELECT count(*) FROM egress_owners").Scan(&owners)
			if staged != 0 || jobs != 1 || owners != 1 {
				t.Fatal(staged, jobs, owners)
			}
			used, e := s.Referenced(ctx, items[0].Hash)
			if e != nil || !used {
				t.Fatal("upgraded input unreferenced", e)
			}
			work, e := s.ClaimEgress(ctx)
			if e != nil || work == nil || work.Hash != items[0].Hash || len(work.Raw) == 0 {
				t.Fatal(work, e)
			}
		})
	}
}
