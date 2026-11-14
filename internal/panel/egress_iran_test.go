package panel

import (
	"context"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/egress"
	"path/filepath"
	"strings"
	"testing"
)

func iranFixture(j egress.Job) egress.Report {
	r := syntheticEgress(j)
	// One observation, in one domain, in one variant is enough for quarantine.
	s := r.Samples[0]
	s.Geo.Country = "IR"
	for i := range s.Geo.Providers {
		s.Geo.Providers[i].Country = "IR"
	}
	r.Samples = []egress.Sample{s}
	r.Checks = r.Checks[:1]
	r.Health = nil
	return r
}
func TestIranEvidenceIndependentAndFailClosed(t *testing.T) {
	s := testStore(t)
	newExportSource(t, s, "iran", sourceURI(560))
	j, _ := s.ClaimEgress(context.Background())
	p := egress.Prepare(j.Raw)
	r := iranFixture(*j)
	if egress.Evaluate(r, p, j.Hash, egress.Now()).Eligible {
		t.Fatal("incomplete route fixture qualified")
	}
	if egress.Iran(r, p, j.Hash, egress.Now()) == nil {
		t.Fatal("partial authenticated IR hidden")
	}
	cases := map[string]func(*egress.Report){
		"control":          func(r *egress.Report) { r.Samples[0].ActualAds = false },
		"other_domain":     func(r *egress.Report) { r.Samples[0].Domain = "example.com" },
		"tls":              func(r *egress.Report) { r.Samples[0].TLS = false },
		"different_stream": func(r *egress.Report) { r.Samples[0].SameStream = false },
		"no_echo":          func(r *egress.Report) { r.Samples[0].Echo = false },
		"binding":          func(r *egress.Report) { r.Hash = strings.Repeat("0", 64) },
		"stale":            func(r *egress.Report) { r.Started -= 600001 },
		"core":             func(r *egress.Report) { r.CoreSHA = strings.Repeat("0", 64) },
		"variant":          func(r *egress.Report) { r.Checks[0].Hash = strings.Repeat("0", 64) },
		"oom":              func(r *egress.Report) { r.ResourcePressure = true },
		"one_provider":     func(r *egress.Report) { r.Samples[0].Geo.Providers = r.Samples[0].Geo.Providers[:1] },
		"same_lineage":     func(r *egress.Report) { r.Samples[0].Geo.Providers[1].Lineage = r.Samples[0].Geo.Providers[0].Lineage },
		"conflict":         func(r *egress.Report) { r.Samples[0].Geo.Providers[1].Country = "CA" },
		"expired":          func(r *egress.Report) { r.Samples[0].Geo.Providers[0].ReleasedAt = 1 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			b, _ := json.Marshal(r)
			var x egress.Report
			json.Unmarshal(b, &x)
			mutate(&x)
			if egress.Iran(x, p, j.Hash, egress.Now()) != nil {
				t.Fatal("invalid Iran evidence accepted")
			}
		})
	}
	// A successful template observation still counts when raw failed.
	r = iranFixture(*j)
	r.Checks = []egress.Check{syntheticEgress(*j).Checks[1]}
	r.Samples[0].Variant = "template"
	if egress.Iran(r, p, j.Hash, egress.Now()) == nil {
		t.Fatal("template IR ignored")
	}
}

