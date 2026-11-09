package panel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/rezajafari0970/Config_manager/internal/probe"
	"github.com/rezajafari0970/Config_manager/internal/repair"
	"github.com/rezajafari0970/Config_manager/internal/smart"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func smartResult(j *probe.Job) probe.Result {
	return probe.Result{Hash: j.Hash, Token: j.Token, Revision: j.Revision, Version: smart.Version, CoreSHA256: probe.CoreSHA256, Attempt: probe.Attempt{SHA256: repair.Digest(j.Raw), Rule: smart.Version, Core: "passed", Connection: "not_run", Code: "synthetic_smart_fixture"}, Candidate: j.Raw}
}
func finishSmart(t *testing.T, s *Store) int {
	t.Helper()
	n := 0
	for n < 100 {
		j, e := s.ClaimSmart(context.Background())
		if e != nil {
			t.Fatal(e)
		}
		if j == nil {
			return n
		}
		if j.Issue != "" {
			t.Fatal(j.Issue)
		}
		if e = s.CompleteSmart(context.Background(), smartResult(j)); e != nil {
			t.Fatal(e)
		}
		n++
	}
	t.Fatal("smart builder did not quiesce")
	return n
}
func smartFixture(t *testing.T) (*Store, []string) {
	t.Helper()
	s := testStore(t)
	a, _ := newExportSource(t, s, "SmartA", sourceURI(1), sourceURI(2), sourceURI(3), sourceURI(4))
	b, _ := newExportSource(t, s, "SmartB", sourceURI(5))
	c, _ := newExportSource(t, s, "SmartC", sourceURI(6), sourceURI(7))
	v, _ := s.SmartSettings(context.Background())
	v.Count = 3
	if e := s.SaveSmartSettings(context.Background(), v); e != nil {
		t.Fatal(e)
	}
	return s, []string{a, b, c}
}
func TestSmartCacheCoverageWarmingAndIndependentRounds(t *testing.T) {
	s, _ := smartFixture(t)
	ctx := context.Background()
	normalBefore, _ := s.Conversions(ctx, 0, true)
	j, e := s.ClaimSmart(ctx)
	if e != nil || j == nil {
		t.Fatal(e)
	}
	if e = s.CompleteSmart(ctx, smartResult(j)); e != nil {
		t.Fatal(e)
	}
	first, body, cycle, e := s.smartNext(ctx)
	if e != nil || first == nil || cycle != 1 || !json.Valid(body) {
		t.Fatal("first", e)
	}
	next, _, same, e := s.smartNext(ctx)
	if e != nil || next == nil || next.ID != first.ID || same != 2 {
		t.Fatal("ready profile blocked by warming heads", e)
	}
	if n := finishSmart(t, s); n != 6 {
		t.Fatal("missing heads", n)
	}
	seen := map[string]bool{}
	for i := 0; i < 7; i++ {
		v, b, c, e := s.smartNext(ctx)
		if e != nil || v == nil || c != 3 || seen[v.ID] || bytes.ContainsAny(b, "\r\n") {
			t.Fatal("round", e, c)
		}
		seen[v.ID] = true
	}
	v, _, c, e := s.smartNext(ctx)
	if e != nil || v == nil || c != 4 {
		t.Fatal("reset", e)
	}
	rows, e := querySmart(ctx, s.db, true, 0)
	if e != nil || len(rows) != 7 {
		t.Fatal(e)
	}
	for _, r := range rows {
		if !r.Available || r.Count != 3 || r.Sources != 3 {
			t.Fatal("coverage", r)
		}
	}
	after, _ := s.Conversions(ctx, 0, true)
	if len(after) != len(normalBefore) {
		t.Fatal("normal outputs changed")
	}
	app, _ := NewServer(s, "http://127.0.0.1:18191")
	h := app.Handler()
	rr := exportRequest(t, h, "GET", app.smartURL(false))
	if rr.Code != 200 || rr.Header().Get("Content-Disposition") != "inline" {
		t.Fatal("not inline")
	}
	lines := bytes.Split(bytes.TrimSpace(rr.Body.Bytes()), []byte{'\n'})
	if len(lines) != 7 {
		t.Fatal("full cache")
	}
	unique := map[string]bool{}
	for _, line := range lines {
		hash := repair.Digest(line)
		if unique[hash] || !json.Valid(line) {
			t.Fatal("duplicate/bad JSONL")
		}
		unique[hash] = true
	}
	before := 0
	s.db.QueryRow("SELECT count(*) FROM smart_rotation_queue").Scan(&before)
	exportRequest(t, h, "HEAD", app.smartURL(true))
	req := exportRequest(t, h, "GET", app.origin+"/smart/claim")
	if req.Code != 404 {
		t.Fatal("private API exposed")
	}
	var remaining int
	s.db.QueryRow("SELECT count(*) FROM smart_rotation_queue").Scan(&remaining)
	if before != remaining {
		t.Fatal("HEAD consumed")
	}
	// Source/config controls immediately invalidate only affected smart profiles.
	source := s.smartSourceForTest(t, first.ID)
	if _, e = s.db.Exec("UPDATE sources SET enabled=0 WHERE id=?", source); e != nil {
		t.Fatal(e)
	}
	if _, e = s.smartBytes(ctx, *first); e == nil {
		t.Fatal("disabled member still exported")
	}
}
func (s *Store) smartSourceForTest(t *testing.T, id string) string {
	t.Helper()
	var source string
	if e := s.db.QueryRow("SELECT source FROM smart_members WHERE head=? ORDER BY position DESC LIMIT 1", id).Scan(&source); e != nil {
		t.Fatal(e)
	}
	return source
}
func TestSmartLeasesSettingsOwnershipExpiryAndBlobs(t *testing.T) {
	for _, action := range []string{"source", "disable", "expiry", "settings", "proof", "blob"} {
		t.Run(action, func(t *testing.T) {
			s, sources := smartFixture(t)
			ctx := context.Background()
			j, e := s.ClaimSmart(ctx)
			if e != nil || j == nil {
				t.Fatal(e)
			}
			if action == "blob" {
				if e = s.CompleteSmart(ctx, smartResult(j)); e != nil {
					t.Fatal(e)
				}
				rows, _ := querySmart(ctx, s.db, true, 0)
				for _, v := range rows {
					if v.Available {
						os.Remove(filepath.Join(s.vault.Directory(), v.Hash+".blob"))
					}
				}
				finishSmart(t, s)
				v, _, _, e := s.smartNext(ctx)
				if e != nil || v == nil || v.ID == j.Hash {
					t.Fatal("missing blob blocks others", e)
				}
				return
			}
			switch action {
			case "source":
				_, e = s.db.Exec("UPDATE sources SET enabled=0 WHERE id=?", sources[1])
			case "disable":
				_, e = s.ManageConfigs(ctx, ConfigBatch{Action: "disable", All: true})
			case "expiry":
				_, e = s.db.Exec("UPDATE artifacts SET expires=0")
			case "settings":
				v, _ := s.SmartSettings(ctx)
				v.Count = 2
				e = s.SaveSmartSettings(ctx, v)
			case "proof":
				_, e = s.db.Exec("UPDATE conversion_results SET requires_connection=1,valid_until=0")
			}
			if e != nil {
				t.Fatal(e)
			}
			if e = s.CompleteSmart(ctx, smartResult(j)); e == nil {
				t.Fatal("stale result accepted")
			}
		})
	}
	// Abandoned leases recover; heartbeat uses separate state.
	s, _ := smartFixture(t)
	ctx := context.Background()
	j, e := s.ClaimSmart(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.RenewSmart(ctx, *j); e != nil {
		t.Fatal(e)
	}
	s.db.Exec("UPDATE smart_heads SET lease_until=0 WHERE id=?", j.Hash)
	finishSmart(t, s)
	if e = s.CompleteSmart(ctx, smartResult(j)); e == nil {
		t.Fatal("reclaimed token accepted")
	}
	normal, _ := s.ProbeWorker(ctx)
	s.SetSmartWorker(ctx, probe.WorkerStatus{Status: "running", Version: smart.Version})
	after, _ := s.ProbeWorker(ctx)
	if normal != after {
		t.Fatal("worker states coupled")
	}
}
func TestSmartPersistentConcurrentRoundsAndMigration(t *testing.T) {
	s, _ := smartFixture(t)
	finishSmart(t, s)
	ctx := context.Background()
	first, _, cycle, e := s.smartNext(ctx)
	if e != nil || first == nil || cycle != 1 {
		t.Fatal(e)
	}
	path := filepath.Join(filepath.Dir(s.vault.Directory()), "panel.db")
	key := bytes.Clone(s.key)
	// Schema8 upgrade preserves the existing normal rotation state and queue.
	dropSplashForMigration(t, s)
	s.db.Exec("PRAGMA user_version=8")
	other, e := Open(path, key)
	if e != nil {
		t.Fatal(e)
	}
	defer other.Close()
	var schema int
	other.db.QueryRow("PRAGMA user_version").Scan(&schema)
	if schema != 13 {
		t.Fatal(schema)
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	seen := map[int64]map[string]bool{1: {first.ID: true}}
	errors := []error{}
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			store := s
			if i%2 == 1 {
				store = other
			}
			v, b, round, e := store.smartNext(ctx)
			mu.Lock()
			defer mu.Unlock()
			if e != nil {
				errors = append(errors, e)
				return
			}
			if v == nil || !json.Valid(b) {
				errors = append(errors, fmt.Errorf("empty output"))
				return
			}
			if seen[round] == nil {
				seen[round] = map[string]bool{}
			}
			if seen[round][v.ID] {
				errors = append(errors, fmt.Errorf("duplicate first position"))
			}
			seen[round][v.ID] = true
		}(i)
	}
	wg.Wait()
	if len(errors) > 0 {
		t.Fatal(errors)
	}
	for _, round := range []int64{1, 2, 3} {
		if len(seen[round]) != 7 {
			t.Fatal("incomplete round", round, len(seen[round]))
		}
	}
	// Fixed TTL and GC: fetching/exporting does not extend the cache lifetime.
	var created int64
	s.db.QueryRow("SELECT created FROM smart_heads LIMIT 1").Scan(&created)
	if e = s.Expire(ctx, time.Now().Add(3*time.Hour)); e != nil {
		t.Fatal(e)
	}
	var n int
	s.db.QueryRow("SELECT count(*) FROM smart_heads").Scan(&n)
	if n != 0 {
		t.Fatal("expired cache retained")
	}
}
func TestSmartHTTPSettingsCSRF(t *testing.T) {
	s, _ := smartFixture(t)
	app, _ := NewServer(s, "http://127.0.0.1:18191")
	h := app.Handler()
	req := exportRequest(t, h, http.MethodPost, app.origin+"/api/smart/settings")
	if req.Code != 403 {
		t.Fatal("CSRF bypass")
	}
}

