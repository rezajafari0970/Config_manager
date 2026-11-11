// Package compat owns narrow, versioned compatibility rules, not network health.
package compat

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"strings"
)

type M = map[string]any

const Version = "client-compat.2026-11-11.v1"
const RelayDriver = "legacy-vless-tcp-loopback.v1"

type Change struct {
	Driver string `json:"driver"`
	Tag    string `json:"outbound_tag"`
	Effect string `json:"effect"`
}
type endpoint struct {
	holder M
	host   string
	port   int
	user   M
}

func number(v any) int {
	switch v := v.(type) {
	case int:
		return v
	case float64:
		if v == float64(int(v)) {
			return int(v)
		}
	case json.Number:
		n, e := strconv.Atoi(string(v))
		if e == nil {
			return n
		}
	}
	return 0
}
func list(v any) []string {
	switch v := v.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, x := range v {
			s, ok := x.(string)
			if !ok {
				return nil
			}
			out = append(out, s)
		}
		return out
	}
	return nil
}
func one(v any, want string) bool { x := list(v); return len(x) == 1 && x[0] == want }
func plain(ob M) bool {
	if ob["protocol"] != "vless" {
		return false
	}
	s, _ := ob["streamSettings"].(M)
	security, _ := s["security"].(string)
	return security == "" || security == "none"
}
func getEndpoint(ob M) (endpoint, error) {
	s, ok := ob["settings"].(M)
	if !ok {
		return endpoint{}, errors.New("relay_invalid_settings")
	}
	e := endpoint{holder: s, user: s}
	if _, exists := s["address"]; !exists {
		nodes, ok := s["vnext"].([]any)
		if !ok || len(nodes) != 1 {
			return e, errors.New("relay_ambiguous_endpoint")
		}
		node, ok := nodes[0].(M)
		if !ok {
			return e, errors.New("relay_invalid_settings")
		}
		users, ok := node["users"].([]any)
		if !ok || len(users) != 1 {
			return e, errors.New("relay_ambiguous_user")
		}
		user, ok := users[0].(M)
		if !ok {
			return e, errors.New("relay_invalid_settings")
		}
		e.holder = node
		e.user = user
	}
	e.host, _ = e.holder["address"].(string)
	e.port = number(e.holder["port"])
	if e.host == "" || e.port < 1 || e.port > 65535 {
		return e, errors.New("relay_invalid_endpoint")
	}
	return e, nil
}
func isLocal(h string) bool {
	ip := net.ParseIP(h)
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified())
}
func destination(h string) (string, string, error) {
	if ip := net.ParseIP(h); ip != nil {
		if isLocal(h) || !ip.IsGlobalUnicast() {
			return "", "", errors.New("relay_nonpublic_destination")
		}
		return "ip", h, nil
	}
	if strings.ContainsAny(h, "/\\:@ \t\r\n") || len(h) > 253 || h == "" {
		return "", "", errors.New("relay_invalid_destination")
	}
	return "domain", "full:" + h, nil
}
func allowedKeys(m M, keys ...string) bool {
	allow := map[string]bool{}
	for _, k := range keys {
		allow[k] = true
	}
	for k := range m {
		if !allow[k] {
			return false
		}
	}
	return true
}
func rulesOf(root M) []any { r, _ := root["routing"].(M); a, _ := r["rules"].([]any); return a }
func socksOf(ob M) M       { s, _ := ob["streamSettings"].(M); o, _ := s["sockopt"].(M); return o }

