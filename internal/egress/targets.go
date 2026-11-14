package egress

import (
	"embed"
	"encoding/json"
)

//go:embed targets.json
var targetAssets embed.FS

type Target struct {
	Domain   string `json:"domain"`
	Group    string `json:"group"`
	Role     string `json:"role"`
	Driver   string `json:"driver"`
	Required bool   `json:"required"`
}

// The compiled registry is authoritative. Helpers receive it in their request;
// a report cannot weaken required coverage or introduce an untrusted endpoint.
var registry = func() []Target {
	b, err := targetAssets.ReadFile("targets.json")
	if err != nil {
		panic(err)
	}
	var targets []Target
	if err = json.Unmarshal(b, &targets); err != nil {
		panic(err)
	}
	seen := map[string]bool{}
	for _, t := range targets {
		if t.Domain == "" || seen[t.Domain] || (t.Role != "ads" && t.Role != "control") {
			panic("invalid target registry")
		}
		seen[t.Domain] = true
	}
	return targets
}()

func Targets() []Target { return append([]Target(nil), registry...) }
func targetNames(role string) []string {
	out := []string{}
	for _, t := range registry {
		if t.Role == role {
			out = append(out, t.Domain)
		}
	}
	return out
}

var Domains = targetNames("ads")
var ControlDomains = targetNames("control")

func targetFor(domain string) (Target, bool) {
	for _, t := range registry {
		if t.Domain == domain {
			return t, true
		}
	}
	return Target{}, false
}

type ServiceDecision struct {
	Variant      string `json:"variant"`
	Domain       string `json:"domain"`
	Group        string `json:"group"`
	Status       string `json:"status"`
	Observations int    `json:"observations"`
	Required     bool   `json:"required"`
}