func TestSmartGrowingInputsFillCapacityAndNewSource(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "GrowingA", sourceURI(1))
	finishSmart(t, s)
	rows, _ := querySmart(ctx, s.db, true, 0)
	if len(rows) != 1 || rows[0].Count != 1 {
		t.Fatal("initial short pool")
	}
	newExportSource(t, s, "GrowingB", sourceURI(2))
	rows, _ = querySmart(ctx, s.db, true, 0)
	if len(rows) != 1 || !rows[0].Available {
		t.Fatal("safe cache lost during growth")
	}
	finishSmart(t, s)
	rows, _ = querySmart(ctx, s.db, true, 0)
	if len(rows) != 2 {
		t.Fatal("growth not rebuilt")
	}
	for _, r := range rows {
		if r.Count != 2 || r.Sources != 2 {
			t.Fatal("missing new source", r)
		}
	}
	v, _ := s.SmartSettings(ctx)
	v.Count = 2
	s.SaveSmartSettings(ctx, v)
	finishSmart(t, s)
	newExportSource(t, s, "GrowingC", sourceURI(1)) // Identical bytes cover a new owner without duplicate entries.
	finishSmart(t, s)
	rows, _ = querySmart(ctx, s.db, true, 0)
	if len(rows) != 2 {
		t.Fatal("shared membership destabilized cache")
	}
}

