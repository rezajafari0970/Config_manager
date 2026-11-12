package probe

import (
	"errors"
	"net/url"
)

// packetEncoding is a share-link option, not an Xray VLESS settings field.
// Explicit none preserves native UDP framing and does not turn on XUDP/Mux.
// Verified against official v2rayNG 2.3.15 and Xray 26.10.31/26.11.9.
// Other protocols and unknown encodings require their own driver.
const packetEncodingVersion = "vless-packet-encoding.v1"

func packetEncodingRule(kind string, q url.Values) (string, error) {
	values, exists := q["packetEncoding"]
	if !exists {
		return "", nil
	}
	if len(values) != 1 {
		return "", errors.New("ambiguous_query_parameter")
	}
	if kind != "vless" {
		return "", errors.New("unsupported_packet_encoding")
	}
	switch values[0] {
	case "none":
		return packetEncodingVersion + ".native-udp", nil
	case "xudp":
		return "", nil // Preserve the already tested explicit XUDP adapter.
	default:
		return "", errors.New("unsupported_packet_encoding")
	}
}
