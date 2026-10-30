package panel

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/collect"
	"github.com/rezajafari0970/Config_manager/internal/probe"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// Explicit isolated integration harness; never linked into a production binary.
func TestConversionWorkerFixture(t *testing.T) {
	dir := os.Getenv("CONFIG_CONVERSION_FIXTURE_DIR")
	if dir == "" {
		t.Skip("integration harness not requested")
	}
	s, e := Open(filepath.Join(dir, "data", "panel.db"), bytes.Repeat([]byte{21}, 32))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	var samples [][]byte
	b, e := os.ReadFile(filepath.Join(dir, "samples.json"))
	if e != nil || json.Unmarshal(b, &samples) != nil {
		t.Fatal("sample file")
	}
	src := testSource()
	src.Min = 3600
	src.Max = 3600
	src.Unit = "seconds"
	id, e := s.Save(ctx, "", 0, src)
	if e != nil {
		t.Fatal(e)
	}
	j := claimOne(t, s, id)
	result := collect.Result{State: "updated", HTTPStatus: 200, Extracted: len(samples)}
	seen := map[string]bool{}
	for _, raw := range samples {
		ref, e := s.vault.Put(ctx, bytes.NewReader(raw))
		if e != nil {
			t.Fatal(e)
		}
		defer s.vault.Release(ref.Hash)
		kind := probe.Prepare(raw).Kind
		identity := collect.Fingerprint(raw, true, kind, ref.Hash)
		if seen[identity] {
			continue
		}
		seen[identity] = true
		result.Items = append(result.Items, collect.Item{Ref: ref, Identity: identity, Kind: kind})
	}
	if e = s.Complete(ctx, j, result, time.Now()); e != nil {
		t.Fatal(e)
	}
	if e = s.SaveProbeSettings(ctx, probe.Settings{Enabled: true, Concurrency: 3, TimeoutSeconds: 6, RetestSeconds: 3600, HealthTTLSeconds: 7200}); e != nil {
		t.Fatal(e)
	}
	listener, e := net.Listen("unix", filepath.Join(dir, "data", "probe.sock"))
	if e != nil {
		t.Fatal(e)
	}
	os.Chmod(filepath.Join(dir, "data", "probe.sock"), 0600)
	private := &http.Server{Handler: s.PrivateProbeHandler(), ReadHeaderTimeout: 5 * time.Second}
	go private.Serve(listener)
	app, _ := NewServer(s, "http://127.0.0.1:18194")
	public := &http.Server{Addr: "127.0.0.1:18194", Handler: app.Handler(), ReadHeaderTimeout: 5 * time.Second}
	go public.ListenAndServe()
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	public.Shutdown(shutdown)
	private.Shutdown(shutdown)
}