func TestSmartIdenticalConvertedCandidatesHaveOneHead(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "IdenticalA", sourceURI(1))
	newExportSource(t, s, "IdenticalB", sourceURI(2))
	var candidate string
	if e := s.db.QueryRow("SELECT candidate_hash FROM conversion_results ORDER BY hash LIMIT 1").Scan(&candidate); e != nil {
		t.Fatal(e)
	}
	if _, e := s.db.Exec("UPDATE conversion_results SET candidate_hash=?", candidate); e != nil {
		t.Fatal(e)
	}
	if n := finishSmart(t, s); n != 1 {
		t.Fatal("duplicate first candidate", n)
	}
	rows, e := querySmart(ctx, s.db, true, 0)
	if e != nil || len(rows) != 1 || !rows[0].Available || rows[0].Count != 1 {
		t.Fatal("deduped sources caused rebuild loop", e, rows)
	}
}

func TestSmartSourceRevocationImmediatelyReleasesBothSlots(t *testing.T) {
	s, sources := smartFixture(t)
	ctx := context.Background()
	a, e := s.ClaimSmart(ctx)
	if e != nil || a == nil {
		t.Fatal(e)
	}
	b, e := s.ClaimSmart(ctx)
	if e != nil || b == nil {
		t.Fatal(e)
	}
	if next, e := s.ClaimSmart(ctx); e != nil || next != nil {
		t.Fatal("capacity before revocation", e)
	}
	var revision int64
	if e = s.db.QueryRow("SELECT revision FROM sources WHERE id=?", sources[1]).Scan(&revision); e != nil {
		t.Fatal(e)
	}
	if e = s.Delete(ctx, sources[1], revision); e != nil {
		t.Fatal(e)
	}
	// The deleted mandatory source has cascaded members but must not leave two
	// unserviceable running jobs holding the whole smart queue for 120 seconds.
	next, e := s.ClaimSmart(ctx)
	if e != nil || next == nil {
		t.Fatal("revoked source blocked all slots", e)
	}
	if next.Token == a.Token || next.Token == b.Token {
		t.Fatal("stale claim reused")
	}
	if e = s.CompleteSmart(ctx, smartResult(a)); e != ErrConflict {
		t.Fatal("old claim resurrected", e)
	}
	if e = s.CompleteSmart(ctx, smartResult(next)); e != nil {
		t.Fatal(e)
	}
}

