package panel

import (
	"bytes"
	"context"
	"errors"
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
	dir := filepath.Join(t.TempDir(), "data")
	s, err := Open(filepath.Join(dir, "panel.db"), bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
func testSource() SourceInput {
	return SourceInput{Name: "منبع آزمایشی", URL: "https://example.com/private/test-token?key=fixture-secret", Min: 15, Max: 20, Unit: "minutes", Enabled: true}
}
func TestSourceValidation(t *testing.T) {
	for _, raw := range []string{"file:///etc/passwd", "https://localhost/a", "http://127.0.0.1/x", "https://[::1]/x", "https://[::ffff:127.0.0.1]/", "http://169.254.169.254/", "http://10.1.2.3/", "http://100.64.0.1/", "https://example.com:99999/", "https://example.com/#private", "https://example.com/\nsecret"} {
		in := testSource()
		in.URL = raw
		if _, err := validateSource(in); err == nil {
			t.Errorf("accepted %q", raw)
		}
	}
	for _, in := range []SourceInput{{Name: "n", URL: "https://example.com", Min: 20, Max: 15, Unit: "minutes"}, {Name: "n", URL: "https://example.com", Min: 1, Max: 9223372036854775807, Unit: "hours"}} {
		if _, err := validateSource(in); err == nil {
			t.Fatal("invalid interval accepted")
		}
	}
	in := testSource()
	in.URL = "https://user:secret@example.com/private?token=hidden"
	display, err := validateSource(in)
	if err != nil || display != "https://example.com/…" {
		t.Fatal(display, err)
	}
}
func TestEncryptedPersistenceAndRevisions(t *testing.T) {
	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "data")
	path := filepath.Join(dir, "panel.db")
	key := bytes.Repeat([]byte{3}, 32)
	s, err := Open(path, key)
	if err != nil {
		t.Fatal(err)
	}
	in := testSource()
	id, err := s.Save(ctx, "", 0, in)
	if err != nil {
		t.Fatal(err)
	}
	original, err := s.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if original.NextRunAt.Before(time.Now().Add(14*time.Minute)) || original.NextRunAt.After(time.Now().Add(21*time.Minute)) {
		t.Fatal("deadline outside interval")
	}
	if _, err = s.Save(ctx, "", 0, in); !errors.Is(err, ErrConflict) {
		t.Fatal("duplicate accepted", err)
	}
	s.Close()
	for _, p := range []string{path, path + "-wal"} {
		data, _ := os.ReadFile(p)
		if bytes.Contains(data, []byte("fixture-secret")) || bytes.Contains(data, []byte("test-token")) {
			t.Fatal("source secret in plaintext database")
		}
	}
	s, err = Open(path, key)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	restored, err := s.Get(ctx, id)
	if err != nil || restored.URL != in.URL || !restored.NextRunAt.Equal(original.NextRunAt) {
		t.Fatal("restart changed data", err)
	}
	list, err := s.List(ctx)
	if err != nil || len(list) != 1 || list[0].URL != "" {
		t.Fatal("list exposed URL")
	}
	in.Name = "updated"
	if _, err = s.Save(ctx, id, 1, in); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Save(ctx, id, 1, in); !errors.Is(err, ErrConflict) {
		t.Fatal("stale edit accepted")
	}
	if err = s.Delete(ctx, id, 1); !errors.Is(err, ErrConflict) {
		t.Fatal("stale delete accepted")
	}
	if err = s.Delete(ctx, id, 2); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Save(ctx, id, 2, in); !errors.Is(err, ErrConflict) {
		t.Fatal("deleted source resurrected")
	}
	s.Close()
	if other, err := Open(path, bytes.Repeat([]byte{9}, 32)); err == nil {
		other.Close()
		t.Fatal("wrong master key accepted")
	}
}
func TestAuthPersistenceAndRevocation(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	password := "synthetic-password-12345"
	if err := s.InitAdmin(ctx, password); err != nil {
		t.Fatal(err)
	}
	if err := s.InitAdmin(ctx, password); err == nil {
		t.Fatal("admin overwritten")
	}
	if _, _, err := s.Authenticate(ctx, "wrong", false); !errors.Is(err, ErrAuth) {
		t.Fatal(err)
	}
	token, expiry, err := s.Authenticate(ctx, password, true)
	if err != nil || !s.Session(ctx, token) || time.Until(expiry) < 29*24*time.Hour {
		t.Fatal("session failed", err)
	}
	var stored string
	s.db.QueryRow("SELECT hash FROM sessions LIMIT 1").Scan(&stored)
	if stored == token {
		t.Fatal("plaintext session token")
	}
	if err = s.ChangePassword(ctx, "wrong", "another-test-password"); !errors.Is(err, ErrAuth) {
		t.Fatal(err)
	}
	if err = s.ChangePassword(ctx, password, "another-test-password"); err != nil {
		t.Fatal(err)
	}
	if s.Session(ctx, token) {
		t.Fatal("session survived password change")
	}
	token, _, err = s.Authenticate(ctx, "another-test-password", false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec("UPDATE sessions SET expires=?", time.Now().Add(-time.Second).Unix()); err != nil {
		t.Fatal(err)
	}
	if s.Session(ctx, token) {
		t.Fatal("expired session accepted")
	}
	token, _, _ = s.Authenticate(ctx, "another-test-password", false)
	if err = s.Logout(ctx, token); err != nil || s.Session(ctx, token) {
		t.Fatal("logout did not revoke")
	}
	for i := 0; i < 10; i++ {
		ok, err := s.AllowAuth(ctx, "192.0.2.1")
		if !ok || err != nil {
			t.Fatal(i, err)
		}
	}
	if ok, _ := s.AllowAuth(ctx, "192.0.2.1"); ok {
		t.Fatal("rate limit absent")
	}
	if ok, _ := s.AllowAuth(ctx, "192.0.2.2"); !ok {
		t.Fatal("client limits incorrectly shared")
	}
}

func request(app http.Handler, method, path string, form url.Values, cookies []*http.Cookie, origin string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "https://panel.example"+path, strings.NewReader(form.Encode()))
	r.RemoteAddr = "192.0.2.2:1234"
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	for _, c := range cookies {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	app.ServeHTTP(w, r)
	return w
}
func csrfFrom(t *testing.T, body string) string {
	t.Helper()
	m := regexp.MustCompile(`name="csrf" value="([^"]+)"`).FindStringSubmatch(body)
	if len(m) != 2 {
		t.Fatal("csrf missing")
	}
	return m[1]
}
func TestHTTPFullCRUDAndSecurity(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if err := s.InitAdmin(ctx, "synthetic-password-12345"); err != nil {
		t.Fatal(err)
	}
	server, err := NewServer(s, "https://panel.example")
	if err != nil {
		t.Fatal(err)
	}
	app := server.Handler()
	w := request(app, "GET", "/", nil, nil, "")
	if w.Code != 303 {
		t.Fatal("unauthenticated access", w.Code)
	}
	w = request(app, "GET", "/login", nil, nil, "")
	loginCookies := w.Result().Cookies()
	nonce := csrfFrom(t, w.Body.String())
	form := url.Values{"csrf": {nonce}, "password": {"synthetic-password-12345"}}
	w = request(app, "POST", "/login", form, loginCookies, "https://evil.example")
	if w.Code != 403 {
		t.Fatal("foreign origin accepted")
	}
	w = request(app, "POST", "/login", form, loginCookies, "")
	if w.Code != 403 {
		t.Fatal("missing origin accepted")
	}
	w = request(app, "POST", "/login", form, loginCookies, "https://panel.example")
	if w.Code != 303 {
		t.Fatal("login", w.Code, w.Body.String())
	}
	var cookies []*http.Cookie
	for _, c := range w.Result().Cookies() {
		if strings.HasSuffix(c.Name, "session") {
			if !c.Secure || !c.HttpOnly || c.SameSite != http.SameSiteStrictMode {
				t.Fatal("weak cookie")
			}
			cookies = append(cookies, c)
		}
	}
	w = request(app, "GET", "/", nil, cookies, "")
	if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("dashboard headers")
	}
	csrf := csrfFrom(t, w.Body.String())
	form = url.Values{"csrf": {csrf}, "name": {"<script>alert('x')</script>"}, "url": {"https://example.com/private-token?secret=yes"}, "min": {"15"}, "max": {"20"}, "unit": {"minutes"}, "enabled": {"on"}}
	w = request(app, "POST", "/sources", form, cookies, "https://panel.example")
	if w.Code != 303 {
		t.Fatal("create", w.Code, w.Body.String())
	}
	items, _ := s.List(ctx)
	if len(items) != 1 {
		t.Fatal("not saved")
	}
	id := items[0].ID
	w = request(app, "GET", "/", nil, cookies, "")
	if strings.Contains(w.Body.String(), "private-token") || strings.Contains(w.Body.String(), "<script>") {
		t.Fatal("secret or HTML injection")
	}
	form.Set("revision", "1")
	form.Set("name", "edited")
	form.Set("csrf", "bad")
	w = request(app, "POST", "/sources/"+id+"/edit", form, cookies, "https://panel.example")
	if w.Code != 403 {
		t.Fatal("bad csrf accepted")
	}
	form.Set("csrf", csrf)
	w = request(app, "POST", "/sources/"+id+"/edit", form, cookies, "https://panel.example")
	if w.Code != 303 {
		t.Fatal("edit", w.Code)
	}
	w = request(app, "POST", "/sources/"+id+"/edit", form, cookies, "https://panel.example")
	if w.Code != 409 {
		t.Fatal("stale form accepted", w.Code)
	}
	form.Set("revision", "2")
	w = request(app, "POST", "/sources/"+id+"/toggle", form, cookies, "https://panel.example")
	if w.Code != 303 {
		t.Fatal("toggle", w.Code)
	}
	item, _ := s.Get(ctx, id)
	if item.Enabled || item.Revision != 3 {
		t.Fatal("toggle persistence")
	}
	w = request(app, "GET", "/sources/"+id+"/delete", nil, cookies, "")
	if w.Code != 200 {
		t.Fatal("confirmation")
	}
	form.Set("revision", "3")
	w = request(app, "POST", "/sources/"+id+"/delete", form, cookies, "https://panel.example")
	if w.Code != 303 {
		t.Fatal("delete")
	}
	w = request(app, "POST", "/logout", url.Values{"csrf": {csrf}}, cookies, "https://panel.example")
	if w.Code != 303 {
		t.Fatal("logout")
	}
	w = request(app, "GET", "/", nil, cookies, "")
	if w.Code != 303 {
		t.Fatal("revoked session accepted")
	}
}
func TestBoundedBodiesAndHealth(t *testing.T) {
	s := testStore(t)
	app, _ := NewServer(s, "https://panel.example")
	w := request(app.Handler(), "POST", "/login", url.Values{"password": {strings.Repeat("x", 17000)}}, nil, "https://panel.example")
	if w.Code != 400 {
		t.Fatal(w.Code)
	}
	w = request(app.Handler(), "GET", "/healthz", nil, nil, "")
	if w.Code != 200 || w.Body.String() != "ok\n" {
		t.Fatal("health")
	}
}

func TestConcurrentRevisionWrites(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	input := testSource()
	id, err := s.Save(ctx, "", 0, input)
	if err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); <-start; _, err := s.Save(ctx, id, 1, input); results <- err }()
	}
	close(start)
	wg.Wait()
	close(results)
	success, conflict := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, ErrConflict) {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("success=%d conflict=%d", success, conflict)
	}
}
