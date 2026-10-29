package main

import (
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/probe"
	"github.com/rezajafari0970/Config_manager/internal/repair"
	"io"
	"os"
)

// Read through stdin; never put credential-bearing configs in process arguments.
func main() {
	b, e := io.ReadAll(io.LimitReader(os.Stdin, repair.MaxBytes+1))
	if e != nil {
		os.Exit(1)
	}
	if json.NewEncoder(os.Stdout).Encode(probe.Prepare(b)) != nil {
		os.Exit(1)
	}
}
