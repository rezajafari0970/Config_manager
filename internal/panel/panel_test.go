package panel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rezajafari0970/Config_manager/internal/collect"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	s, e := Open(filepath.Join(t.TempDir(), "data", "panel.db"), bytes.Repeat([]byte{7}, 32))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
func testSource() SourceInput {
	return SourceInput{Name: "", URL: "https://example.com/Api/Verde/trun.php?secret=fixture", Min: 2, Max: 3, Unit: "minutes", Enabled: true}
}
func TestIntervalNamingAndNoBusinessCaps(t *testing.T) {
	for raw, want := range map[string]string{"https://iran.fitn1.ir/Api/Verde/trun.php": "Verde", "https://example.com/Api/Alan/main.php?url=https://other.example/a": "Alan", "https://example.com/": "example.com", "https://example.com/Api/Rosa/": "Rosa"} {
		if got := AutoName(raw); got != want {
			t.Errorf("%s: %s", raw, got)
		}
	}
	for _, text := range []string{"۲ تا ۳", "٢ - ٣", "2–3", "2—3"} {
		a, b, e := parseInterval(text)
		if e != nil || a != 2 || b != 3 {
			t.Fatal(text, a, b, e)
		}
	}
	in := testSource()
	in.Name = strings.Repeat("n", 2000)
	in.URL = "https://example.com/" + strings.Repeat("x", 10000)
	in.Min = 50000
	in.Max = 100000
	if _, e := validateSource(in); e != nil {
		t.Fatal("legacy business cap remains", e)
	}
	for _, raw := range []string{"file:///etc/passwd", "https://", "https://example.com/\nhttps://example.net/"} {
		in = testSource()
		in.URL = raw
		if _, e := validateSource(in); e == nil {
			t.Fatal("invalid source accepted")
		}
	}
}
func TestEncryptedSourceAndRevisionIsolation(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	in := testSource()
	id, e := s.Save(ctx, "", 0, in)
	if e != nil {
		t.Fatal(e)
	}
	got, e := s.Get(ctx, id)
	if e != nil || got.URL != in.URL || got.Name != "Verde" {
		t.Fatal("save mismatch", e)
	}
	if _, e = s.Save(ctx, "", 0, in); !errors.Is(e, ErrConflict) {
		t.Fatal("duplicate accepted")
	}
	results := make(chan error, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); <-start; _, e := s.Save(ctx, id, 1, in); results <- e }()
	}
	close(start)
	wg.Wait()
	close(results)
	success, conflict := 0, 0
	for e := range results {
		if e == nil {
			success++
		} else if errors.Is(e, ErrConflict) {
			conflict++
		} else {
			t.Fatal(e)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatal("concurrent revision escaped")
	}
	if e = s.Delete(ctx, id, 1); !errors.Is(e, ErrConflict) {
		t.Fatal("stale deletion")
	}
	if e = s.Delete(ctx, id, 2); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Save(ctx, id, 2, in); !errors.Is(e, ErrConflict) {
		t.Fatal("deleted source resurrected")
	}
}
func TestMigrationAndReopen(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "data", "panel.db")
	key := bytes.Repeat([]byte{9}, 32)
	s, e := Open(path, key)
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.db.Exec(`DROP VIEW conversion_inputs;DROP TABLE conversion_results;DROP TABLE conversion_jobs;DROP TABLE conversion_settings;DROP TABLE conversion_worker;DROP TABLE repairs;DROP TABLE config_controls;DROP TABLE staged;DROP TABLE memberships;DROP TABLE artifacts;DROP TABLE responses;DROP TABLE settings;DROP TABLE sources;CREATE TABLE sources(id TEXT PRIMARY KEY,name TEXT NOT NULL,url BLOB NOT NULL,url_hash TEXT NOT NULL UNIQUE,display_url TEXT NOT NULL,min INTEGER NOT NULL,max INTEGER NOT NULL,unit TEXT NOT NULL,enabled INTEGER NOT NULL,revision INTEGER NOT NULL,created INTEGER NOT NULL,updated INTEGER NOT NULL,next_run INTEGER NOT NULL);CREATE TABLE admin(id INTEGER PRIMARY KEY,password_hash TEXT);CREATE TABLE sessions(hash TEXT PRIMARY KEY,expires INTEGER);CREATE TABLE attempts(key TEXT PRIMARY KEY,count INTEGER,window INTEGER);PRAGMA user_version=1;`)
	if e != nil {
		t.Fatal(e)
	}
	raw := "https://example.com/private-token?key=fixture-secret"
	_, e = s.db.Exec("INSERT INTO sources VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)", "legacy", "legacy", s.seal("legacy", []byte(raw)), s.digest("source-url", raw), "https://example.com/…", 15, 20, "minutes", 1, 3, 1, 1, 1)
	if e != nil {
		t.Fatal(e)
	}
	s.Close()
	s, e = Open(path, key)
	if e != nil {
		t.Fatal(e)
	}
	got, e := s.Get(ctx, "legacy")
	if e != nil || got.URL != raw || got.Revision != 3 {
		t.Fatal("migration lost source", e)
	}
	var n int
	s.db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name IN ('admin','sessions','attempts')").Scan(&n)
	if n != 0 {
		t.Fatal("password tables retained")
	}
	s.Close()
	for _, file := range []string{path, path + "-wal"} {
		data, _ := os.ReadFile(file)
		if bytes.Contains(data, []byte("fixture-secret")) || bytes.Contains(data, []byte("private-token")) {
			t.Fatal("source secret stored plaintext")
		}
	}
	if other, e := Open(path, bytes.Repeat([]byte{8}, 32)); e == nil {
		other.Close()
		t.Fatal("wrong master key accepted")
	}
}
func claimOne(t *testing.T, s *Store, id string) collect.Job {
	t.Helper()
	if e := s.RunNow(context.Background(), id); e != nil {
		t.Fatal(e)
	}
	jobs, e := s.Claim(context.Background(), time.Now().Add(time.Second), 1, time.Minute)
	if e != nil || len(jobs) != 1 || jobs[0].ID != id {
		t.Fatal("claim mismatch", jobs, e)
	}
	return jobs[0]
}
func fixtureResult(t *testing.T, s *Store, raw string) collect.Result {
	t.Helper()
	ref, e := s.vault.Put(context.Background(), strings.NewReader(raw))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.vault.Release(ref.Hash) })
	identity := collect.Fingerprint([]byte(raw), true, "vless", ref.Hash)
	return collect.Result{State: "updated", HTTPStatus: 200, Raw: ref, ETag: `"fixture"`, Extracted: 1, Items: []collect.Item{{Ref: ref, Identity: identity, Kind: "vless"}}}
}
func TestAtomicGenerationGlobalDedupAndSourceIndependence(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	a, _ := s.Save(ctx, "", 0, testSource())
	j := claimOne(t, s, a)
	result := fixtureResult(t, s, "vless://synthetic@vpn.example:443#original\r\n")
	if e := s.Complete(ctx, j, result, time.Now()); e != nil {
		t.Fatal(e)
	}
	in := testSource()
	in.URL = "https://example.net/Api/Rosa/trun.php"
	b, _ := s.Save(ctx, "", 0, in)
	j = claimOne(t, s, b)
	if e := s.Complete(ctx, j, result, time.Now()); e != nil {
		t.Fatal(e)
	}
	items, e := s.Outputs(ctx, "", 0, true)
	if e != nil || len(items) != 1 || items[0].Sources != 2 {
		t.Fatal("global dedup failed", items, e)
	}
	if e = s.Delete(ctx, a, 1); e != nil {
		t.Fatal(e)
	}
	items, _ = s.Outputs(ctx, "", 0, true)
	if len(items) != 1 || items[0].Sources != 1 {
		t.Fatal("one deletion removed another source config")
	}
	j = claimOne(t, s, b)
	in.URL = "https://example.net/Api/Changed/trun.php"
	if _, e = s.Save(ctx, b, 1, in); e != nil {
		t.Fatal(e)
	}
	if e = s.Complete(ctx, j, result, time.Now()); !errors.Is(e, ErrConflict) {
		t.Fatal("stale job committed")
	}
	items, _ = s.Outputs(ctx, "", 0, true)
	if len(items) != 0 {
		t.Fatal("changed URL kept old membership")
	}
}
func TestFixedTTLReplacementEmptyAnd304(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	id, _ := s.Save(ctx, "", 0, testSource())
	first := time.Now()
	j := claimOne(t, s, id)
	result := fixtureResult(t, s, "vless://synthetic@vpn.example:443#original\r\n")
	if e := s.Complete(ctx, j, result, first); e != nil {
		t.Fatal(e)
	}
	items, _ := s.Outputs(ctx, "", 0, true)
	expiry := items[0].Expires
	j = claimOne(t, s, id)
	if j.ETag == "" {
		t.Fatal("HTTP validator absent")
	}
	if e := s.Complete(ctx, j, result, first.Add(time.Hour)); e != nil {
		t.Fatal(e)
	}
	items, _ = s.Outputs(ctx, "", 0, true)
	if items[0].Expires != expiry {
		t.Fatal("repeat extended TTL")
	}
	j = claimOne(t, s, id)
	if e := s.Complete(ctx, j, collect.Result{State: "empty", HTTPStatus: 200}, first.Add(90*time.Minute)); e != nil {
		t.Fatal(e)
	}
	items, _ = s.Outputs(ctx, "", 0, true)
	if len(items) != 1 {
		t.Fatal("empty erased retained raw config")
	}
	j = claimOne(t, s, id)
	if e := s.Complete(ctx, j, collect.Result{State: "unchanged", HTTPStatus: 304}, first.Add(100*time.Minute)); e != nil {
		t.Fatal(e)
	}
	items, _ = s.Outputs(ctx, "", 0, true)
	if items[0].Expires != expiry {
		t.Fatal("304 extended TTL")
	}
	if e := s.Expire(ctx, first.Add(2*time.Hour)); e != nil {
		t.Fatal(e)
	}
	items, _ = s.Outputs(ctx, "", 0, true)
	if len(items) != 0 {
		t.Fatal("expired output remains")
	}
	j = claimOne(t, s, id)
	if j.ETag != "" {
		t.Fatal("expired cache requested 304")
	}
}
func TestSuccessfulGenerationReplacesOldAndRecovery(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	id, _ := s.Save(ctx, "", 0, testSource())
	j := claimOne(t, s, id)
	if e := s.Complete(ctx, j, fixtureResult(t, s, "vless://one@vpn.example:443#one"), time.Now()); e != nil {
		t.Fatal(e)
	}
	j = claimOne(t, s, id)
	if e := s.Complete(ctx, j, fixtureResult(t, s, "vless://two@vpn.example:443#two"), time.Now()); e != nil {
		t.Fatal(e)
	}
	items, _ := s.Outputs(ctx, "", 0, true)
	if len(items) != 1 {
		t.Fatal("generation appended rather than replaced")
	}
	if _, e := s.Claim(ctx, time.Now(), 1, time.Minute); e != nil {
		t.Fatal(e)
	}
	j = claimOne(t, s, id)
	if e := s.Recover(ctx); e != nil {
		t.Fatal(e)
	}
	jobs, e := s.Claim(ctx, time.Now().Add(time.Second), 1, time.Minute)
	if e != nil || len(jobs) != 1 || jobs[0].Token == j.Token {
		t.Fatal("interrupted lease not recovered")
	}
	if e = s.Complete(ctx, j, collect.Result{State: "empty"}, time.Now()); !errors.Is(e, ErrConflict) {
		t.Fatal("pre-restart completion accepted")
	}
}
func request(app http.Handler, method, path, body, csrf string, cookies []*http.Cookie, origin string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "https://panel.example"+path, strings.NewReader(body))
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	r.Header.Set("X-CSRF-Token", csrf)
	r.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	app.ServeHTTP(w, r)
	return w
}
func TestPasswordlessBulkAPIAndCSRF(t *testing.T) {
	s := testStore(t)
	app, e := NewServer(s, "https://panel.example")
	if e != nil {
		t.Fatal(e)
	}
	handler := app.Handler()
	w := request(handler, "GET", "/input", "", "", nil, "")
	if w.Code != 200 || strings.Contains(w.Body.String(), "type=\"password\"") {
		t.Fatal("login still required")
	}
	match := regexp.MustCompile(`name="csrf-token" content="([^"]+)"`).FindStringSubmatch(w.Body.String())
	if len(match) != 2 {
		t.Fatal("CSRF token missing")
	}
	csrf := match[1]
	cookies := w.Result().Cookies()
	if !cookies[0].Secure || !cookies[0].HttpOnly {
		t.Fatal("weak workspace cookie")
	}
	w = request(handler, "GET", "/api/state", "", "", nil, "")
	if w.Code != 200 {
		t.Fatal("state unexpectedly authenticated")
	}
	path := "/api/sources/import?" + url.Values{"interval": {"۲ تا ۳"}, "unit": {"minutes"}}.Encode()
	links := "https://example.com/Api/Verde/trun.php\nhttps://example.com/Api/Rosa/trun.php\nhttps://example.com/Api/Verde/trun.php\ninvalid"
	w = request(handler, "POST", path, links, csrf, cookies, "https://evil.example")
	if w.Code != 403 {
		t.Fatal("cross-site write allowed")
	}
	w = request(handler, "POST", path, links, csrf, cookies, "https://panel.example")
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	var result struct {
		Added      int   `json:"added"`
		Duplicates int   `json:"duplicates"`
		Invalid    []int `json:"invalid_lines"`
	}
	json.Unmarshal(w.Body.Bytes(), &result)
	if result.Added != 2 || result.Duplicates != 1 || len(result.Invalid) != 1 || result.Invalid[0] != 4 {
		t.Fatal(result)
	}
	items, _ := s.List(context.Background())
	if len(items) != 2 {
		t.Fatal("bulk failed")
	}
	for _, item := range items {
		if item.Min != 2 || item.Max != 3 || (item.Name != "Verde" && item.Name != "Rosa") {
			t.Fatal(item)
		}
	}
	var bulk strings.Builder
	for i := 0; i < 1003; i++ {
		fmt.Fprintf(&bulk, "https://example.net/Api/Source%d/trun.php\n", i)
	}
	w = request(handler, "POST", path, bulk.String(), csrf, cookies, "https://panel.example")
	if w.Code != 200 {
		t.Fatal("legacy count/body cap", w.Code)
	}
	stats, _ := s.Stats(context.Background())
	if stats.Sources != 1005 {
		t.Fatal("source count limited", stats.Sources)
	}
}

