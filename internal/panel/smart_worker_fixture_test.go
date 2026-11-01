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
	"time"
)

// Synthetic/current accepted inputs are isolated from production state. Never shipped.
func TestSmartWorkerFixture(t *testing.T) {
	dir := os.Getenv("CONFIG_SMART_FIXTURE_DIR")
	if dir == "" {
		t.Skip("smart integration not requested")
	}
	s, e := Open(filepath.Join(dir, "data", "panel.db"), bytes.Repeat([]byte{23}, 32))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	var samples map[string][]string
	data, e := os.ReadFile(filepath.Join(dir, "samples.json"))
	if e != nil || json.Unmarshal(data, &samples) != nil {
		t.Fatal("samples unavailable")
	}
	for name, raws := range samples {
		newExportSource(t, s, name, raws...)
	}
	v, _ := s.SmartSettings(ctx)
	v.Count = 50
	if e = s.SaveSmartSettings(ctx, v); e != nil {
		t.Fatal(e)
	}
	listener, e := net.Listen("unix", filepath.Join(dir, "data", "probe.sock"))
	if e != nil {
		t.Fatal(e)
	}
	os.Chmod(filepath.Join(dir, "data", "probe.sock"), 0600)
	private := &http.Server{Handler: s.PrivateProbeHandler(), ReadHeaderTimeout: 5 * time.Second}
	go private.Serve(listener)
	app, _ := NewServer(s, "http://127.0.0.1:18195")
	public := &http.Server{Addr: "127.0.0.1:18195", Handler: app.Handler(), ReadHeaderTimeout: 5 * time.Second}
	go public.ListenAndServe()
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	public.Shutdown(shutdown)
	private.Shutdown(shutdown)
}
