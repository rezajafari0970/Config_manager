package smart

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rezajafari0970/Config_manager/internal/probe"
	"github.com/rezajafari0970/Config_manager/internal/repair"
	"github.com/rezajafari0970/Config_manager/internal/target"
	"sort"
	"strconv"
	"strings"
)

type M = map[string]any

const base = `{"remarks":"@shenvpn-smart-parallel-v6b","log":{"loglevel":"none","access":"none","error":"none","dnsLog":false},"dns":{"tag":"dns-module","queryStrategy":"UseIPv4","servers":[{"address":"https://8.8.8.8/dns-query","domains":[],"skipFallback":true,"tag":"dns-bootstrap","timeoutMs":2500},{"address":"https://1.1.1.1/dns-query","domains":[],"skipFallback":true,"tag":"dns-bootstrap","timeoutMs":2500},{"address":"https://1.1.1.1/dns-query","skipFallback":false,"timeoutMs":2500},{"address":"https://8.8.8.8/dns-query","skipFallback":false,"timeoutMs":2500}],"disableFallbackIfMatch":true,"disableCache":false,"serveStale":true,"serveExpiredTTL":60,"enableParallelQuery":true},"inbounds":[{"tag":"socks","listen":"127.0.0.1","port":10808,"protocol":"socks","settings":{"auth":"noauth","udp":true},"sniffing":{"enabled":true,"destOverride":["http","tls","quic"],"routeOnly":true}}],"outbounds":[],"routing":{"domainStrategy":"AsIs","balancers":[{"tag":"smart-parallel-balancer","selector":["proxy-"],"strategy":{"type":"leastLoad","settings":{"expected":3,"maxRTT":"2s","tolerance":0.15}},"fallbackTag":"block"},{"tag":"bootstrap-ip-balancer","selector":[],"strategy":{"type":"leastLoad","settings":{"expected":3,"maxRTT":"2s","tolerance":0.15}},"fallbackTag":"block"}],"rules":[{"type":"field","inboundTag":["socks","tun"],"port":"53","network":"tcp,udp","outboundTag":"smart-dns-out"},{"type":"field","inboundTag":["dns-bootstrap"],"balancerTag":"bootstrap-ip-balancer"},{"type":"field","inboundTag":["dns-module","smart-dns-egress"],"balancerTag":"smart-parallel-balancer"},{"type":"field","network":"tcp,udp","balancerTag":"smart-parallel-balancer"}]},"burstObservatory":{"subjectSelector":["proxy-"],"pingConfig":{"destination":"https://www.gstatic.com/generate_204","interval":"15s","sampling":3,"timeout":"5s","httpMethod":"GET"}},"policy":{"levels":{"0":{"handshake":15,"connIdle":600,"uplinkOnly":2,"downlinkOnly":5}},"system":{"statsInboundUplink":true,"statsInboundDownlink":true,"statsOutboundUplink":true,"statsOutboundDownlink":true}},"stats":{},"metrics":{"tag":"debug-metrics","listen":"127.0.0.1:11111"}}`

func object(b []byte) (M, error) {
	var m M
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	if d.Decode(&m) != nil || m == nil {
		return nil, errors.New("smart_invalid_member")
	}
	return m, nil
}
func Prepare(b []byte) probe.Plan {
	p := probe.Plan{Target: target.Pinned().ID, Version: Version, OriginalSHA256: repair.Digest(b), Kind: "smart", Code: "prepared", AndroidImport: "not_run"}
	if len(b) > repair.MaxBytes || !json.Valid(b) {
		p.Code = "smart_invalid_profile"
		return p
	}
	p.Candidates = []probe.Candidate{{Rule: Version, SHA256: repair.Digest(b), Config: bytes.Clone(b)}}
	return p
}

