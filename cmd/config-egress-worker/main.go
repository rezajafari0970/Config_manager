// Bounded adaptive supervisor for finite Ads egress investigations.
package main

import (
	"bufio"
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
	"runtime"
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
func execute(ctx context.Context, c client, driver *driverSession, j egress.Job) {
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
		input, _ := json.Marshal(map[string]any{"hash": j.Hash, "started": started, "plan": p, "targets": egress.Targets(), "deep": j.Deep, "attempt": j.Attempt, "accepted": j.Accepted})
		output, err := driver.exchange(job, input)
		if err != nil || json.Unmarshal(output, &r) != nil {
			driver.stop()
			r = egress.Report{Version: egress.Version, Hash: j.Hash, Started: started, Finished: egress.Now(), Code: "driver_unavailable"}
		}
	}
	if r.Finished == 0 {
		r.Finished = egress.Now()
	}
	j.Raw = nil
	if e := c.call(ctx, "complete", egress.Completion{Job: j, Report: r}, nil); e != nil {
		log.Print("egress completion deferred; lease will recover")
	}
}

// One trusted helper and network namespace per slot. Every request launches a
// fresh capability-free core; the helper kills its whole cgroup and denies all
// egress before acknowledging a result. Failure destroys the slot and restarts it.
type driverSession struct {
	owner               context.Context
	helper, prefix, run string
	slot                int
	cmd                 *exec.Cmd
	input               io.WriteCloser
	responses           <-chan []byte
	closed              chan struct{}
}

func (d *driverSession) start() error {
	d.run = egress.Hash([]byte(fmt.Sprint(os.Getpid(), d.slot, time.Now().UnixNano())))[:6]
	cmd := exec.CommandContext(d.owner, "/usr/bin/python3", d.helper, "--prefix", d.prefix, "--slot", fmt.Sprint(d.slot), "--run-id", d.run, "--serve")
	cmd.Stderr = io.Discard
	cmd.WaitDelay = 3 * time.Second
	input, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		input.Close()
		return err
	}
	if err = cmd.Start(); err != nil {
		input.Close()
		return err
	}
	responses := make(chan []byte, 1)
	closed := make(chan struct{})
	d.cmd = cmd
	d.input = input
	d.responses = responses
	d.closed = closed
	go func() {
		defer close(responses)
		scanner := bufio.NewScanner(output)
		scanner.Buffer(make([]byte, 65536), 1<<20)
		for scanner.Scan() {
			body := append([]byte(nil), scanner.Bytes()...)
			select {
			case responses <- body:
			case <-closed:
				return
			}
		}
	}()
	return nil
}
func (d *driverSession) stop() {
	if d.cmd == nil {
		return
	}
	close(d.closed)
	d.input.Close()
	_ = d.cmd.Process.Kill()
	_ = d.cmd.Wait()
	clean(d.helper, d.prefix, d.slot, d.run)
	d.cmd = nil
	d.input = nil
	d.responses = nil
}
func (d *driverSession) exchange(ctx context.Context, input []byte) ([]byte, error) {
	if d.cmd == nil {
		if err := d.start(); err != nil {
			return nil, err
		}
	}
	written := make(chan error, 1)
	writer := d.input
	go func() { _, err := writer.Write(append(input, '\n')); written <- err }()
	select {
	case err := <-written:
		if err != nil {
			d.stop()
			return nil, err
		}
	case <-ctx.Done():
		d.stop()
		return nil, ctx.Err()
	}
	select {
	case result, ok := <-d.responses:
		if !ok {
			d.stop()
			return nil, errors.New("driver closed")
		}
		return result, nil
	case <-ctx.Done():
		d.stop()
		return nil, ctx.Err()
	}
}

func run() error {
	socket := flag.String("socket", "/var/lib/config-repair/probe.sock", "local protected queue")
	helper := flag.String("helper", "/opt/config-repair/current/scripts/egress-probe.py", "trusted driver")
	prefix := flag.String("prefix", "cge", "isolated scope")
	lockPath := flag.String("lock-file", "/run/config-repair-egress.lock", "single supervisor")
	workers := flag.Int("workers", min(egress.MaxWorkers, runtime.NumCPU()*2+2), "maximum concurrent isolated jobs")
	flag.Parse()
	if *workers < 1 || *workers > egress.MaxWorkers {
		return errors.New("invalid worker limit")
	}
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
	governor := resourceGovernor{current: resourceCapacity(*workers, sampleResources()), changed: time.Now()}
	var capacityLimit atomic.Int32
	capacityLimit.Store(int32(governor.current))
	var active atomic.Int32
	var wg sync.WaitGroup
	for slot := 1; slot <= *workers; slot++ {
		wg.Add(1)
		go func(slot int) {
			defer wg.Done()
			driver := &driverSession{owner: ctx, helper: *helper, prefix: *prefix, slot: slot}
			defer driver.stop()
			var claims uint64
			for ctx.Err() == nil {
				var j *egress.Job
				capacity := int(capacityLimit.Load())
				lane := investigationLane(slot, capacity, claims)
				if slot <= capacity && c.call(ctx, "claim", map[string]string{"lane": lane}, &j) == nil && j != nil {
					claims++
					active.Add(1)
					execute(ctx, c, driver, *j)
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
	notifySystemd("READY=1")
	defer notifySystemd("STOPPING=1")
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for ctx.Err() == nil {
		notifySystemd("WATCHDOG=1")
		capacityLimit.Store(int32(governor.update(resourceCapacity(*workers, sampleResources()), time.Now())))
		_ = c.call(ctx, "status", map[string]any{"status": "running", "active": active.Load(), "code": fmt.Sprintf("capacity_%d", capacityLimit.Load())}, nil)
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
