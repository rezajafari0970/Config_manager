package panel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/rezajafari0970/Config_manager/internal/collect"
	"github.com/rezajafari0970/Config_manager/internal/convert"
	"github.com/rezajafari0970/Config_manager/internal/probe"
	"io"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const conversionSample = "vless://00000000-0000-4000-8000-000000000001@192.0.2.10:443?security=tls&sni=example.com#synthetic"

func addConversion(t *testing.T, s *Store, raw string) (string, probe.Job) {
	t.Helper()
	ctx := context.Background()
	src := testSource()
	src.URL = "https://example.com/" + randomToken()
	id, e := s.Save(ctx, "", 0, src)
	if e != nil {
		t.Fatal(e)
	}
	j := claimOne(t, s, id)
	if e = s.Complete(ctx, j, fixtureResult(t, s, raw), time.Now()); e != nil {
		t.Fatal(e)
	}
	job, e := s.ClaimProbe(ctx)
	if e != nil || job == nil {
		t.Fatal("conversion claim", e)
	}
	return id, *job
}
func conversionResult(j probe.Job, idx int, connected bool) probe.Result {
	p := convert.Prepare(j.Raw)
	c := p.Candidates[idx]
	a := probe.Attempt{SHA256: c.SHA256, Rule: c.Rule, Core: "passed", Connection: "failed", Code: "proxy_request_failed", RuntimeSHA256: c.SHA256}
	if connected {
		a.Connection = "passed"
		a.Code = "https_proxy_verified"
		a.EndpointsPassed = 2
	}
	return probe.Result{Hash: j.Hash, Token: j.Token, Revision: j.Revision, Version: convert.Version, CoreSHA256: probe.CoreSHA256, Attempt: a, Candidate: c.Config}
}
func TestConversionExportJSONLinesAndRemovedBank(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	_, j := addConversion(t, s, conversionSample)
	if e := s.CompleteProbe(ctx, conversionResult(j, 0, false)); e != nil {
		t.Fatal(e)
	}
	// Identical connection with a different label must not duplicate the output.
	src := testSource()
	src.URL = "https://example.com/second"
	id, _ := s.Save(ctx, "", 0, src)
	job := claimOne(t, s, id)
	if e := s.Complete(ctx, job, fixtureResult(t, s, conversionSample+"label"), time.Now()); e != nil {
		t.Fatal(e)
	}
	server, _ := NewServer(s, "http://127.0.0.1:18191")
	h := server.Handler()
	path := "/subscription/" + s.digest("subscription", "template-jsonl-v1") + "/configs.jsonl"
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "http://127.0.0.1:18191"+path, nil)
	h.ServeHTTP(rr, req)
	if rr.Code != 200 || rr.Header().Get("Content-Type") != "application/x-ndjson; charset=utf-8" {
		t.Fatal(rr.Code, rr.Body.String())
	}
	lines := strings.Split(strings.TrimSuffix(rr.Body.String(), "\n"), "\n")
	if len(lines) != 1 || !json.Valid([]byte(lines[0])) || lines[0][0] != '{' {
		t.Fatal("invalid JSONL")
	}
	if rr.Header().Get("X-Android-Import") != "not-run" {
		t.Fatal("unearned evidence")
	}
	for _, path := range []string{"/api/bank", "/api/bank/report/x", "/bank/raw/x", "/sub/x/all.json", "/assets/bank.js"} {
		rr = httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest("GET", "http://127.0.0.1:18191"+path, nil))
		if rr.Code != 404 {
			t.Fatal("old bank route remains", path, rr.Code)
		}
	}
	body, e := s.vault.Open(j.Hash)
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := io.ReadAll(body)
	body.Close()
	if string(raw) != conversionSample {
		t.Fatal("original mutated")
	}
	var n int
	s.db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name LIKE 'bank_%' OR name LIKE 'probe_%'").Scan(&n)
	if n != 0 {
		t.Fatal("bank schema remains")
	}
}
func TestConversionLifetimeLeasesAndSourceIsolation(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	id, j := addConversion(t, s, conversionSample)
	if e := s.CompleteProbe(ctx, conversionResult(j, 0, true)); e != nil {
		t.Fatal(e)
	}
	// Empty response preserves the current generation, never extends its TTL.
	fetch := claimOne(t, s, id)
	if e := s.Complete(ctx, fetch, collect.Result{State: "empty"}, time.Now()); e != nil {
		t.Fatal(e)
	}
	rows, _ := s.Conversions(ctx, 0, true)
	if len(rows) != 1 || !rows[0].Available {
		t.Fatal("empty fetch dropped conversion")
	}
	candidate := rows[0].CandidateHash
	s.db.Exec("UPDATE conversion_jobs SET next_run=0")
	old, e := s.ClaimProbe(ctx)
	if e != nil || old == nil {
		t.Fatal(e)
	}
	if e = s.Delete(ctx, id, 1); e != nil {
		t.Fatal(e)
	}
	if e = s.CompleteProbe(ctx, conversionResult(*old, 0, true)); !errors.Is(e, ErrConflict) {
		t.Fatal("deleted completion accepted", e)
	}
	if rows, _ = s.Conversions(ctx, 0, true); len(rows) != 0 {
		t.Fatal("orphan published")
	}
	if used, _ := s.Referenced(ctx, candidate); used {
		t.Fatal("orphan blob retained")
	}
	_, fresh := addConversion(t, s, conversionSample)
	if e = s.CompleteProbe(ctx, conversionResult(fresh, 0, false)); e != nil {
		t.Fatal(e)
	}
	if e = s.Expire(ctx, time.Now().Add(3*time.Hour)); e != nil {
		t.Fatal(e)
	}
	var n int
	s.db.QueryRow("SELECT count(*) FROM conversion_jobs").Scan(&n)
	if n != 0 {
		t.Fatal("TTL left jobs")
	}
}
func TestConversionHypothesisRequiresFreshConnection(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	_, j := addConversion(t, s, "trojan://synthetic@192.0.2.1:443?security=none&sni=example.com")
	if e := s.CompleteProbe(ctx, conversionResult(j, 1, false)); e != nil {
		t.Fatal(e)
	}
	rows, _ := s.Conversions(ctx, 0, true)
	if rows[0].Available {
		t.Fatal("unconnected TLS guess exported")
	}
	s.db.Exec("UPDATE conversion_jobs SET next_run=0")
	next, _ := s.ClaimProbe(ctx)
	if e := s.CompleteProbe(ctx, conversionResult(*next, 1, true)); e != nil {
		t.Fatal(e)
	}
	rows, _ = s.Conversions(ctx, 0, true)
	if !rows[0].Available {
		t.Fatal("verified repair hidden")
	}
	s.db.Exec("UPDATE conversion_results SET valid_until=0")
	rows, _ = s.Conversions(ctx, 0, true)
	if rows[0].Available {
		t.Fatal("expired hypothesis exported")
	}
}
func TestConversionRejectsUnrelatedCandidateAndStaleWorker(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	_, j := addConversion(t, s, conversionSample)
	r := conversionResult(j, 0, true)
	r.Candidate = bytes.ReplaceAll(r.Candidate, []byte("@shenvpn"), []byte("changed"))
	if e := s.CompleteProbe(ctx, r); !errors.Is(e, ErrInvalid) {
		t.Fatal("arbitrary payload accepted", e)
	}
	s.db.Exec("UPDATE conversion_jobs SET lease_until=0")
	next, _ := s.ClaimProbe(ctx)
	if next == nil || next.Token == j.Token {
		t.Fatal("lease recovery")
	}
	if e := s.CompleteProbe(ctx, conversionResult(j, 0, true)); !errors.Is(e, ErrConflict) {
		t.Fatal("stale token accepted")
	}
	settings, _ := s.ProbeSettings(ctx)
	settings.Enabled = false
	s.SaveProbeSettings(ctx, settings)
	if e := s.CompleteProbe(ctx, conversionResult(*next, 0, true)); !errors.Is(e, ErrConflict) {
		t.Fatal("paused completion accepted")
	}
}
func TestV5BankRemovalPreservesSources(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	id, _ := s.Save(ctx, "", 0, testSource())
	_, e := s.db.Exec(`DROP VIEW conversion_inputs;DROP TABLE conversion_results;DROP TABLE conversion_jobs;DROP TABLE conversion_settings;DROP TABLE conversion_worker;
 CREATE TABLE bank_samples(hash TEXT PRIMARY KEY);CREATE TABLE probe_jobs(hash TEXT REFERENCES bank_samples(hash));CREATE TABLE probe_results(hash TEXT REFERENCES bank_samples(hash));CREATE TABLE bank_variants(hash TEXT REFERENCES bank_samples(hash));CREATE TABLE probe_models(x);CREATE TABLE probe_rule_evidence(x);CREATE TABLE probe_settings(id,enabled,concurrency,timeout_seconds,retest_seconds,health_ttl_seconds);INSERT INTO probe_settings VALUES(1,1,3,7,120,300);CREATE TABLE probe_worker(x);INSERT INTO bank_samples VALUES('synthetic');PRAGMA user_version=5;`)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.migrateV6(); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Get(ctx, id); e != nil {
		t.Fatal("migration lost source", e)
	}
	var v int
	s.db.QueryRow("PRAGMA user_version").Scan(&v)
	if v != 6 {
		t.Fatal(v)
	}
	settings, _ := s.ProbeSettings(ctx)
	if settings.Concurrency != 3 || settings.TimeoutSeconds != 7 {
		t.Fatal("lost worker settings")
	}
}

