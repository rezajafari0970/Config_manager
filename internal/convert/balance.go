package convert

import (
	"encoding/json"
	"errors"
	"strings"
)

// Only explicit universal routes identify the entry of a multi-proxy config.
// Selective routes are replaced by the requested shared envelope, not promoted
// into a default route. Unknown predicates are never assumed universal.
func routedEntry(source M) (outbound, balancer string, found bool, err error) {
	routing, _ := source["routing"].(M)
	rules, _ := routing["rules"].([]any)
	for _, item := range rules {
		r, ok := item.(M)
		if !ok {
			continue
		}
		universal := true
		for key, value := range r {
			switch key {
			case "type":
				universal = universal && (value == "field" || value == "")
			case "ruleTag", "outboundTag", "balancerTag":
			case "port":
				universal = universal && (value == "0-65535" || value == "1-65535")
			case "network":
				values, e := stringList(value)
				if e != nil {
					universal = false
					continue
				}
				set := map[string]bool{}
				for _, v := range values {
					for _, n := range strings.Split(v, ",") {
						set[strings.TrimSpace(n)] = true
					}
				}
				universal = universal && len(set) == 2 && set["tcp"] && set["udp"]
			default:
				universal = false
			}
		}
		if !universal {
			continue
		}
		outbound, _ = r["outboundTag"].(string)
		balancer, _ = r["balancerTag"].(string)
		if (outbound == "") == (balancer == "") {
			return "", "", false, errors.New("ambiguous_route_target")
		}
		return outbound, balancer, true, nil
	}
	return "", "", false, nil
}

func isProxy(ob M) bool {
	switch ob["protocol"] {
	case "freedom", "blackhole", "dns", "loopback", nil, "":
		return false
	}
	return true
}

func stringList(value any) ([]string, error) {
	if s, ok := value.(string); ok && s != "" {
		return []string{s}, nil
	}
	items, ok := value.([]any)
	if !ok || len(items) == 0 {
		return nil, errors.New("invalid_tag_selector")
	}
	out := make([]string, 0, len(items))
	for _, v := range items {
		s, ok := v.(string)
		if !ok || s == "" {
			return nil, errors.New("invalid_tag_selector")
		}
		out = append(out, s)
	}
	return out, nil
}

func matches(tag string, selectors []string) bool {
	for _, prefix := range selectors {
		if strings.HasPrefix(tag, prefix) {
			return true
		}
	}
	return false
}

// Keep original tags so leastLoad cost matchers, selectors, fallbacks and chain
// references retain their meaning. Infrastructure tags must not enter the pool.
func infrastructureTag(base string, used map[string]bool, selectors []string) (string, error) {
	for _, prefix := range []string{"", "cm-", "__cm-", "~cm-"} {
		tag := prefix + base
		if !used[tag] && !matches(tag, selectors) {
			used[tag] = true
			return tag, nil
		}
	}
	return "", errors.New("reserved_balancer_tag_collision")
}

