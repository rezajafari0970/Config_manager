package panel

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/blob"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestEgressWorkerFixture(t *testing.T) {
	dir := os.Getenv("EGRESS_WORKER_FIXTURE")
	if dir == "" {
		t.Skip("isolated fixture not requested")
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	s, e := Open(filepath.Join(dir, "panel.db"), bytes.Repeat([]byte{71}, 32))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	var count int
	s.db.QueryRow("SELECT count(*) FROM sources").Scan(&count)
	if count == 0 {
		private, e := os.ReadFile("/etc/config-repair/panel.json")
		if e != nil {
			t.Fatal(e)
		}
		var c struct {
			Database string `json:"database"`
			Key      string `json:"master_key"`
		}
		json.Unmarshal(private, &c)
		key, _ := base64.StdEncoding.DecodeString(c.Key)
		vault, e := blob.New(filepath.Join(filepath.Dir(c.Database), "blobs"), key)
		if e != nil {
			t.Fatal(e)
		}
		original, e := sql.Open("sqlite", "file:"+c.Database+"?mode=ro")
		if e != nil {
			t.Fatal(e)
		}
		rows, e := original.Query("SELECT DISTINCT r.hash FROM conversion_results r JOIN memberships m ON m.raw_hash=r.hash WHERE r.connection='passed' AND r.core='passed' ORDER BY r.tested DESC LIMIT 3")
		if e != nil {
			t.Fatal(e)
		}
		raws := []string{}
		for rows.Next() {
			var h string
			rows.Scan(&h)
			f, e := vault.Open(h)
			if e != nil {
				continue
			}
			b, e := io.ReadAll(io.LimitReader(f, 8<<20))
			f.Close()
			if e == nil {
				raws = append(raws, string(b))
			}
		}
		rows.Close()
		original.Close()
		raws = append(raws, sourceURI(241), "hysteria2://synthetic@192.0.2.242:443?sni=example.com")
		for _, raw := range raws {
			src := testSource()
			src.URL = "https://example.com/fixture/" + randomToken()
			id, e := s.Save(ctx, "", 0, src)
			if e != nil {
				t.Fatal(e)
			}
			j := claimOne(t, s, id)
			if e = s.Complete(ctx, j, fixtureResult(t, s, raw), time.Now()); e != nil {
				t.Fatal(e)
			}
		}
	}
	socket := filepath.Join(dir, "probe.sock")
	os.Remove(socket)
	ln, e := net.Listen("unix", socket)
	if e != nil {
		t.Fatal(e)
	}
	defer ln.Close()
	os.Chmod(socket, 0600)
	privateServer := &http.Server{Handler: s.PrivateProbeHandler()}
	defer privateServer.Close()
	go privateServer.Serve(ln)
	tcp, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer tcp.Close()
	origin := "http://" + tcp.Addr().String()
	app, e := NewServer(s, origin)
	if e != nil {
		t.Fatal(e)
	}
	web := &http.Server{Handler: app.Handler()}
	defer web.Close()
	go web.Serve(tcp)
	ready, _ := json.Marshal(map[string]any{"pid": os.Getpid(), "socket": socket, "origin": origin})
	os.WriteFile(filepath.Join(dir, "ready.json"), ready, 0600)
	<-ctx.Done()
}
