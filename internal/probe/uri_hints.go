package probe

import (
	"errors"
	"net/url"
)

// 3x-ui emits this positive compatibility hint for Mihomo. Pinned Xray REALITY
// already builds its native uTLS ClientHello (including supported ML-KEM shares)
// and has no JSON field for this hint. Preserve the fingerprint and credentials;
// never invent an unknown core field or accept an unsupported disable request.
func realityMLKEMHint(kind string, q url.Values) error {
	values, exists := q["support-x25519mlkem768"]
	if !exists {
		return nil
	}
	if len(values) != 1 {
		return errors.New("ambiguous_query_parameter")
	}
	if kind != "vless" || q.Get("security") != "reality" {
		return errors.New("reality_mlkem_hint_incompatible")
	}
	switch values[0] {
	case "true", "1":
		return nil
	case "false", "0":
		return errors.New("reality_mlkem_disable_unsupported")
	default:
		return errors.New("reality_mlkem_hint_invalid")
	}
}
