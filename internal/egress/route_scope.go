package egress

import (
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/compat"
	"strings"
)

// Only exact, bounded rule semantics are proved locally. Unknown conditions,
// DNS-dependent rules, regex/geosite and balancers never become invented blocks.
func localBlocked(v Variant, domain string) bool {
	var root compat.M
	if json.Unmarshal(v.Config, &root) != nil {
		return false
	}
	if _, err := compat.Logical(root); err != nil {
		return false
	}
	b, _ := json.Marshal(root)
	var cfg struct {
		Outbounds []struct{ Tag, Protocol string }
		Inbounds  []struct {
			Tag, Protocol string
			Port          int
		}
		Routing struct {
			DomainStrategy string
			Rules          []map[string]json.RawMessage
		}
	}
	if json.Unmarshal(b, &cfg) != nil || len(cfg.Outbounds) == 0 {
		return false
	}
	inbound := ""
	for _, i := range cfg.Inbounds {
		if i.Protocol == "socks" && i.Port == 10808 {
			inbound = i.Tag
		}
	}
	kinds := map[string]string{}
	for _, o := range cfg.Outbounds {
		kinds[o.Tag] = o.Protocol
	}
	for _, r := range cfg.Routing.Rules {
		var domains, ins []string
		var network, port, tag, kind string
		if x, ok := r["inboundTag"]; ok {
			if json.Unmarshal(x, &ins) != nil {
				return false
			}
			found := false
			for _, i := range ins {
				if i == inbound {
					found = true
				}
			}
			if !found {
				continue
			}
		}
		if x, ok := r["network"]; ok {
			if json.Unmarshal(x, &network) != nil {
				return false
			}
			if network == "udp" {
				continue
			}
			if network != "tcp" && network != "tcp,udp" {
				return false
			}
		}
		if x, ok := r["port"]; ok {
			if json.Unmarshal(x, &port) != nil {
				return false
			}
			if port == "53" {
				continue
			}
			if port != "443" {
				return false
			}
		}
		matched := true
		if x, ok := r["domain"]; ok {
			if json.Unmarshal(x, &domains) != nil || len(domains) == 0 {
				return false
			}
			matched = false
			for _, d := range domains {
				switch {
				case strings.HasPrefix(d, "full:"):
					matched = matched || domain == strings.TrimPrefix(d, "full:")
				case strings.HasPrefix(d, "domain:"):
					s := strings.TrimPrefix(d, "domain:")
					matched = matched || domain == s || strings.HasSuffix(domain, "."+s)
				default:
					return false
				}
			}
			if !matched {
				continue
			}
		}
		allowed := map[string]bool{"type": true, "domain": true, "network": true, "port": true, "inboundTag": true, "outboundTag": true, "ruleTag": true}
		for k := range r {
			if !allowed[k] {
				return false
			}
		}
		if x, ok := r["type"]; ok && (json.Unmarshal(x, &kind) != nil || kind != "field") {
			return false
		}
		if json.Unmarshal(r["outboundTag"], &tag) != nil {
			return false
		}
		return kinds[tag] == "blackhole"
	}
	return cfg.Outbounds[0].Protocol == "blackhole"
}
