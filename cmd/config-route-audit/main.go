// config-route-audit is an isolated, opt-in sample investigation. It never opens
// the production database or changes publication. Supply private config on stdin.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/rezajafari0970/Config_manager/internal/adsroute"
	"github.com/rezajafari0970/Config_manager/internal/egress"
)

func main() {
	if err := run(); err != nil {
		// Fixed errors only; helper stderr and private configs never enter stdout.
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	helper := flag.String("helper", "", "absolute path to this revision's egress-probe.py")
	flag.Parse()
	if os.Geteuid() != 0 || !filepath.IsAbs(*helper) || filepath.Base(*helper) != "egress-probe.py" {
		return errors.New("root and an explicit absolute helper path are required")
	}
	lock, err := os.OpenFile("/run/config-route-audit.lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return errors.New("audit_lock_unavailable")
	}
	defer lock.Close()
	if syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) != nil {
		return errors.New("another_sample_audit_is_running")
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	raw, err := io.ReadAll(io.LimitReader(os.Stdin, (8<<20)+1))
	if err != nil || len(raw) == 0 || len(raw) > 8<<20 {
		return errors.New("invalid_private_input")
	}
	p := egress.Prepare(raw)
	if !p.Complete || len(p.Variants) != 2 {
		return errors.New("exact_input_driver_unavailable")
	}
	var token [3]byte
	if _, err = rand.Read(token[:]); err != nil {
		return errors.New("run_id_unavailable")
	}
	id := hex.EncodeToString(token[:])
	args := []string{*helper, "--prefix", "cgt", "--slot", "31", "--run-id", id}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		exec.CommandContext(ctx, "python3", append(args, "--cleanup")...).Run()
	}()
	request, _ := json.Marshal(map[string]any{"hash": egress.Hash(raw), "plan": p, "started": egress.Now(), "deep": true, "accepted": true, "discovery": true, "targets": adsroute.Targets()})
	ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "python3", args...)
	cmd.Stdin = bytes.NewReader(request)
	cmd.WaitDelay = 5 * time.Second
	body, err := cmd.Output()
	if err != nil || len(body) > 2<<20 {
		return errors.New("isolated_driver_failed_or_timed_out")
	}
	var report egress.Report
	if json.Unmarshal(body, &report) != nil {
		return errors.New("invalid_driver_receipt")
	}
	return json.NewEncoder(os.Stdout).Encode(struct {
		Result adsroute.Result `json:"result"`
		Report egress.Report   `json:"report"`
	}{adsroute.Analyze(report, p, egress.Hash(raw), egress.Now()), report})
}
