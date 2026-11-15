package panel

import (
	"context"
	"strings"
	"testing"

	"github.com/rezajafari0970/Config_manager/internal/egress"
)

func TestEgressEligibilityLookupUpgradePreservesEvidenceAndGates(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "eligibility-upgrade", sourceURI(275))
	j, err := s.ClaimEgress(ctx)
	if err != nil || j == nil {
		t.Fatal(j, err)
	}
	if err = s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: syntheticEgress(*j)}); err != nil {
		t.Fatal(err)
	}
	var view, report, decision string
	var since, attempts, tested int64
	if err = s.db.QueryRow("SELECT sql FROM sqlite_master WHERE name='egress_eligible'").Scan(&view); err != nil {
		t.Fatal(err)
	}
	legacy := strings.Replace(view, egressInventoryLookup, egressInventorySet, 1)
	if legacy == view {
		t.Fatal("fixture did not install legacy scope lookup")
	}
	if _, err = s.db.Exec("DROP VIEW egress_eligible;" + legacy); err != nil {
		t.Fatal(err)
	}
	if err = s.db.QueryRow("SELECT report,decision,attempts,tested,(SELECT since FROM egress_metrics) FROM egress_jobs WHERE hash=?", j.Hash).Scan(&report, &decision, &attempts, &tested, &since); err != nil {
		t.Fatal(err)
	}
	for pass := 0; pass < 2; pass++ {
		if err = s.ensureEgressEligibilityLookup(); err != nil {
			t.Fatal(err)
		}
		var unchanged bool
		if err = s.db.QueryRow("SELECT report=? AND decision=? AND attempts=? AND tested=? AND (SELECT since FROM egress_metrics)=? FROM egress_jobs WHERE hash=?", report, decision, attempts, tested, since, j.Hash).Scan(&unchanged); err != nil || !unchanged {
			t.Fatal("evidence or epoch changed", err)
		}
		var eligible, blocked, schema int
		if err = s.db.QueryRow("SELECT (SELECT count(*) FROM egress_eligible),(SELECT count(*) FROM input_blocked WHERE hash=?)", j.Hash).Scan(&eligible, &blocked); err != nil || eligible != 1 || blocked != 0 {
			t.Fatal("qualified member changed", eligible, blocked, err)
		}
		if err = s.db.QueryRow("PRAGMA user_version").Scan(&schema); err != nil || schema != 37 {
			t.Fatal("schema changed", schema, err)
		}
	}
	if _, err = s.db.Exec("UPDATE egress_jobs SET valid_until=1 WHERE hash=?", j.Hash); err != nil {
		t.Fatal(err)
	}
	var eligible, blocked int
	if err = s.db.QueryRow("SELECT (SELECT count(*) FROM egress_eligible),(SELECT count(*) FROM input_blocked WHERE hash=?)", j.Hash).Scan(&eligible, &blocked); err != nil || eligible != 0 || blocked != 1 {
		t.Fatal("expired proof bypassed current gates", eligible, blocked, err)
	}
}
