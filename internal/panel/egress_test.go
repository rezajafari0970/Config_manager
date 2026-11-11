package panel

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/rezajafari0970/Config_manager/internal/egress"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func dropEgressForMigration(t *testing.T, s *Store) {
	t.Helper()
	for _, table := range []string{"conversion_inputs", "smart_inputs", "splash_smart_inputs"} {
		var q string
		if e := s.db.QueryRow("SELECT sql FROM sqlite_master WHERE name=?", table).Scan(&q); e != nil {
			t.Fatal(e)
		}
		q = strings.ReplaceAll(q, " AND m.raw_hash NOT IN (SELECT hash FROM input_blocked) AND m.artifact_id NOT IN (SELECT identity FROM input_blocked_identities)", "")
		q = strings.ReplaceAll(q, " UNION SELECT hash,kind,size FROM egress_eligible", "")
		if _, e := s.db.Exec("DROP VIEW " + table + ";" + q); e != nil {
			t.Fatal(e)
		}
	}
	for _, q := range []string{"DROP TRIGGER egress_ingest", "DROP VIEW input_blocked_identities", "DROP VIEW input_blocked", "DROP VIEW egress_eligible", "DROP TABLE egress_owners", "DROP TABLE egress_jobs", "DROP TABLE egress_turns", "DROP TABLE egress_events", "DROP TABLE egress_settings", "DROP TABLE egress_worker"} {
		if _, e := s.db.Exec(q); e != nil {
			t.Fatal(e)
		}
	}
}
func syntheticEgress(j egress.Job) egress.Report {
	p := egress.Prepare(j.Raw)
	now := egress.Now()
	r := egress.Report{Version: egress.Version, Hash: j.Hash, Started: now, Finished: now, Core: "xray-26.11.9", CoreSHA: egress.Cores["xray-26.11.9"], Infrastructure: true}
	for _, v := range p.Variants {
		r.Checks = append(r.Checks, egress.Check{Name: v.Name, Hash: v.Hash, RuntimeHash: v.Hash, Core: true})
		for wave := 1; wave <= 3; wave++ {
			for _, d := range append(append([]string{}, egress.Domains...), "www.cloudflare.com", "api.ipify.org") {
				ads := d != "www.cloudflare.com" && d != "api.ipify.org"
				ip := "1.1.1.1"
				if ads {
					ip = "8.8.8.8"
				}
				n := egress.Hash([]byte(fmt.Sprint(v.Name, d, wave)))[:32]
				r.Samples = append(r.Samples, egress.Sample{Variant: v.Name, Domain: d, Wave: wave, At: now, IP: ip, Family: 4, Transport: "tcp", Nonce: n, Connection: n, TLS: true, SameStream: true, Echo: true, ActualAds: ads, Certificate: strings.Repeat("c", 64), Response: strings.Repeat("f", 64)})
			}
		}
	}
	r.Facts = []egress.Fact{{IP: "8.8.8.8", Provider: "proxycheck-v3", Class: "Residential", At: now, Expires: now + 3600000, Digest: strings.Repeat("a", 64)}}
	return r
}
func TestEgressQueueLeaseExportAndExpiry(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	uris := []string{}
	for i := 1; i <= egress.MaxWorkers+1; i++ {
		uris = append(uris, sourceURI(i))
	}
	newExportSource(t, s, "all", uris...)
	j, e := s.ClaimEgress(ctx)
	if e != nil || j == nil {
		t.Fatal(e)
	}
	j2, e := s.ClaimEgress(ctx)
	if e != nil || j2 == nil || j2.Hash == j.Hash {
		t.Fatal(e)
	}
	for i := 2; i < egress.MaxWorkers; i++ {
		extra, err := s.ClaimEgress(ctx)
		if err != nil || extra == nil {
			t.Fatal("parallel claim failed", i, err)
		}
	}
	third, e := s.ClaimEgress(ctx)
	if e != nil || third != nil {
		t.Fatal("unbounded concurrency", e)
	}
	r := syntheticEgress(*j)
	wrong := *j
	wrong.Token = "wrong"
	if e = s.CompleteEgress(ctx, egress.Completion{Job: wrong, Report: r}); e != ErrConflict {
		t.Fatal("stale lease accepted", e)
	}
	if e = s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: r}); e != nil {
		t.Fatal(e)
	}
	app, e := NewServer(s, "http://127.0.0.1:18191")
	if e != nil {
		t.Fatal(e)
	}
	h := app.Handler()
	base := app.origin + "/egress-subscription/" + s.digest("egress-export-v2", "all") + "/"
	for _, f := range []string{"configs.json", "configs.jsonl", "next.json", "raw.txt"} {
		rr := exportRequest(t, h, "GET", base+f)
		if rr.Code != 200 {
			t.Fatal(f, rr.Code, rr.Body.String())
		}
		if f == "configs.json" || f == "next.json" {
			if !json.Valid(rr.Body.Bytes()) {
				t.Fatal("invalid json")
			}
		}
	}
	if _, e = s.db.Exec("UPDATE egress_jobs SET first_seen=1 WHERE hash=?", j.Hash); e != nil {
		t.Fatal(e)
	}
	if e = s.Expire(ctx, time.Now()); e != nil {
		t.Fatal(e)
	}
	var state string
	s.db.QueryRow("SELECT state FROM egress_jobs WHERE hash=?", j.Hash).Scan(&state)
	if state != "RESIDENTIAL" {
		t.Fatal("accepted expired by retention", state)
	}
	s.db.Exec("UPDATE egress_jobs SET valid_until=1 WHERE hash=?", j.Hash)
	if rr := exportRequest(t, h, "GET", base+"configs.json"); rr.Code != 503 {
		t.Fatal("stale proof exported", rr.Code)
	}
	for _, path := range []string{"/api/residential", "/residential-subscription/old/configs.json", "/ads-different-subscription/old/raw.txt"} {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest("GET", app.origin+path, nil))
		if rr.Code != 410 {
			t.Fatal(path, rr.Code)
		}
	}
	var legacy int
	s.db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name GLOB 'residential_*'").Scan(&legacy)
	if legacy != 0 {
		t.Fatal("legacy remains")
	}
}
func TestEgressUnknownQuarantinesAndConfirmedLossPurges(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "member", sourceURI(4))
	j, _ := s.ClaimEgress(ctx)
	if j == nil {
		t.Fatal("no input")
	}
	if e := s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: syntheticEgress(*j)}); e != nil {
		t.Fatal(e)
	}
	for step := 0; step < 4; step++ {
		s.db.Exec("UPDATE egress_jobs SET next_run=0,last_failure=0 WHERE hash=?", j.Hash)
		next, e := s.ClaimEgress(ctx)
		if e != nil || next == nil {
			t.Fatal(e)
		}
		r := syntheticEgress(*next)
		if step == 0 {
			r.Facts = nil
		} else {
			r.Facts[0].Class = "Hosting"
		}
		if e = s.CompleteEgress(ctx, egress.Completion{Job: *next, Report: r}); e != nil {
			t.Fatal(e)
		}
		var state string
		s.db.QueryRow("SELECT state FROM egress_jobs WHERE hash=?", j.Hash).Scan(&state)
		want := "QUARANTINED"
		if step == 3 {
			want = "DELETED"
		}
		if state != want {
			t.Fatal(step, state, want)
		}
	}
	var n int
	s.db.QueryRow("SELECT count(*) FROM conversion_inputs WHERE hash=?", j.Hash).Scan(&n)
	if n != 0 {
		t.Fatal("purged member in ordinary output")
	}
	s.db.QueryRow("SELECT count(*) FROM input_exclusions WHERE hash=?", j.Hash).Scan(&n)
	if n == 0 {
		t.Fatal("purge not durable")
	}
}

