package panel

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/residential"
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
	// Explicit synthetic compatibility presentation; never a live core receipt.
	attempts, _ := json.Marshal([]residential.CoreAttempt{{CoreID: "xray-26.5.9", Variants: []residential.Variant{{Name: "raw", Core: "passed", Health: "HEALTHY", Code: "fixture_only"}, {Name: "template", Core: "passed", Health: "HEALTHY", Code: "fixture_only"}}}})
	if _, e = store.db.Exec("UPDATE residential_records SET decision=json_set(decision,'$.core_id','xray-26.5.9','$.compatibility',json(?))", string(attempts)); e != nil {
		t.Fatal(e)
	}
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