// Compose retains complete member outbounds inside the responsive Smart envelope.
// Accepted member JSON is decoded into new objects; raw stored bytes are never edited.
func Compose(members []Member) ([]byte, error) {
	if len(members) == 0 {
		return nil, errors.New("smart_empty_profile")
	}
	sources := make([]M, len(members))
	total := 0
	for slot, member := range members {
		source, e := object(member.Body)
		if e != nil {
			return nil, e
		}
		nodes, ok := source["outbounds"].([]any)
		if !ok || len(nodes) == 0 {
			return nil, errors.New("smart_missing_outbounds")
		}
		seen := map[string]bool{}
		for _, node := range nodes {
			ob, ok := node.(M)
			if !ok {
				return nil, errors.New("smart_invalid_outbound")
			}
			tag, _ := ob["tag"].(string)
			if tag == "" || seen[tag] {
				return nil, errors.New("smart_duplicate_tag")
			}
			seen[tag] = true
		}
		selected, e := entryTags(source, nodes)
		if e != nil {
			return nil, e
		}
		total += len(selected)
		sources[slot] = source
	}
	width := max(2, len(strconv.Itoa(total)))
	entries, helpers := []any{}, []any{}
	bootstrap := map[string]bool{}
	bootstrapTags := []string{}
	for slot, source := range sources {
		nodes := source["outbounds"].([]any)
		byTag := map[string]M{}
		tags := map[string]string{}
		for i, value := range nodes {
			ob, ok := value.(M)
			if !ok {
				return nil, errors.New("smart_invalid_outbound")
			}
			tag, _ := ob["tag"].(string)
			if tag == "" || byTag[tag] != nil {
				return nil, errors.New("smart_duplicate_tag")
			}
			collectEndpointDomains(ob, bootstrap)
			byTag[tag] = ob
			tags[tag] = fmt.Sprintf("member-%04d-out-%04d_", slot+1, i)
		}
		selected, e := entryTags(source, nodes)
		if e != nil {
			return nil, e
		}
		selectedSet := map[string]bool{}
		for _, tag := range selected {
			if byTag[tag] == nil {
				return nil, errors.New("smart_missing_dependency")
			}
			switch byTag[tag]["protocol"] {
			case "freedom", "blackhole", "dns", "loopback":
				return nil, errors.New("smart_requires_source_routing")
			}
			selectedSet[tag] = true
			tags[tag] = fmt.Sprintf("proxy-%0*d", width, len(entries)+1)
			entries = append(entries, byTag[tag])
		}
		// A copied entry must remain usable with the global proxy routes.
		// Active loopback chains need source-specific routing and cannot be
		// silently represented by this template. Inactive helpers are still copied.
		visiting, visited := map[string]bool{}, map[string]bool{}
		var check func(string) error
		check = func(tag string) error {
			if visited[tag] {
				return nil
			}
			if visiting[tag] {
				return errors.New("smart_dependency_cycle")
			}
			ob := byTag[tag]
			if ob == nil {
				return errors.New("smart_missing_dependency")
			}
			if ob["protocol"] == "loopback" {
				return errors.New("smart_requires_source_routing")
			}
			visiting[tag] = true
			for _, ref := range outboundReferences(ob) {
				if e := check(ref); e != nil {
					return e
				}
			}
			delete(visiting, tag)
			visited[tag] = true
			return nil
		}
		for _, tag := range selected {
			if e := check(tag); e != nil {
				return nil, e
			}
		}
		for _, tag := range selected {
			if independentIP(tag, byTag, map[string]bool{}) {
				bootstrapTags = append(bootstrapTags, tags[tag])
			}
		}
		for _, value := range nodes {
			ob := value.(M)
			old := ob["tag"].(string)
			ob["tag"] = tags[old]
			applySmartSocketPolicy(ob)
			if e := rewriteReferences(ob, tags, fmt.Sprintf("member-%04d-", slot+1)); e != nil {
				return nil, e
			}
			if !selectedSet[old] {
				helpers = append(helpers, ob)
			}
		}
	}
	if len(bootstrapTags) == 0 {
		return nil, errors.New("smart_bootstrap_no_ip")
	}
	// AndroidLibXrayLite MeasureOutboundDelay removes routing/DNS/observers.
	// Its default outbound must therefore be a real, DNS-independent proxy.
	// Normal operation still uses the catch-all leastLoad rule and block fallback.
	for i, entry := range entries {
		if entry.(M)["tag"] == bootstrapTags[0] {
			entries[0], entries[i] = entries[i], entries[0]
			break
		}
	}
	out := append([]any{}, entries...)
	out = append(out, helpers...)
	out = append(out, M{"tag": "block", "protocol": "blackhole", "settings": M{}})
	out = append(out, M{"tag": "smart-dns-out", "protocol": "dns", "settings": M{"rewriteNetwork": "tcp", "rewriteAddress": "1.1.1.1", "rewritePort": 53, "rules": []any{M{"action": "hijack", "qType": "1,28"}, M{"action": "direct"}}}, "streamSettings": M{"sockopt": M{"dialerProxy": "smart-dns-relay"}}}, M{"tag": "smart-dns-relay", "protocol": "loopback", "settings": M{"inboundTag": "smart-dns-egress"}})
	var body bytes.Buffer
	if e := orderedJSON(&body, out); e != nil {
		return nil, e
	}
	// Raw replacement preserves fixed field order; bootstrap is scoped to endpoint names.
	result := []byte(base)
	domains := make([]string, 0, len(bootstrap))
	for name := range bootstrap {
		domains = append(domains, "full:"+name)
	}
	sort.Strings(domains)
	encoded, _ := json.Marshal(domains)
	result = bytes.ReplaceAll(result, []byte("\"domains\":[]"), append([]byte("\"domains\":"), encoded...))
	encoded, _ = json.Marshal(bootstrapTags)
	result = bytes.Replace(result, []byte("\"selector\":[]"), append([]byte("\"selector\":"), encoded...), 1)
	result = bytes.Replace(result, []byte("\"outbounds\":[]"), append([]byte("\"outbounds\":"), body.Bytes()...), 1)
	if len(result) > repair.MaxBytes {
		return nil, errors.New("smart_profile_size_limit")
	}
	return result, nil
}

