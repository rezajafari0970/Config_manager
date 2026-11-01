package panel

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestSmartSingleReadyRoundSkipsRebuildAndNeverReturnsEmptySuccess(t *testing.T) {
	s, _ := smartFixture(t)
	ctx := context.Background()
	app, e := NewServer(s, "http://127.0.0.1:18191")
	if e != nil {
		t.Fatal(e)
	}
	h := app.Handler()
	empty := exportRequest(t, h, http.MethodGet, app.smartURL(true))
	if empty.Code != http.StatusServiceUnavailable || empty.Header().Get("Retry-After") == "" || empty.Body.Len() == 0 {
		t.Fatal("empty cache is not explicit", empty.Code)
	}
	var cycle int64
	s.db.QueryRow("SELECT cycle FROM smart_rotation_state WHERE id=1").Scan(&cycle)
	if cycle != 0 {
		t.Fatal("empty request consumed round")
	}
	finishSmart(t, s)
	first, _, round, e := s.smartNext(ctx)
	if e != nil || first == nil {
		t.Fatal(e)
	}
	// Simulate a cache rebuilt in the background while a durable round is open.
	var queued string
	if e = s.db.QueryRow("SELECT head FROM smart_rotation_queue ORDER BY position LIMIT 1").Scan(&queued); e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec("UPDATE smart_heads SET state='queued' WHERE id=?", queued); e != nil {
		t.Fatal(e)
	}
	seen := map[string]bool{first.ID: true}
	for i := 0; i < 5; i++ {
		v, b, c, e := s.smartNext(ctx)
		if e != nil || v == nil || v.ID == queued || seen[v.ID] || c != round || !json.Valid(b) {
			t.Fatal("ready round", e)
		}
		seen[v.ID] = true
	}
	rr := exportRequest(t, h, http.MethodGet, app.smartURL(true))
	if rr.Code != 200 || rr.Header().Get("X-Config-Count") != "1" || rr.Header().Get("Content-Disposition") != "inline" {
		t.Fatal("warming head froze ready cache", rr.Code)
	}
	if !json.Valid(bytes.TrimSpace(rr.Body.Bytes())) || bytes.Count(rr.Body.Bytes(), []byte{'\n'}) != 1 {
		t.Fatal("single output is not one compact JSON line")
	}
	var q int
	s.db.QueryRow("SELECT count(*) FROM smart_rotation_queue").Scan(&q)
	exportRequest(t, h, http.MethodHead, app.smartURL(true))
	var after int
	s.db.QueryRow("SELECT count(*) FROM smart_rotation_queue").Scan(&after)
	if q != after {
		t.Fatal("HEAD consumed ready turn")
	}
}
