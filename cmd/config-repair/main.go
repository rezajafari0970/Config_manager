package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/rezajafari0970/Config_manager/internal/repair"
	"github.com/rezajafari0970/Config_manager/internal/target"
)

func main() { os.Exit(run(os.Args[1:])) }
func run(args []string) int {
	if len(args) == 1 && args[0] == "target" {
		_ = json.NewEncoder(os.Stdout).Encode(target.Pinned())
		return 0
	}
	if len(args) != 2 && len(args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: config-repair target | inspect FILE | repair-stats INPUT NEW_OUTPUT")
		return 2
	}
	if args[0] != "inspect" && args[0] != "repair-stats" {
		fmt.Fprintln(os.Stderr, "unknown command")
		return 2
	}
	if args[0] == "inspect" && len(args) != 2 || args[0] == "repair-stats" && len(args) != 3 {
		return 2
	}
	f, err := os.Open(args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "input_open_failed")
		return 2
	}
	b, err := io.ReadAll(io.LimitReader(f, repair.MaxBytes+1))
	_ = f.Close()
	if err != nil {
		fmt.Fprintln(os.Stderr, "input_read_failed")
		return 2
	}
	if args[0] == "inspect" {
		r := repair.Analyze(b)
		_ = json.NewEncoder(os.Stdout).Encode(r)
		if r.HasErrors() {
			return 1
		}
		return 0
	}
	out, patch, err := repair.RepairStats(b)
	if err != nil {
		fmt.Fprintln(os.Stderr, "repair_rejected")
		return 1
	}
	d, err := os.OpenFile(args[2], os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		fmt.Fprintln(os.Stderr, "output_create_failed_no_overwrite")
		return 2
	}
	_, err = d.Write(out)
	if err == nil {
		err = d.Sync()
	}
	closeErr := d.Close()
	if err != nil || closeErr != nil {
		_ = os.Remove(args[2])
		fmt.Fprintln(os.Stderr, "output_write_failed")
		return 2
	}
	_ = json.NewEncoder(os.Stdout).Encode(struct {
		Patches any           `json:"patches"`
		Report  repair.Report `json:"report"`
	}{patch, repair.Analyze(out)})
	return 0
}