func TestSQLiteRecycledConnectionsKeepCascadesAndDurability(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	id, j := addConversion(t, s, conversionSample)
	if e := s.CompleteProbe(ctx, conversionResult(j, 0, true)); e != nil {
		t.Fatal(e)
	}
	// Force a different physical SQLite connection for each subsequent operation.
	s.db.SetMaxIdleConns(0)
	for pragma, want := range map[string]int{"foreign_keys": 1, "busy_timeout": 5000, "synchronous": 2} {
		var got int
		if e := s.db.QueryRow("PRAGMA " + pragma).Scan(&got); e != nil || got != want {
			t.Fatal("recycled connection", pragma, got, e)
		}
	}
	if e := s.Delete(ctx, id, 1); e != nil {
		t.Fatal(e)
	}
	var count int
	s.db.QueryRow("SELECT count(*) FROM conversion_results").Scan(&count)
	if count != 0 {
		t.Fatal("orphaned conversion result", count)
	}
	rows, e := s.db.Query("PRAGMA foreign_key_check")
	if e != nil {
		t.Fatal(e)
	}
	defer rows.Close()
	if rows.Next() {
		t.Fatal("foreign key violation")
	}
}
func TestStartupCleansLegacyOrphanConversionResults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private", "panel.db")
	key := bytes.Repeat([]byte{31}, 32)
	s, e := Open(path, key)
	if e != nil {
		t.Fatal(e)
	}
	_, j := addConversion(t, s, conversionSample)
	if e = s.CompleteProbe(context.Background(), conversionResult(j, 0, true)); e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec("PRAGMA foreign_keys=OFF;DELETE FROM conversion_jobs;INSERT INTO memberships SELECT source_id,'missing-artifact',generation,raw_hash,kind,size FROM memberships LIMIT 1;"); e != nil {
		t.Fatal(e)
	}
	s.Close()
	s, e = Open(path, key)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	var count int
	s.db.QueryRow("SELECT count(*) FROM conversion_results").Scan(&count)
	if count != 0 {
		t.Fatal("legacy orphan survived restart")
	}
	sources, e := s.ListPage(context.Background(), "", 0)
	if e != nil || len(sources) != 1 {
		t.Fatal("source damaged", e)
	}
	body, e := s.vault.Open(j.Hash)
	if e != nil {
		t.Fatal("raw input damaged", e)
	}
	body.Close()
	check, e := s.db.Query("PRAGMA foreign_key_check")
	if e != nil {
		t.Fatal(e)
	}
	defer check.Close()
	if check.Next() {
		t.Fatal("legacy foreign key violation remains")
	}
}

