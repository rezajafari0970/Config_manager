package panel

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestResidentialFixedLifetimeAndHealthCacheAreIndependent(t *testing.T) {
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
	// Expiry at read-time must win even before scheduled cleanup runs.
	if _, e := s.db.Exec("UPDATE residential_records SET retain_until=? WHERE hash=?", time.Now().Add(-time.Second).UnixMilli(), j.Hash); e != nil {
		t.Fatal(e)
	}
	for _, f := range []string{"configs.jsonl", "next.json", "raw.txt"} {
		if w := resRequest(s, "all", f); w.Code != 503 {
			t.Fatal(f, w.Code)
		}
	}
	if e := s.Expire(ctx, time.Now()); e != nil {
		t.Fatal(e)
	}
	var state string
	s.db.QueryRow("SELECT state FROM residential_records WHERE hash=?", j.Hash).Scan(&state)
	if state != "DELETED" {
		t.Fatal(state)
	}
	if e := s.CompleteResidential(ctx, resCompletion(*j)); !errors.Is(e, ErrConflict) {
		t.Fatal("stale completion after expiry", e)
	}
	if used, e := s.Referenced(ctx, j.Hash); e != nil || used {
		t.Fatal("expired original retained", used, e)
	}
	// A later increase cannot resurrect the expired record or its old proof.
	cfg.RetentionSeconds = 72000
	if e := s.SaveSettings(ctx, cfg); e != nil {
		t.Fatal(e)
	}
	if w := resRequest(s, "all", "raw.txt"); w.Code != 503 {
		t.Fatal("expired proof resurrected", w.Code)
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
	if after != before-18000000 {
		t.Fatal("shorter setting not applied")
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