// Logical recognizes a complete fixed-destination relay graph before composition.
// Source-supplied graph names alone are never trusted as an authorization marker.
func Logical(root M) ([]Change, error) {
	obs, _ := root["outbounds"].([]any)
	ins, _ := root["inbounds"].([]any)
	rules := rulesOf(root)
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
	removeIn, removeRule, removeOut := map[int]bool{}, map[int]bool{}, map[string]bool{}
	type restore struct {
		ob   M
		e    endpoint
		host string
		port int
		leg  M
	}
	changes := []Change{}
	restores := []restore{}
	for _, v := range obs {
		ob := v.(M)
		if !plain(ob) {
			continue
		}
		e, err := getEndpoint(ob)
		if err != nil {
			continue
		}
		if e.host != "127.0.0.1" {
			continue
		}
		found := -1
		var inbound M
		for i, v := range ins {
			in, ok := v.(M)
			if !ok {
				continue
			}
			if in["listen"] == "127.0.0.1" && number(in["port"]) == e.port && (in["protocol"] == "dokodemo-door" || in["protocol"] == "tunnel") {
				if found >= 0 {
					return nil, errors.New("relay_duplicate_listener")
				}
				found = i
				inbound = in
			}
		}
		if found < 0 {
			continue
		}
		if err := eligible(ob, e); err != nil {
			return nil, err
		}
		if removeIn[found] {
			return nil, errors.New("relay_shared_listener")
		}
		if !allowedKeys(inbound, "tag", "listen", "port", "protocol", "settings") {
			return nil, errors.New("relay_inbound_extensions")
		}
		tag, _ := inbound["tag"].(string)
		s, _ := inbound["settings"].(M)
		if tag == "" || !allowedKeys(s, "address", "port", "network", "followRedirect") || s["network"] != "tcp" || s["followRedirect"] != false {
			return nil, errors.New("relay_unbounded_listener")
		}
		host, _ := s["address"].(string)
		port := number(s["port"])
		key, value, err := destination(host)
		if err != nil || port < 1 || port > 65535 {
			return nil, errors.New("relay_invalid_destination")
		}
		allow, deny := -1, -1
		legTag := ""
		for i, v := range rules {
			r, ok := v.(M)
			if !ok {
				return nil, errors.New("relay_invalid_rule")
			}
			inboundTags := list(r["inboundTag"])
			if len(inboundTags) > 0 {
				includes := false
				for _, t := range inboundTags {
					if t == tag {
						includes = true
					}
				}
				if !includes {
					continue
				}
				if len(inboundTags) != 1 {
					return nil, errors.New("relay_shared_route")
				}
			} else if deny < 0 {
				return nil, errors.New("relay_unscoped_route")
			}
			if deny >= 0 {
				break
			}
			if allow < 0 {
				if !one(r["inboundTag"], tag) || r["type"] != "field" || r["network"] != "tcp" || !one(r[key], value) || r["port"] != strconv.Itoa(port) || !allowedKeys(r, "type", "inboundTag", "network", key, "port", "outboundTag") {
					return nil, errors.New("relay_destination_not_scoped")
				}
				legTag, _ = r["outboundTag"].(string)
				allow = i
			} else {
				blockTag, _ := r["outboundTag"].(string)
				block := byTag[blockTag]
				if !one(r["inboundTag"], tag) || r["type"] != "field" || r["network"] != "tcp,udp" || block == nil || block["protocol"] != "blackhole" || !allowedKeys(r, "type", "inboundTag", "network", "outboundTag") {
					return nil, errors.New("relay_missing_deny")
				}
				deny = i
			}
		}
		leg := byTag[legTag]
		if allow < 0 || deny < 0 || leg == nil || leg["protocol"] != "freedom" || !allowedKeys(leg, "tag", "protocol", "settings", "streamSettings", "sendThrough") {
			return nil, errors.New("relay_egress_not_scoped")
		}
		settings, settingsOK := leg["settings"].(M)
		if !settingsOK || len(settings) != 0 {
			return nil, errors.New("relay_egress_extensions")
		}
		stream, _ := leg["streamSettings"].(M)
		if !allowedKeys(stream, "sockopt") {
			return nil, errors.New("relay_egress_extensions")
		}
		if len(socksOf(ob)) != 0 {
			return nil, errors.New("relay_local_socket_extensions")
		}
		if _, exists := ob["sendThrough"]; exists {
			return nil, errors.New("relay_local_bind")
		}
		removeIn[found] = true
		removeRule[allow] = true
		removeRule[deny] = true
		removeOut[legTag] = true
		restores = append(restores, restore{ob, e, host, port, leg})
	}
	if len(restores) == 0 {
		return changes, nil
	}
	for i, v := range rules {
		if removeRule[i] {
			continue
		}
		r := v.(M)
		tag, _ := r["outboundTag"].(string)
		if removeOut[tag] {
			return nil, errors.New("relay_egress_shared")
		}
	}
	for _, v := range obs {
		ob := v.(M)
		proxy, _ := ob["proxySettings"].(M)
		for _, holder := range []M{socksOf(ob), proxy} {
			for _, key := range []string{"dialerProxy", "tag"} {
				tag, _ := holder[key].(string)
				if removeOut[tag] {
					return nil, errors.New("relay_egress_shared")
				}
			}
		}
	}
	for _, r := range restores {
		r.e.holder["address"] = r.host
		r.e.holder["port"] = r.port
		stream, _ := r.ob["streamSettings"].(M)
		if stream == nil {
			stream = M{}
			r.ob["streamSettings"] = stream
		}
		if sock := socksOf(r.leg); len(sock) > 0 {
			stream["sockopt"] = sock
		}
		if through, exists := r.leg["sendThrough"]; exists {
			r.ob["sendThrough"] = through
		}
		tag, _ := r.ob["tag"].(string)
		changes = append(changes, Change{RelayDriver, tag, "legacy_plaintext_no_added_encryption"})
	}
	newIns, newRules, newObs := []any{}, []any{}, []any{}
	for i, v := range ins {
		if !removeIn[i] {
			newIns = append(newIns, v)
		}
	}
	for i, v := range rules {
		if !removeRule[i] {
			newRules = append(newRules, v)
		}
	}
	for _, v := range obs {
		tag, _ := v.(M)["tag"].(string)
		if !removeOut[tag] {
			newObs = append(newObs, v)
		}
	}
	root["inbounds"] = newIns
	root["outbounds"] = newObs
	root["routing"].(M)["rules"] = newRules
	return changes, nil
}

