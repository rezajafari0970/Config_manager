package compat

import (
	"encoding/json"
	"sort"
	"strings"
)

// Family is bounded diagnostic metadata, never credentials or health evidence.
func Family(body []byte) string {
	var root M
	if json.Unmarshal(body, &root) != nil {
		return "unclassified"
	}
	labels := map[string]bool{}
	obs, _ := root["outbounds"].([]any)
	for _, v := range obs {
		ob, ok := v.(M)
		if !ok {
			continue
		}
		p, _ := ob["protocol"].(string)
		switch p {
		case "vless", "vmess", "trojan", "shadowsocks", "hysteria", "wireguard", "socks", "http":
		default:
			continue
		}
		s, _ := ob["streamSettings"].(M)
		n, _ := s["network"].(string)
		sec, _ := s["security"].(string)
		if method, ok := s["method"].(string); ok && method != "" {
			n = method
		}
		switch n {
		case "", "raw":
			n = "tcp"
		case "websocket":
			n = "ws"
		case "splithttp":
			n = "xhttp"
		case "mkcp":
			n = "kcp"
		}
		if sec == "" {
			sec = "none"
		}
		if n == "tcp" {
			tcp, _ := s["tcpSettings"].(M)
			if raw, ok := s["rawSettings"].(M); ok {
				tcp = raw
			}
			header, _ := tcp["header"].(M)
			if header["type"] == "http" {
				n = "tcp-http"
			}
		}
		if p == "wireguard" {
			n = "udp"
			sec = "native"
		}
		if p == "hysteria" {
			p = "hysteria2"
		}
		if len(n) > 20 || len(sec) > 20 {
			labels[p+"+unclassified"] = true
			continue
		}
		safe := true
		for _, c := range n + sec {
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
				safe = false
			}
		}
		if safe {
			labels[p+"+"+n+"+"+sec] = true
		} else {
			labels[p+"+unclassified"] = true
		}
	}
	all := []string{}
	for label := range labels {
		all = append(all, label)
	}
	sort.Strings(all)
	if len(all) == 0 {
		return "unclassified"
	}
	if len(all) > 4 {
		return "mixed"
	}
	return strings.Join(all, ",")
}