func TestIranQuarantineAllExportsAndStickyRoundConfirmation(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "guard", sourceURI(561))
	j, _ := s.ClaimEgress(ctx)
	if e := s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: syntheticEgress(*j)}); e != nil {
		t.Fatal(e)
	}
	app, e := NewServer(s, "http://127.0.0.1:18191")
	if e != nil {
		t.Fatal(e)
	}
	base := app.origin + "/egress-subscription/" + s.digest("egress-export-v2", "all") + "/"
	next := func() *egress.Job {
		t.Helper()
		s.db.Exec("UPDATE egress_jobs SET next_run=0 WHERE hash=?", j.Hash)
		x, e := s.ClaimEgress(ctx)
		if e != nil || x == nil {
			t.Fatal(e)
		}
		return x
	}
	finish := func(x *egress.Job, r egress.Report) {
		t.Helper()
		if e := s.CompleteEgress(ctx, egress.Completion{Job: *x, Report: r}); e != nil {
			t.Fatal(e)
		}
	}
	j = next()
	r := iranFixture(*j)
	finish(j, r)
	if e := s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: r}); e != ErrConflict {
		t.Fatal("duplicate completion counted", e)
	}
	checkBlocked := func() {
		t.Helper()
		for _, f := range []string{"configs.json", "configs.jsonl", "next.json", "raw.txt"} {
			rr := exportRequest(t, app.Handler(), "GET", base+f)
			if rr.Code != 503 {
				t.Fatal("IR leaked", f, rr.Code)
			}
		}
		for _, view := range []string{"egress_eligible", "egress_export_eligible", "conversion_inputs", "smart_inputs", "splash_smart_inputs"} {
			var n int
			if e := s.db.QueryRow("SELECT count(*) FROM " + view).Scan(&n); e != nil || n != 0 {
				t.Fatal(view, n, e)
			}
		}
	}
	checkBlocked()
	var positives int
	var state string
	var tested, due int64
	s.db.QueryRow("SELECT state,positives FROM egress_iran_guard").Scan(&state, &positives)
	s.db.QueryRow("SELECT tested,next_run FROM egress_jobs WHERE hash=?", j.Hash).Scan(&tested, &due)
	if state != "QUARANTINED" || positives != 1 || due-tested != 60000 {
		t.Fatal(state, positives, due-tested)
	}
	// Early replay in a fresh lease must not confirm. A clean round cannot clear.
	j = next()
	finish(j, iranFixture(*j))
	s.db.QueryRow("SELECT positives FROM egress_iran_guard").Scan(&positives)
	if positives != 1 {
		t.Fatal("unspaced duplicate confirmed")
	}
	j = next()
	finish(j, syntheticEgress(*j))
	checkBlocked()
	// Rename and refetch while quarantined: identity gate applies before probing.
	newExportSource(t, s, "renamed", strings.Split(sourceURI(561), "#")[0]+"#renamed")
	checkBlocked()
	// Established aliases must not pull one another back to 60s forever.
	s.db.Exec("UPDATE egress_jobs SET state='QUARANTINED',reason='ads_iran_quarantined',valid_until=0,next_run=? WHERE hash!=?", egress.Now()+500000, j.Hash)
	j = next()
	finish(j, syntheticEgress(*j))
	var aliasDue int64
	s.db.QueryRow("SELECT min(next_run) FROM egress_jobs WHERE hash!=?", j.Hash).Scan(&aliasDue)
	if aliasDue < egress.Now()+400000 {
		t.Fatal("alias retry storm", aliasDue)
	}
	// Simulate passage of spacing in durable state without sleeping in tests.
	s.db.Exec("UPDATE egress_iran_guard SET last_round=last_round-61000")
	j = next()
	finish(j, iranFixture(*j))
	s.db.QueryRow("SELECT state,positives FROM egress_iran_guard").Scan(&state, &positives)
	if state != "BLOCKED" || positives != 2 {
		t.Fatal(state, positives)
	}
	var remaining int
	s.db.QueryRow("SELECT count(*) FROM egress_jobs WHERE state!='DELETED'").Scan(&remaining)
	if remaining != 0 {
		t.Fatal("alias survived global removal", remaining)
	}
	newExportSource(t, s, "refetched", strings.Split(sourceURI(561), "#")[0]+"#again")
	s.db.QueryRow("SELECT count(*) FROM artifacts").Scan(&remaining)
	if remaining != 0 {
		t.Fatal("tombstone bypass", remaining)
	}
	checkBlocked()
}

func TestIranUnknownDoesNotDeleteAndCountryAnyVeto(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "mixed", sourceURI(562))
	j, _ := s.ClaimEgress(ctx)
	r := syntheticEgress(*j)
	for i := range r.Samples {
		if !r.Samples[i].ActualAds {
			continue
		}
		c := "TW"
		if i == 0 {
			c = "IR"
		}
		r.Samples[i].Geo.Country = c
		for k := range r.Samples[i].Geo.Providers {
			r.Samples[i].Geo.Providers[k].Country = c
		}
	}
	s.db.Exec(`UPDATE egress_country_policy SET allowed='["TW","IR"]'`)
	if e := s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: r}); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 4; i++ {
		s.db.Exec("UPDATE egress_iran_guard SET last_round=last_round-61000")
		s.db.Exec("UPDATE egress_jobs SET next_run=0 WHERE hash=?", j.Hash)
		j, _ = s.ClaimEgress(ctx)
		r = iranFixture(*j)
		r.ResourcePressure = true
		if e := s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: r}); e != nil {
			t.Fatal(e)
		}
	}
	var state string
	var positives, n int
	s.db.QueryRow("SELECT state,positives FROM egress_iran_guard").Scan(&state, &positives)
	s.db.QueryRow("SELECT count(*) FROM egress_export_eligible").Scan(&n)
	if state != "QUARANTINED" || positives != 1 || n != 0 {
		t.Fatal(state, positives, n)
	}
	var due, tested int64
	s.db.QueryRow("SELECT next_run,tested FROM egress_jobs WHERE hash=?", j.Hash).Scan(&due, &tested)
	if due-tested > 600000 {
		t.Fatal("guard retry exceeds ten minutes")
	}
}

