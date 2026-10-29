// The privileged supervisor only accepts jobs from a protected local Unix
// socket. Untrusted configuration execution belongs exclusively to the sandbox.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/rezajafari0970/Config_manager/internal/probe"
	"github.com/rezajafari0970/Config_manager/internal/repair"
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
	req, e := http.NewRequestWithContext(ctx, "POST", "http://unix"+path, bytes.NewReader(b))
	if e != nil {
		return e
	}
	req.Header.Set("Content-Type", "application/json")
	res, e := c.http.Do(req)
	if e != nil {
		return e
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("api_status_%d", res.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(io.LimitReader(res.Body, 4<<20)).Decode(out)
	}
	return nil
}

type helperResult struct {
	probe.Attempt
	Attempts []probe.Attempt `json:"attempts"`
}

func execute(ctx context.Context, c client, helper, core string, slot int, j probe.Job) {
	p := probe.Prepare(j.Raw)
	code := p.Code
	if repair.Digest(j.Raw) != j.Hash {
		code = "original_blob_unavailable"
		p.Candidates = nil
	}
	if j.Issue != "" {
		code = j.Issue
		p.Candidates = nil
	}
	if p.Kind == "" {
		p.Kind = j.Kind
	}
	p.Code = code
	r := probe.Result{Hash: j.Hash, Token: j.Token, Revision: j.Revision, Version: probe.Version, CoreSHA256: probe.CoreSHA256, Model: probe.Model(p), Attempt: probe.Attempt{SHA256: j.Hash, Rule: "preparation", Core: "not_run", Connection: "not_run", Code: code}}
	if len(p.Candidates) > 0 {
		var preferred string
		if c.call(ctx, "/strategy", map[string]string{"model": r.Model}, &preferred) == nil && preferred != "" {
			for i, v := range p.Candidates {
				if v.Rule == preferred {
					p.Candidates[0], p.Candidates[i] = p.Candidates[i], p.Candidates[0]
					break
				}
			}
		}
		deadline := time.Duration(len(p.Candidates)*(50+2*j.Settings.TimeoutSeconds)) * time.Second
		jobCtx, cancel := context.WithTimeout(ctx, deadline)
		defer cancel()
		done := make(chan struct{})
		defer close(done)
		go func() {
			tick := time.NewTicker(20 * time.Second)
			defer tick.Stop()
			for {
				select {
				case <-done:
					return
				case <-jobCtx.Done():
					return
				case <-tick.C:
					leaseCtx, stop := context.WithTimeout(jobCtx, 10*time.Second)
					err := c.call(leaseCtx, "/renew", probe.Job{Hash: j.Hash, Token: j.Token, Revision: j.Revision}, nil)
					stop()
					if err != nil {
						cancel()
						return
					}
				}
			}
		}()
		input, _ := json.Marshal(map[string]any{"plan": p, "timeout": j.Settings.TimeoutSeconds})
		runID := repair.Digest([]byte(j.Token))[:6]
		command := exec.CommandContext(jobCtx, "/usr/bin/python3", helper, "--core", core, "--slot", fmt.Sprint(slot), "--run-id", runID)
		command.Stdin = bytes.NewReader(input)
		command.WaitDelay = 5 * time.Second
		var out bytes.Buffer
		command.Stdout = &out
		command.Stderr = io.Discard
		e := command.Run()
		var h helperResult
		if e == nil && json.Unmarshal(out.Bytes(), &h) == nil && h.Code != "" {
			r.Attempt = h.Attempt
			r.Attempts = h.Attempts
			for _, candidate := range p.Candidates {
				if candidate.SHA256 == h.SHA256 {
					r.Candidate = candidate.Config
					break
				}
			}
		} else {
			r.Core = "blocked"
			r.Code = "probe_infrastructure_error"
			if errors.Is(jobCtx.Err(), context.DeadlineExceeded) {
				r.Code = "probe_job_timeout"
			}
		}
		cleanCtx, stop := context.WithTimeout(context.Background(), 20*time.Second)
		cleanup := exec.CommandContext(cleanCtx, "/usr/bin/python3", helper, "--cleanup-slot", fmt.Sprint(slot), "--cleanup-run", runID)
		cleanup.Stdout = io.Discard
		cleanup.Stderr = io.Discard
		_ = cleanup.Run()
		stop()
	}
	completeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if e := c.call(completeCtx, "/complete", r, nil); e != nil {
		log.Print("probe completion deferred; lease will recover")
	}
}
func run() error {
	socket := flag.String("socket", "/var/lib/config-repair/probe.sock", "protected Unix queue socket")
	helper := flag.String("helper", "/opt/config-repair/current/scripts/probe-one.py", "trusted sandbox driver")
	core := flag.String("core", "/opt/config-repair/core-26.7.28/xray", "pinned core")
	flag.Parse()
	if os.Geteuid() != 0 {
		return errors.New("privileged sandbox supervisor required")
	}
	lock, e := os.OpenFile("/run/config-repair-prober.lock", os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return e
	}
	defer lock.Close()
	if e = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); e != nil {
		return errors.New("probe supervisor already running")
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	c := client{&http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", *socket)
	}}, Timeout: 30 * time.Second}}
	status := func(state, code string, active int) {
		sc, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_ = c.call(sc, "/status", probe.WorkerStatus{Status: state, Code: code, Version: probe.Version, Active: active}, nil)
	}
	clean := exec.CommandContext(ctx, "/usr/bin/python3", *helper, "--cleanup")
	if e = clean.Run(); e != nil {
		return errors.New("probe_cleanup_failed")
	}
	status("starting", "sandbox_preflight", 0)
	checkCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	check := exec.CommandContext(checkCtx, "/usr/bin/python3", *helper, "--core", *core, "--self-test")
	e = check.Run()
	cancel()
	if e != nil {
		status("blocked", "sandbox_preflight_failed", 0)
		return errors.New("sandbox_preflight_failed")
	}
	var active atomic.Int32
	var wg sync.WaitGroup
	for slot := 1; slot <= 8; slot++ {
		wg.Add(1)
		go func(slot int) {
			defer wg.Done()
			tick := time.NewTicker(time.Second)
			defer tick.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-tick.C:
					var j *probe.Job
					if e := c.call(ctx, "/claim", nil, &j); e != nil || j == nil {
						continue
					}
					active.Add(1)
					execute(ctx, c, *helper, *core, slot, *j)
					active.Add(-1)
				}
			}
		}(slot)
	}
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	status("running", "", 0)
	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return nil
		case <-tick.C:
			status("running", "", int(active.Load()))
		}
	}
}
func main() {
	if e := run(); e != nil {
		log.Print(e)
		os.Exit(1)
	}
}
