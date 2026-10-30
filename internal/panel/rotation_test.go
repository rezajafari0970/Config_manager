package panel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func seedRotation(t *testing.T, s *Store, start, count int) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	for i := start; i < start+count; i++ {
		raw := strings.Replace(conversionSample, "192.0.2.10", fmt.Sprintf("192.0.2.%d", i), 1)
		_, job := addConversion(t, s, raw)
		result := conversionResult(job, 0, true)
		if e := s.CompleteProbe(context.Background(), result); e != nil {
			t.Fatal(e)
		}
		out[result.SHA256] = result.Candidate
	}
	return out
}
func nextRotation(t *testing.T, s *Store) *rotatingConfig {
	t.Helper()
	c, e := s.NextConversion(context.Background())
	if e != nil || c == nil {
		t.Fatal("next config", e)
	}
	return c
}
func rotationPath(s *Store) string {
	return "http://127.0.0.1:18191/subscription/" + s.digest("subscription", "template-single-v1") + "/next.json"
}

func TestRotationCyclesRestartAndSchema6Upgrade(t *testing.T) {
	s := testStore(t)
	expected := seedRotation(t, s, 1, 3)
	path := filepath.Join(filepath.Dir(s.vault.Directory()), "panel.db")
	key := append([]byte(nil), s.key...)
	// Simulate an existing schema6 installation, preserving all active data.
	if _, e := s.db.Exec("DROP TABLE export_rotation_queue;DROP TABLE export_rotation_state;PRAGMA user_version=6;"); e != nil {
		t.Fatal(e)
	}
	s.Close()
	s, e := Open(path, key)
	if e != nil {
		t.Fatal(e)
	}
	var version int
	if e = s.db.QueryRow("PRAGMA user_version").Scan(&version); e != nil || version != 8 {
		t.Fatal("schema8", version, e)
	}
	first := nextRotation(t, s)
	if first.Cycle != 1 {
		t.Fatal("first cycle", first.Cycle)
	}
	s.Close()
	s, e = Open(path, key)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	seen := map[string]bool{first.Conversion.CandidateHash: true}
	for i := 0; i < 2; i++ {
		c := nextRotation(t, s)
		if c.Cycle != 1 || seen[c.Conversion.CandidateHash] || !bytes.Equal(c.Bytes, expected[c.Conversion.CandidateHash]) {
			t.Fatal("repeated/lost/changed after restart")
		}
		seen[c.Conversion.CandidateHash] = true
	}
	for i := 0; i < 3; i++ {
		c := nextRotation(t, s)
		if c.Cycle != 2 || !seen[c.Conversion.CandidateHash] {
			t.Fatal("reset before all configs or missed config")
		}
		delete(seen, c.Conversion.CandidateHash)
	}
	if len(seen) != 0 {
		t.Fatal("incomplete second round")
	}
}

func TestRotationConcurrentStoresAndGarbageCollection(t *testing.T) {
	s := testStore(t)
	expected := seedRotation(t, s, 1, 5)
	other, e := Open(filepath.Join(filepath.Dir(s.vault.Directory()), "panel.db"), s.key)
	if e != nil {
		t.Fatal(e)
	}
	defer other.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	type result struct {
		c *rotatingConfig
		e error
	}
	results := make(chan result, 25)
	var wg sync.WaitGroup
	for i := 0; i < 25; i++ {
		wg.Add(1)
		store := s
		if i%2 == 1 {
			store = other
		}
		go func() { defer wg.Done(); c, e := store.NextConversion(ctx); results <- result{c, e} }()
	}
	gcDone := make(chan error, 1)
	go func() { gcDone <- s.vault.Sweep(ctx, s.Referenced) }()
	wg.Wait()
	close(results)
	rounds := map[int64]map[string]bool{}
	for r := range results {
		if r.e != nil || r.c == nil {
			t.Fatal("parallel next", r.e)
		}
		c := r.c
		if !bytes.Equal(c.Bytes, expected[c.Conversion.CandidateHash]) {
			t.Fatal("wrong config")
		}
		if rounds[c.Cycle] == nil {
			rounds[c.Cycle] = map[string]bool{}
		}
		if rounds[c.Cycle][c.Conversion.CandidateHash] {
			t.Fatal("duplicate parallel claim", c.Cycle)
		}
		rounds[c.Cycle][c.Conversion.CandidateHash] = true
	}
	if e := <-gcDone; e != nil {
		t.Fatal("GC deadlock/failure", e)
	}
	if len(rounds) != 5 {
		t.Fatal("wrong round count", len(rounds))
	}
	for _, round := range rounds {
		if len(round) != 5 {
			t.Fatal("incomplete parallel round")
		}
	}
}

