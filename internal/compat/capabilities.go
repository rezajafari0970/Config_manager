package compat

import "errors"

// ValidateCapabilities is deliberately conservative and versioned. Unknown
// optional fields remain available to the exact pinned core's own validator.
func ValidateCapabilities(root M) error {
	obs, _ := root["outbounds"].([]any)
	for _, v := range obs {
		ob, ok := v.(M)
		if !ok {
			return errors.New("outbound_object_required")
		}
		protocol, _ := ob["protocol"].(string)
		switch protocol {
		case "hysteria1", "hy", "tuic":
			return errors.New("protocol_requires_other_client_core")
		}
		s, _ := ob["streamSettings"].(M)
		network, _ := s["network"].(string)
		if method, ok := s["method"].(string); ok && method != "" {
			network = method
		}
		switch network {
		case "h2", "h3", "http":
			return errors.New("legacy_http_transport_removed")
		case "quic":
			return errors.New("legacy_quic_transport_removed")
		}
		security, _ := s["security"].(string)
		if security == "xtls" {
			return errors.New("legacy_xtls_security_removed")
		}
		if security == "reality" && network != "" && network != "tcp" && network != "raw" && network != "xhttp" && network != "splithttp" && network != "grpc" {
			return errors.New("reality_transport_unsupported")
		}
		if protocol == "hysteria" {
			settings, _ := ob["settings"].(M)
			if number(settings["version"]) != 2 {
				return errors.New("hysteria_version_unsupported")
			}
			if network != "hysteria" || security != "tls" {
				return errors.New("hysteria_transport_security_mismatch")
			}
		}
		if protocol == "wireguard" && (network != "" || security != "" && security != "none") {
			return errors.New("wireguard_external_stream_requires_driver")
		}
	}
	return nil
}
