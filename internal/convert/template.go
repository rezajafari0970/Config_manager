// Package convert applies the user's common envelope to recoverable Xray
// outbounds. It does not infer credentials or promise connectivity.
package convert

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"strconv"

	"github.com/rezajafari0970/Config_manager/internal/probe"
	"github.com/rezajafari0970/Config_manager/internal/repair"
)

const Version = "xray-26.7.28.template.v1"

type M = map[string]any

// The endpoint and transport are deliberately absent from the shared template.
const envelope = `{"remarks":"@shenvpn","log":{"loglevel":"info","access":"none","error":"","dnsLog":true},"dns":{"servers":["https://1.1.1.1/dns-query","https://8.8.8.8/dns-query","https://9.9.9.9/dns-query"],"queryStrategy":"UseIPv4","tag":"dns-module"},"inbounds":[{"tag":"socks","listen":"127.0.0.1","port":10808,"protocol":"socks","settings":{"auth":"noauth","udp":true},"sniffing":{"enabled":true,"destOverride":["http","tls","quic"],"routeOnly":true}}],"outbounds":[],"routing":{"domainStrategy":"AsIs","rules":[{"type":"field","inboundTag":["dns-module"],"outboundTag":"proxy"},{"type":"field","inboundTag":["socks"],"port":"53","network":"tcp,udp","outboundTag":"dns-out"},{"type":"field","network":"tcp,udp","outboundTag":"proxy"}]},"stats":{},"policy":{"system":{"statsOutboundUplink":true,"statsOutboundDownlink":true}}}`

func Prepare(raw []byte) probe.Plan {
	p := probe.Prepare(raw)
	p.Version = Version
	candidates := p.Candidates
	// Custom JSON may need the same explicit TLS hypothesis as a Trojan URI.
	// This does not alter the original candidate, and cannot be exported until
	// this exact alternative actually connects.
	if p.Kind == "json" || p.Kind == "custom" {
		for _, c := range p.Candidates {
			m, e := object(c.Config)
			if e != nil {
				continue
			}
			list, _ := m["outbounds"].([]any)
			changed := false
			for _, v := range list {
				ob, ok := v.(M)
				if !ok || ob["protocol"] != "trojan" {
					continue
				}
				stream, _ := ob["streamSettings"].(M)
				if stream == nil {
					stream = M{}
					ob["streamSettings"] = stream
				}
				security, _ := stream["security"].(string)
				if security != "" && security != "none" {
					continue
				}
				stream["security"] = "tls"
				tls, _ := stream["tlsSettings"].(M)
				if tls == nil {
					tls = M{}
					stream["tlsSettings"] = tls
				}
				tls["allowInsecure"] = false
				changed = true
			}
			if changed {
				b, _ := json.Marshal(m)
				candidates = append(candidates, probe.Candidate{Rule: "custom-trojan-tls-hypothesis", Config: b, RequiresConnection: true})
			}
		}
	}
	p.Candidates = []probe.Candidate{}
	seen := map[string]bool{}
	for _, c := range candidates {
		b, e := Apply(c.Config)
		if e != nil {
			p.Code = e.Error()
			continue
		}
		h := repair.Digest(b)
		if seen[h] {
			continue
		}
		seen[h] = true
		p.Candidates = append(p.Candidates, probe.Candidate{Rule: Version + "." + c.Rule, SHA256: h, Config: b, RequiresConnection: c.RequiresConnection})
	}
	if len(p.Candidates) > 0 {
		p.Code = "prepared"
	}
	return p
}

func object(b []byte) (M, error) {
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	var m M
	if d.Decode(&m) != nil || m == nil {
		return nil, errors.New("invalid_json_object")
	}
	return m, nil
}

