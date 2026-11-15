package panel

import (
	"context"
	"github.com/rezajafari0970/Config_manager/internal/egress"
	"testing"
)

func TestHardeningFollowupFIFOAndUrgentIran(t *testing.T) {
	for _, iran := range []bool{false, true} {
		s := testStore(t)
		ctx := context.Background()
		newExportSource(t, s, "fair", sourceURI(751), sourceURI(752))
		a, _ := s.ClaimEgress(ctx)
		b, _ := s.ClaimEgress(ctx)
		if a == nil || b == nil {
			t.Fatal("fixture")
		}
		s.db.Exec("UPDATE egress_jobs SET token='',lease_until=0,ever_accepted=1,tested=?,state='RESIDENTIAL',valid_until=?,next_run=2 WHERE hash=?", egress.Now(), egress.Now()+300000, a.Hash)
		if iran {
			s.db.Exec("UPDATE egress_jobs SET token='',lease_until=0,ever_accepted=0,tested=0,reason='ads_iran_quarantined',next_run=1 WHERE hash=?", b.Hash)
		} else {
			s.db.Exec("UPDATE egress_jobs SET token='',lease_until=0,tested=?,deep_completed=1,reason='ads_delivery_unproven',state='UNKNOWN',next_run=1 WHERE hash=?", egress.Now(), b.Hash)
		}
		next, err := s.ClaimEgressLane(ctx, "followup")
		if err != nil || next == nil || next.Hash != b.Hash {
			t.Fatal("due never-accepted safety/recovery starved", iran, err)
		}
	}
}
func TestHardeningUrgentExpiryOverridesFutureNextRun(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "deadline", sourceURI(753))
	j, _ := s.ClaimEgress(ctx)
	s.db.Exec("UPDATE egress_jobs SET token='',lease_until=0,ever_accepted=1,tested=?,state='RESIDENTIAL',valid_until=?,next_run=? WHERE hash=?", egress.Now(), egress.Now()+60000, egress.Now()+180000, j.Hash)
	next, err := s.ClaimEgressLane(ctx, "initial")
	if err != nil || next == nil || next.Hash != j.Hash {
		t.Fatal("deadline missed by stale next_run", err)
	}
}
func TestHardeningWithdrawalSurvivesWindowExpiryAndNeedsOneWindow(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "sticky", sourceURI(754))
	j, _ := s.ClaimEgress(ctx)
	tx, _ := s.db.Begin()
	defer tx.Rollback()
	now := egress.Now()
	for i, dt := range []int64{0, 540000, 1080000} {
		retired, hits, err := s.retirement(ctx, tx, j.Hash, "confirmed_ads_delivery_unhealthy", false, now+dt, now+dt, now+dt)
		want := i + 1
		if i == 2 {
			want = 1
		}
		if err != nil || retired || hits != want {
			t.Fatal("spanning windows retired input", i, retired, hits, err)
		}
	}
	_, hits, err := s.retirement(ctx, tx, j.Hash, "", false, now+1800000, now+1800000, now+1800000)
	if err != nil || hits != 0 {
		t.Fatal(hits, err)
	}
	var blocked int
	if err = tx.QueryRow("SELECT count(*) FROM input_blocked WHERE hash=?", j.Hash).Scan(&blocked); err != nil || blocked != 1 {
		t.Fatal("unknown lifted sticky withdrawal", blocked, err)
	}
	if _, _, err = s.retirement(ctx, tx, j.Hash, "", true, now+1800001, now+1800001, now+1800001); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow("SELECT count(*) FROM egress_retirements WHERE hash=?", j.Hash).Scan(&blocked); err != nil || blocked != 0 {
		t.Fatal("eligible recovery did not clear withdrawal", blocked, err)
	}
}
func TestHardeningDefinitiveCountsUniqueInputsNotRenewals(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "counter", sourceURI(755))
	for i := 0; i < 2; i++ {
		j, err := s.ClaimEgress(ctx)
		if err != nil || j == nil {
			t.Fatal(err)
		}
		if err = s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: syntheticEgress(*j)}); err != nil {
			t.Fatal(err)
		}
		s.db.Exec("UPDATE egress_jobs SET next_run=0 WHERE hash=?", j.Hash)
	}
	var completed, first, definitive int
	s.db.QueryRow("SELECT completed,first_assessments,definitive FROM egress_metrics").Scan(&completed, &first, &definitive)
	if completed != 2 || first != 1 || definitive != 1 {
		t.Fatal(completed, first, definitive)
	}
}
func TestHardeningMigrationPreservesEpochProofAndStickyWithdrawal(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "migration", sourceURI(756))
	j, _ := s.ClaimEgress(ctx)
	if err := s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: syntheticEgress(*j)}); err != nil {
		t.Fatal(err)
	}
	var since int64
	var report string
	s.db.QueryRow("SELECT since FROM egress_metrics").Scan(&since)
	s.db.QueryRow("SELECT report FROM egress_jobs WHERE hash=?", j.Hash).Scan(&report)
	for _, q := range []string{"DROP TABLE egress_settlements", "ALTER TABLE egress_retirements DROP COLUMN first_at", "UPDATE egress_metrics SET definitive=99", "PRAGMA user_version=36"} {
		if _, err := s.db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	s.db.Exec("INSERT INTO egress_retirements(hash,reason,hits,at) VALUES(?,'confirmed_ads_delivery_unhealthy',2,?)", j.Hash, egress.Now())
	for i := 0; i < 2; i++ {
		if err := s.migrateV37(); err != nil {
			t.Fatal(err)
		}
	}
	var clock int64
	var def, hits, rows, version int
	var after string
	s.db.QueryRow("SELECT since,definitive FROM egress_metrics").Scan(&clock, &def)
	s.db.QueryRow("SELECT report FROM egress_jobs WHERE hash=?", j.Hash).Scan(&after)
	s.db.QueryRow("SELECT count(*),coalesce(sum(hits),0) FROM egress_retirements").Scan(&rows, &hits)
	s.db.QueryRow("PRAGMA user_version").Scan(&version)
	if clock != since || after != report || def != 1 || rows != 1 || hits != 0 || version != 37 {
		t.Fatal(clock, since, def, rows, hits, version)
	}
}
