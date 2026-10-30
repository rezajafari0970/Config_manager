package convert

import "errors"

// Open client-side UDP/443 gates without changing the wire protocol, endpoint,
// credentials, TLS verification or whether multiplexing is enabled.
// QUIC is UDP traffic, not an additional Xray routing network.
func allowProxyTraffic(ob, settings M) error {
	if ob["protocol"] == "vless" {
		allowVision := func(user M) {
			if user["flow"] == "xtls-rprx-vision" {
				user["flow"] = "xtls-rprx-vision-udp443"
			}
		}
		allowVision(settings)
		if nodes, ok := settings["vnext"].([]any); ok {
			for _, node := range nodes {
				if n, ok := node.(M); ok {
					if users, ok := n["users"].([]any); ok {
						for _, u := range users {
							if user, ok := u.(M); ok {
								allowVision(user)
							}
						}
					}
				}
			}
		}
	}
	if value, exists := ob["mux"]; exists && value != nil {
		mux, ok := value.(M)
		if !ok {
			return errors.New("invalid_mux_settings")
		}
		switch mux["xudpProxyUDP443"] {
		case nil, "", "reject":
			mux["xudpProxyUDP443"] = "allow"
		case "allow", "skip": // Both permit UDP/443; retain an explicit native-UDP bypass.
		default:
			return errors.New("invalid_mux_udp443_policy")
		}
	}
	return nil
}