type controlledDriver struct {
	mu          sync.Mutex
	active, max int
	slow        chan struct{}
	started     chan struct{}
	once        sync.Once
}

func (*controlledDriver) Name() string { return "test.v1" }
func (d *controlledDriver) Fetch(ctx context.Context, j collect.Job) (collect.Response, error) {
	d.mu.Lock()
	d.active++
	if d.active > d.max {
		d.max = d.active
	}
	d.mu.Unlock()
	defer func() { d.mu.Lock(); d.active--; d.mu.Unlock() }()
	if strings.Contains(j.URL, "Slow") {
		d.once.Do(func() { close(d.started) })
		select {
		case <-ctx.Done():
			return collect.Response{}, ctx.Err()
		case <-d.slow:
		}
	}
	select {
	case <-d.started:
	case <-ctx.Done():
		return collect.Response{}, ctx.Err()
	}
	if strings.Contains(j.URL, "Panic") {
		panic("synthetic plugin panic")
	}
	return collect.Response{Status: 200, Body: io.NopCloser(strings.NewReader("vless://synthetic@vpn.example:443#exact\r\n"))}, nil
}
func TestEngineParallelSlowSourceCancellationAndPanicRecovery(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	driver := &controlledDriver{slow: make(chan struct{}), started: make(chan struct{})}
	engine := collect.NewEngine(s, s.vault, driver)
	in := testSource()
	in.URL = "https://example.com/Api/Slow/trun.php"
	slow, _ := s.Save(ctx, "", 0, in)
	in.URL = "https://example.com/Api/Fast/trun.php"
	fast, _ := s.Save(ctx, "", 0, in)
	in.URL = "https://example.com/Api/Panic/trun.php"
	panicID, _ := s.Save(ctx, "", 0, in)
	done := make(chan error, 1)
	go func() { done <- engine.Run(ctx) }()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		v, _ := s.Get(ctx, fast)
		p, _ := s.Get(ctx, panicID)
		if v.State == "updated" && p.ErrorCode == "worker_recovered" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	v, _ := s.Get(ctx, fast)
	p, _ := s.Get(ctx, panicID)
	if v.State != "updated" || p.ErrorCode != "worker_recovered" {
		t.Fatal("slow/panicking source blocked independent work", v.State, p.ErrorCode)
	}
	if e := s.Delete(ctx, slow, 1); e != nil {
		t.Fatal(e)
	}
	engine.Wake(slow, true)
	cancel()
	select {
	case e := <-done:
		if e != nil {
			t.Fatal(e)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("engine did not stop cleanly")
	}
	driver.mu.Lock()
	max := driver.max
	driver.mu.Unlock()
	if max < 2 {
		t.Fatal("fetches did not overlap")
	}
	if _, e := s.Get(context.Background(), slow); !errors.Is(e, ErrNotFound) {
		t.Fatal("deleted source returned")
	}
}

func TestSharedIdentityPreservesEachSourcesOriginalRepresentation(t *testing.T) {
	ctx := context.Background()
	s := testStore(t)
	a, _ := s.Save(ctx, "", 0, testSource())
	j := claimOne(t, s, a)
	first := fixtureResult(t, s, "vless://synthetic@vpn.example:443#Label-A\r\n")
	if e := s.Complete(ctx, j, first, time.Now()); e != nil {
		t.Fatal(e)
	}
	in := testSource()
	in.URL = "https://example.net/Api/B/trun.php"
	b, _ := s.Save(ctx, "", 0, in)
	j = claimOne(t, s, b)
	second := fixtureResult(t, s, "vless://synthetic@vpn.example:443#Label-B\r\n")
	if first.Items[0].Identity != second.Items[0].Identity {
		t.Fatal("fixture identity mismatch")
	}
	if e := s.Complete(ctx, j, second, time.Now()); e != nil {
		t.Fatal(e)
	}
	if e := s.Delete(ctx, a, 1); e != nil {
		t.Fatal(e)
	}
	items, e := s.Outputs(ctx, "", 0, true)
	if e != nil || len(items) != 1 || items[0].Hash != second.Raw.Hash {
		t.Fatal("deleting one source changed the other's raw bytes", e)
	}
	reader, e := s.vault.Open(items[0].Hash)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := io.ReadAll(reader)
	reader.Close()
	if e != nil || string(raw) != "vless://synthetic@vpn.example:443#Label-B\r\n" {
		t.Fatal("remaining source bytes changed", e)
	}
}
