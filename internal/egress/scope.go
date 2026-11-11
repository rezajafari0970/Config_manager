package egress

import "encoding/json"

// These exceptions cannot match the named HTTPS probes. The sandbox continues
// to block direct DNS; original runtime bytes are preserved. Keep the matching
// Python driver rule in egress-preflight.py and its adversarial tests in sync.
func dnsBypass(raw json.RawMessage) bool {
	var r map[string]json.RawMessage
	if json.Unmarshal(raw, &r) != nil {
		return false
	}
	allowed := map[string]bool{"type": true, "port": true, "network": true, "inboundTag": true, "ip": true, "outboundTag": true}
	for k := range r {
		if !allowed[k] {
			return false
		}
	}
	var kind, port string
	if v, ok := r["type"]; ok && (json.Unmarshal(v, &kind) != nil || kind != "field") {
		return false
	}
	if json.Unmarshal(r["port"], &port) == nil && port == "53" {
		return true
	}
	var ips []string
	if json.Unmarshal(r["ip"], &ips) != nil || len(ips) == 0 {
		return false
	}
	resolvers := map[string]bool{"1.1.1.1": true, "1.0.0.1": true, "8.8.8.8": true, "8.8.4.4": true, "9.9.9.9": true, "149.112.112.112": true, "104.16.248.249": true, "104.16.249.249": true}
	for _, ip := range ips {
		if !resolvers[ip] {
			return false
		}
	}
	return true
}
