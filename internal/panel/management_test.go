package panel

import (
	"context"
	"errors"
	"github.com/rezajafari0970/Config_manager/internal/collect"
	"testing"
	"time"
)

func TestBatchSourcesAtomicRevisionAndActiveJobIsolation(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	ids := []string{}
	for _, name := range []string{"A", "B", "C"} {
		in := testSource()
		in.URL = "https://example.com/" + name
		id, e := s.Save(ctx, "", 0, in)
		if e != nil {
			t.Fatal(e)
		}
		ids = append(ids, id)
	}
	jobs, e := s.Claim(ctx, time.Now().Add(time.Second), 3, time.Minute)
	if e != nil || len(jobs) != 3 {
		t.Fatal(e, len(jobs))
	}
	byID := map[string]collect.Job{}
	for _, j := range jobs {
		byID[j.ID] = j
	}
	_, e = s.ManageSources(ctx, SourceBatch{Action: "disable", Items: []SourceSelection{{ids[0], 1}, {ids[1], 99}}})
	if !errors.Is(e, ErrConflict) {
		t.Fatal("stale batch accepted", e)
	}
	a, _ := s.Get(ctx, ids[0])
	if !a.Enabled || a.Revision != 1 {
		t.Fatal("batch was not atomic")
	}
	if _, e = s.ManageSources(ctx, SourceBatch{Action: "disable", Items: []SourceSelection{{ids[0], 1}}}); e != nil {
		t.Fatal(e)
	}
	result := fixtureResult(t, s, "vless://synthetic@vpn.example:443#raw\r\n")
	if e = s.Complete(ctx, byID[ids[0]], result, time.Now()); !errors.Is(e, ErrConflict) {
		t.Fatal("disabled job revived source", e)
	}
	if e = s.Complete(ctx, byID[ids[1]], result, time.Now()); e != nil {
		t.Fatal("unrelated source blocked", e)
	}
	if _, e = s.ManageSources(ctx, SourceBatch{Action: "delete", All: true}); e != nil {
		t.Fatal(e)
	}
	if e = s.Complete(ctx, byID[ids[2]], result, time.Now()); !errors.Is(e, ErrConflict) {
		t.Fatal("deleted job revived source", e)
	}
	st, _ := s.Stats(ctx)
	if st.Sources != 0 || st.Configs != 0 {
		t.Fatal(st)
	}
}

