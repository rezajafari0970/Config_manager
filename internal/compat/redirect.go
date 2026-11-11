package compat

import (
	"errors"
	"net"
	"strconv"
	"strings"
)

// Logical restores only a recognized, exclusively owned fixed-destination
// adapter. This lets normal, Smart and Ads compose the same original graph.
func Logical(root M) ([]Change, error) {
	normalized := append(NormalizeEndpointLiterals(root), EnforceStrictTLS(root)...)
	changes, err := logicalRedirect(root)
	changes = append(normalized, changes...)
	if err != nil {
		return nil, err
	}
	old, err := logicalListener(root)
	return append(changes, old...), err
}

func logicalRedirect(root M) ([]Change, error) {
	obs, _ := root["outbounds"].([]any)
	byTag := map[string]M{}
	for _, v := range obs {
		ob, ok := v.(M)
		if !ok {
			return nil, errors.New("relay_invalid_outbound")
		}
		tag, _ := ob["tag"].(string)
		if tag != "" {
			if byTag[tag] != nil {
				return nil, errors.New("duplicate_outbound_tag")
			}
			byTag[tag] = ob
		}
	}
	type restore struct {
		ob   M
		ep   endpoint
		leg  M
		host string
		port int
		tag  string
	}
	pending := []restore{}
	owners := map[string]M{}
	for _, v := range obs {
		ob := v.(M)
		if !plain(ob) {
			continue
		}
		ep, err := getEndpoint(ob)
		if err != nil || ep.host != "127.0.0.1" {
			continue
		}
		ref, _ := socksOf(ob)["dialerProxy"].(string)
		leg := byTag[ref]
		if leg == nil || leg["protocol"] != "freedom" {
			continue
		}
		settings, _ := leg["settings"].(M)
		redirect, ok := settings["redirect"].(string)
		if !ok {
			continue
		}
		if err := eligible(ob, ep); err != nil {
			return nil, err
		}
		if !allowedKeys(socksOf(ob), "dialerProxy") || ob["sendThrough"] != nil {
			return nil, errors.New("relay_local_socket_extensions")
		}
		if !allowedKeys(leg, "tag", "protocol", "settings", "streamSettings", "sendThrough") ||
			!allowedKeys(settings, "redirect") {
			return nil, errors.New("relay_egress_extensions")
		}
		stream, _ := leg["streamSettings"].(M)
		if !allowedKeys(stream, "sockopt") {
			return nil, errors.New("relay_egress_extensions")
		}
		host, p, err := net.SplitHostPort(redirect)
		port, pe := strconv.Atoi(p)
		if err != nil || pe != nil || port < 1 || port > 65535 || port != ep.port {
			return nil, errors.New("relay_invalid_destination")
		}
		if _, _, err := destination(host); err != nil {
			return nil, err
		}
		if owners[ref] != nil {
			return nil, errors.New("relay_egress_shared")
		}
		owners[ref] = ob
		pending = append(pending, restore{ob, ep, leg, host, port, ref})
	}
	if len(pending) == 0 {
		return nil, nil
	}
	// Never discard a helper selected by routing, a balancer, another dialer,
	// nested download settings, or an observer. Names are not trust markers.
	for tag, owner := range owners {
		count := 0
		var walk func(any)
		walk = func(value any) {
			switch x := value.(type) {
			case M:
				for k, v := range x {
					if k == "outboundTag" || k == "fallbackTag" || k == "dialerProxy" {
						if s, ok := v.(string); ok && s == tag {
							count++
						}
					}
					if k == "proxySettings" {
						if p, ok := v.(M); ok && p["tag"] == tag {
							count++
						}
					}
					if k == "selector" || k == "subjectSelector" {
						for _, prefix := range list(v) {
							if len(prefix) <= len(tag) && tag[:len(prefix)] == prefix {
								count++
							}
						}
					}
					walk(v)
				}
			case []any:
				for _, v := range x {
					walk(v)
				}
			}
		}
		walk(root)
		if count != 1 || socksOf(owner)["dialerProxy"] != tag {
			return nil, errors.New("relay_egress_shared")
		}
	}
	changes := []Change{}
	for _, r := range pending {
		r.ep.holder["address"] = r.host
		r.ep.holder["port"] = r.port
		stream := r.ob["streamSettings"].(M)
		legStream, _ := r.leg["streamSettings"].(M)
		if opt, exists := legStream["sockopt"]; exists {
			stream["sockopt"] = opt
		} else {
			delete(stream, "sockopt")
		}
		if through, exists := r.leg["sendThrough"]; exists {
			r.ob["sendThrough"] = through
		}
		tag, _ := r.ob["tag"].(string)
		changes = append(changes, Change{RelayDriver, tag, "legacy_plaintext_no_added_encryption"})
	}
	kept := []any{}
	for _, v := range obs {
		ob := v.(M)
		tag, _ := ob["tag"].(string)
		if owners[tag] == nil {
			kept = append(kept, ob)
		}
	}
	root["outbounds"] = kept
	return changes, nil
}

