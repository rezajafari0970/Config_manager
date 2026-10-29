package panel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/rezajafari0970/Config_manager/internal/collect"
	"github.com/rezajafari0970/Config_manager/internal/probe"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const sampleVLESS = "vless://00000000-0000-4000-8000-000000000001@192.0.2.10:443?security=tls&sni=example.com#synthetic"

func addBank(t *testing.T, s *Store, raw string) probe.Job {
	t.Helper()
	ctx := context.Background()
	if _, e := s.ImportBank(ctx, strings.NewReader(raw)); e != nil {
		t.Fatal(e)
	}
	j, e := s.ClaimProbe(ctx)
	if e != nil || j == nil {
		t.Fatal("claim", e)
	}
	return *j
}
func passedResult(j probe.Job) probe.Result {
	p := probe.Prepare(j.Raw)
	c := p.Candidates[0]
	a := probe.Attempt{SHA256: c.SHA256, Rule: c.Rule, Core: "passed", Connection: "passed", Code: "https_proxy_verified", RuntimeSHA256: c.SHA256, EndpointsPassed: 2}
	return probe.Result{Hash: j.Hash, Token: j.Token, Revision: j.Revision, Version: probe.Version, CoreSHA256: probe.CoreSHA256, Model: probe.Model(p), Attempt: a, Candidate: c.Config, Attempts: []probe.Attempt{a}}
}
func TestBankPreservesSamplesAcrossSourceGenerationsAndExpiry(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	id, e := s.Save(ctx, "", 0, testSource())
	if e != nil {
		t.Fatal(e)
	}
	j := claimOne(t, s, id)
	raw := []byte(sampleVLESS)
	ref, e := s.vault.Put(ctx, bytes.NewReader(raw))
	if e != nil {
		t.Fatal(e)
	}
	defer s.vault.Release(ref.Hash)
	item := collect.Item{Ref: ref, Identity: ref.Hash, Kind: "vless", Quality: "valid"}
	if _, e = s.Stage(ctx, j, item); e != nil {
		t.Fatal(e)
	}
	if rows, e := s.Bank(ctx, "", "", 0); e != nil || len(rows) != 0 {
		t.Fatal("unaccepted staging entered bank", e)
	}
	if e = s.Complete(ctx, j, collect.Result{State: "updated", Staged: true, Extracted: 1}, time.Now()); e != nil {
		t.Fatal(e)
	}
	if e = s.Expire(ctx, time.Now().Add(3*time.Hour)); e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec("DELETE FROM sources WHERE id=?", id); e != nil {
		t.Fatal(e)
	}
	if used, e := s.Referenced(ctx, ref.Hash); e != nil || !used {
		t.Fatal("persistent sample unprotected", e)
	}
	v, e := s.BankOne(ctx, ref.Hash)
	if e != nil || v.Hash != ref.Hash {
		t.Fatal("sample lost", e)
	}
	f, e := s.vault.Open(v.Hash)
	if e != nil {
		t.Fatal(e)
	}
	got, _ := io.ReadAll(f)
	f.Close()
	if !bytes.Equal(got, raw) {
		t.Fatal("raw bytes changed")
	}
	if _, e = s.ManageBank(ctx, ConfigBatch{Action: "delete", IDs: []string{ref.Hash}}); e != nil {
		t.Fatal(e)
	}
	if _, e = s.ImportBank(ctx, bytes.NewReader(raw)); e != nil {
		t.Fatal(e)
	}
	if _, e = s.BankOne(ctx, ref.Hash); !errors.Is(e, ErrNotFound) {
		t.Fatal("explicit delete resurrected")
	}
}
func TestProbeEvidenceGateAndStaleLeases(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	j := addBank(t, s, sampleVLESS)
	r := passedResult(j)
	if e := s.CompleteProbe(ctx, r); e != nil {
		t.Fatal(e)
	}
	rule, e := s.PreferredRule(ctx, r.Model)
	if e != nil || rule != r.Rule {
		t.Fatal("successful rule not learned", e)
	}
	all, e := s.Subscription(ctx, false)
	if e != nil || len(all) != 1 || !bytes.Equal(all[0], r.Candidate) {
		t.Fatal("verified bytes missing", e)
	}
	if repaired, e := s.Subscription(ctx, true); e != nil || len(repaired) != 0 {
		t.Fatal("conversion mislabelled repair", e)
	}
	if e = s.CompleteProbe(ctx, r); !errors.Is(e, ErrConflict) {
		t.Fatal("duplicate completion accepted", e)
	}
	if _, e = s.ManageBank(ctx, ConfigBatch{Action: "retest", IDs: []string{j.Hash}}); e != nil {
		t.Fatal(e)
	}
	if all, _ = s.Subscription(ctx, false); len(all) != 0 {
		t.Fatal("invalidated evidence exported")
	}
	next, e := s.ClaimProbe(ctx)
	if e != nil || next == nil {
		t.Fatal(e)
	}
	if next.Revision <= j.Revision {
		t.Fatal("revision not advanced")
	}
	if _, e = s.ManageBank(ctx, ConfigBatch{Action: "disable", IDs: []string{j.Hash}}); e != nil {
		t.Fatal(e)
	}
	if e = s.CompleteProbe(ctx, passedResult(*next)); !errors.Is(e, ErrConflict) {
		t.Fatal("disabled sample revived", e)
	}
	if e = s.RenewProbe(ctx, *next); !errors.Is(e, ErrConflict) {
		t.Fatal("disabled lease renewed", e)
	}
}
func TestSubscriptionsDedupeAndRejectPartialExpiredOrWrongVersion(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	j := addBank(t, s, sampleVLESS)
	r := passedResult(j)
	bad := r
	bad.EndpointsPassed = 1
	if e := s.CompleteProbe(ctx, bad); !errors.Is(e, ErrInvalid) {
		t.Fatal("one destination approved", e)
	}
	bad = r
	bad.CoreSHA256 = strings.Repeat("0", 64)
	if e := s.CompleteProbe(ctx, bad); !errors.Is(e, ErrInvalid) {
		t.Fatal("wrong core approved", e)
	}
	if e := s.CompleteProbe(ctx, r); e != nil {
		t.Fatal(e)
	}
	j2 := addBank(t, s, strings.Replace(sampleVLESS, "#synthetic", "#other-label", 1))
	if e := s.CompleteProbe(ctx, passedResult(j2)); e != nil {
		t.Fatal(e)
	}
	all, e := s.Subscription(ctx, false)
	if e != nil || len(all) != 1 {
		t.Fatal("duplicate subscription", len(all), e)
	}
	s.db.Exec("UPDATE probe_results SET valid_until=0")
	all, e = s.Subscription(ctx, false)
	if e != nil || len(all) != 0 {
		t.Fatal("expired proof exported", e)
	}
	s.db.Exec("UPDATE probe_results SET valid_until=?,version='older'", time.Now().Add(time.Hour).UnixMilli())
	all, _ = s.Subscription(ctx, false)
	if len(all) != 0 {
		t.Fatal("old rule proof exported")
	}
}
func TestPrivateQueueNotPublicAndSubscriptionJSON(t *testing.T) {
	s := testStore(t)
	j := addBank(t, s, sampleVLESS)
	if e := s.CompleteProbe(context.Background(), passedResult(j)); e != nil {
		t.Fatal(e)
	}
	app, _ := NewServer(s, "http://127.0.0.1:18192")
	h := app.Handler()
	for _, path := range []string{"/claim", "/renew", "/complete", "/status"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "http://127.0.0.1:18192"+path, nil))
		if w.Code != 404 && w.Code != 405 {
			t.Fatal("queue publicly mounted", path, w.Code)
		}
	}
	report := httptest.NewRecorder()
	h.ServeHTTP(report, httptest.NewRequest("GET", "http://127.0.0.1:18192/api/bank/report/"+j.Hash, nil))
	var history struct {
		Variants []bankVariant `json:"variants"`
	}
	if report.Code != 200 || json.Unmarshal(report.Body.Bytes(), &history) != nil || len(history.Variants) != 1 {
		t.Fatal("candidate history API failed")
	}
	variant := httptest.NewRecorder()
	h.ServeHTTP(variant, httptest.NewRequest("GET", "http://127.0.0.1:18192/bank/variant/"+j.Hash+"/"+history.Variants[0].Hash, nil))
	if variant.Code != 200 || !bytes.Equal(variant.Body.Bytes(), passedResult(j).Candidate) {
		t.Fatal("candidate history bytes changed")
	}
	token := s.digest("subscription", "server-verified-v1")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "http://127.0.0.1:18192/sub/"+token+"/all.json", nil))
	var out []json.RawMessage
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &out) != nil || len(out) != 1 || w.Header().Get("X-Android-Import") != "not-run" {
		t.Fatal("subscription contract", w.Code)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "http://127.0.0.1:18192/sub/wrong/all.json", nil))
	if w.Code != 404 {
		t.Fatal("token gate")
	}
}
func TestProbeExpiredLeaseRecovery(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	j := addBank(t, s, sampleVLESS)
	s.db.Exec("UPDATE probe_jobs SET lease_until=0")
	next, e := s.ClaimProbe(ctx)
	if e != nil || next == nil || next.Token == j.Token {
		t.Fatal("lease not recovered", e)
	}
	if e = s.CompleteProbe(ctx, passedResult(j)); !errors.Is(e, ErrConflict) {
		t.Fatal("stale lease result accepted", e)
	}
}