func TestIranGuardSurvivesRestartAndExpiredLeaseRecovery(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "restart-guard", sourceURI(563))
	j, _ := s.ClaimEgress(ctx)
	if e := s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: iranFixture(*j)}); e != nil {
		t.Fatal(e)
	}
	s.db.Exec("UPDATE egress_jobs SET next_run=0 WHERE hash=?", j.Hash)
	old, _ := s.ClaimEgress(ctx)
	s.db.Exec("UPDATE egress_jobs SET lease_until=1 WHERE hash=?", j.Hash)
	path := filepath.Join(filepath.Dir(s.vault.Directory()), "panel.db")
	key := s.key
	s.Close()
	reopened, e := Open(path, key)
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.Close()
	j, e = reopened.ClaimEgress(ctx)
	if e != nil || j == nil || j.Token == old.Token {
		t.Fatal("lost restart lease", e)
	}
	if e = reopened.CompleteEgress(ctx, egress.Completion{Job: *old, Report: iranFixture(*old)}); e != ErrConflict {
		t.Fatal("stale worker changed guard", e)
	}
	var n int
	reopened.db.QueryRow("SELECT positives FROM egress_iran_guard").Scan(&n)
	if n != 1 {
		t.Fatal("lost durable signal")
	}
	reopened.db.Exec("UPDATE egress_iran_guard SET last_round=last_round-61000")
	if e = reopened.CompleteEgress(ctx, egress.Completion{Job: *j, Report: iranFixture(*j)}); e != nil {
		t.Fatal(e)
	}
	reopened.Close()
	reopened, e = Open(path, key)
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.Close()
	var state string
	reopened.db.QueryRow("SELECT state FROM egress_iran_guard").Scan(&state)
	if state != "BLOCKED" {
		t.Fatal("lost tombstone")
	}
	newExportSource(t, reopened, "after-restart", strings.Split(sourceURI(563), "#")[0]+"#changed")
	reopened.db.QueryRow("SELECT count(*) FROM artifacts").Scan(&n)
	if n != 0 {
		t.Fatal("restart restored revoked input")
	}
}

func TestIranMigrationPreservesPolicyAndQuarantinesHistoricalIR(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "migration", sourceURI(564))
	j, _ := s.ClaimEgress(ctx)
	if e := s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: syntheticEgress(*j)}); e != nil {
		t.Fatal(e)
	}
	s.db.Exec(`UPDATE egress_country_policy SET allowed='["TW","CY"]',revision=8`)
	s.db.Exec("INSERT INTO egress_countries VALUES(?,'IR',?,?)", j.Hash, egress.Now(), egress.Now())
	var view string
	s.db.QueryRow("SELECT sql FROM sqlite_master WHERE name='egress_eligible'").Scan(&view)
	cut := strings.Index(view, " AND j.tested>=(SELECT started FROM egress_iran_policy")
	if cut < 0 {
		t.Fatal("missing guard view")
	}
	_, e := s.db.Exec("DROP VIEW egress_eligible;" + view[:cut] + ";DROP TABLE egress_iran_guard;PRAGMA user_version=30;")
	if e != nil {
		t.Fatal(e)
	}
	if e = s.migrateV31(); e != nil {
		t.Fatal(e)
	}
	var state, allowed string
	var positives, rev, n int
	s.db.QueryRow("SELECT state,positives FROM egress_iran_guard").Scan(&state, &positives)
	s.db.QueryRow("SELECT allowed,revision FROM egress_country_policy").Scan(&allowed, &rev)
	s.db.QueryRow("SELECT count(*) FROM egress_eligible").Scan(&n)
	if state != "QUARANTINED" || positives != 0 || rev != 8 || allowed != `["TW","CY"]` || n != 0 {
		t.Fatal(state, positives, allowed, rev, n)
	}
	if e = s.migrateV31(); e != nil {
		t.Fatal("not idempotent", e)
	}
}

