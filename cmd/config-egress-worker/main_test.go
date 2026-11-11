package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDriverSessionReuseAndTimeoutRecovery(t *testing.T) {
	helper := filepath.Join(t.TempDir(), "driver.py")
	source := `import sys,json,os,time
if '--cleanup' in sys.argv:sys.exit(0)
for line in sys.stdin:
 req=json.loads(line)
 if req.get('delay'):time.sleep(10)
 print(json.dumps({'pid':os.getpid(),'nonce':req.get('nonce')}),flush=True)
`
	if err := os.WriteFile(helper, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	d := &driverSession{owner: context.Background(), helper: helper, prefix: "cgt", slot: 31}
	defer d.stop()
	call := func(nonce string) int {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		b, e := d.exchange(ctx, []byte(`{"nonce":"`+nonce+`"}`))
		if e != nil {
			t.Fatal(e)
		}
		var r struct {
			PID   int    `json:"pid"`
			Nonce string `json:"nonce"`
		}
		if json.Unmarshal(b, &r) != nil || r.Nonce != nonce {
			t.Fatal("stale or malformed reply")
		}
		return r.PID
	}
	first := call("one")
	if second := call("two"); first != second {
		t.Fatal("helper was not reused")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := d.exchange(ctx, []byte(`{"delay":true}`)); err == nil {
		t.Fatal("deadline ignored")
	}
	if next := call("three"); next == first {
		t.Fatal("timed-out helper reused")
	}
}

func TestDriverOversizedOutputFailsClosed(t *testing.T) {
	helper := filepath.Join(t.TempDir(), "driver.py")
	source := `import sys
if '--cleanup' in sys.argv:sys.exit(0)
for line in sys.stdin:print('x'*(2*1024*1024),flush=True)
`
	if err := os.WriteFile(helper, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	d := &driverSession{owner: context.Background(), helper: helper, prefix: "cgt", slot: 30}
	defer d.stop()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := d.exchange(ctx, []byte(`{}`)); err == nil {
		t.Fatal("oversized response accepted")
	}
}
