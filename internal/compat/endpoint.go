package compat

import (
	"net"
	"strings"
)

const EndpointLiteralDriver = "endpoint-ipv6-literal.v1"

// Brackets belong to URI authority / host:port syntax, not the JSON address
// field. Only a complete, valid IPv6 literal is unwrapped. Credentials, port,
// HTTP Host, SNI, routes, and raw source bytes are not inferred or rewritten.
func NormalizeEndpointLiterals(root M) []Change {
	changes := []Change{}
	obs, _ := root["outbounds"].([]any)
	for _, v := range obs {
		ob, ok := v.(M)
		if !ok {
			continue
		}
		switch ob["protocol"] {
		case "vless", "vmess", "trojan", "shadowsocks", "socks", "http", "hysteria":
		default:
			continue
		}
		settings, _ := ob["settings"].(M)
		holders := []M{settings}
		for _, key := range []string{"vnext", "servers"} {
			rows, _ := settings[key].([]any)
			for _, v := range rows {
				if row, ok := v.(M); ok {
					holders = append(holders, row)
				}
			}
		}
		for _, holder := range holders {
			address, _ := holder["address"].(string)
			if len(address) < 4 || address[0] != '[' || address[len(address)-1] != ']' {
				continue
			}
			literal := address[1 : len(address)-1]
			if !strings.Contains(literal, ":") || net.ParseIP(literal) == nil {
				continue
			}
			holder["address"] = literal
			tag, _ := ob["tag"].(string)
			changes = append(changes, Change{EndpointLiteralDriver, tag, "ipv6_address_brackets_removed"})
		}
	}
	return changes
}
