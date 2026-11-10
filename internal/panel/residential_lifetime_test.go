package panel

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestResidentialAgeExemptionAndHealthCacheAreIndependent(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	resPolicy(t, s)
	cfg, _ := s.Settings(ctx)
	cfg.RetentionSeconds = 36000
	if e := s.SaveSettings(ctx, cfg); e != nil {
		t.Fatal(e)
	}
	health, e := s.ProbeSettings(ctx)
	if e != nil {
		t.Fatal(e)
	}
	health.HealthTTLSeconds = 36000
	if e = s.SaveProbeSettings(ctx, health); e != nil {
		t.Fatal(e)
	}
	policy, _, e := residentialSettings(ctx, s.db)
	if e != nil || policy.EvidenceTTLSeconds >= 36000 {
		t.Fatal("health cache extended Ads evidence", e)
	}
	_, j, _ := resSeed(t, s, 1)
	var first, until int64
	if e := s.db.QueryRow("SELECT first_seen,retain_until FROM residential_records WHERE hash=?", j.Hash).Scan(&first, &until); e != nil {
		t.Fatal(e)
	}
	if until-first != 36000000 {
		t.Fatal("not a ten-hour lifetime", until-first)
	}
	// Accepted configuration age is independent of evidence freshness.
	if _, e = s.db.Exec("UPDATE residential_records SET retain_until=1,first_seen=first_seen-108000000 WHERE hash=?", j.Hash); e != nil {
		t.Fatal(e)
	}
	if e = s.Expire(ctx, time.Now()); e != nil {
		t.Fatal(e)
	}
	for _, f := range []string{"configs.jsonl", "next.json", "raw.txt"} {
		if w := resRequest(s, "all", f); w.Code != 200 {
			t.Fatal(f, w.Code)
		}
	}
	if used, e := s.Referenced(ctx, j.Hash); e != nil || !used {
		t.Fatal("accepted original lost", used, e)
	}
	s.db.Exec("UPDATE residential_records SET valid_until=0 WHERE hash=?", j.Hash)
	for _, f := range []string{"configs.jsonl", "next.json", "raw.txt"} {
		if w := resRequest(s, "all", f); w.Code != 503 {
			t.Fatal("expired evidence published", f, w.Code)
		}
	}
	if e = s.Expire(ctx, time.Now()); e != nil {
		t.Fatal(e)
	}
	var state string
	s.db.QueryRow("SELECT state FROM residential_records WHERE hash=?", j.Hash).Scan(&state)
	if state != "QUARANTINED" {
		t.Fatal("expired evidence deleted retained member", state)
	}
	s.db.Exec("UPDATE residential_records SET next_run=0 WHERE hash=?", j.Hash)
	fresh, e := s.ClaimResidential(ctx)
	if e != nil || fresh == nil {
		t.Fatal("old retained record not retested", e)
	}
	if e = s.CompleteResidential(ctx, resCompletion(*fresh)); e != nil {
		t.Fatal(e)
	}
	if w := resRequest(s, "all", "raw.txt"); w.Code != 200 {
		t.Fatal("fresh proof not restored", w.Code)
	}

}

func TestResidentialLifetimeDoesNotRenewOnRetestOrSettingIncrease(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	resPolicy(t, s)
	cfg, _ := s.Settings(ctx)
	cfg.RetentionSeconds = 36000
	s.SaveSettings(ctx, cfg)
	_, j, _ := resSeed(t, s, 1)
	var before, after int64
	s.db.QueryRow("SELECT retain_until FROM residential_records WHERE hash=?", j.Hash).Scan(&before)
	s.db.Exec("UPDATE residential_records SET next_run=0 WHERE hash=?", j.Hash)
	fresh, e := s.ClaimResidential(ctx)
	if e != nil || fresh == nil {
		t.Fatal(e)
	}
	if e = s.CompleteResidential(ctx, resCompletion(*fresh)); e != nil {
		t.Fatal(e)
	}
	cfg.RetentionSeconds = 72000
	if e = s.SaveSettings(ctx, cfg); e != nil {
		t.Fatal(e)
	}
	s.db.QueryRow("SELECT retain_until FROM residential_records WHERE hash=?", j.Hash).Scan(&after)
	if after != before {
		t.Fatal("lifetime was extended")
	}
	cfg.RetentionSeconds = 18000
	s.SaveSettings(ctx, cfg)
	s.db.QueryRow("SELECT retain_until FROM residential_records WHERE hash=?", j.Hash).Scan(&after)
	if after != before {
		t.Fatal("general setting changed exempt lifetime")
	}
}

func TestResidentialOldZIPLinkRedirectsToPrintedRaw(t *testing.T) {
	s := testStore(t)
	server, _ := NewServer(s, "http://127.0.0.1:18191")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, httptest.NewRequest("GET", server.residentialURL("all", "raw.zip"), nil))
	if w.Code != http.StatusPermanentRedirect || w.Header().Get("Location") != server.residentialURL("all", "raw.txt") {
		t.Fatal(w.Code, w.Header())
	}
	if w.Header().Get("Content-Type") == "application/zip" {
		t.Fatal("ZIP still delivered")
	}
}

func TestResidentialFirstAssessmentCannotExpireUnexamined(t *testing.T) {
	s := testStore(t)
	resPolicy(t, s)
	ctx := context.Background()
	source, _ := addConversion(t, s, conversionSample)
	s.db.Exec("UPDATE residential_records SET retain_until=1;UPDATE artifacts SET expires=1")
	if e := s.Expire(ctx, time.Now()); e != nil {
		t.Fatal(e)
	}
	if e := s.Delete(ctx, source, 1); e != nil {
		t.Fatal(e)
	}
	if e := s.vault.Sweep(ctx, s.Referenced); e != nil {
		t.Fatal(e)
	}
	j, e := s.ClaimResidential(ctx)
	if e != nil || j == nil || len(j.Raw) == 0 {
		t.Fatal("untested input discarded", e)
	}
	if e = s.RenewResidential(ctx, *j); e != nil {
		t.Fatal(e)
	}
	c := resCompletion(*j)
	c.Report.IPFacts = nil
	if e = s.CompleteResidential(ctx, c); e != nil {
		t.Fatal(e)
	}
	if e = s.Expire(ctx, time.Now()); e != nil {
		t.Fatal(e)
	}
	var state string
	s.db.QueryRow("SELECT state FROM residential_records WHERE hash=?", j.Hash).Scan(&state)
	if state != "DELETED" {
		t.Fatal("general TTL should apply after first unqualified assessment", state)
	}
}