func TestIranQuarantineSurvivesUnrelatedGeneralHealthFailures(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "unhealthy-guard", sourceURI(565))
	j, _ := s.ClaimEgress(ctx)
	if e := s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: syntheticEgress(*j)}); e != nil {
		t.Fatal(e)
	}
	s.db.Exec("UPDATE egress_jobs SET next_run=0 WHERE hash=?", j.Hash)
	j, _ = s.ClaimEgress(ctx)
	if e := s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: iranFixture(*j)}); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 3; i++ {
		s.db.Exec("UPDATE egress_jobs SET next_run=0,last_failure=0 WHERE hash=?", j.Hash)
		j, _ = s.ClaimEgress(ctx)
		r := syntheticEgress(*j)
		host := r.Health[0]
		host.Variant = "host"
		host.HostControl = true
		host.Providers = append([]egress.HealthProvider(nil), host.Providers...)
		for k := range r.Health {
			for m := range r.Health[k].Providers {
				r.Health[k].Providers[m].Download = egress.Transfer{CurlExit: 28}
				r.Health[k].Providers[m].Upload = egress.Transfer{CurlExit: 28}
			}
		}
		r.Health = append(r.Health, host)
		if e := s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: r}); e != nil {
			t.Fatal(e)
		}
	}
	var state, reason string
	s.db.QueryRow("SELECT state,reason FROM egress_jobs WHERE hash=?", j.Hash).Scan(&state, &reason)
	if state != "QUARANTINED" || reason != "ads_iran_quarantined" {
		t.Fatal("generic health improperly changed Iran quarantine", state, reason)
	}
	var positives int
	s.db.QueryRow("SELECT positives FROM egress_iran_guard").Scan(&positives)
	if positives != 1 {
		t.Fatal("health failure counted as IR")
	}
}

func TestIranRetainsEarlierAttemptEvidence(t *testing.T) {
	s := testStore(t)
	newExportSource(t, s, "earlier-ir", sourceURI(580))
	j, _ := s.ClaimEgress(context.Background())
	p := egress.Prepare(j.Raw)
	r := syntheticEgress(*j)
	old := iranFixture(*j)
	old.Started = r.Started
	old.Finished = r.Finished
	r.Evidence = []egress.Report{old}
	if egress.Iran(r, p, j.Hash, egress.Now()) == nil {
		t.Fatal("clean selected attempt hid earlier IR")
	}
	r.Evidence[0].CoreSHA = "invalid"
	if egress.Iran(r, p, j.Hash, egress.Now()) != nil {
		t.Fatal("unbound attempt counted")
	}
}
func TestCountryMinorityIranBlocksPublicationWithoutConfirmedIR(t *testing.T) {
	s := testStore(t)
	newExportSource(t, s, "conflict-ir", sourceURI(581))
	j, _ := s.ClaimEgress(context.Background())
	p := egress.Prepare(j.Raw)
	r := syntheticEgress(*j)
	for i := range r.Samples {
		if r.Samples[i].ActualAds {
			g := &r.Samples[i].Geo
			x := g.Providers[0]
			x.Name = "independent-ir"
			x.Lineage = "independent-ir"
			x.Country = "IR"
			g.Providers = append(g.Providers, x)
		}
	}
	d := egress.Evaluate(r, p, j.Hash, egress.Now())
	c := egress.Countries(r, d, egress.Now())
	if c.Complete || c.Status != "country_iran_conflict" {
		t.Fatal(c)
	}
	if egress.Iran(r, p, j.Hash, egress.Now()) != nil {
		t.Fatal("minority report promoted to confirmed IR")
	}
}
func TestV32VersionInvalidatesProofWithoutErasingHistory(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "version-history", sourceURI(582))
	j, _ := s.ClaimEgress(ctx)
	if e := s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: syntheticEgress(*j)}); e != nil {
		t.Fatal(e)
	}
	var beforeReport, beforeDecision string
	var tested, attempts int64
	s.db.QueryRow("SELECT report,decision,tested,attempts FROM egress_jobs WHERE hash=?", j.Hash).Scan(&beforeReport, &beforeDecision, &tested, &attempts)
	s.db.Exec("UPDATE egress_settings SET version='ads-egress.v5|old'")
	if e := s.reconcileEgressVersion(); e != nil {
		t.Fatal(e)
	}
	var report, decision, state string
	var afterTested, afterAttempts, until, healthyUntil int64
	s.db.QueryRow("SELECT report,decision,tested,attempts,valid_until,state FROM egress_jobs WHERE hash=?", j.Hash).Scan(&report, &decision, &afterTested, &afterAttempts, &until, &state)
	s.db.QueryRow("SELECT valid_until FROM egress_health WHERE hash=?", j.Hash).Scan(&healthyUntil)
	if report != beforeReport || decision != beforeDecision || tested <= 0 || afterTested != 0 || attempts != afterAttempts || until != 0 || healthyUntil != 0 || state != "QUARANTINED" {
		t.Fatal("history lost or old proof eligible")
	}
	var history int
	s.db.QueryRow("SELECT count(*) FROM egress_events WHERE hash=? AND at=?", j.Hash, tested).Scan(&history)
	if history != 1 {
		t.Fatal("old assessment history lost", history)
	}
}
