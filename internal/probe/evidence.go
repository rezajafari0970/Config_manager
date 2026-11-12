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

// VerifiedPayload is a finite 2 MiB download / 1 MiB upload plus post-load check.
// It is separate from core acceptance, general HTTPS and device/app performance.
func VerifiedPayload(e map[string]string) bool {
	return e["payload_version"] == "payload_v1" &&
		e["payload_download"] == "passed" && e["payload_upload"] == "passed" &&
		e["payload_recheck"] == "passed" && e["payload_download_bytes"] == "2097152" &&
		e["payload_upload_bytes"] == "1048576"
}