// Apply is deterministic and idempotent. Custom chains are preserved under
// stable tags; ambiguous entry selection or broken/cyclic dependencies fail.
func Apply(b []byte) ([]byte, error) {
	source, e := object(b)
	if e != nil {
		return nil, e
	}
	list, ok := source["outbounds"].([]any)
	if !ok || len(list) == 0 {
		return nil, errors.New("missing_proxy_outbound")
	}
	byTag := map[string]M{}
	var proxies []M
	for _, v := range list {
		ob, ok := v.(M)
		if !ok {
			return nil, errors.New("invalid_outbound")
		}
		tag, _ := ob["tag"].(string)
		if tag != "" {
			if byTag[tag] != nil {
				return nil, errors.New("duplicate_outbound_tag")
			}
			byTag[tag] = ob
		}
		switch ob["protocol"] {
		case "freedom", "blackhole", "dns", "loopback":
		default:
			proxies = append(proxies, ob)
		}
	}
	if len(proxies) == 0 {
		return nil, errors.New("missing_proxy_outbound")
	}
	var primary M
	for _, ob := range proxies {
		if ob["tag"] == "proxy" {
			primary = ob
		}
	}
	if primary == nil {
		if len(proxies) != 1 {
			return nil, errors.New("ambiguous_proxy_selection")
		}
		primary = proxies[0]
	}
	chain := []M{}
	visiting := map[string]bool{}
	done := map[string]bool{}
	var visit func(M) error
	visit = func(ob M) error {
		tag, _ := ob["tag"].(string)
		if len(chain) >= 64 {
			return errors.New("proxy_chain_depth_limit")
		}
		if visiting[tag] {
			return errors.New("cyclic_proxy_chain")
		}
		if done[tag] {
			return nil
		}
		visiting[tag] = true
		chain = append(chain, ob)
		for _, ref := range references(ob) {
			dep := byTag[ref]
			if dep == nil {
				return errors.New("missing_proxy_dependency")
			}
			if e := visit(dep); e != nil {
				return e
			}
		}
		delete(visiting, tag)
		done[tag] = true
		return nil
	}
	if e = visit(primary); e != nil {
		return nil, e
	}
	tags := map[string]string{}
	for i, ob := range chain {
		old, _ := ob["tag"].(string)
		tag := "proxy"
		if i > 0 {
			tag = "chain-" + strconv.Itoa(i)
		}
		tags[old] = tag
	}
	out := []any{}
	for _, ob := range chain {
		old, _ := ob["tag"].(string)
		ob["tag"] = tags[old]
		if e = normalize(ob); e != nil {
			return nil, e
		}
		rewriteReferences(ob, tags)
		out = append(out, ob)
	}
	root, _ := object([]byte(envelope))
	out = append(out, M{"tag": "dns-out", "protocol": "dns", "settings": M{}, "streamSettings": M{"sockopt": M{"dialerProxy": "proxy"}}}, M{"tag": "direct", "protocol": "freedom"}, M{"tag": "block", "protocol": "blackhole"})
	root["outbounds"] = out
	// Bootstrap only the proxy's own hostnames locally. Sending these queries
	// through the not-yet-connected proxy creates a DNS recursion. All other
	// queries retain the three proxied DoH resolvers in the user's template.
	var domains []string
	seen := map[string]bool{}
	for _, ob := range chain {
		for _, host := range hostnames(ob) {
			if !seen[host] {
				seen[host] = true
				domains = append(domains, "full:"+host)
			}
		}
	}
	if len(domains) > 0 {
		dns := root["dns"].(M)
		servers := dns["servers"].([]any)
		dns["servers"] = append([]any{M{"address": "https+local://1.1.1.1/dns-query", "domains": domains, "skipFallback": true}}, servers...)
	}
	return json.Marshal(root)
}

