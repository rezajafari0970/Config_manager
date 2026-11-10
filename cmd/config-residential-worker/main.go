// The privileged supervisor executes only the compiled trusted driver. Core
// configuration execution remains in the existing unprivileged closed sandbox.
package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
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

	"github.com/rezajafari0970/Config_manager/internal/convert"
	"github.com/rezajafari0970/Config_manager/internal/probe"
	"github.com/rezajafari0970/Config_manager/internal/residential"
)

type client struct{ http *http.Client }

func (c client) call(ctx context.Context, path string, in, out any) error {
	b, e := json.Marshal(in)
	if e != nil {
		return e
	}
	req, e := http.NewRequestWithContext(ctx, "POST", "http://unix/residential"+path, bytes.NewReader(b))
	if e != nil {
		return e
	}
	req.Header.Set("Content-Type", "application/json")
	response, e := c.http.Do(req)
	if e != nil {
		return e
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return fmt.Errorf("private_status_%d", response.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(io.LimitReader(response.Body, 6<<20)).Decode(out)
	}
	return nil
}

type boundedOutput struct{ bytes.Buffer }

func (b *boundedOutput) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 4<<20 {
		return 0, errors.New("driver_output_limit")
	}
	return b.Buffer.Write(p)
}
func execute(ctx context.Context, c client, helper, core, workerID, epoch string, slot int, j residential.Job, factKey []byte) error {
	var cleanupErr error
	started := time.Now().UnixMilli()
	plan := residential.NewCoreRegistry().Prepare("xray-26.7.28", j.Raw)
	report := residential.Report{Version: residential.Version, PolicyHash: j.Policy.Digest(), TemplateVersion: convert.Version, CoreSHA256: probe.CoreSHA256, Worker: workerID, InfrastructureEpoch: epoch, Started: started, Finished: started, InfrastructureOK: true, Variants: []residential.Variant{}, Observations: []residential.Observation{}, Controls: []residential.Observation{}, IPFacts: []residential.IPFact{}}
	completion := residential.Completion{Hash: j.Hash, Token: j.Token, Revision: j.Revision, Epoch: j.Epoch, TrustHash: j.TrustHash, Report: report}
	if residential.Digest(j.Raw) != j.Hash || j.Issue != "" {
		plan.Variants = nil
		plan.Code = "original_blob_unavailable"
		completion.Report.InfrastructureOK = false
	}
	if len(plan.Variants) != 2 {
		for _, name := range []string{"raw", "template"} {
			completion.Report.Variants = append(completion.Report.Variants, residential.Variant{Name: name, Core: "unsupported", Health: "UNKNOWN", Code: plan.Code})
		}
	} else {
		jobCtx, cancel := context.WithTimeout(ctx, 4*time.Minute)
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
					err := c.call(leaseCtx, "/renew", residential.Job{Hash: j.Hash, Token: j.Token, Revision: j.Revision, Epoch: j.Epoch, Policy: j.Policy, TrustHash: j.TrustHash}, nil)
					stop()
					if err != nil {
						cancel()
						return
					}
				}
			}
		}()
		input, _ := json.Marshal(map[string]any{"plan": plan, "deep": j.Deep, "policy": j.Policy, "policy_hash": j.Policy.Digest(), "template_version": convert.Version, "worker": workerID, "infrastructure_epoch": epoch})
		runID := residential.Digest([]byte(j.Token))[:6]
		command := exec.CommandContext(jobCtx, "/usr/bin/python3", helper, "--core", core, "--slot", fmt.Sprint(slot), "--run-id", runID)
		command.Env = append(os.Environ(), "SHEN_RESIDENTIAL_PARENT_PID="+fmt.Sprint(os.Getpid()))
		command.Stdin = bytes.NewReader(input)
		command.WaitDelay = 5 * time.Second
		var output boundedOutput
		command.Stdout = &output
		command.Stderr = io.Discard
		err := command.Run()
		if err == nil && json.Unmarshal(output.Bytes(), &report) == nil && report.Version == residential.Version {
			completion.Report = report
			completion.Candidate = plan.Variants[1].Config
		} else {
			completion.Report.InfrastructureOK = false
			completion.Report.Finished = time.Now().UnixMilli()
		}
		cleanCtx, stop := context.WithTimeout(context.Background(), 20*time.Second)
		cleanup := exec.CommandContext(cleanCtx, "/usr/bin/python3", helper, "--cleanup-slot", fmt.Sprint(slot), "--cleanup-run", runID)
		cleanup.Stdout = io.Discard
		cleanup.Stderr = io.Discard
		cleanupErr = cleanup.Run()
		stop()
		if cleanupErr != nil {
			completion.Report.InfrastructureOK = false
		}
	}
	for i := range completion.Report.IPFacts {
		completion.Report.IPFacts[i].MAC = residential.IPFactMAC(completion.Report.IPFacts[i], factKey)
	}
	completion.Report.Finished = max(completion.Report.Finished, started)
	completeCtx, stop := context.WithTimeout(ctx, 30*time.Second)
	defer stop()
	if c.call(completeCtx, "/complete", completion, nil) != nil {
		log.Print("residential completion deferred; lease recovery scheduled")
	}
	return cleanupErr
}
func run() error {
	keyPath := flag.String("signing-config", "/etc/config-repair/panel.json", "private owner configuration")
	socket := flag.String("socket", "/var/lib/config-repair/probe.sock", "private queue socket")
	helper := flag.String("helper", "/opt/config-repair/current/scripts/residential-matrix.py", "trusted compatibility driver")
	core := flag.String("core", "/opt/config-repair/core-26.7.28/xray", "pinned core")
	lockPath := flag.String("lock-file", "/run/config-repair-residential.lock", "supervisor lock")
	slots := flag.Int("slots", 2, "bounded persistent worker slots")
	flag.Parse()
	if os.Geteuid() != 0 || *slots < 1 || *slots > 2 {
		return errors.New("invalid supervisor")
	}
	keyFile, e := os.Open(*keyPath)
	if e != nil {
		return errors.New("fact_authentication_key_unavailable")
	}
	var keys struct {
		MasterKey string `json:"master_key"`
	}
	decodeErr := json.NewDecoder(io.LimitReader(keyFile, 16384)).Decode(&keys)
	keyFile.Close()
	factKey, e := base64.StdEncoding.DecodeString(keys.MasterKey)
	if decodeErr != nil || e != nil || len(factKey) != 32 {
		return errors.New("invalid_fact_authentication_key")
	}
	lock, e := os.OpenFile(*lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return e
	}
	defer lock.Close()
	if syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) != nil {
		return errors.New("supervisor already running")
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	c := client{http: &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", *socket)
	}}, Timeout: 30 * time.Second}}
	var active atomic.Int32
	var cleanupFailed atomic.Bool
	status := func(state, code string) {
		sc, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_ = c.call(sc, "/status", probe.WorkerStatus{Status: state, Code: code, Version: residential.Version, Active: int(active.Load())}, nil)
	}
	status("starting", "sandbox_preflight")
	for _, args := range [][]string{{"--cleanup"}, {"--core", *core, "--self-test"}} {
		sc, cancel := context.WithTimeout(ctx, 60*time.Second)
		cmd := exec.CommandContext(sc, "/usr/bin/python3", append([]string{*helper}, args...)...)
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		e = cmd.Run()
		cancel()
		if e != nil {
			status("blocked", "sandbox_preflight_failed")
			return errors.New("sandbox_preflight_failed")
		}
	}
	epoch := fmt.Sprintf("boot-%x", time.Now().UnixNano())
	var wg sync.WaitGroup
	for slot := 1; slot <= *slots; slot++ {
		wg.Add(1)
		go func(slot int) {
			defer wg.Done()
			delay := time.Second
			for {
				timer := time.NewTimer(delay)
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
				var j *residential.Job
				if e := c.call(ctx, "/claim", map[string]string{"worker": fmt.Sprintf("local-slot-%d", slot)}, &j); e != nil || j == nil {
					delay = min(5*time.Second, delay*2)
					continue
				}
				active.Add(1)
				err := execute(ctx, c, *helper, *core, fmt.Sprintf("local-slot-%d", slot), epoch, slot, *j, factKey)
				active.Add(-1)
				if err != nil {
					cleanupFailed.Store(true)
					status("blocked", "sandbox_cleanup_failed")
					stop()
					return
				}
				delay = 100 * time.Millisecond
			}
		}(slot)
	}
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	status("running", "")
	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			if cleanupFailed.Load() {
				return errors.New("sandbox_cleanup_failed")
			}
			return nil
		case <-tick.C:
			status("running", "")
		}
	}
}
func main() {
	if e := run(); e != nil {
		log.Fatal(e)
	}
}
