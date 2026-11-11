package probe

// VerifiedEvidence gates publication; VN is not an HTTP/3 certificate.
func VerifiedEvidence(e map[string]string) bool {
	for _, key := range []string{"https", "dns_tcp", "doh_cloudflare", "doh_google", "target_core", "bundled_core", "app_import_contract"} {
		if e[key] != "passed" {
			return false
		}
	}
	for _, key := range []string{"dns_udp", "quic_vn"} {
		if e[key] != "passed" && e[key] != "unsupported" {
			return false
		}
	}
	return true
}
