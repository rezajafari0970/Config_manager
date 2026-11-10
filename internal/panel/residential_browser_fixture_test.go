package panel

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
)

// Synthetic isolated UI fixture, never compiled into production binaries.
func TestResidentialBrowserFixture(t *testing.T) {
	dir := os.Getenv("CONFIG_RESIDENTIAL_BROWSER_FIXTURE_DIR")
	if dir == "" {
		t.Skip("explicit browser fixture only")
	}
	store, e := Open(filepath.Join(dir, "data", "panel.db"), bytes.Repeat([]byte{7}, 32))
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	resPolicy(t, store)
	resSeed(t, store, 1)
	resSeed(t, store, 2)
	listener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	origin := "http://" + listener.Addr().String()
	app, e := NewServer(store, origin)
	if e != nil {
		t.Fatal(e)
	}
	server := &http.Server{Handler: app.Handler()}
	go server.Serve(listener)
	defer server.Close()
	ready, _ := json.Marshal(map[string]string{"origin": origin, "evidence": "synthetic-fixture-only"})
	if e = os.WriteFile(filepath.Join(dir, "public.json"), ready, 0600); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()
	<-ctx.Done()
}