func eligible(ob M, e endpoint) error {
	if v, _ := e.user["encryption"].(string); v != "" && v != "none" {
		return errors.New("relay_encrypted_vless")
	}
	if v, _ := e.user["flow"].(string); v != "" {
		return errors.New("relay_flow_requires_driver")
	}
	s, _ := ob["streamSettings"].(M)
	network, _ := s["network"].(string)
	if method, ok := s["method"].(string); ok && method != "" {
		network = method
	}
	if network != "" && network != "tcp" && network != "raw" {
		return errors.New("legacy_vless_transport_requires_driver")
	}
	for _, key := range []string{"finalmask", "address", "port"} {
		if _, ok := s[key]; ok {
			return errors.New("relay_stream_override_requires_driver")
		}
	}
	if _, ok := ob["proxySettings"]; ok {
		return errors.New("relay_proxy_settings_requires_driver")
	}
	for _, key := range []string{"tcpSettings", "rawSettings"} {
		tcp, _ := s[key].(M)
		h, _ := tcp["header"].(M)
		kind, _ := h["type"].(string)
		if kind != "" && kind != "none" && kind != "http" {
			return errors.New("relay_header_requires_driver")
		}
	}
	return nil
}

// Wrap replaces only the carrier destination; protocol/header/credentials stay intact.
// The carrier is still plaintext. The local exception adds no encryption.
func Wrap(root M) ([]Change, error) {
	if e := ValidateCapabilities(root); e != nil {
		return nil, e
	}
	if _, err := Logical(root); err != nil {
		return nil, err
	}
	obs, _ := root["outbounds"].([]any)
	ins, _ := root["inbounds"].([]any)
	usedTags := map[string]bool{}
	usedPorts := map[int]bool{10808: true, 11111: true}
	for _, group := range [][]any{obs, ins} {
		for _, v := range group {
			m, ok := v.(M)
			if !ok {
				return nil, errors.New("relay_invalid_object")
			}
			if tag, ok := m["tag"].(string); ok {
				usedTags[tag] = true
			}
			if port := number(m["port"]); port != 0 {
				usedPorts[port] = true
			}
		}
	}
	unique := func(base string) string {
		tag := base
		for n := 1; usedTags[tag]; n++ {
			tag = base + "-" + strconv.Itoa(n)
		}
		usedTags[tag] = true
		return tag
	}
	port := 20000
	allocate := func() (int, error) {
		for port <= 59999 && usedPorts[port] {
			port++
		}
		if port > 59999 {
			return 0, errors.New("relay_port_exhausted")
		}
		p := port
		usedPorts[p] = true
		port++
		return p, nil
	}
	changes := []Change{}
	prefix := []any{}
	legs := []any{}
	blockTag := ""
	for _, v := range obs {
		ob := v.(M)
		if ob["protocol"] == "blackhole" {
			if t, ok := ob["tag"].(string); ok && t != "" {
				blockTag = t
				break
			}
		}
	}
	for _, v := range obs {
		ob := v.(M)
		if !plain(ob) {
			continue
		}
		e, err := getEndpoint(ob)
		if err != nil {
			return nil, err
		}
		if enc, _ := e.user["encryption"].(string); enc != "" && enc != "none" {
			continue
		}
		if isLocal(e.host) {
			continue
		}
		if err := eligible(ob, e); err != nil {
			return nil, err
		}
		key, value, err := destination(e.host)
		if err != nil {
			return nil, err
		}
		if len(changes) >= 256 {
			return nil, errors.New("relay_limit_exceeded")
		}
		p, err := allocate()
		if err != nil {
			return nil, err
		}
		index := strconv.Itoa(len(changes) + 1)
		inTag := unique("compat-relay-" + index)
		legTag := unique("compat-transport-" + index)
		if blockTag == "" {
			blockTag = unique("compat-deny")
			legs = append(legs, M{"tag": blockTag, "protocol": "blackhole", "settings": M{}})
		}
		leg := M{"tag": legTag, "protocol": "freedom", "settings": M{}}
		stream, _ := ob["streamSettings"].(M)
		if sock, exists := stream["sockopt"]; exists {
			leg["streamSettings"] = M{"sockopt": sock}
			delete(stream, "sockopt")
		}
		if through, exists := ob["sendThrough"]; exists {
			leg["sendThrough"] = through
			delete(ob, "sendThrough")
		}
		ins = append(ins, M{"tag": inTag, "listen": "127.0.0.1", "port": p, "protocol": "dokodemo-door", "settings": M{"address": e.host, "port": e.port, "network": "tcp", "followRedirect": false}})
		prefix = append(prefix,
			M{"type": "field", "inboundTag": []any{inTag}, "network": "tcp", key: []any{value}, "port": strconv.Itoa(e.port), "outboundTag": legTag},
			M{"type": "field", "inboundTag": []any{inTag}, "network": "tcp,udp", "outboundTag": blockTag})
		e.holder["address"] = "127.0.0.1"
		e.holder["port"] = p
		legs = append(legs, leg)
		tag, _ := ob["tag"].(string)
		changes = append(changes, Change{RelayDriver, tag, "legacy_plaintext_no_added_encryption"})
	}
	if len(changes) == 0 {
		return changes, nil
	}
	routing, _ := root["routing"].(M)
	if routing == nil {
		return nil, errors.New("relay_routing_required")
	}
	rules, _ := routing["rules"].([]any)
	if len(rules) == 0 {
		return nil, errors.New("relay_routing_required")
	}
	root["inbounds"] = ins
	root["outbounds"] = append(obs, legs...)
	routing["rules"] = append(prefix, rules...)
	return changes, nil
}

// ApplyJSON is the single final compatibility boundary for all output builders.
func ApplyJSON(body []byte) ([]byte, []Change, error) {
	if !json.Valid(body) {
		return nil, nil, errors.New("invalid_json")
	}
	var root M
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&root); err != nil {
		return nil, nil, err
	}
	changes, err := Wrap(root)
	if err != nil {
		return nil, nil, err
	}
	if len(changes) == 0 {
		return body, changes, nil
	}
	b, err := json.Marshal(root)
	return b, changes, err
}
