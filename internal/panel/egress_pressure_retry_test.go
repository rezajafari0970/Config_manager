package panel

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/rezajafari0970/Config_manager/internal/egress"
)

func TestEgressPressureCannotExhaustUnassessedIdentity(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "pressure-recovery", sourceURI(277))
	previousDelay := int64(0)
	for round := 0; round < 3; round++ {
		if _, err := s.db.Exec("UPDATE egress_jobs SET next_run=0"); err != nil {
			t.Fatal(err)
		}
		job, err := s.ClaimEgress(ctx)
		if err != nil || job == nil {
			t.Fatalf("identity stranded after %d resource-pressure reports: job=%v error=%v", round, job, err)
		}
		report := syntheticEgress(*job)
		report.ResourcePressure = true
		report.Samples = nil
		if err = s.CompleteEgress(ctx, egress.Completion{Job: *job, Report: report}); err != nil {
			t.Fatal(err)
		}
		var delay int64
		if err = s.db.QueryRow("SELECT next_run-tested FROM egress_jobs WHERE hash=?", job.Hash).Scan(&delay); err != nil || delay < 300000 || delay < previousDelay || delay > 3600000 {
			t.Fatal("pressure retry did not back off", delay, previousDelay, err)
		}
		previousDelay = delay
		var eligible int
		if err = s.db.QueryRow("SELECT count(*) FROM egress_eligible").Scan(&eligible); err != nil || eligible != 0 {
			t.Fatal("pressure published", eligible, err)
		}
	}
	if _, err := s.db.Exec("UPDATE egress_jobs SET next_run=0"); err != nil {
		t.Fatal(err)
	}
	job, err := s.ClaimEgress(ctx)
	if err != nil || job == nil {
		t.Fatal("recovery was never tested", job, err)
	}
	if err = s.CompleteEgress(ctx, egress.Completion{Job: *job, Report: syntheticEgress(*job)}); err != nil {
		t.Fatal(err)
	}
	var eligible int
	if err = s.db.QueryRow("SELECT count(*) FROM egress_eligible").Scan(&eligible); err != nil || eligible != 1 {
		t.Fatal("fresh positive did not qualify", eligible, err)
	}
}

func TestEgressPressureKeepsNetworkAssessmentBudget(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "mixed-pressure", sourceURI(278))
	for _, pressure := range []bool{false, true, false} {
		if _, err := s.db.Exec("UPDATE egress_jobs SET next_run=0"); err != nil {
			t.Fatal(err)
		}
		job, err := s.ClaimEgress(ctx)
		if err != nil || job == nil {
			t.Fatal("assessment budget lost", job, err)
		}
		report := syntheticEgress(*job)
		report.ResourcePressure, report.Samples = pressure, nil
		if err = s.CompleteEgress(ctx, egress.Completion{Job: *job, Report: report}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.db.Exec("UPDATE egress_jobs SET next_run=0"); err != nil {
		t.Fatal(err)
	}
	job, err := s.ClaimEgress(ctx)
	if err != nil || job != nil {
		t.Fatal("ordinary inconclusive network retries became unlimited", job, err)
	}
}

func TestEgressPressureRecoveryOnOpenPreservesCacheAndGuards(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	names := []string{"pressure", "network", "negative", "blocked", "archived", "excluded", "leased"}
	hashes := map[string]string{}
	for i, name := range names {
		id, _ := newExportSource(t, s, "reopen-"+name, sourceURI(281+i))
		var hash, identity string
		if err := s.db.QueryRow("SELECT u.hash,u.identity FROM egress_unique u JOIN memberships m ON m.artifact_id=u.identity WHERE m.source_id=?", id).Scan(&hash, &identity); err != nil {
			t.Fatal(err)
		}
		hashes[name] = hash
		if _, err := s.db.Exec(`UPDATE egress_jobs SET state='UNKNOWN',reason='local_resource_pressure',tested=123,attempts=9,report='{"keep":true}',next_run=456,first_completed=1,deep_completed=1,retry_streak=2 WHERE hash=?`, hash); err != nil {
			t.Fatal(err)
		}
		var err error
		switch name {
		case "network":
			_, err = s.db.Exec("UPDATE egress_jobs SET reason='controls_incomplete' WHERE hash=?", hash)
		case "negative":
			_, err = s.db.Exec("UPDATE egress_jobs SET state='NO_SPLIT',reason='non_ads_uses_ads_egress' WHERE hash=?", hash)
		case "blocked":
			_, err = s.db.Exec("INSERT INTO egress_iran_guard(identity,state,first_seen) VALUES(?,'BLOCKED',1)", identity)
		case "archived":
			_, err = s.db.Exec("DELETE FROM memberships WHERE source_id=?", id)
		case "excluded":
			_, err = s.db.Exec("INSERT INTO input_exclusions(identity,at,hash,manual) VALUES(?,1,?,1)", identity, hash)
		case "leased":
			_, err = s.db.Exec("UPDATE egress_jobs SET token='owned',lease_until=9223372036854775807 WHERE hash=?", hash)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	before, err := egressMetricState(ctx, s.db)
	if err != nil {
		t.Fatal(err)
	}
	var path string
	if err = s.db.QueryRow("SELECT file FROM pragma_database_list WHERE name='main'").Scan(&path); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path, bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	after, err := egressMetricState(ctx, s.db)
	if err != nil || before != after {
		t.Fatal("statistics changed", before, after, err)
	}
	for _, name := range names {
		var deep, retries, tested, attempts, next, first int
		var report string
		if err = s.db.QueryRow("SELECT deep_completed,retry_streak,tested,attempts,report,next_run,first_completed FROM egress_jobs WHERE hash=?", hashes[name]).Scan(&deep, &retries, &tested, &attempts, &report, &next, &first); err != nil {
			t.Fatal(err)
		}
		wantDeep, wantRetries := 1, 2
		if name == "pressure" || name == "archived" {
			wantDeep, wantRetries = 0, 0
		}
		if deep != wantDeep || retries != wantRetries || tested != 123 || attempts != 9 || report != `{"keep":true}` || next != 456 || first != 1 {
			t.Fatal("recovery modified unrelated evidence or guard", name, deep, retries, tested, attempts, report, next, first)
		}
	}
	if err = s.recoverEgressPressureBudget(); err != nil {
		t.Fatal(err)
	}
	var changes, work int
	if err = s.db.QueryRow("SELECT changes()").Scan(&changes); err != nil || changes != 0 {
		t.Fatal("recovery not idempotent", changes, err)
	}
	if err = s.db.QueryRow("SELECT count(*) FROM egress_work").Scan(&work); err != nil || work != 1 {
		t.Fatal("archive/cache/guard entered work queue", work, err)
	}
	job, err := s.ClaimEgress(ctx)
	if err != nil || job == nil || job.Hash != hashes["pressure"] {
		t.Fatal("current pressure job not recoverable", job, err)
	}
	newExportSource(t, s, "returned-archive", sourceURI(285))
	job, err = s.ClaimEgress(ctx)
	if err != nil || job == nil || job.Hash != hashes["archived"] {
		t.Fatal(fmt.Sprint("returning identity did not recover: ", job, " ", err))
	}
}