// Wrap uses an outbound-only fixed redirect. Native MeasureOutboundDelay drops
// inbounds/router/DNS but retains these outbounds. No local socket is opened,
// no TLS is invented, and the helper cannot dial an arbitrary user destination.
func Wrap(root M) ([]Change, error) {
	if err := ValidateCapabilities(root); err != nil {
		return nil, err
	}
	normalizedChanges := append(NormalizeEndpointLiterals(root), EnforceStrictTLS(root)...)
	_, err := Logical(root)
	if err != nil {
		return nil, err
	}
	obs, _ := root["outbounds"].([]any)
	used := map[string]bool{}
	for _, v := range obs {
		ob, ok := v.(M)
		if !ok {
			return nil, errors.New("relay_invalid_object")
		}
		tag, _ := ob["tag"].(string)
		used[tag] = true
	}
	changes := append([]Change{}, normalizedChanges...)
	legs := []any{}
	for _, v := range obs {
		ob := v.(M)
		if !plain(ob) {
			continue
		}
		ep, err := getEndpoint(ob)
		if err != nil {
			return nil, err
		}
		enc, _ := ep.user["encryption"].(string)
		if enc != "" && enc != "none" || isLocal(ep.host) {
			continue
		}
		if err := eligible(ob, ep); err != nil {
			return nil, err
		}
		if _, _, err := destination(ep.host); err != nil {
			return nil, err
		}
		if len(legs) >= 256 {
			return nil, errors.New("relay_limit_exceeded")
		}
		base := "compat-transport-" + strconv.Itoa(len(legs)+1)
		tag := base
		for n := 1; used[tag]; n++ {
			tag = base + "-" + strconv.Itoa(n)
		}
		used[tag] = true
		leg := M{"tag": tag, "protocol": "freedom", "settings": M{"redirect": net.JoinHostPort(ep.host, strconv.Itoa(ep.port))}}
		stream, _ := ob["streamSettings"].(M)
		if stream == nil {
			stream = M{}
			ob["streamSettings"] = stream
		}
		if opt, exists := stream["sockopt"]; exists {
			leg["streamSettings"] = M{"sockopt": opt}
		}
		if through, exists := ob["sendThrough"]; exists {
			leg["sendThrough"] = through
			delete(ob, "sendThrough")
		}
		// Preserve the authority that the official transport otherwise derives
		// from the original destination. Explicit Host/authority is never changed.
		network, _ := stream["network"].(string)
		if method, ok := stream["method"].(string); ok && method != "" {
			network = method
		}
		key, field := "", "host"
		switch network {
		case "ws", "websocket":
			key = "wsSettings"
		case "httpupgrade":
			key = "httpupgradeSettings"
		case "grpc":
			key = "grpcSettings"
			field = "authority"
		case "xhttp", "splithttp":
			key = "xhttpSettings"
			if stream["splithttpSettings"] != nil && stream["xhttpSettings"] == nil {
				key = "splithttpSettings"
			}
		}
		if key != "" {
			settings, _ := stream[key].(M)
			if settings == nil {
				settings = M{}
				stream[key] = settings
			}
			host, _ := settings[field].(string)
			// Xray still promotes legacy WS headers.Host when host is absent.
			// Preserve that precedence before materializing an implicit authority.
			if host == "" && key == "wsSettings" {
				headers, _ := settings["headers"].(M)
				for name, value := range headers {
					if !strings.EqualFold(name, "Host") {
						continue
					}
					v, ok := value.(string)
					if !ok || host != "" && host != v {
						return nil, errors.New("ambiguous_ws_host")
					}
					host = v
				}
				if host != "" {
					settings[field] = host
				}
			}
			if host == "" {
				host = ep.host
				if ip := net.ParseIP(host); ip != nil && ip.To4() == nil {
					host = "[" + host + "]"
				}
				if network == "grpc" && net.ParseIP(ep.host) != nil {
					host = net.JoinHostPort(ep.host, strconv.Itoa(ep.port))
				}
				settings[field] = host
			}
		}
		stream["sockopt"] = M{"dialerProxy": tag}
		ep.holder["address"] = "127.0.0.1"
		legs = append(legs, leg)
		obTag, _ := ob["tag"].(string)
		changes = append(changes, Change{RelayDriver, obTag, "legacy_plaintext_no_added_encryption"})
	}
	if len(changes) > 0 {
		root["outbounds"] = append(obs, legs...)
	}
	return changes, nil
}
