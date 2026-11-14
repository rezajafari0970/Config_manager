package panel

import (
	"context"
	"github.com/rezajafari0970/Config_manager/internal/egress"
	"testing"
)

func TestEgressResetFencesOldWorkPreservesAuditAndPrioritizesAccepted(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "reset", sourceURI(250), sourceURI(251), sourceURI(252))
	accepted, err := s.ClaimEgress(ctx)
	if err != nil || accepted == nil {
		t.Fatal(err)
	}
	if err = s.CompleteEgress(ctx, egress.Completion{Job: *accepted, Report: syntheticEgress(*accepted)}); err != nil {
		t.Fatal(err)
	}
	old, err := s.ClaimEgress(ctx)
	if err != nil || old == nil {
		t.Fatal(err)
	}
	stale := egress.Completion{Job: *old, Report: syntheticEgress(*old)}
	before, _ := egressMetricState(ctx, s.db)
	id, err := s.ResetEgress(ctx, 0)
	if err != nil || id != 1 {
		t.Fatal(id, err)
	}
	after, _ := egressMetricState(ctx, s.db)
	if before != after {
		t.Fatal("lifetime counters reset", before, after)
	}
	if _, err = s.ResetEgress(ctx, 0); err != ErrConflict {
		t.Fatal("duplicate reset", err)
	}
	if err = s.CompleteEgress(ctx, stale); err != ErrConflict {
		t.Fatal("old worker overwrote reset", err)
	}
	var eligible, oldReports int
	s.db.QueryRow("SELECT count(*) FROM egress_eligible").Scan(&eligible)
	s.db.QueryRow("SELECT count(*) FROM egress_jobs WHERE report!='{}' AND tested=0").Scan(&oldReports)
	if eligible != 0 || oldReports != 1 {
		t.Fatal("freshness/archive", eligible, oldReports)
	}
	fresh, err := s.ClaimEgressLane(ctx, "initial")
	if err != nil || fresh == nil || fresh.Hash != accepted.Hash || !fresh.Accepted || !fresh.Deep || fresh.Attempt != 1 {
		t.Fatal("accepted priority/deep check", fresh, err)
	}
	complete := egress.Completion{Job: *fresh, Report: syntheticEgress(*fresh)}
	if err = s.CompleteEgress(ctx, complete); err != nil {
		t.Fatal(err)
	}
	if err = s.CompleteEgress(ctx, complete); err != ErrConflict {
		t.Fatal("duplicate completion", err)
	}
	run, err := egressRunState(ctx, s.db)
	if err != nil || run["completed"] != 1 || run["pending"] != 2 || run["accepted_completed"] != 1 {
		t.Fatal(run, err)
	}
	next, err := s.ClaimEgressLane(ctx, "initial")
	if err != nil || next == nil || next.Deep || next.Attempt != 1 {
		t.Fatal("fresh round inherited old attempts", next, err)
	}
	if err = s.CompleteEgress(ctx, egress.Completion{Job: *next, Report: syntheticEgress(*next)}); err != nil {
		t.Fatal(err)
	}
	after, _ = egressMetricState(ctx, s.db)
	if after.Completed != before.Completed+2 || after.FirstAssessments != before.FirstAssessments+1 {
		t.Fatal("lifetime first assessments invented", before, after)
	}
	// Another reset is explicit, CAS protected, and previous cohort results survive.
	id, err = s.ResetEgress(ctx, 1)
	if err != nil || id != 2 {
		t.Fatal(id, err)
	}
	var prior int
	s.db.QueryRow("SELECT count(*) FROM egress_run_members WHERE run=1 AND status='completed'").Scan(&prior)
	if prior != 2 {
		t.Fatal("prior round lost", prior)
	}
}