func TestEgressCurrentDeepRetryPrecedesRetainedFirstAssessment(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "active", sourceURI(121))
	j, err := s.ClaimEgress(ctx)
	if err != nil || j == nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec("UPDATE egress_jobs SET token='',lease_until=0,tested=?,attempts=1,next_run=0 WHERE hash=?", egress.Now(), j.Hash); err != nil {
		t.Fatal(err)
	}
	archived := strings.Repeat("a", 64)
	if _, err = s.db.Exec("INSERT INTO egress_jobs(hash,kind,size,first_seen,last_seen) VALUES(?, 'fixture',1,1,1)", archived); err != nil {
		t.Fatal(err)
	}
	next, err := s.ClaimEgress(ctx)
	if err != nil || next == nil || next.Hash != j.Hash || !next.Deep {
		t.Fatal("live retry starved by history", next, err)
	}
	s.db.Exec("UPDATE egress_jobs SET token='',lease_until=0 WHERE hash=?", j.Hash)
	next, err = s.ClaimEgress(ctx)
	if err != nil || next == nil || next.Hash != archived {
		t.Fatal("history starved by repeated retry", next, err)
	}
}

func TestEgressManualRecheckPrecedesCurrentFirstAssessment(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "active", sourceURI(131), sourceURI(132))
	var target string
	s.db.QueryRow("SELECT hash FROM egress_jobs ORDER BY hash DESC LIMIT 1").Scan(&target)
	if _, e := s.db.Exec("UPDATE egress_jobs SET reason='manual_recheck',tested=?,attempts=2,first_seen=? WHERE hash=?", egress.Now(), egress.Now()+1000, target); e != nil {
		t.Fatal(e)
	}
	j, e := s.ClaimEgress(ctx)
	if e != nil || j == nil || j.Hash != target {
		t.Fatal("manual check starved", j, e)
	}
}

func TestEgressPartialFirstAttemptRetriesSoonAndRemainsVisible(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "partial source", sourceURI(141))
	j, e := s.ClaimEgress(ctx)
	if e != nil || j == nil {
		t.Fatal(e)
	}
	r := syntheticEgress(*j)
	for i := range r.Samples {
		if r.Samples[i].Variant == "raw" && r.Samples[i].Wave == 1 && r.Samples[i].Domain == "api.ipify.org" {
			r.Samples[i].Echo = false
		}
	}
	if e = s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: r}); e != nil {
		t.Fatal(e)
	}
	var tested, next int64
	s.db.QueryRow("SELECT tested,next_run FROM egress_jobs WHERE hash=?", j.Hash).Scan(&tested, &next)
	if next-tested != 30000 {
		t.Fatal("partial first attempt delayed", next-tested)
	}
	app, e := NewServer(s, "http://127.0.0.1:18191")
	if e != nil {
		t.Fatal(e)
	}
	rr := exportRequest(t, app.Handler(), "GET", app.origin+"/api/egress")
	var body struct {
		Eligible, Partial int
		Entries           []struct {
			State    string
			Sources  []string
			Decision egress.Decision
		}
	}
	if rr.Code != 200 || json.Unmarshal(rr.Body.Bytes(), &body) != nil || body.Partial != 1 || body.Eligible != 0 || len(body.Entries) != 1 || body.Entries[0].Decision.Eligible || len(body.Entries[0].Sources) != 1 {
		t.Fatal("partial hidden or incorrectly eligible", rr.Code, rr.Body.String())
	}
}