func TestConfigDisableDeleteAndClearDoNotResurrect(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	a, _ := s.Save(ctx, "", 0, testSource())
	in := testSource()
	in.URL = "https://example.net/Api/B/trun.php"
	b, _ := s.Save(ctx, "", 0, in)
	first := fixtureResult(t, s, "vless://one@vpn.example:443#Keep original\r\n")
	second := fixtureResult(t, s, "vless://two@vpn.example:443#Original 2\r\n")
	jobs, e := s.Claim(ctx, time.Now().Add(time.Second), 2, time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	for _, j := range jobs {
		r := first
		if j.ID == b {
			r = second
		}
		if e = s.Complete(ctx, j, r, time.Now()); e != nil {
			t.Fatal(e)
		}
	}
	id := first.Items[0].Identity
	other := second.Items[0].Identity
	if _, e = s.ManageConfigs(ctx, ConfigBatch{Action: "disable", IDs: []string{id, id}}); e != nil {
		t.Fatal(e)
	}
	original, e := s.Output(ctx, id)
	if e != nil || !original.Disabled || original.Hash != first.Raw.Hash {
		t.Fatal("disable changed original", e, original)
	}
	n := 0
	if e = s.WalkOutputs(ctx, func(a Artifact) error {
		n++
		if a.ID == id {
			t.Fatal("disabled config exported")
		}
		return nil
	}); e != nil || n != 1 {
		t.Fatal(n, e)
	}
	j := claimOne(t, s, a)
	if e = s.Complete(ctx, j, first, time.Now()); e != nil {
		t.Fatal(e)
	}
	original, _ = s.Output(ctx, id)
	if !original.Disabled {
		t.Fatal("fetch reenabled config")
	}
	j = claimOne(t, s, b)
	if _, e = s.ManageConfigs(ctx, ConfigBatch{Action: "delete", IDs: []string{other}}); e != nil {
		t.Fatal(e)
	}
	if e = s.Complete(ctx, j, second, time.Now()); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Output(ctx, other); !errors.Is(e, ErrNotFound) {
		t.Fatal("inflight fetch resurrected deletion", e)
	}
	if _, e = s.ManageConfigs(ctx, ConfigBatch{Action: "enable", IDs: []string{id}}); e != nil {
		t.Fatal(e)
	}
	original, _ = s.Output(ctx, id)
	if original.Disabled {
		t.Fatal("enable failed")
	}
	j = claimOne(t, s, a)
	if _, e = s.ManageConfigs(ctx, ConfigBatch{Action: "delete", All: true}); e != nil {
		t.Fatal(e)
	}
	if e = s.Complete(ctx, j, first, time.Now()); !errors.Is(e, ErrConflict) {
		t.Fatal("clear did not reject old job", e)
	}
	st, _ := s.Stats(ctx)
	if st.Sources != 2 || st.Configs != 0 {
		t.Fatal("clear removed sources or retained configs", st)
	}
	if _, e = s.Response(ctx, a); !errors.Is(e, ErrNotFound) {
		t.Fatal("clear retained raw responses")
	}
	if e = s.Expire(ctx, time.Now().Add(3*time.Hour)); e != nil {
		t.Fatal(e)
	}
	var controls int
	s.db.QueryRow("SELECT count(*) FROM config_controls").Scan(&controls)
	if controls != 0 {
		t.Fatal("controls retained forever")
	}
}

func TestCorruptSourceAndTimeoutAreIndependent(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	in := testSource()
	bad, _ := s.Save(ctx, "", 0, in)
	in.URL = "https://example.net/source"
	in.TimeoutSeconds = 83
	good, _ := s.Save(ctx, "", 0, in)
	if _, e := s.db.Exec("UPDATE sources SET url=x'00' WHERE id=?", bad); e != nil {
		t.Fatal(e)
	}
	now := time.Now().Add(time.Second)
	jobs, e := s.Claim(ctx, now, 2, time.Minute)
	if e != nil || len(jobs) != 1 || jobs[0].ID != good || jobs[0].TimeoutSeconds != 83 {
		t.Fatal("corrupt source blocked another", jobs, e)
	}
	var lease int64
	if e = s.db.QueryRow("SELECT lease_until FROM sources WHERE id=?", good).Scan(&lease); e != nil || lease < now.Add(113*time.Second).UnixMilli() {
		t.Fatal("lease ignores timeout", e, lease)
	}
	var state, code string
	s.db.QueryRow("SELECT state,error_code FROM sources WHERE id=?", bad).Scan(&state, &code)
	if state != "error" || code != "source_data_error" {
		t.Fatal(state, code)
	}
}

func TestManagementRequiresCSRFAndExplicitSelection(t *testing.T) {
	s := testStore(t)
	app, _ := NewServer(s, "https://panel.example")
	handler := app.Handler()
	for _, path := range []string{"/api/sources/batch", "/api/configs/batch"} {
		w := request(handler, "POST", path, `{"action":"delete","all":true}`, "", nil, "https://attacker.example")
		if w.Code != 403 {
			t.Fatal("destructive CSRF", path, w.Code)
		}
	}
	if _, e := s.ManageConfigs(context.Background(), ConfigBatch{Action: "delete"}); !errors.Is(e, ErrInvalid) {
		t.Fatal("implicit global deletion")
	}
	if _, e := s.ManageSources(context.Background(), SourceBatch{Action: "delete"}); !errors.Is(e, ErrInvalid) {
		t.Fatal("implicit global deletion")
	}
}

func TestPerSourceTimeoutDoesNotBlockOtherSources(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	driver := &controlledDriver{slow: make(chan struct{}), started: make(chan struct{})}
	engine := collect.NewEngine(s, s.vault, driver)
	in := testSource()
	in.URL = "https://example.com/Slow"
	in.TimeoutSeconds = 1
	slow, e := s.Save(ctx, "", 0, in)
	if e != nil {
		t.Fatal(e)
	}
	in.URL = "https://example.com/Fast"
	in.TimeoutSeconds = 0
	fast, e := s.Save(ctx, "", 0, in)
	if e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	go func() { done <- engine.Run(ctx) }()
	defer func() {
		cancel()
		select {
		case e := <-done:
			if e != nil {
				t.Error(e)
			}
		case <-time.After(3 * time.Second):
			t.Error("worker shutdown timeout")
		}
	}()
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		a, _ := s.Get(ctx, slow)
		b, _ := s.Get(ctx, fast)
		if a.ErrorCode == "timeout" && b.State == "updated" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("per-source deadline or unrelated completion failed")
}
