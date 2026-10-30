package panel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rezajafari0970/Config_manager/internal/collect"
	"github.com/rezajafari0970/Config_manager/internal/convert"
	"github.com/rezajafari0970/Config_manager/internal/repair"
)

func sourceURI(n int) string {
	return strings.Replace(conversionSample, "192.0.2.10", fmt.Sprintf("192.0.2.%d", n), 1)
}
func fillExportSource(t *testing.T, s *Store, id string, raws ...string) map[string][]byte {
	t.Helper()
	ctx := context.Background()
	job := claimOne(t, s, id)
	ref, e := s.vault.Put(ctx, strings.NewReader(strings.Join(raws, "\n")))
	if e != nil {
		t.Fatal(e)
	}
	defer s.vault.Release(ref.Hash)
	result := collect.Result{State: "updated", HTTPStatus: 200, Raw: ref, Extracted: len(raws)}
	expected := map[string][]byte{}
	for _, raw := range raws {
		result.Items = append(result.Items, fixtureResult(t, s, raw).Items...)
		candidate := convert.Prepare([]byte(raw)).Candidates[0]
		expected[candidate.SHA256] = candidate.Config
	}
	if e = s.Complete(ctx, job, result, time.Now()); e != nil {
		t.Fatal(e)
	}
	for {
		next, e := s.ClaimProbe(ctx)
		if e != nil {
			t.Fatal(e)
		}
		if next == nil {
			break
		}
		if e = s.CompleteProbe(ctx, conversionResult(*next, 0, true)); e != nil {
			t.Fatal(e)
		}
	}
	return expected
}
func newExportSource(t *testing.T, s *Store, name string, raws ...string) (string, map[string][]byte) {
	t.Helper()
	input := testSource()
	input.Name = name
	input.URL = "https://example.com/Api/" + name + "/trun.php"
	id, e := s.Save(context.Background(), "", 0, input)
	if e != nil {
		t.Fatal(e)
	}
	return id, fillExportSource(t, s, id, raws...)
}
func exportRequest(t *testing.T, h http.Handler, method, url string) *httptest.ResponseRecorder {
	t.Helper()
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(method, url, nil))
	return rr
}
func assertExportLines(t *testing.T, h http.Handler, url string, expected map[string][]byte) {
	t.Helper()
	rr := exportRequest(t, h, "GET", url)
	if rr.Code != 200 || rr.Header().Get("Content-Type") != "text/plain; charset=utf-8" || rr.Header().Get("Content-Disposition") != "inline" {
		t.Fatal("bad export response", rr.Code)
	}
	seen := map[string]bool{}
	for _, line := range bytes.Split(bytes.TrimSuffix(rr.Body.Bytes(), []byte{'\n'}), []byte{'\n'}) {
		if len(line) == 0 {
			continue
		}
		hash := repair.Digest(line)
		if seen[hash] || !bytes.Equal(expected[hash], line) {
			t.Fatal("unowned, duplicate or modified candidate")
		}
		seen[hash] = true
	}
	if len(seen) != len(expected) {
		t.Fatal("missing source configs", len(seen), len(expected))
	}
}
func TestSourceExportsOwnershipTokensAndIndependentRounds(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	a, wantA := newExportSource(t, s, "A", sourceURI(1), sourceURI(2))
	b, wantB := newExportSource(t, s, "B", sourceURI(1), sourceURI(3))
	app, _ := NewServer(s, "http://127.0.0.1:18191")
	h := app.Handler()
	global := map[string][]byte{}
	for k, v := range wantA {
		global[k] = v
	}
	for k, v := range wantB {
		global[k] = v
	}
	assertExportLines(t, h, app.exportURL("", false), global)
	assertExportLines(t, h, app.exportURL(a, false), wantA)
	assertExportLines(t, h, app.exportURL(b, false), wantB)
	invalid := strings.Replace(app.exportURL(a, false), "/source/"+a+"/", "/source/"+b+"/", 1)
	if exportRequest(t, h, "GET", invalid).Code != 404 {
		t.Fatal("token reused across sources")
	}
	wrongMode := strings.TrimSuffix(app.exportURL(a, false), "configs.jsonl") + "next.json"
	if exportRequest(t, h, "GET", wrongMode).Code != 404 {
		t.Fatal("token reused across output modes")
	}
	for _, scope := range []string{a, b} {
		rr := exportRequest(t, h, "GET", app.origin+"/api/conversions?source="+scope)
		var data struct {
			Entries []Conversion `json:"entries"`
			URL     string       `json:"export_url"`
			Single  string       `json:"single_export_url"`
			Source  string       `json:"source"`
		}
		if json.Unmarshal(rr.Body.Bytes(), &data) != nil || data.Source != scope || len(data.Entries) != 2 || data.URL != app.exportURL(scope, false) || data.Single != app.exportURL(scope, true) {
			t.Fatal("source API scope")
		}
		exportRequest(t, h, "HEAD", data.Single)
	}
	var count int
	s.db.QueryRow("SELECT count(*) FROM export_rotation_state").Scan(&count)
	if count != 1 {
		t.Fatal("listing, full exports or HEAD created a source round")
	}
	seen := map[string]bool{}
	firstGlobal := nextRotation(t, s)
	for i := 0; i < 2; i++ {
		c, e := s.nextSourceConversion(ctx, a)
		if e != nil || c.Cycle != 1 || seen[c.Conversion.CandidateHash] || wantA[c.Conversion.CandidateHash] == nil {
			t.Fatal("source A round", e)
		}
		seen[c.Conversion.CandidateHash] = true
	}
	nextA, e := s.nextSourceConversion(ctx, a)
	if e != nil || nextA.Cycle != 2 {
		t.Fatal("source A reset", e)
	}
	firstB, e := s.nextSourceConversion(ctx, b)
	if e != nil || firstB.Cycle != 1 || wantB[firstB.Conversion.CandidateHash] == nil {
		t.Fatal("A advanced B", e)
	}
	secondGlobal := nextRotation(t, s)
	if secondGlobal.Cycle != 1 || firstGlobal.Conversion.CandidateHash == secondGlobal.Conversion.CandidateHash {
		t.Fatal("source advanced global")
	}
}
func TestSourceSelectionPrecedesIdentityDeduplication(t *testing.T) {
	s := testStore(t)
	rawA := sourceURI(1) + "-A"
	rawB := sourceURI(1) + "-B"
	a, wantA := newExportSource(t, s, "A", rawA)
	b, _ := newExportSource(t, s, "B", rawB)
	// Both originals share canonical identity, but B's exact representation has
	// no accepted result. A's passing result must never be borrowed by B.
	if _, e := s.db.Exec("UPDATE conversion_results SET core='failed',candidate_hash='' WHERE hash=?", repair.Digest([]byte(rawB))); e != nil {
		t.Fatal(e)
	}
	app, _ := NewServer(s, "http://127.0.0.1:18191")
	h := app.Handler()
	assertExportLines(t, h, app.exportURL(a, false), wantA)
	assertExportLines(t, h, app.exportURL(b, false), map[string][]byte{})
	rows, e := querySourceConversions(context.Background(), s.db, b, 0, true)
	if e != nil || len(rows) != 1 || rows[0].Available || rows[0].Hash != repair.Digest([]byte(rawB)) {
		t.Fatal("wrong owner's conversion chosen", e)
	}
	if exportRequest(t, h, "GET", app.exportURL(b, true)).Code != 204 {
		t.Fatal("borrowed other source candidate")
	}
	if exportRequest(t, h, "GET", app.origin+"/output/json/"+rows[0].ID+"?source="+b).Code != 404 {
		t.Fatal("scoped individual copy borrowed other source")
	}
}
func TestSourceRoundLifecycleAndCascade(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	a, _ := newExportSource(t, s, "A", sourceURI(1), sourceURI(2))
	b, wantB := newExportSource(t, s, "B", sourceURI(1), sourceURI(3))
	app, _ := NewServer(s, "http://127.0.0.1:18191")
	h := app.Handler()
	old, e := s.prepareScopedRotation(ctx, a)
	if e != nil {
		t.Fatal(e)
	}
	firstB, e := s.nextSourceConversion(ctx, b)
	if e != nil {
		t.Fatal(e)
	}
	newA := fillExportSource(t, s, a, sourceURI(4))
	if claimed, e := s.claimRotation(ctx, old); e != nil || claimed {
		t.Fatal("source replacement exported stale bytes", e)
	}
	assertExportLines(t, h, app.exportURL(a, false), newA)
	assertExportLines(t, h, app.exportURL(b, false), wantB)
	if _, e = s.ManageSources(ctx, SourceBatch{Action: "disable", Items: []SourceSelection{{ID: a, Revision: 1}}}); e != nil {
		t.Fatal(e)
	}
	assertExportLines(t, h, app.exportURL(a, false), map[string][]byte{})
	if rr := exportRequest(t, h, "GET", app.exportURL(a, true)); rr.Code != 204 {
		t.Fatal("disabled source yielded config")
	}
	if e = s.Delete(ctx, a, 2); e != nil {
		t.Fatal(e)
	}
	for _, url := range []string{app.exportURL(a, false), app.exportURL(a, true), app.origin + "/api/conversions?source=" + a} {
		if exportRequest(t, h, "GET", url).Code != 404 {
			t.Fatal("deleted source route alive")
		}
	}
	var n int
	s.db.QueryRow("SELECT count(*) FROM export_rotation_state WHERE scope=?", a).Scan(&n)
	if n != 0 {
		t.Fatal("source state did not cascade")
	}
	s.db.QueryRow("SELECT count(*) FROM export_rotation_queue WHERE scope=?", a).Scan(&n)
	if n != 0 {
		t.Fatal("source queue did not cascade")
	}
	secondB, e := s.nextSourceConversion(ctx, b)
	if e != nil || secondB.Cycle != 1 || secondB.Conversion.CandidateHash == firstB.Conversion.CandidateHash {
		t.Fatal("delete A reset B", e)
	}
	assertExportLines(t, h, app.exportURL(b, false), wantB)
}
func TestSourceRotationLocksAreIndependentAndCancelable(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	a, _ := newExportSource(t, s, "A", sourceURI(1))
	b, _ := newExportSource(t, s, "B", sourceURI(2))
	unlock, e := s.lockRotation(ctx, a)
	if e != nil {
		t.Fatal(e)
	}
	bctx, bcancel := context.WithTimeout(ctx, time.Second)
	defer bcancel()
	if c, e := s.nextSourceConversion(bctx, b); e != nil || c == nil {
		t.Fatal("A lock blocked B", e)
	}
	actx, acancel := context.WithTimeout(ctx, 20*time.Millisecond)
	defer acancel()
	if _, e = s.nextSourceConversion(actx, a); e == nil {
		t.Fatal("canceled wait consumed a turn")
	}
	unlock()
	s.rotationMu.Lock()
	n := len(s.rotationLocks)
	s.rotationMu.Unlock()
	if n != 0 {
		t.Fatal("idle source locks leaked")
	}
}
func TestParallelSourceRoundsAcrossStores(t *testing.T) {
	s := testStore(t)
	a, wantA := newExportSource(t, s, "A", sourceURI(1), sourceURI(2), sourceURI(3))
	b, wantB := newExportSource(t, s, "B", sourceURI(1), sourceURI(4), sourceURI(5))
	all := map[string][]byte{}
	for k, v := range wantA {
		all[k] = v
	}
	for k, v := range wantB {
		all[k] = v
	}
	other, e := Open(filepath.Join(filepath.Dir(s.vault.Directory()), "panel.db"), s.key)
	if e != nil {
		t.Fatal(e)
	}
	defer other.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	expected := map[string]map[string][]byte{a: wantA, b: wantB, "": all}
	type result struct {
		scope string
		c     *rotatingConfig
		e     error
	}
	out := make(chan result, 22)
	var wg sync.WaitGroup
	for scope, configs := range expected {
		for i := 0; i < len(configs)*2; i++ {
			wg.Add(1)
			store := s
			if i%2 == 1 {
				store = other
			}
			go func() { defer wg.Done(); c, e := store.nextSourceConversion(ctx, scope); out <- result{scope, c, e} }()
		}
	}
	wg.Wait()
	close(out)
	seen := map[string]map[int64]map[string]bool{}
	for r := range out {
		if r.e != nil || r.c == nil {
			t.Fatal("concurrent scoped request", r.e)
		}
		c := r.c
		if !bytes.Equal(c.Bytes, expected[r.scope][c.Conversion.CandidateHash]) {
			t.Fatal("cross-source contamination")
		}
		if seen[r.scope] == nil {
			seen[r.scope] = map[int64]map[string]bool{}
		}
		if seen[r.scope][c.Cycle] == nil {
			seen[r.scope][c.Cycle] = map[string]bool{}
		}
		if seen[r.scope][c.Cycle][c.Conversion.CandidateHash] {
			t.Fatal("duplicate within scoped round")
		}
		seen[r.scope][c.Cycle][c.Conversion.CandidateHash] = true
	}
	for scope, rounds := range seen {
		if len(rounds) != 2 {
			t.Fatal("wrong scoped cycle count")
		}
		for _, hashes := range rounds {
			if len(hashes) != len(expected[scope]) {
				t.Fatal("incomplete scoped cycle")
			}
		}
	}
}
func TestSchema7MigrationPreservesGlobalQueue(t *testing.T) {
	s := testStore(t)
	seedRotation(t, s, 1, 3)
	nextRotation(t, s)
	pending, e := s.prepareRotation(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	// Recreate the old schema7 rotation layout with an already-started round.
	_, e = s.db.Exec(`ALTER TABLE export_rotation_queue RENAME TO scoped_queue;ALTER TABLE export_rotation_state RENAME TO scoped_state;
 CREATE TABLE export_rotation_state(id INTEGER PRIMARY KEY,cycle INTEGER NOT NULL);INSERT INTO export_rotation_state SELECT 1,cycle FROM scoped_state WHERE scope='';
 CREATE TABLE export_rotation_queue(position INTEGER PRIMARY KEY,candidate_hash TEXT NOT NULL UNIQUE);INSERT INTO export_rotation_queue SELECT position,candidate_hash FROM scoped_queue WHERE scope='';
 DROP TABLE scoped_queue;DROP TABLE scoped_state;PRAGMA user_version=7;`)
	if e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(filepath.Dir(s.vault.Directory()), "panel.db")
	key := s.key
	s.Close()
	s, e = Open(path, key)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	got := nextRotation(t, s)
	if got.Cycle != pending.Cycle || got.Conversion.CandidateHash != pending.Conversion.CandidateHash {
		t.Fatal("migration reset global rotation")
	}
	var version int
	s.db.QueryRow("PRAGMA user_version").Scan(&version)
	if version != 8 {
		t.Fatal("schema version", version)
	}
}
