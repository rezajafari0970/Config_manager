package panel

import (
	"bytes"
	"context"
	"errors"
	"github.com/rezajafari0970/Config_manager/internal/collect"
	"github.com/rezajafari0970/Config_manager/internal/convert"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

const browserURI = "vless://00000000-0000-4000-8000-000000000001@192.0.2.1:443?security=tls&sni=vpn.example#Original%20Label\r\n"
const browserJSON = "{ \"outbounds\" : [{\"protocol\":\"vless\",\"settings\":{\"address\":\"vpn.example\",\"port\":443,\"id\":\"00000000-0000-4000-8000-000000000002\"}}], \"stats\" : [], \"opaque\" : 9007199254740993 }\r\n"

type browserDriver struct{}

func (browserDriver) Name() string { return "synthetic-browser-fixture.v1" }
func (browserDriver) Fetch(ctx context.Context, job collect.Job) (collect.Response, error) {
	if dir := os.Getenv("CONFIG_PANEL_BROWSER_FIXTURE_DIR"); dir != "" {
		if _, e := os.Stat(filepath.Join(dir, "upstream-offline")); e == nil {
			return collect.Response{}, errors.New("synthetic upstream outage")
		}
	}
	if strings.Contains(job.URL, "Broken") {
		return collect.Response{}, errors.New("synthetic failure")
	}
	if strings.Contains(job.URL, "Slow") {
		<-ctx.Done()
		return collect.Response{}, ctx.Err()
	}
	body := browserURI
	if strings.Contains(job.URL, "Splash") {
		body = sourceURI(71) + "\n" + sourceURI(72) + "\n"
	}
	if strings.Contains(job.URL, "Verde") {
		body = browserURI + browserURI + browserJSON
	}
	if strings.Contains(job.URL, "Empty") {
		body = "\n"
	}
	return collect.Response{Status: 200, Body: io.NopCloser(strings.NewReader(body)), ETag: `"synthetic-browser"`}, nil
}

// Explicit development harness, skipped by normal CI. Never compiled into cmd/config-panel.
func TestBrowserFixtureServer(t *testing.T) {
	dir := os.Getenv("CONFIG_PANEL_BROWSER_FIXTURE_DIR")
	if dir == "" {
		t.Skip("browser harness not requested")
	}
	store, e := Open(filepath.Join(dir, "data", "panel.db"), bytes.Repeat([]byte{17}, 32))
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	app, e := NewServer(store, "http://127.0.0.1:18192")
	if e != nil {
		t.Fatal(e)
	}
	engine := collect.NewEngine(store, store.vault, browserDriver{})
	app.SetEngine(engine)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	done := make(chan error, 1)
	go func() { done <- engine.Run(ctx) }()
	// Synthetic UI evidence only; this harness is never part of a shipped binary.
	converted := make(chan struct{})
	go func() {
		defer close(converted)
		tick := time.NewTicker(100 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				j, e := store.ClaimProbe(ctx)
				if e != nil || j == nil {
					continue
				}
				if len(convert.Prepare(j.Raw).Candidates) == 0 {
					continue
				}
				r := conversionResult(*j, 0, true)
				// Synthetic connection evidence for the UI harness only.
				r.Code = "https_multi_site_verified" // synthetic UI receipt only; no network measurement
				_ = store.CompleteProbe(ctx, r)
			}
		}
	}()

	smartDone := make(chan struct{})
	go func() {
		defer close(smartDone)
		tick := time.NewTicker(100 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				for _, p := range []*SmartPipeline{store.Smart(), store.SplashSmart(), store.AdsSmart()} {
					j, e := p.ClaimSmart(ctx)
					if e == nil && j != nil {
						r := smartResult(j)
						r.Version = p.version()
						r.Rule = p.version()
						if j.Issue != "" {
							r.Core = "blocked"
							r.Connection = "not_run"
							r.Code = j.Issue
							r.Candidate = nil
						}
						_ = p.CompleteSmart(ctx, r)
					}
				}
			}
		}
	}()

	server := &http.Server{Addr: "127.0.0.1:18192", Handler: app.Handler(), ReadHeaderTimeout: 5 * time.Second}
	httpDone := make(chan error, 1)
	go func() { httpDone <- server.ListenAndServe() }()
	select {
	case <-ctx.Done():
	case e := <-httpDone:
		t.Fatal(e)
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server.Shutdown(shutdown)
	<-converted
	<-smartDone
	if e := <-done; e != nil {
		t.Fatal(e)
	}
}