// entryTags flattens only the selected top-level pool into the single requested
// latency balancer. Other outbound definitions remain copied helpers.
func entryTags(source M, nodes []any) ([]string, error) {
	routing, _ := source["routing"].(M)
	rules, _ := routing["rules"].([]any)
	primary, balanced := "", ""
	for _, value := range rules {
		r, _ := value.(M)
		if r["network"] == "tcp,udp" && r["inboundTag"] == nil && r["port"] == nil {
			// Refuse hidden predicates instead of treating a conditional route as universal.
			for key := range r {
				if key != "type" && key != "network" && key != "outboundTag" && key != "balancerTag" {
					return nil, errors.New("smart_ambiguous_entry")
				}
			}
			if primary != "" || balanced != "" {
				return nil, errors.New("smart_ambiguous_entry")
			}
			primary, _ = r["outboundTag"].(string)
			balanced, _ = r["balancerTag"].(string)
		}
	}
	if (primary == "") == (balanced == "") {
		return nil, errors.New("smart_ambiguous_entry")
	}
	if primary != "" {
		return []string{primary}, nil
	}
	balancers, _ := routing["balancers"].([]any)
	var pool M
	for _, value := range balancers {
		b, _ := value.(M)
		if b["tag"] == balanced {
			if pool != nil {
				return nil, errors.New("smart_duplicate_balancer")
			}
			pool = b
		}
	}
	if pool == nil {
		return nil, errors.New("smart_missing_balancer")
	}
	selector, ok := pool["selector"].([]any)
	if !ok || len(selector) == 0 {
		return nil, errors.New("smart_invalid_selector")
	}
	for _, value := range selector {
		if text, ok := value.(string); !ok || text == "" {
			return nil, errors.New("smart_invalid_selector")
		}
	}
	chosen := []string{}
	for _, value := range nodes {
		tag := value.(M)["tag"].(string)
		for _, prefix := range selector {
			if strings.HasPrefix(tag, prefix.(string)) {
				chosen = append(chosen, tag)
				break
			}
		}
	}
	if len(chosen) == 0 {
		return nil, errors.New("smart_empty_balancer")
	}
	return chosen, nil
}