func applyBalanced(source M, list []any, byTag map[string]M, target string) ([]byte, error) {
	routing, _ := source["routing"].(M)
	balancers, ok := routing["balancers"].([]any)
	if !ok {
		return nil, errors.New("missing_balancer")
	}
	var selected M
	seen := map[string]bool{}
	for _, item := range balancers {
		b, ok := item.(M)
		if !ok {
			return nil, errors.New("invalid_balancer")
		}
		tag, _ := b["tag"].(string)
		if tag == "" || seen[tag] {
			return nil, errors.New("duplicate_balancer_tag")
		}
		seen[tag] = true
		if tag == target {
			selected = b
		}
	}
	if selected == nil {
		return nil, errors.New("missing_balancer")
	}
	selectors, err := stringList(selected["selector"])
	if err != nil {
		return nil, err
	}
	needed, visiting := map[string]bool{}, map[string]bool{}
	var visit func(string) error
	visit = func(tag string) error {
		ob := byTag[tag]
		if ob == nil {
			return errors.New("missing_proxy_dependency")
		}
		if visiting[tag] {
			return errors.New("cyclic_proxy_chain")
		}
		if needed[tag] {
			return nil
		}
		if len(needed) >= 64 {
			return errors.New("proxy_chain_depth_limit")
		}
		visiting[tag] = true
		needed[tag] = true
		for _, ref := range references(ob) {
			if e := visit(ref); e != nil {
				return e
			}
		}
		delete(visiting, tag)
		return nil
	}
	members := 0
	for _, item := range list {
		ob := item.(M)
		tag, _ := ob["tag"].(string)
		if tag != "" && matches(tag, selectors) {
			if !isProxy(ob) {
				return nil, errors.New("balancer_nonproxy_member")
			}
			members++
			if err = visit(tag); err != nil {
				return nil, err
			}
		}
	}
	if members == 0 {
		return nil, errors.New("empty_balancer_pool")
	}
	if value, exists := selected["fallbackTag"]; exists && value != "" {
		tag, ok := value.(string)
		if !ok || byTag[tag] == nil {
			return nil, errors.New("missing_balancer_fallback")
		}
		if !isProxy(byTag[tag]) {
			return nil, errors.New("balancer_nonproxy_fallback")
		}
		if err = visit(tag); err != nil {
			return nil, err
		}
	}
	// Xray falls back to the first outbound when a balancer yields no target.
	// Keep that exact real proxy first; making loopback the default would loop.
	first := list[0].(M)
	firstTag, _ := first["tag"].(string)
	if !needed[firstTag] || !isProxy(first) {
		return nil, errors.New("ambiguous_balancer_default")
	}
	chain := []M{}
	out := []any{}
	used := map[string]bool{}
	for _, item := range list {
		ob := item.(M)
		tag, _ := ob["tag"].(string)
		if !needed[tag] {
			continue
		}
		if err = normalize(ob); err != nil {
			return nil, err
		}
		chain = append(chain, ob)
		out = append(out, ob)
		used[tag] = true
	}
	root, _ := object([]byte(envelope))
	// Observer configuration and tag-based strategy settings remain unchanged.
	// Do not let newly generated DNS/loopback/direct helpers become probe targets.
	allSelectors := append([]string(nil), selectors...)
	for _, key := range []string{"observatory", "burstObservatory"} {
		if value, exists := source[key]; exists {
			observer, ok := value.(M)
			if !ok {
				return nil, errors.New("invalid_observatory")
			}
			if v, exists := observer["subjectSelector"]; exists {
				ss, e := stringList(v)
				if e != nil {
					return nil, e
				}
				allSelectors = append(allSelectors, ss...)
			}
			root[key] = observer
		}
	}
	proxyTag, err := infrastructureTag("proxy", used, allSelectors)
	if err != nil {
		return nil, err
	}
	dnsTag, err := infrastructureTag("dns-out", used, allSelectors)
	if err != nil {
		return nil, err
	}
	directTag, err := infrastructureTag("direct", used, allSelectors)
	if err != nil {
		return nil, err
	}
	blockTag, err := infrastructureTag("block", used, allSelectors)
	if err != nil {
		return nil, err
	}
	const routeTag = "cm-balancer-dispatch"
	out = append(out,
		M{"tag": proxyTag, "protocol": "loopback", "settings": M{"inboundTag": routeTag}},
		M{"tag": dnsTag, "protocol": "dns", "settings": M{}, "streamSettings": M{"sockopt": M{"dialerProxy": proxyTag}}},
		M{"tag": directTag, "protocol": "freedom"},
		M{"tag": blockTag, "protocol": "blackhole"})
	root["outbounds"] = out
	root["routing"] = M{"domainStrategy": "AsIs", "balancers": []any{selected}, "rules": []any{
		M{"type": "field", "inboundTag": []string{routeTag}, "balancerTag": target},
		M{"type": "field", "inboundTag": []string{"dns-module"}, "balancerTag": target},
		M{"type": "field", "inboundTag": []string{"socks"}, "port": "53", "network": "tcp,udp", "outboundTag": dnsTag},
		M{"type": "field", "network": "tcp,udp", "balancerTag": target},
	}}
	addBootstrap(root, chain)
	return json.Marshal(root)
}
