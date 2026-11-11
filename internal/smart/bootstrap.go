package smart

import (
	"encoding/json"
	"net"
	"strings"
)

func realProxy(protocol any) bool {
	switch protocol {
	case "vless", "vmess", "trojan", "shadowsocks", "socks", "http", "hysteria", "wireguard":
		return true
	}
	return false
}

// Only known endpoint fields are inspected. SNI, HTTP Host, source DNS and
// arbitrary extension fields must never become endpoint-bootstrap domains.
func endpointAddresses(ob M) []string {
	result := []string{}
	add := func(value any) {
		if host, ok := value.(string); ok && host != "" {
			result = append(result, strings.TrimSuffix(strings.ToLower(host), "."))
		}
	}
	settings, _ := ob["settings"].(M)
	add(settings["address"])
	for _, key := range []string{"vnext", "servers"} {
		items, _ := settings[key].([]any)
		for _, v := range items {
			item, _ := v.(M)
			add(item["address"])
		}
	}
	peers, _ := settings["peers"].([]any)
	for _, v := range peers {
		item, _ := v.(M)
		if endpoint, ok := item["endpoint"].(string); ok {
			if host, _, e := net.SplitHostPort(endpoint); e == nil {
				add(host)
			} else {
				add(endpoint)
			}
		}
	}
	walkDownloadStreams(ob["streamSettings"], func(stream M) { add(stream["address"]) })
	return result
}
func collectEndpointDomains(ob M, found map[string]bool) {
	if !realProxy(ob["protocol"]) {
		return
	}
	for _, host := range endpointAddresses(ob) {
		if net.ParseIP(strings.Trim(host, "[]")) == nil {
			found[host] = true
		}
	}
}

// Every possible endpoint and dialer dependency must be independent of DNS.
// An IP-looking entry chained through a domain is deliberately excluded.
func independentIP(tag string, nodes map[string]M, visiting map[string]bool) bool {
	ob := nodes[tag]
	if ob == nil || visiting[tag] || !realProxy(ob["protocol"]) {
		return false
	}
	addresses := endpointAddresses(ob)
	if len(addresses) == 0 {
		return false
	}
	for _, host := range addresses {
		if net.ParseIP(strings.Trim(host, "[]")) == nil {
			return false
		}
	}
	visiting[tag] = true
	defer delete(visiting, tag)
	for _, dependency := range outboundReferences(ob) {
		if !independentIP(dependency, nodes, visiting) {
			return false
		}
	}
	return true
}
func walkDownloadStreams(value any, visit func(M)) {
	switch stream := value.(type) {
	case M:
		// Follow only the XHTTP transport path, not unrelated extension/credential data.
		if x, ok := stream["xhttpSettings"].(M); ok {
			holders := []M{x}
			if extra, ok := x["extra"].(M); ok {
				holders = append(holders, extra)
			}
			for _, holder := range holders {
				if download, ok := holder["downloadSettings"].(M); ok {
					visit(download)
					walkDownloadStreams(download, visit)
				}
			}
		}
	}
}
func applySmartSocketPolicy(ob M) {
	if !realProxy(ob["protocol"]) {
		return
	}
	stream, ok := ob["streamSettings"].(M)
	if !ok {
		stream = M{}
		ob["streamSettings"] = stream
	}
	apply := func(stream M) {
		socket, ok := stream["sockopt"].(M)
		if !ok {
			socket = M{}
			stream["sockopt"] = socket
		}
		socket["domainStrategy"] = "ForceIPv4"
		socket["tcpKeepAliveIdle"] = json.Number("45")
		socket["tcpKeepAliveInterval"] = json.Number("15")
	}
	apply(stream)
	walkDownloadStreams(stream, apply)
}
