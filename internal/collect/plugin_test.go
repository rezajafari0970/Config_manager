package collect

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

type reentrantFormat struct{ r *Registry }

func (reentrantFormat) Name() string                                   { return "fixture" }
func (f reentrantFormat) Match([]byte) bool                            { f.r.Register(f); return true }
func (reentrantFormat) Extract(context.Context, string, Emitter) error { return nil }

type fixtureInspector struct{}

func (fixtureInspector) Kind() string                { return "fixture" }
func (fixtureInspector) Inspect([]byte, bool) string { return "valid" }

func TestPluginCallbacksAndConcurrentRegistration(t *testing.T) {
	r := NewRegistry()
	r.Register(reentrantFormat{r})
	path := filepath.Join(t.TempDir(), "input")
	if e := os.WriteFile(path, []byte("fixture"), 0600); e != nil {
		t.Fatal(e)
	}
	inspectors := NewInspectionRegistry()
	var wg sync.WaitGroup
	for n := 0; n < 4; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				inspectors.Register(fixtureInspector{})
				if inspectors.Inspect(nil, true, "fixture") != "valid" {
					t.Error("registered inspector missing")
				}
				if e := r.Extract(context.Background(), path, func(context.Context, io.Reader, string) error { return nil }); e != nil {
					t.Error(e)
				}
			}
		}()
	}
	wg.Wait()
}
