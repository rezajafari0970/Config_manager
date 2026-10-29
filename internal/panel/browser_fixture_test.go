package panel

import (
	"bytes"
	"context"
	"errors"
	"github.com/rezajafari0970/Config_manager/internal/collect"
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

const browserURI = "vless://00000000-0000-4000-8000-000000000001@vpn.example:443?security=tls#Original%20Label\r\n"
const browserJSON = "{ \"outbounds\" : [{\"protocol\":\"vless\",\"settings\":{\"address\":\"vpn.example\",\"port\":443,\"id\":\"00000000-0000-4000-8000-000000000002\"}}], \"stats\" : [], \"opaque\" : 9007199254740993 }\r\n"

type browserDriver struct{}

func (browserDriver) Name() string { return "synthetic-browser-fixture.v1" }
func (browserDriver) Fetch(ctx context.Context, job collect.Job) (collect.Response, error) {
	if strings.Contains(job.URL, "Broken") {
		return collect.Response{}, errors.New("synthetic failure")
	}
	if strings.Contains(job.URL, "Slow") {
		<-ctx.Done()
		return collect.Response{}, ctx.Err()
	}
	body := browserURI
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
	if e := <-done; e != nil {
		t.Fatal(e)
	}
}
