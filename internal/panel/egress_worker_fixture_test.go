package panel

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/rezajafari0970/Config_manager/internal/blob"
	"github.com/rezajafari0970/Config_manager/internal/collect"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
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
	s.db.QueryRow("SELECT count(*) FROM egress_jobs").Scan(&count)
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
		query := "SELECT min(r.hash) FROM conversion_results r JOIN memberships m ON m.raw_hash=r.hash WHERE r.connection='passed' AND r.core='passed' GROUP BY m.artifact_id ORDER BY max(r.tested) DESC LIMIT 3"
		benchmark, _ := strconv.Atoi(os.Getenv("EGRESS_BENCHMARK_INPUTS"))
		if benchmark > 0 && benchmark <= 2000 {
			query = fmt.Sprintf("SELECT hash FROM egress_inventory WHERE paused=0 ORDER BY ever_accepted DESC,hash LIMIT %d", benchmark)
		}
		args := []any{}
		if selection := os.Getenv("EGRESS_FIXTURE_HASHES_FILE"); selection != "" {
			body, err := os.ReadFile(selection)
			var hashes []string
			if err != nil || json.Unmarshal(body, &hashes) != nil || len(hashes) < 1 || len(hashes) > 32 {
				t.Fatal("invalid private canary selection")
			}
			for _, h := range hashes {
				if !validDigest(h) {
					t.Fatal("invalid private canary hash")
				}
				args = append(args, h)
			}
			benchmark = len(hashes)
			query = "SELECT hash FROM egress_inventory WHERE paused=0 AND hash IN(" + strings.TrimRight(strings.Repeat("?,", len(hashes)), ",") + ") ORDER BY hash"
		}
		rows, e := original.Query(query, args...)
		if e != nil {
			t.Fatal(e)
		}
		raws := []string{}
		acceptedHashes := map[string]bool{}
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
		if benchmark > 0 {
			ar, err := original.Query("SELECT hash FROM egress_jobs WHERE ever_accepted=1 AND state!='DELETED'")
			if err != nil {
				t.Fatal(err)
			}
			for ar.Next() {
				var hash string
				if err = ar.Scan(&hash); err != nil {
					t.Fatal(err)
				}
				acceptedHashes[hash] = true
			}
			ar.Close()
		}
		original.Close()
		if benchmark == 0 {
			raws = append(raws, sourceURI(241), "hysteria2://synthetic@192.0.2.242:443?sni=example.com")
		}
		if benchmark > 0 {
			// Use real committed membership so the benchmark exercises identity admission.
			src := testSource()
			src.URL = "https://example.com/benchmark/" + randomToken()
			id, err := s.Save(ctx, "", 0, src)
			if err != nil {
				t.Fatal(err)
			}
			job := claimOne(t, s, id)
			ref, err := s.vault.Put(ctx, strings.NewReader(strings.Join(raws, "\n")))
			if err != nil {
				t.Fatal(err)
			}
			result := collect.Result{State: "updated", HTTPStatus: 200, Raw: ref, Extracted: len(raws)}
			for _, raw := range raws {
				result.Items = append(result.Items, fixtureResult(t, s, raw).Items...)
			}
			if err = s.Complete(ctx, job, result, time.Now()); err != nil {
				t.Fatal(err)
			}
			s.vault.Release(ref.Hash)
			for hash := range acceptedHashes {
				if _, err = s.db.Exec("UPDATE egress_jobs SET ever_accepted=1,state='QUARANTINED' WHERE hash=?", hash); err != nil {
					t.Fatal(err)
				}
			}

		} else {
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
	ready, _ := json.Marshal(map[string]any{"pid": os.Getpid(), "socket": socket, "origin": origin, "benchmark": os.Getenv("EGRESS_BENCHMARK_INPUTS")})
	os.WriteFile(filepath.Join(dir, "ready.json"), ready, 0600)
	<-ctx.Done()
}
