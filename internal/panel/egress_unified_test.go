package panel

import (
	"context"
	"github.com/rezajafari0970/Config_manager/internal/egress"
	"strings"
	"testing"
)

func badDelivery(j egress.Job) egress.Report {
	r := syntheticEgress(j)
	host := r.Delivery[0]
	host.Assets = append([]egress.DeliveryAsset(nil), host.Assets...)
	host.HostControl = true
	host.Variant = "host"
	for i := range host.Assets {
		host.Assets[i].Nonce = egress.Hash([]byte("host" + host.Assets[i].Profile))[:32]
	}
	for i := range r.Delivery {
		for k := range r.Delivery[i].Assets {
			r.Delivery[i].Assets[k].CurlExit = 28
			r.Delivery[i].Assets[k].HTTPStatus = 0
			r.Delivery[i].Assets[k].TLSVerified = false
			r.Delivery[i].Assets[k].Complete = false
			r.Delivery[i].Assets[k].ContentMatch = false
			r.Delivery[i].Assets[k].Bytes = 0
			r.Delivery[i].Assets[k].BodySHA = ""
		}
	}
	r.Delivery = append(r.Delivery, host)
	return r
}
func TestUnifiedDeliveryRevokesAllAliasesAndBlocksReimport(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	raw := sourceURI(731)
	newExportSource(t, s, "one", raw)
	newExportSource(t, s, "two", raw+"-cosmetic")
	j, err := s.ClaimEgress(ctx)
	if err != nil || j == nil {
		t.Fatal(err)
	}
	hash := j.Hash
	for step := 0; step < 3; step++ {
		if step > 0 {
			s.db.Exec("UPDATE egress_retirements SET at=? WHERE hash=?", egress.Now()-60000, hash)
			s.db.Exec("UPDATE egress_jobs SET next_run=0 WHERE hash=?", hash)
			j, err = s.ClaimEgress(ctx)
			if err != nil || j == nil || j.Hash != hash {
				t.Fatal("negative confirmation lost", j, err)
			}
		}
		if err = s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: badDelivery(*j)}); err != nil {
			t.Fatal(err)
		}
		var published int
		s.db.QueryRow("SELECT count(*) FROM egress_eligible").Scan(&published)
		var blocked int
		s.db.QueryRow("SELECT count(*) FROM input_blocked WHERE hash=?", hash).Scan(&blocked)
		if blocked != 1 {
			t.Fatal("Ads failure did not block all outputs", blocked)
		}
		if published != 0 {
			t.Fatal("bad Ads assets published")
		}
	}
	var excluded, members, live, runnable, responses int
	s.db.QueryRow("SELECT count(*) FROM input_exclusions").Scan(&excluded)
	s.db.QueryRow("SELECT count(*) FROM memberships").Scan(&members)
	s.db.QueryRow("SELECT count(*) FROM responses").Scan(&responses)
	s.db.QueryRow("SELECT count(*) FROM egress_jobs WHERE state!='DELETED'").Scan(&live)
	s.db.QueryRow("SELECT count(*) FROM egress_jobs WHERE candidate_hash!='' OR report!='{}'").Scan(&runnable)
	if excluded != 1 || members != 0 || live != 0 || runnable != 0 || responses != 0 {
		t.Fatal(excluded, members, live, runnable, responses)
	}
	newExportSource(t, s, "three", raw+"-again")
	if j, err = s.ClaimEgress(ctx); err != nil || j != nil {
		t.Fatal("revoked identity returned", j, err)
	}
}
func TestUnifiedRetirementNeedsIndependentSpacedRoundsAndRecoveryClears(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "one", sourceURI(732))
	j, _ := s.ClaimEgress(ctx)
	tx, _ := s.db.Begin()
	defer tx.Rollback()
	now := egress.Now()
	for i := 0; i < 4; i++ {
		removed, hits, err := s.retirement(ctx, tx, j.Hash, "confirmed_ads_route_lost", false, now, now, now)
		if err != nil || removed || hits != 1 {
			t.Fatal(removed, hits, err)
		}
	}
	if _, hits, err := s.retirement(ctx, tx, j.Hash, "", true, now, now, now); err != nil || hits != 0 {
		t.Fatal(hits, err)
	}
	if _, hits, err := s.retirement(ctx, tx, j.Hash, "", false, now, now, now); err != nil || hits != 0 {
		t.Fatal("unknown created negative", hits, err)
	}
}
func TestUnifiedResetIsOneTimeAndPreservesSourcePolicyAndExclusions(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "reset", sourceURI(733))
	j, _ := s.ClaimEgress(ctx)
	if err := s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: syntheticEgress(*j)}); err != nil {
		t.Fatal(err)
	}
	s.db.Exec("INSERT INTO input_exclusions VALUES('fixture-exclusion',1,'fixture-hash',1)")
	var sources string
	s.db.QueryRow("SELECT group_concat(id||':'||revision) FROM sources").Scan(&sources)
	var blockedView string
	s.db.QueryRow("SELECT sql FROM sqlite_master WHERE name='input_blocked'").Scan(&blockedView)
	blockedView = strings.ReplaceAll(blockedView, " UNION SELECT hash FROM egress_retirements UNION SELECT hash FROM egress_jobs WHERE json_extract(decision,'$.country_conflict')=1", "")
	if _, err := s.db.Exec("DROP VIEW input_blocked;" + blockedView); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec("DROP VIEW egress_work;CREATE VIEW egress_work AS SELECT * FROM egress_inventory WHERE tested=0 OR ever_accepted=1"); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{"DROP TABLE egress_retirements", "ALTER TABLE egress_run_members DROP COLUMN settled_at", "ALTER TABLE egress_run_members DROP COLUMN settled_state", "PRAGMA user_version=35"} {
		if _, err := s.db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.migrateV36(); err != nil {
		t.Fatal(err)
	}
	var tested, attempts, completed, excluded int
	var since int64
	var after string
	s.db.QueryRow("SELECT tested,attempts FROM egress_jobs WHERE hash=?", j.Hash).Scan(&tested, &attempts)
	s.db.QueryRow("SELECT completed,since FROM egress_metrics").Scan(&completed, &since)
	s.db.QueryRow("SELECT count(*) FROM input_exclusions").Scan(&excluded)
	s.db.QueryRow("SELECT group_concat(id||':'||revision) FROM sources").Scan(&after)
	if tested != 0 || attempts != 0 || completed != 0 || excluded != 1 || after != sources {
		t.Fatal(tested, attempts, completed, excluded)
	}
	if err := s.migrateV35(); err != nil {
		t.Fatal(err)
	}
	if err := s.migrateV36(); err != nil {
		t.Fatal(err)
	}
	var v int
	var again int64
	s.db.QueryRow("PRAGMA user_version").Scan(&v)
	s.db.QueryRow("SELECT since FROM egress_metrics").Scan(&again)
	if v != 36 || again != since {
		t.Fatal("restart repeated reset", v, again, since)
	}
}
func TestSettledRescanResultDoesNotOverwriteFirstAssessment(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "settled", sourceURI(734))
	j, _ := s.ClaimEgress(ctx)
	run, err := s.ResetEgress(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	var started int64
	s.db.QueryRow("SELECT started FROM egress_runs WHERE id=?", run).Scan(&started)
	tx, _ := s.db.Begin()
	defer tx.Rollback()
	if err = recordEgressRun(ctx, tx, j.Hash, "UNKNOWN", false, started, started+100, 100); err != nil {
		t.Fatal(err)
	}
	if err = recordEgressRun(ctx, tx, j.Hash, "NO_SPLIT", true, started+200, started+300, 100); err != nil {
		t.Fatal(err)
	}
	var first, settled int64
	var state string
	tx.QueryRow("SELECT completed_at,settled_at,state FROM egress_run_members WHERE hash=?", j.Hash).Scan(&first, &settled, &state)
	if first != started+100 || settled != started+300 || state != "UNKNOWN" {
		t.Fatal(first, settled, state)
	}
}

