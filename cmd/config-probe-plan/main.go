package main

import (
	"encoding/json"
	"flag"
	"github.com/rezajafari0970/Config_manager/internal/compat"
	"github.com/rezajafari0970/Config_manager/internal/convert"
	"github.com/rezajafari0970/Config_manager/internal/probe"
	"github.com/rezajafari0970/Config_manager/internal/repair"
	"github.com/rezajafari0970/Config_manager/internal/smart"
	"io"
	"os"
)

// Read through stdin; never put credential-bearing configs in process arguments.
func main() {
	template := flag.Bool("template", false, "prepare final shared-template bytes")
	smartMode := flag.Bool("smart", false, "compose an array of final member objects")
	uploadCandidate := flag.Bool("experimental-upload-candidate", false, "append unqualified legacy TCP mux/native UDP alternative; default off")
	flag.Parse()
	b, e := io.ReadAll(io.LimitReader(os.Stdin, repair.MaxBytes+1))
	if e != nil {
		os.Exit(1)
	}
	plan := probe.Prepare(b)
	if *template {
		plan = convert.Prepare(b)
	}
	if *smartMode {
		var inputs []json.RawMessage
		if json.Unmarshal(b, &inputs) != nil {
			os.Exit(2)
		}
		members := []smart.Member{}
		for _, input := range inputs {
			members = append(members, smart.Member{Body: input})
		}
		body, err := smart.Compose(members)
		if err != nil {
			plan = probe.Plan{Code: err.Error()}
		} else {
			plan = smart.Prepare(body)
		}
	}
	if *uploadCandidate && plan.Code == "prepared" {
		original := append([]probe.Candidate{}, plan.Candidates...)
		for _, c := range original {
			if len(plan.Candidates) >= 12 {
				break
			}
			candidate, changes, err := compat.UploadAlternative(c.Config)
			if err == nil && len(changes) > 0 {
				plan.Candidates = append(plan.Candidates, probe.Candidate{Rule: compat.UploadDriver + "." + c.Rule, SHA256: repair.Digest(candidate), Config: candidate, RequiresConnection: true})
			}
		}
		plan.Version += ".experimental-upload-candidates.v1"
	}
	if json.NewEncoder(os.Stdout).Encode(plan) != nil {
		os.Exit(1)
	}
}
