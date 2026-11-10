package panel

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSmartAvailabilityIsTruthfulAndDoesNotRotate(t *testing.T) {
	s, _ := smartFixture(t)
	app, e := NewServer(s, "http://127.0.0.1:18191")
	if e != nil {
		t.Fatal(e)
	}
	h := app.Handler()
	for _, p := range []*SmartPipeline{s.Smart(), s.SplashSmart(), s.AdsSmart()} {
		single := app.pipelineURL(p, true)
		statusURL := strings.TrimSuffix(single, "next.json") + "status.json"
		r := exportRequest(t, h, "GET", statusURL)
		var v SmartAvailability
		if r.Code != 200 || json.Unmarshal(r.Body.Bytes(), &v) != nil || v.Available {
			t.Fatalf("initial availability %d %s", r.Code, r.Body.String())
		}
		if p.ads || p.splash {
			if v.State != "no_inputs" {
				t.Fatal(v)
			}
		}
		head := exportRequest(t, h, "HEAD", single)
		if head.Code != 503 || head.Body.Len() != 0 || head.Header().Get("X-Config-Count") != "0" {
			t.Fatal("HEAD reported ready", head)
		}
		raw := exportRequest(t, h, "GET", single)
		if raw.Code != 503 || !strings.HasPrefix(raw.Header().Get("Content-Type"), "text/plain") || strings.Contains(raw.Body.String(), "<html") {
			t.Fatal("client response corrupted")
		}
		req := httptest.NewRequest("GET", single, nil)
		req.Header.Set("Accept", "text/html,application/xhtml+xml")
		html := httptest.NewRecorder()
		h.ServeHTTP(html, req)
		if html.Code != 503 || !strings.Contains(html.Body.String(), "/assets/waiting.js") || !strings.Contains(html.Body.String(), "name=\"viewport\"") {
			t.Fatal("browser has no status page", html.Body.String())
		}
	}
	finishSmart(t, s)
	before := 0
	s.db.QueryRow("SELECT cycle FROM smart_rotation_state WHERE id=1").Scan(&before)
	for i := 0; i < 3; i++ {
		statusURL := strings.TrimSuffix(app.smartURL(true), "next.json") + "status.json"
		r := exportRequest(t, h, "GET", statusURL)
		var v SmartAvailability
		json.Unmarshal(r.Body.Bytes(), &v)
		if !v.Available || v.State != "ready" {
			t.Fatal(v)
		}
		if r = exportRequest(t, h, "HEAD", app.smartURL(true)); r.Code != 200 {
			t.Fatal(r.Code)
		}
	}
	after := 0
	s.db.QueryRow("SELECT cycle FROM smart_rotation_state WHERE id=1").Scan(&after)
	if before != after {
		t.Fatal("read-only status consumed a delivery turn")
	}
	r := exportRequest(t, h, "GET", app.smartURL(true))
	if r.Code != 200 || !json.Valid(r.Body.Bytes()) {
		t.Fatal("ready JSON changed")
	}
}

func TestSmartAdsAvailabilityDistinguishesFailedSplash(t *testing.T) {
	s, _ := smartFixture(t)
	id, _ := newExportSource(t, s, "FailedSplash", sourceURI(70))
	if _, e := s.db.Exec("UPDATE sources SET splash=1 WHERE id=?", id); e != nil {
		t.Fatal(e)
	}
	if _, e := s.db.Exec("UPDATE conversion_results SET connection='failed',code='reality_authentication_rejected' WHERE hash IN(SELECT raw_hash FROM memberships WHERE source_id=?)", id); e != nil {
		t.Fatal(e)
	}
	v, e := s.AdsSmart().availability(context.Background())
	if e != nil || v.Available || v.State != "ads_health_unavailable" || v.Inputs != 1 || v.Failed != 1 {
		t.Fatal(v, e)
	}
}

func TestStatusTokenIsScoped(t *testing.T) {
	s := testStore(t)
	app, _ := NewServer(s, "http://127.0.0.1:18191")
	url := strings.Replace(strings.TrimSuffix(app.smartURL(true), "next.json")+"status.json", "/smart-subscription/", "/smart-ads-subscription/", 1)
	if r := exportRequest(t, app.Handler(), http.MethodGet, url); r.Code != 404 {
		t.Fatal("scope token accepted")
	}
}
