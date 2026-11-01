package smart

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rezajafari0970/Config_manager/internal/probe"
	"github.com/rezajafari0970/Config_manager/internal/repair"
	"github.com/rezajafari0970/Config_manager/internal/target"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type M = map[string]any

const base = `{"remarks":"@shenvpn-smart-parallel-v6b","log":{"loglevel":"debug","dnsLog":true},"dns":{"tag":"dns-module","queryStrategy":"UseIPv4","servers":[{"address":"https://1.1.1.1/dns-query","skipFallback":false},{"address":"https://8.8.8.8/dns-query","skipFallback":false},{"address":"https://9.9.9.9/dns-query","skipFallback":false}]},"inbounds":[{"tag":"socks","listen":"127.0.0.1","port":10808,"protocol":"socks","settings":{"auth":"noauth","udp":true},"sniffing":{"enabled":true,"destOverride":["http","tls","quic"],"routeOnly":true}}],"outbounds":[],"routing":{"domainStrategy":"AsIs","balancers":[],"rules":[]},"burstObservatory":{"subjectSelector":["proxy-"],"pingConfig":{"destination":"https://www.gstatic.com/generate_204","interval":"20s","sampling":3,"timeout":"4s","httpMethod":"GET"}},"policy":{"levels":{"0":{"handshake":15,"connIdle":600,"uplinkOnly":2,"downlinkOnly":5}}},"stats":{}}`

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
func Compose(members []Member) ([]byte, error) {
	if len(members) == 0 {
		return nil, errors.New("smart_empty_profile")
	}
	root, _ := object([]byte(base))
	out := []any{}
	entries := []any{}
	rules := []any{}
	balancers := []any{}
	subjects := []string{"proxy-"}
	bootstrap := map[string]bool{}
	for slot, member := range members {
		source, e := object(member.Body)
		if e != nil {
			return nil, e
		}
		nodes, ok := source["outbounds"].([]any)
		if !ok || len(nodes) == 0 {
			return nil, errors.New("smart_missing_outbounds")
		}
		routing, _ := source["routing"].(M)
		originalRules, _ := routing["rules"].([]any)
		originalBalancers, _ := routing["balancers"].([]any)
		primary, balanced := "", ""
		for _, v := range originalRules {
			r, _ := v.(M)
			if r["network"] == "tcp,udp" && r["inboundTag"] == nil && r["port"] == nil {
				primary, _ = r["outboundTag"].(string)
				balanced, _ = r["balancerTag"].(string)
			}
		}
		if (primary == "") == (balanced == "") {
			return nil, errors.New("smart_ambiguous_entry")
		}
		prefix := fmt.Sprintf("member-%04d-", slot+1)
		entry := fmt.Sprintf("proxy-%04d", slot+1)
		tags := map[string]string{}
		bTags := map[string]string{}
		inTags := map[string]string{}
		oldOrder := []string{}
		for i, v := range nodes {
			ob, ok := v.(M)
			if !ok {
				return nil, errors.New("smart_invalid_outbound")
			}
			tag, _ := ob["tag"].(string)
			if tag == "" || tags[tag] != "" {
				return nil, errors.New("smart_duplicate_tag")
			}
			tags[tag] = fmt.Sprintf("%sout-%04d_", prefix, i)
			oldOrder = append(oldOrder, tag)
		}
		if primary != "" {
			if tags[primary] == "" {
				return nil, errors.New("smart_missing_dependency")
			}
			tags[primary] = entry
		}
		for i, v := range originalBalancers {
			b, ok := v.(M)
			if !ok {
				return nil, errors.New("smart_invalid_balancer")
			}
			tag, _ := b["tag"].(string)
			if tag == "" || bTags[tag] != "" {
				return nil, errors.New("smart_duplicate_balancer")
			}
			bTags[tag] = fmt.Sprintf("%sbal-%04d_", prefix, i)
		}
		if balanced != "" && bTags[balanced] == "" {
			return nil, errors.New("smart_missing_balancer")
		}
		inName := func(old string) string {
			if inTags[old] == "" {
				inTags[old] = fmt.Sprintf("%sin-%04d_", prefix, len(inTags))
			}
			return inTags[old]
		}
		var rewrite func(any) error
		rewrite = func(value any) error {
			switch v := value.(type) {
			case M:
				for k, child := range v {
					switch k {
					case "outboundTag", "dialerProxy":
						if old, ok := child.(string); ok && old != "" {
							if tags[old] == "" {
								return errors.New("smart_missing_dependency")
							}
							v[k] = tags[old]
							continue
						}
					case "balancerTag":
						if old, ok := child.(string); ok && old != "" {
							if bTags[old] == "" {
								return errors.New("smart_missing_balancer")
							}
							v[k] = bTags[old]
							continue
						}
					case "inboundTag":
						switch old := child.(type) {
						case string:
							v[k] = inName(old)
						case []any:
							values := []string{}
							for _, x := range old {
								s, ok := x.(string)
								if !ok {
									return errors.New("smart_invalid_route")
								}
								values = append(values, inName(s))
							}
							v[k] = values
						}
						continue
					case "proxySettings":
						if p, ok := child.(M); ok {
							if old, ok := p["tag"].(string); ok && old != "" {
								if tags[old] == "" {
									return errors.New("smart_missing_dependency")
								}
								p["tag"] = tags[old]
							}
						}
					}
					if e := rewrite(child); e != nil {
						return e
					}
				}
			case []any:
				for _, x := range v {
					if e := rewrite(x); e != nil {
						return e
					}
				}
			}
			return nil
		}
		expand := func(value any) ([]string, error) {
			raw, ok := value.([]any)
			if !ok {
				return nil, errors.New("smart_invalid_selector")
			}
			chosen := []string{}
			for _, tag := range oldOrder {
				for _, p := range raw {
					s, ok := p.(string)
					if !ok || s == "" {
						return nil, errors.New("smart_invalid_selector")
					}
					if strings.HasPrefix(tag, s) {
						chosen = append(chosen, tags[tag])
						break
					}
				}
			}
			return chosen, nil
		}
		for _, v := range originalBalancers {
			b := v.(M)
			selector, e := expand(b["selector"])
			if e != nil || len(selector) == 0 {
				return nil, errors.New("smart_empty_balancer")
			}
			b["selector"] = selector
			b["tag"] = bTags[b["tag"].(string)]
			if fallback, ok := b["fallbackTag"].(string); ok && fallback != "" {
				if tags[fallback] == "" {
					return nil, errors.New("smart_missing_dependency")
				}
				b["fallbackTag"] = tags[fallback]
			} else {
				b["fallbackTag"] = tags[oldOrder[0]]
			}
			strategy, _ := b["strategy"].(M)
			settings, _ := strategy["settings"].(M)
			if costs, ok := settings["costs"].([]any); ok {
				mapped := []any{}
				for _, item := range costs {
					cost, ok := item.(M)
					if !ok {
						return nil, errors.New("smart_invalid_cost")
					}
					match, _ := cost["match"].(string)
					isRegex, _ := cost["regexp"].(bool)
					var re *regexp.Regexp
					if isRegex {
						re, e = regexp.Compile(match)
						if e != nil {
							return nil, errors.New("smart_invalid_cost")
						}
					}
					for _, tag := range oldOrder {
						matched := ""
						if strings.Contains(tag, match) {
							matched = match
						}
						if isRegex {
							matched = re.FindString(tag)
						}
						if matched != "" {
							copy := M{}
							for k, x := range cost {
								copy[k] = x
							}
							// Freeze the old core's substring/regexp auto cost before renaming.
							value, _ := strconv.ParseFloat(fmt.Sprint(cost["value"]), 64)
							if value <= 0 {
								value = 1
								number := regexp.MustCompile(`\d+(\.\d+)?`).FindString(matched)
								if number != "" {
									value, _ = strconv.ParseFloat(number, 64)
								}
							}
							copy["value"] = value
							copy["match"] = tags[tag]
							copy["regexp"] = false
							mapped = append(mapped, copy)
						}
					}
				}
				settings["costs"] = mapped
			}
			balancers = append(balancers, b)
			subjects = append(subjects, selector...)
		}
		// Retain internal loopback dispatch rules, but replace source-global SOCKS/DNS
		// routing with the requested smart envelope.
		for _, v := range originalRules {
			r, _ := v.(M)
			ins, ok := r["inboundTag"].([]any)
			if !ok {
				continue
			}
			internal := false
			for _, x := range ins {
				if x != "socks" && x != "dns-module" {
					internal = true
				}
			}
			if !internal {
				continue
			}
			if e := rewrite(r); e != nil {
				return nil, e
			}
			rules = append(rules, r)
		}
		if balanced != "" {
			in := inName("smart-entry")
			entries = append(entries, M{"tag": entry, "protocol": "loopback", "settings": M{"inboundTag": in}})
			rules = append(rules, M{"type": "field", "inboundTag": []string{in}, "balancerTag": bTags[balanced]})
		}
		// Place the selected input first; preserve each component's real default as
		// the explicit fallback for its inner balancers.
		ordered := nodes
		if primary != "" {
			ordered = []any{}
			for _, v := range nodes {
				if v.(M)["tag"] == primary {
					ordered = append(ordered, v)
				}
			}
			for _, v := range nodes {
				if v.(M)["tag"] != primary {
					ordered = append(ordered, v)
				}
			}
		}
		for _, v := range ordered {
			ob := v.(M)
			tag := ob["tag"].(string)
			ob["tag"] = tags[tag]
			if e := rewrite(ob); e != nil {
				return nil, e
			}
			if stream, ok := ob["streamSettings"].(M); ok {
				opt, _ := stream["sockopt"].(M)
				if opt == nil {
					opt = M{}
					stream["sockopt"] = opt
				}
				if opt["tcpKeepAliveIdle"] == nil {
					opt["tcpKeepAliveIdle"] = 30
				}
			}
			if ob["tag"] == entry {
				entries = append(entries, ob)
			} else {
				out = append(out, ob)
			}
		}
		dns, _ := source["dns"].(M)
		servers, _ := dns["servers"].([]any)
		for _, v := range servers {
			server, _ := v.(M)
			address, _ := server["address"].(string)
			if strings.HasPrefix(address, "https+local://") {
				domains, _ := server["domains"].([]any)
				for _, d := range domains {
					if text, ok := d.(string); ok && strings.HasPrefix(text, "full:") {
						bootstrap[text] = true
					}
				}
			}
		}
	}
	out = append(entries, out...)
	out = append(out, M{"tag": "direct", "protocol": "freedom", "settings": M{}}, M{"tag": "block", "protocol": "blackhole", "settings": M{}})
	balancers = append(balancers, M{"tag": "smart-parallel-balancer", "selector": []string{"proxy-"}, "fallbackTag": "proxy-0001", "strategy": M{"type": "leastLoad", "settings": M{"expected": min(3, len(members)), "maxRTT": "2s", "tolerance": 0.15}}})
	rules = append(rules, M{"type": "field", "inboundTag": []string{"dns-module"}, "balancerTag": "smart-parallel-balancer"}, M{"type": "field", "network": "tcp,udp", "balancerTag": "smart-parallel-balancer"})
	root["outbounds"] = out
	root["routing"] = M{"domainStrategy": "AsIs", "balancers": balancers, "rules": rules}
	root["burstObservatory"].(M)["subjectSelector"] = subjects
	if len(bootstrap) > 0 {
		domains := []string{}
		for d := range bootstrap {
			domains = append(domains, d)
		}
		sort.Strings(domains)
		dns := root["dns"].(M)
		dns["servers"] = append([]any{M{"address": "https+local://1.1.1.1/dns-query", "domains": domains, "skipFallback": true}}, dns["servers"].([]any)...)
	}
	b, e := json.Marshal(root)
	if len(b) > repair.MaxBytes {
		return nil, errors.New("smart_profile_size_limit")
	}
	return b, e
}
