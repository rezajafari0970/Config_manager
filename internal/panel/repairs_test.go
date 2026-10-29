package panel

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/collect"
	"github.com/rezajafari0970/Config_manager/internal/repair"
	"io"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDerivedRepairLifecycleAndRawPreservation(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	id, e := s.Save(ctx, "", 0, testSource())
	if e != nil {
		t.Fatal(e)
	}
	job := claimOne(t, s, id)
	raw := []byte(`{"outbounds":[{"protocol":"vless","settings":{}}],"stats":[],"opaque":9007199254740993}`)
	d := repair.NewRegistry().Propose(raw, true, "json", "invalid")
	if d.Status != "repaired" {
		t.Fatal(d)
	}
	ref, e := s.vault.Put(ctx, bytes.NewReader(raw))
	if e != nil {
		t.Fatal(e)
	}
	defer s.vault.Release(ref.Hash)
	candidate, e := s.vault.Put(ctx, bytes.NewReader(d.Candidate))
	if e != nil {
		t.Fatal(e)
	}
	defer s.vault.Release(candidate.Hash)
	item := collect.Item{Ref: ref, Identity: ref.Hash, Kind: "json", Quality: "invalid", Repair: d, Candidate: candidate}
	if _, e = s.Stage(ctx, job, item); e != nil {
		t.Fatal(e)
	}
	if used, e := s.Referenced(ctx, candidate.Hash); e != nil || !used {
		t.Fatal("staged candidate unprotected", e)
	}
	if e = s.Complete(ctx, job, collect.Result{State: "updated", Staged: true, Extracted: 1}, time.Now()); e != nil {
		t.Fatal(e)
	}
	a, e := s.Output(ctx, item.Identity)
	if e != nil || a.Hash != ref.Hash {
		t.Fatal("original overwritten", e)
	}
	body, e := s.vault.Open(a.Hash)
	if e != nil {
		t.Fatal(e)
	}
	got, _ := io.ReadAll(body)
	body.Close()
	if !bytes.Equal(got, raw) {
		t.Fatal("original bytes changed")
	}
	app, _ := NewServer(s, "http://127.0.0.1:18192")
	get := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "http://127.0.0.1:18192"+path, nil)
		app.Handler().ServeHTTP(w, r)
		return w
	}
	w := get("/output/repaired/" + a.ID)
	if w.Code != 200 || !bytes.Equal(w.Body.Bytes(), d.Candidate) || w.Header().Get("X-Original-SHA256") != a.Hash {
		t.Fatal("wrong derived download", w.Code)
	}
	w = get("/api/repairs/" + a.ID)
	var report repair.Diagnostic
	if json.Unmarshal(w.Body.Bytes(), &report) != nil || report.Status != "repaired" || report.CoreValidation != "not_run" {
		t.Fatal("wrong report")
	}
	filtered, e := s.OutputsFiltered(ctx, "", "repaired", 0, false)
	if e != nil || len(filtered) != 1 {
		t.Fatal("repair filter missing", e)
	}
	filtered, e = s.OutputsFiltered(ctx, "", "blocked", 0, false)
	if e != nil || len(filtered) != 0 {
		t.Fatal("repair filter leaked", e)
	}
	summary, e := s.RepairSummary(ctx, []Artifact{a})
	if e != nil || summary.Counts["repaired"] != 1 || summary.Sources[id]["repaired"] != 1 {
		t.Fatal(summary, e)
	}
	if _, e = s.ManageConfigs(ctx, ConfigBatch{Action: "disable", IDs: []string{a.ID}}); e != nil {
		t.Fatal(e)
	}
	if get("/output/repaired/"+a.ID).Code != 409 {
		t.Fatal("disabled repair exported")
	}
	if e = s.Expire(ctx, time.Now().Add(3*time.Hour)); e != nil {
		t.Fatal(e)
	}
	if get("/output/repaired/"+a.ID).Code != 404 || get("/api/repairs/"+a.ID).Code != 404 {
		t.Fatal("expired repair available")
	}
	if used, e := s.Referenced(ctx, candidate.Hash); e != nil || used {
		t.Fatal("orphan candidate retained", e)
	}
	var n int
	s.db.QueryRow("SELECT count(*) FROM repairs").Scan(&n)
	if n != 0 {
		t.Fatal("repair metadata outlives original")
	}
}

func TestAutomaticRepairMixedSourceCannotBecomeWorkerFailure(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	input := testSource()
	input.URL = "https://example.com/Verde"
	id, e := s.Save(ctx, "", 0, input)
	if e != nil {
		t.Fatal(e)
	}
	engine := collect.NewEngine(s, s.vault, browserDriver{})
	done := make(chan error, 1)
	go func() { done <- engine.Run(ctx) }()
	defer func() {
		cancel()
		if e := <-done; e != nil {
			t.Error(e)
		}
	}()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		source, e := s.Get(ctx, id)
		if e != nil {
			t.Fatal(e)
		}
		if source.FetchCount > 0 {
			if source.State != "updated" || source.ContentState != "partial" || source.ConfigCount != 2 {
				t.Fatal("mixed source worker failed", source.State, source.ErrorCode)
			}
			out, e := s.Outputs(ctx, "", 0, true)
			if e != nil {
				t.Fatal(e)
			}
			summary, e := s.RepairSummary(ctx, out)
			if e != nil || summary.Counts["repaired"] != 1 {
				t.Fatal("automatic repair missing", summary, e)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("automatic repair did not complete")
}
