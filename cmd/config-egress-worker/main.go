// Independent two-slot supervisor for finite Ads egress investigations.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/rezajafari0970/Config_manager/internal/egress"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type client struct{ http *http.Client }

func (c client) call(ctx context.Context, path string, in, out any) error {
	b, e := json.Marshal(in)
	if e != nil {
		return e
	}
	req, e := http.NewRequestWithContext(ctx, "POST", "http://unix/egress/"+path, bytes.NewReader(b))
	if e != nil {
		return e
	}
	req.Header.Set("Content-Type", "application/json")
	r, e := c.http.Do(req)
	if e != nil {
		return e
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		return fmt.Errorf("api_status_%d", r.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(io.LimitReader(r.Body, 16<<20)).Decode(out)
	}
	return nil
}

type bounded struct {
	bytes.Buffer
	max int
}

func (b *bounded) Write(p []byte) (int, error) {
	if b.Len()+len(p) > b.max {
		return 0, errors.New("driver output limit")
	}
	return b.Buffer.Write(p)
}
func clean(helper, prefix string, slot int, run string) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	args := []string{helper, "--prefix", prefix, "--slot", fmt.Sprint(slot), "--cleanup"}
	if run != "" {
		args = append(args, "--run-id", run)
	}
	cmd := exec.CommandContext(ctx, "/usr/bin/python3", args...)
	cmd.WaitDelay = 2 * time.Second
	_ = cmd.Run()
}
func execute(ctx context.Context, c client, helper, prefix string, slot int, j egress.Job) {
	started := egress.Now()
	p := egress.Prepare(j.Raw)
	r := egress.Report{Version: egress.Version, Hash: j.Hash, Started: started, Finished: started, Code: "driver_unavailable"}
	if len(p.Variants) == 2 && egress.Hash(j.Raw) == j.Hash && j.Issue == "" {
		job, cancel := context.WithTimeout(ctx, 220*time.Second)
		defer cancel()
		done := make(chan struct{})
		defer close(done)
		go func() {
			ticker := time.NewTicker(20 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-done:
					return
				case <-job.Done():
					return
				case <-ticker.C:
					if c.call(job, "renew", egress.Job{Hash: j.Hash, Token: j.Token, Revision: j.Revision}, nil) != nil {
						cancel()
						return
					}
				}
			}
		}()
		run := egress.Hash([]byte(j.Token))[:6]
		input, _ := json.Marshal(map[string]any{"hash": j.Hash, "started": started, "plan": p, "deep": j.Deep})
		out := &bounded{max: 1 << 20}
		cmd := exec.CommandContext(job, "/usr/bin/python3", helper, "--prefix", prefix, "--slot", fmt.Sprint(slot), "--run-id", run)
		cmd.Stdin = bytes.NewReader(input)
		cmd.Stdout = out
		cmd.Stderr = io.Discard
		cmd.WaitDelay = 3 * time.Second
		if e := cmd.Run(); e != nil || json.Unmarshal(out.Bytes(), &r) != nil {
			r = egress.Report{Version: egress.Version, Hash: j.Hash, Started: started, Finished: egress.Now(), Code: "driver_unavailable"}
		}
		clean(helper, prefix, slot, run)
	}
	if r.Finished == 0 {
		r.Finished = egress.Now()
	}
	j.Raw = nil
	if e := c.call(ctx, "complete", egress.Completion{Job: j, Report: r}, nil); e != nil {
		log.Print("egress completion deferred; lease will recover")
	}
}
func run() error {
	socket := flag.String("socket", "/var/lib/config-repair/probe.sock", "local protected queue")
	helper := flag.String("helper", "/opt/config-repair/current/scripts/egress-probe.py", "trusted driver")
	prefix := flag.String("prefix", "cge", "isolated scope")
	lockPath := flag.String("lock-file", "/run/config-repair-egress.lock", "single supervisor")
	flag.Parse()
	if os.Geteuid() != 0 || (*prefix != "cge" && *prefix != "cgt") {
		return errors.New("invalid supervisor environment")
	}
	lock, e := os.OpenFile(*lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return e
	}
	defer lock.Close()
	if e = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		return errors.New("supervisor already running")
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	clean(*helper, *prefix, 0, "")
	c := client{&http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", *socket)
	}}}}
	var active atomic.Int32
	var wg sync.WaitGroup
	for slot := 1; slot <= 2; slot++ {
		wg.Add(1)
		go func(slot int) {
			defer wg.Done()
			for ctx.Err() == nil {
				var j *egress.Job
				if c.call(ctx, "claim", nil, &j) == nil && j != nil {
					active.Add(1)
					execute(ctx, c, *helper, *prefix, slot, *j)
					active.Add(-1)
					continue
				}
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Second):
				}
			}
		}(slot)
	}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for ctx.Err() == nil {
		_ = c.call(ctx, "status", map[string]any{"status": "running", "active": active.Load(), "code": ""}, nil)
		select {
		case <-ctx.Done():
		case <-ticker.C:
		}
	}
	wg.Wait()
	clean(*helper, *prefix, 0, "")
	return nil
}
func main() {
	if e := run(); e != nil {
		log.Fatal(e)
	}
}