func TestEgressResetPreservesBlockedPausedAndManualExclusions(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "guards", sourceURI(253), sourceURI(254), sourceURI(255), sourceURI(256))
	rows, err := s.db.Query("SELECT hash,identity FROM egress_owners ORDER BY hash")
	if err != nil {
		t.Fatal(err)
	}
	type item struct{ hash, identity string }
	all := []item{}
	for rows.Next() {
		var i item
		if err = rows.Scan(&i.hash, &i.identity); err != nil {
			t.Fatal(err)
		}
		all = append(all, i)
	}
	rows.Close()
	if len(all) != 4 {
		t.Fatal(len(all))
	}
	if _, err = s.db.Exec("INSERT INTO egress_iran_guard(identity,state,first_seen) VALUES(?,'BLOCKED',1)", all[0].identity); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec("INSERT INTO input_exclusions(identity,at,hash,manual) VALUES(?,1,?,1)", all[1].identity, all[1].hash); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec("UPDATE egress_jobs SET paused=1 WHERE hash=?", all[2].hash); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ResetEgress(ctx, 0); err != nil {
		t.Fatal(err)
	}
	run, err := egressRunState(ctx, s.db)
	if err != nil || run["total"] != 1 {
		t.Fatal(run, err)
	}
	var guards, exclusions, paused int
	s.db.QueryRow("SELECT count(*) FROM egress_iran_guard WHERE state='BLOCKED'").Scan(&guards)
	s.db.QueryRow("SELECT count(*) FROM input_exclusions").Scan(&exclusions)
	s.db.QueryRow("SELECT count(*) FROM egress_jobs WHERE paused=1").Scan(&paused)
	if guards != 1 || exclusions != 1 || paused != 1 {
		t.Fatal(guards, exclusions, paused)
	}
	tx, _ := s.db.Begin()
	if err = s.revokeEgress(ctx, tx, all[3].hash, "manual_delete", egress.Now()); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	run, err = egressRunState(ctx, s.db)
	if err != nil || run["excluded"] != 1 || run["completed"] != 0 || run["finished"] != true {
		t.Fatal(run, err)
	}
}

func TestEgressRunCountsConfirmedIranCompletionAsTested(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "iran-round", sourceURI(257))
	if _, err := s.ResetEgress(ctx, 0); err != nil {
		t.Fatal(err)
	}
	j, err := s.ClaimEgress(ctx)
	if err != nil || j == nil {
		t.Fatal(err)
	}
	// Revocation marks the other members excluded, while the actual measured
	// member must be completed, even when the deletion trigger ran first.
	tx, _ := s.db.Begin()
	if err = s.revokeEgress(ctx, tx, j.Hash, "confirmed_ads_iran", egress.Now()); err != nil {
		t.Fatal(err)
	}
	if err = recordEgressRun(ctx, tx, j.Hash, "DELETED", false, egress.Now(), egress.Now(), 20); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	run, err := egressRunState(ctx, s.db)
	if err != nil || run["completed"] != 1 || run["excluded"] != 0 || run["definitive"] != 0 {
		t.Fatal(run, err)
	}
}

func TestEgressManualDeleteFencesAllAliasesWithoutInventingHealthDeletion(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "alias-delete", sourceURI(258), sourceURI(259))
	first, _ := s.ClaimEgress(ctx)
	second, _ := s.ClaimEgress(ctx)
	var identity string
	if err := s.db.QueryRow("SELECT identity FROM egress_owners WHERE hash=? LIMIT 1", first.Hash).Scan(&identity); err != nil {
		t.Fatal(err)
	}
	// Bind an existing in-flight alternate representation to the same identity.
	if _, err := s.db.Exec("UPDATE egress_owners SET identity=? WHERE hash=?", identity, second.Hash); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ResetEgress(ctx, 0); err != nil {
		t.Fatal(err)
	}
	stale, _ := s.ClaimEgress(ctx)
	tx, _ := s.db.Begin()
	if err := s.revokeEgress(ctx, tx, first.Hash, "manual_delete", egress.Now()); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	var live int
	s.db.QueryRow("SELECT count(*) FROM egress_jobs WHERE state!='DELETED'").Scan(&live)
	if live != 0 {
		t.Fatal("alias remained in active queue", live)
	}
	if err := s.CompleteEgress(ctx, egress.Completion{Job: *stale, Report: syntheticEgress(*stale)}); err != ErrConflict {
		t.Fatal("alias lease survived exclusion", err)
	}
	run, err := egressRunState(ctx, s.db)
	if err != nil || run["excluded"] != 2 || run["pending"] != 0 {
		t.Fatal(run, err)
	}
	metrics, _ := egressMetricState(ctx, s.db)
	if metrics.UnhealthyDeleted != 0 {
		t.Fatal("manual deletion counted as health failure")
	}
}