func references(ob M) []string {
	var out []string
	if p, ok := ob["proxySettings"].(M); ok {
		if tag, ok := p["tag"].(string); ok && tag != "" {
			out = append(out, tag)
		}
	}
	if s, ok := ob["streamSettings"].(M); ok {
		if o, ok := s["sockopt"].(M); ok {
			if tag, ok := o["dialerProxy"].(string); ok && tag != "" {
				out = append(out, tag)
			}
		}
	}
	return out
}
func rewriteReferences(ob M, tags map[string]string) {
	if p, ok := ob["proxySettings"].(M); ok {
		if tag, ok := p["tag"].(string); ok && tag != "" {
			p["tag"] = tags[tag]
		}
	}
	if s, ok := ob["streamSettings"].(M); ok {
		if o, ok := s["sockopt"].(M); ok {
			if tag, ok := o["dialerProxy"].(string); ok && tag != "" {
				o["dialerProxy"] = tags[tag]
			}
		}
	}
}
func normalize(ob M) error {
	protocol, _ := ob["protocol"].(string)
	switch protocol {
	case "vless", "vmess", "trojan", "shadowsocks", "socks", "hysteria", "wireguard", "http":
	default:
		return errors.New("unsupported_template_protocol")
	}
	s, ok := ob["settings"].(M)
	if !ok {
		return errors.New("missing_protocol_settings")
	}
	if protocol == "hysteria" && s["version"] != json.Number("2") {
		return errors.New("hysteria1_unsupported_by_target_core")
	}
	// An absent VLESS encryption field has the unambiguous standard value
	// "none". This supplies syntax only; it does not add TLS or remote support.
	if protocol == "vless" {
		if s["address"] != nil && s["encryption"] == nil {
			s["encryption"] = "none"
		}
		if nodes, ok := s["vnext"].([]any); ok {
			for _, node := range nodes {
				if n, ok := node.(M); ok {
					if users, ok := n["users"].([]any); ok {
						for _, u := range users {
							if user, ok := u.(M); ok && user["encryption"] == nil {
								user["encryption"] = "none"
							}
						}
					}
				}
			}
		}
	}
	// Pinned core supports both forms. Preserve advanced flat options; use the
	// requested vnext shape when it can be represented without losing fields.
	if (protocol == "vless" || protocol == "vmess") && s["address"] != nil && s["reverse"] == nil {
		if s["vnext"] != nil {
			return errors.New("ambiguous_endpoint_schema")
		}
		user := M{}
		for k, v := range s {
			if k != "address" && k != "port" {
				user[k] = v
			}
		}
		ob["settings"] = M{"vnext": []any{M{"address": s["address"], "port": s["port"], "users": []any{user}}}}
	}
	if protocol != "wireguard" {
		stream, ok := ob["streamSettings"].(M)
		if !ok {
			if ob["streamSettings"] != nil {
				return errors.New("invalid_stream_settings")
			}
			stream = M{}
			ob["streamSettings"] = stream
		}
		opt, ok := stream["sockopt"].(M)
		if !ok {
			if stream["sockopt"] != nil {
				return errors.New("invalid_socket_options")
			}
			opt = M{}
			stream["sockopt"] = opt
		}
		if opt["domainStrategy"] == nil {
			opt["domainStrategy"] = "UseIPv4"
		}
	} else if s["domainStrategy"] == nil {
		s["domainStrategy"] = "ForceIPv4"
	}
	return nil
}
func hostnames(ob M) []string {
	var hosts []string
	s, _ := ob["settings"].(M)
	add := func(v any) {
		if h, ok := v.(string); ok && h != "" && net.ParseIP(h) == nil {
			hosts = append(hosts, h)
		}
	}
	add(s["address"])
	for _, key := range []string{"vnext", "servers"} {
		if list, ok := s[key].([]any); ok {
			for _, v := range list {
				if o, ok := v.(M); ok {
					add(o["address"])
				}
			}
		}
	}
	if list, ok := s["peers"].([]any); ok {
		for _, v := range list {
			if o, ok := v.(M); ok {
				if ep, ok := o["endpoint"].(string); ok {
					h, _, e := net.SplitHostPort(ep)
					if e == nil {
						add(h)
					}
				}
			}
		}
	}
	return hosts
}
