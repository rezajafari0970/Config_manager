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
	pipeline := s.Smart()
	splash := os.Getenv("CONFIG_SMART_FIXTURE_SCOPE") == "splash"
	ads := os.Getenv("CONFIG_SMART_FIXTURE_SCOPE") == "ads"
	if ads {
		pipeline = s.AdsSmart()
		mainA, mainB := []string{}, []string{}
		for n := 201; n <= 255; n++ {
			if n <= 245 {
				mainA = append(mainA, sourceURI(n))
			} else {
				mainB = append(mainB, sourceURI(n))
			}
		}
		newExportSource(t, s, "MainA", mainA...)
		newExportSource(t, s, "MainB", mainB...)
	}
	if splash {
		pipeline = s.SplashSmart()
		newExportSource(t, s, "NormalExcluded", sourceURI(200))
	}
	for name, raws := range samples {
		id, _ := newExportSource(t, s, name, raws...)
		if splash || ads {
			markSplash(t, s, id, true)
		}
	}
	v, _ := pipeline.SmartSettings(ctx)
	v.Count = 50
	if e = pipeline.SaveSmartSettings(ctx, v); e != nil {
		t.Fatal(e)
	}
	listener, e := net.Listen("unix", filepath.Join(dir, "data", "probe.sock"))
	if e != nil {
		t.Fatal(e)
	}
	os.Chmod(filepath.Join(dir, "data", "probe.sock"), 0600)
	private := &http.Server{Handler: s.PrivateProbeHandler(), ReadHeaderTimeout: 5 * time.Second}
	go private.Serve(listener)
	publicListener, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer publicListener.Close()
	origin := "http://" + publicListener.Addr().String()
	app, e := NewServer(s, origin)
	if e != nil {
		t.Fatal(e)
	}
	public := &http.Server{Handler: app.Handler(), ReadHeaderTimeout: 5 * time.Second}
	go public.Serve(publicListener)
	ready, e := json.Marshal(map[string]string{"origin": origin, "scope": os.Getenv("CONFIG_SMART_FIXTURE_SCOPE")})
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(dir, "public.json"), ready, 0600); e != nil {
		t.Fatal(e)
	}
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	public.Shutdown(shutdown)
	private.Shutdown(shutdown)
}