func TestUnavailableConversionDoesNotBreakJSONLinesAndRequeues(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	_, a := addConversion(t, s, conversionSample)
	if e := s.CompleteProbe(ctx, conversionResult(a, 0, true)); e != nil {
		t.Fatal(e)
	}
	_, b := addConversion(t, s, strings.Replace(conversionSample, "192.0.2.10", "192.0.2.11", 1))
	if e := s.CompleteProbe(ctx, conversionResult(b, 0, true)); e != nil {
		t.Fatal(e)
	}
	bad := conversionResult(a, 0, true)
	if e := os.Remove(filepath.Join(s.vault.Directory(), bad.SHA256+".blob")); e != nil {
		t.Fatal(e)
	}
	server, _ := NewServer(s, "http://127.0.0.1:18191")
	rr := httptest.NewRecorder()
	path := "http://127.0.0.1:18191/subscription/" + s.digest("subscription", "template-jsonl-v1") + "/configs.jsonl"
	server.Handler().ServeHTTP(rr, httptest.NewRequest("GET", path, nil))
	if rr.Code != 200 || rr.Header().Get("X-Config-Count") != "1" || !json.Valid(bytes.TrimSpace(rr.Body.Bytes())) {
		t.Fatal("one missing blob broke unrelated output", rr.Code)
	}
	j, e := s.ClaimProbe(ctx)
	if e != nil || j == nil || j.Hash != a.Hash {
		t.Fatal("missing blob was not requeued", e)
	}
	if e = s.CompleteProbe(ctx, conversionResult(*j, 0, true)); e != nil {
		t.Fatal(e)
	}
	rows, _ := s.Conversions(ctx, 0, true)
	if len(rows) != 2 || !rows[0].Available || !rows[1].Available {
		t.Fatal("conversion did not recover")
	}
	// A stale list snapshot must not revoke a newer different candidate.
	for _, c := range rows {
		if c.Hash == a.Hash {
			c.CandidateHash = strings.Repeat("0", 64)
			if e = s.retryConversionBlob(ctx, c); e != nil {
				t.Fatal(e)
			}
		}
	}
	rows, _ = s.Conversions(ctx, 0, true)
	if !rows[0].Available || !rows[1].Available {
		t.Fatal("stale snapshot revoked fresh result")
	}
}
