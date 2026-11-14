package adsroute

import (
	_ "embed"
	"encoding/json"

	"github.com/rezajafari0970/Config_manager/internal/egress"
)

// The discovery registry is separate from the live v7 publication registry.
// Google documents ipv4.google.com as an IPv4-only hostname. We still use the
// hostname through SOCKS, authenticating that exact TLS name; never replace the
// destination with a locally resolved IP or infer the route of www.google.com.
// https://developers.google.com/speed/public-dns/docs/using
//
//go:embed targets.json
var targetJSON []byte

var registry = func() []egress.Target {
	var targets []egress.Target
	if json.Unmarshal(targetJSON, &targets) != nil || len(targets) == 0 || len(targets) > 32 {
		panic("invalid route discovery registry")
	}
	seen := map[string]bool{}
	for _, t := range targets {
		if seen[t.Domain] || t.Domain == "" || t.Group == "" || (t.Role != "ads" && t.Role != "control") {
			panic("invalid route discovery target")
		}
		seen[t.Domain] = true
	}
	return targets
}()

func Targets() []egress.Target { return append([]egress.Target(nil), registry...) }
