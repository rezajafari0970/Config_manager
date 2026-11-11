package compat

import "errors"

// FailClosedSocketDNS preserves IP-family preference but prevents the system
// resolver fallback in Xray's DialSystem when its internal DNS lookup fails.
// It touches only stream sockets, including the separate XHTTP download leg.
func FailClosedSocketDNS(stream M) error {
	return failClosedSocketDNS(stream, 0)
}

func failClosedSocketDNS(stream M, depth int) error {
	if depth > 16 {
		return errors.New("socket_dns_stream_depth")
	}
	opt, ok := stream["sockopt"].(M)
	if !ok {
		if stream["sockopt"] != nil {
			return errors.New("invalid_socket_options")
		}
		opt = M{}
		stream["sockopt"] = opt
	}
	switch opt["domainStrategy"] {
	case nil, "", "UseIPv4":
		opt["domainStrategy"] = "ForceIPv4"
	case "AsIs", "UseIP":
		opt["domainStrategy"] = "ForceIP"
	case "UseIPv6":
		opt["domainStrategy"] = "ForceIPv6"
	case "UseIPv4v6":
		opt["domainStrategy"] = "ForceIPv4v6"
	case "UseIPv6v4":
		opt["domainStrategy"] = "ForceIPv6v4"
	}
	if x, ok := stream["xhttpSettings"].(M); ok {
		holders := []M{x}
		if extra, ok := x["extra"].(M); ok {
			holders = append(holders, extra)
		}
		for _, holder := range holders {
			if download, ok := holder["downloadSettings"].(M); ok {
				if err := failClosedSocketDNS(download, depth+1); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