func TestUnifiedConflictWithdrawalUsesSparseIndex(t *testing.T) {
	s := testStore(t)
	rows, err := s.db.Query("EXPLAIN QUERY PLAN SELECT hash FROM egress_jobs WHERE json_extract(decision,'$.country_conflict')=1")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	used := false
	for rows.Next() {
		var id, parent, unused int
		var detail string
		if err = rows.Scan(&id, &parent, &unused, &detail); err != nil {
			t.Fatal(err)
		}
		used = used || strings.Contains(detail, "egress_iran_conflicts")
	}
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !used {
		t.Fatal("global withdrawal scans archived decision bodies")
	}
}

func TestUnifiedUnprovedDeliveryRemainsRecoverableWithoutReimport(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "asset-recovery", sourceURI(735))
	var hash string
	for i := 0; i < 2; i++ {
		j, err := s.ClaimEgress(ctx)
		if err != nil || j == nil {
			t.Fatal(j, err)
		}
		hash = j.Hash
		r := syntheticEgress(*j)
		r.Delivery = nil
		if err = s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: r}); err != nil {
			t.Fatal(err)
		}
		if _, err = s.db.Exec("UPDATE egress_jobs SET next_run=0 WHERE hash=?", hash); err != nil {
			t.Fatal(err)
		}
	}
	var work int
	if err := s.db.QueryRow("SELECT count(*) FROM egress_work WHERE hash=?", hash).Scan(&work); err != nil || work != 1 {
		t.Fatal("asset outage became permanently forgotten", work, err)
	}
	j, err := s.ClaimEgress(ctx)
	if err != nil || j == nil || j.Hash != hash {
		t.Fatal(j, err)
	}
	if err = s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: syntheticEgress(*j)}); err != nil {
		t.Fatal(err)
	}
	var eligible int
	s.db.QueryRow("SELECT count(*) FROM egress_eligible WHERE hash=?", hash).Scan(&eligible)
	if eligible != 1 {
		t.Fatal("asset recovery did not restore eligibility")
	}
}