func TestSmartTemplateVersionRevokesOldCacheAndOccupiedLeases(t *testing.T) {
	s, _ := smartFixture(t)
	ctx := context.Background()
	finishSmart(t, s)
	normal, _ := s.Conversions(ctx, 0, true)
	settings, _ := s.SmartSettings(ctx)
	first, _, cycle, e := s.smartNext(ctx)
	if e != nil || first == nil {
		t.Fatal(e)
	}
	var before int
	s.db.QueryRow("SELECT count(*) FROM smart_rotation_queue").Scan(&before)
	if _, e = s.db.Exec("UPDATE smart_heads SET version='xray-26.7.28.smart.v7';UPDATE smart_cache SET version='xray-26.7.28.smart.v7'"); e != nil {
		t.Fatal(e)
	}
	rows, e := querySmart(ctx, s.db, true, 0)
	if e != nil {
		t.Fatal(e)
	}
	for _, r := range rows {
		if r.Available {
			t.Fatal("old template exported")
		}
	}
	if _, e = s.db.Exec("UPDATE smart_heads SET token='old-worker',lease_until=?,state='running' WHERE id IN (SELECT id FROM smart_heads LIMIT 2)", time.Now().Add(time.Minute).UnixMilli()); e != nil {
		t.Fatal(e)
	}
	job, e := s.ClaimSmart(ctx)
	if e != nil || job == nil || job.Issue != "" {
		t.Fatal("old leases blocked new builder", e, job)
	}
	if e = s.CompleteSmart(ctx, smartResult(job)); e != nil {
		t.Fatal(e)
	}
	finishSmart(t, s)
	after, _ := s.SmartSettings(ctx)
	normalAfter, _ := s.Conversions(ctx, 0, true)
	var remaining int
	s.db.QueryRow("SELECT count(*) FROM smart_rotation_queue").Scan(&remaining)
	if after != settings || before != remaining || len(normalAfter) != len(normal) {
		t.Fatal("version update reset settings, turns or normal outputs")
	}
	next, body, nextCycle, e := s.smartNext(ctx)
	if e != nil || next == nil || next.ID == first.ID || nextCycle != cycle || !bytes.Contains(body, []byte(`"loglevel":"none"`)) {
		t.Fatal("updated rotation/template", e)
	}
}