func TestGlobalPauseRevokesLeaseAndHealthTTLClamps(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	j := addBank(t, s, sampleVLESS)
	v, e := s.ProbeSettings(ctx)
	if e != nil {
		t.Fatal(e)
	}
	v.Enabled = false
	if e = s.SaveProbeSettings(ctx, v); e != nil {
		t.Fatal(e)
	}
	if e = s.CompleteProbe(ctx, passedResult(j)); !errors.Is(e, ErrConflict) {
		t.Fatal("paused lease completed", e)
	}
	if next, e := s.ClaimProbe(ctx); e != nil || next != nil {
		t.Fatal("paused claim", e)
	}
	v.Enabled = true
	v.RetestSeconds = 30
	v.HealthTTLSeconds = 30
	if e = s.SaveProbeSettings(ctx, v); e != nil {
		t.Fatal(e)
	}
	next, e := s.ClaimProbe(ctx)
	if e != nil || next == nil {
		t.Fatal(e)
	}
	r := passedResult(*next)
	if e = s.CompleteProbe(ctx, r); e != nil {
		t.Fatal(e)
	}
	var variants int
	if e = s.db.QueryRow("SELECT count(*) FROM bank_variants WHERE hash=?", j.Hash).Scan(&variants); e != nil || variants != 1 {
		t.Fatal("candidate history missing", e)
	}
	v.Enabled = false
	if e = s.SaveProbeSettings(ctx, v); e != nil {
		t.Fatal(e)
	}
	var tested, until int64
	s.db.QueryRow("SELECT tested,valid_until FROM probe_results WHERE hash=?", j.Hash).Scan(&tested, &until)
	if until-tested > 30000 {
		t.Fatal("proof TTL not clamped")
	}
}

func TestNewSamplesCannotStarveDueRetests(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	first := addBank(t, s, sampleVLESS)
	if e := s.CompleteProbe(ctx, passedResult(first)); e != nil {
		t.Fatal(e)
	}
	s.db.Exec("UPDATE probe_jobs SET next_run=1 WHERE hash=?", first.Hash)
	for _, suffix := range []string{"a", "b", "c", "d"} {
		if _, e := s.ImportBank(ctx, strings.NewReader(sampleVLESS+suffix)); e != nil {
			t.Fatal(e)
		}
	}
	s.probeTurn.Store(0)
	for i := 0; i < 4; i++ {
		j, e := s.ClaimProbe(ctx)
		if e != nil || j == nil {
			t.Fatal(e)
		}
		if (i == 3) != (j.Hash == first.Hash) {
			t.Fatal("new sample/retest fairness violated", i)
		}
	}
}