// References are rewritten in the same parsed copy, including nested download
// streams. Credentials, TLS, transport, Mux and unknown extension values survive.
func outboundReferences(value any) []string {
	refs := []string{}
	var walk func(any)
	walk = func(value any) {
		switch v := value.(type) {
		case M:
			for k, child := range v {
				if k == "dialerProxy" || k == "outboundTag" {
					if s, ok := child.(string); ok && s != "" {
						refs = append(refs, s)
					}
				}
				if k == "proxySettings" {
					if p, ok := child.(M); ok {
						if s, ok := p["tag"].(string); ok && s != "" {
							refs = append(refs, s)
						}
					}
				}
				walk(child)
			}
		case []any:
			for _, child := range v {
				walk(child)
			}
		}
	}
	walk(value)
	return refs
}
func rewriteReferences(value any, tags map[string]string, prefix string) error {
	switch v := value.(type) {
	case M:
		for k, child := range v {
			switch k {
			case "dialerProxy", "outboundTag":
				if s, ok := child.(string); ok && s != "" {
					if tags[s] == "" {
						return errors.New("smart_missing_dependency")
					}
					v[k] = tags[s]
					continue
				}
			case "proxySettings":
				if p, ok := child.(M); ok {
					if s, ok := p["tag"].(string); ok && s != "" {
						if tags[s] == "" {
							return errors.New("smart_missing_dependency")
						}
						p["tag"] = tags[s]
					}
				}
			}
			if e := rewriteReferences(child, tags, prefix); e != nil {
				return e
			}
		}
		// Loopback inboundTag is a separate namespace, not an outbound reference.
		if v["protocol"] == "loopback" {
			if settings, ok := v["settings"].(M); ok {
				if tag, ok := settings["inboundTag"].(string); ok && tag != "" {
					settings["inboundTag"] = prefix + "in-" + tag
				}
			}
		}
	case []any:
		for _, child := range v {
			if e := rewriteReferences(child, tags, prefix); e != nil {
				return e
			}
		}
	}
	return nil
}

// Stable schema-friendly field order without discarding unknown JSON fields or
// coercing numbers. Arrays (including users/servers/ALPN) retain source order.
func orderedJSON(out *bytes.Buffer, value any) error {
	switch v := value.(type) {
	case M:
		order := []string{"tag", "protocol", "sendThrough", "settings", "streamSettings", "proxySettings", "mux", "vnext", "servers", "address", "port", "users", "id", "alterId", "encryption", "flow", "method", "password", "network", "security", "tcpSettings", "rawSettings", "xhttpSettings", "wsSettings", "grpcSettings", "httpSettings", "tlsSettings", "realitySettings", "sockopt", "header", "type", "request", "version", "mode", "path", "headers", "show", "serverName", "fingerprint", "publicKey", "shortId", "spiderX", "alpn"}
		rank := map[string]int{}
		for i, k := range order {
			rank[k] = i + 1
		}
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			a, b := rank[keys[i]], rank[keys[j]]
			if a == 0 {
				a = len(order) + 1
			}
			if b == 0 {
				b = len(order) + 1
			}
			if a != b {
				return a < b
			}
			return keys[i] < keys[j]
		})
		out.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				out.WriteByte(',')
			}
			b, _ := json.Marshal(k)
			out.Write(b)
			out.WriteByte(':')
			if e := orderedJSON(out, v[k]); e != nil {
				return e
			}
		}
		out.WriteByte('}')
	case []any:
		out.WriteByte('[')
		for i, child := range v {
			if i > 0 {
				out.WriteByte(',')
			}
			if e := orderedJSON(out, child); e != nil {
				return e
			}
		}
		out.WriteByte(']')
	default:
		b, e := json.Marshal(v)
		if e != nil {
			return e
		}
		out.Write(b)
	}
	return nil
}