func TestRotationNewArrivalsWaitAndDisabledEntriesAreSkipped(t *testing.T) {
	s := testStore(t)
	seedRotation(t, s, 1, 3)
	first := nextRotation(t, s)
	next, e := s.prepareRotation(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	newItems := seedRotation(t, s, 10, 1)
	if _, e = s.ManageConfigs(context.Background(), ConfigBatch{Action: "disable", IDs: []string{next.Conversion.ID}}); e != nil {
		t.Fatal(e)
	}
	if claimed, e := s.claimRotation(context.Background(), next); e != nil || claimed {
		t.Fatal("disabled candidate claimed", e)
	}
	third := nextRotation(t, s)
	if third.Cycle != 1 || third.Conversion.CandidateHash == first.Conversion.CandidateHash || newItems[third.Conversion.CandidateHash] != nil {
		t.Fatal("new arrival interrupted current round")
	}
	seenNew := false
	for i := 0; i < 3; i++ {
		c := nextRotation(t, s)
		if c.Cycle != 2 || c.Conversion.CandidateHash == next.Conversion.CandidateHash {
			t.Fatal("bad second round")
		}
		seenNew = seenNew || newItems[c.Conversion.CandidateHash] != nil
	}
	if !seenNew {
		t.Fatal("new config never joined")
	}
	if claimed, e := s.claimRotation(context.Background(), next); e != nil || claimed {
		t.Fatal("stale round claim accepted", e)
	}
}

func TestRotationMissingBlobDeletionExpiryAndEmptyOutput(t *testing.T) {
	for _, reason := range []string{"blob", "delete", "expiry"} {
		t.Run(reason, func(t *testing.T) {
			s := testStore(t)
			seedRotation(t, s, 1, 2)
			c, e := s.prepareRotation(context.Background())
			if e != nil {
				t.Fatal(e)
			}
			switch reason {
			case "blob":
				e = os.Remove(filepath.Join(s.vault.Directory(), c.Conversion.CandidateHash+".blob"))
			case "delete":
				_, e = s.ManageConfigs(context.Background(), ConfigBatch{Action: "delete", IDs: []string{c.Conversion.ID}})
			case "expiry":
				_, e = s.db.Exec("UPDATE artifacts SET expires=0 WHERE id=?", c.Conversion.ID)
			}
			if e != nil {
				t.Fatal(e)
			}
			valid := nextRotation(t, s)
			if valid.Cycle != 1 || valid.Conversion.CandidateHash == c.Conversion.CandidateHash {
				t.Fatal("unavailable candidate served")
			}
			if _, e = s.ManageConfigs(context.Background(), ConfigBatch{Action: "disable", All: true}); e != nil {
				t.Fatal(e)
			}
			server, _ := NewServer(s, "http://127.0.0.1:18191")
			rr := httptest.NewRecorder()
			server.Handler().ServeHTTP(rr, httptest.NewRequest("GET", rotationPath(s), nil))
			if rr.Code != 204 || rr.Body.Len() != 0 || rr.Header().Get("X-Config-Count") != "0" {
				t.Fatal("empty response must not contain a config or wrapper")
			}
		})
	}
}

func TestRotationHTTPNonConsumingRequestsAndExactSingleJSON(t *testing.T) {
	s := testStore(t)
	expected := seedRotation(t, s, 1, 3)
	server, _ := NewServer(s, "http://127.0.0.1:18191")
	handler := server.Handler()
	path := rotationPath(s)
	for _, kind := range []string{"head", "prefetch", "invalid", "all", "api"} {
		method, url := "GET", path
		if kind == "head" {
			method = "HEAD"
		}
		if kind == "invalid" {
			url = strings.Replace(path, s.digest("subscription", "template-single-v1"), "invalid", 1)
		}
		if kind == "all" {
			url = "http://127.0.0.1:18191/subscription/" + s.digest("subscription", "template-jsonl-v1") + "/configs.jsonl"
		}
		if kind == "api" {
			url = "http://127.0.0.1:18191/api/conversions"
		}
		req := httptest.NewRequest(method, url, nil)
		if kind == "prefetch" {
			req.Header.Set("Sec-Purpose", "prefetch;prerender")
		}
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if kind == "invalid" && rr.Code != 404 {
			t.Fatal("bad token accepted")
		}
		if kind == "api" {
			var state map[string]any
			if json.Unmarshal(rr.Body.Bytes(), &state) != nil || state["single_export_url"] != path {
				t.Fatal("missing single link")
			}
		}
	}
	var cycle int
	if e := s.db.QueryRow("SELECT cycle FROM export_rotation_state").Scan(&cycle); e != nil || cycle != 0 {
		t.Fatal("nonconsuming request advanced round", e)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := s.NextConversion(cancelled); e == nil {
		t.Fatal("cancelled request consumed")
	}
	seen := map[string]bool{}
	for i := 0; i < 4; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		hash := rr.Header().Get("X-Content-SHA256")
		if rr.Code != 200 || rr.Header().Get("Content-Type") != "text/plain; charset=utf-8" || rr.Header().Get("Content-Disposition") != "inline" || rr.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("HTTP contract")
		}
		if !bytes.Equal(rr.Body.Bytes(), append(append([]byte(nil), expected[hash]...), '\n')) || bytes.Count(rr.Body.Bytes(), []byte{'\n'}) != 1 {
			t.Fatal("not exact single line")
		}
		if i < 3 {
			if seen[hash] || rr.Header().Get("X-Export-Cycle") != "1" {
				t.Fatal("duplicate before round complete")
			}
			seen[hash] = true
		} else if rr.Header().Get("X-Export-Cycle") != "2" {
			t.Fatal("round did not reset")
		}
	}
}
