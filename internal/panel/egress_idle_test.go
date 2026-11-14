package panel

import (
	"context"
	"testing"
	"time"

	"github.com/rezajafari0970/Config_manager/internal/egress"
)

func TestEgressIdleClaimDoesNotAcquireWriter(t *testing.T) {
	for _, state := range []string{"empty", "future", "cached", "disabled", "leased"} {
		t.Run(state, func(t *testing.T) {
			s := testStore(t)
			if state != "empty" {
				newExportSource(t, s, "idle", sourceURI(270))
				var statement string
				switch state {
				case "future":
					statement = "UPDATE egress_jobs SET next_run=9223372036854775807"
				case "cached":
					statement = "UPDATE egress_jobs SET state='NO_SPLIT',reason='non_ads_uses_ads_egress',tested=1,first_completed=1,next_run=0"
				case "disabled":
					statement = "UPDATE egress_settings SET enabled=0"
				case "leased":
					statement = "UPDATE egress_jobs SET token='busy',lease_until=9223372036854775807"
				}
				if _, err := s.db.Exec(statement); err != nil {
					t.Fatal(err)
				}
			}
			tx, err := s.db.Begin()
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			// Hold the sole writer throughout all idle polls. WAL preflight must still
			// finish; queue inspection must not delay collector/proof publication.
			for _, lane := range []string{"", "initial", "followup"} {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				job, err := s.ClaimEgressLane(ctx, lane)
				cancel()
				if err != nil || job != nil {
					t.Fatalf("idle lane %q acquired writer: job=%v error=%v", lane, job, err)
				}
			}
		})
	}
}

func TestEgressIdlePreflightPreservesScopeMaintenance(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "removed", sourceURI(271))
	if _, err := s.ResetEgress(ctx, 0); err != nil {
		t.Fatal(err)
	}
	job, err := s.ClaimEgress(ctx)
	if err != nil || job == nil {
		t.Fatal(job, err)
	}
	if _, err = s.db.Exec("DELETE FROM memberships; UPDATE egress_settings SET enabled=0"); err != nil {
		t.Fatal(err)
	}
	next, err := s.ClaimEgress(ctx)
	if err != nil || next != nil {
		t.Fatal(next, err)
	}
	var token, status string
	var revision int64
	if err = s.db.QueryRow("SELECT token,revision FROM egress_jobs WHERE hash=?", job.Hash).Scan(&token, &revision); err != nil {
		t.Fatal(err)
	}
	if err = s.db.QueryRow("SELECT status FROM egress_run_members WHERE hash=?", job.Hash).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if token != "" || revision <= job.Revision || status != "excluded" {
		t.Fatal("scope cleanup skipped", token, revision, status)
	}
	if err = s.CompleteEgress(ctx, egress.Completion{Job: *job, Report: syntheticEgress(*job)}); err != ErrConflict {
		t.Fatal("stale completion accepted", err)
	}
}

func TestEgressIdlePreflightRechecksAfterWriterWait(t *testing.T) {
	s := testStore(t)
	newExportSource(t, s, "removed-during-wait", sourceURI(272))
	tx, err := s.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	before := s.db.Stats().WaitCount
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		job, err := s.ClaimEgress(ctx)
		if job != nil {
			result <- ErrConflict
			return
		}
		result <- err
	}()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for s.db.Stats().WaitCount == before {
		select {
		case err := <-result:
			t.Fatalf("claim returned before writer fence: %v", err)
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-ticker.C:
		}
	}
	if _, err = tx.Exec("DELETE FROM memberships"); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err = <-result; err != nil {
		t.Fatal("preflight bypassed writer scope recheck", err)
	}
	var attempts int
	if err = s.db.QueryRow("SELECT sum(attempts) FROM egress_jobs").Scan(&attempts); err != nil || attempts != 0 {
		t.Fatal(attempts, err)
	}
}
