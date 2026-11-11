package compat

const StrictTLSDriver = "strict-tls-verification.v1"

// Disable the removed bypass in outbound TLS settings only. Pins, trust roots,
// SNI and HTTP headers stay intact. XHTTP may have a separate download stream.
func EnforceStrictTLS(root M) []Change {
	changes := []Change{}
	obs, _ := root["outbounds"].([]any)
	for _, v := range obs {
		ob, ok := v.(M)
		if !ok {
			continue
		}
		changed := false
		var visit func(M)
		visit = func(stream M) {
			if tls, ok := stream["tlsSettings"].(M); ok && tls["allowInsecure"] == true {
				tls["allowInsecure"] = false
				changed = true
			}
			xhttp, _ := stream["xhttpSettings"].(M)
			extra, _ := xhttp["extra"].(M)
			for _, holder := range []M{xhttp, extra} {
				if download, ok := holder["downloadSettings"].(M); ok {
					visit(download)
				}
			}
		}
		stream, _ := ob["streamSettings"].(M)
		visit(stream)
		if changed {
			tag, _ := ob["tag"].(string)
			changes = append(changes, Change{StrictTLSDriver, tag, "allow_insecure_disabled"})
		}
	}
	return changes
}
