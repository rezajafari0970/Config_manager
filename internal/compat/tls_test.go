package compat

import (
	"encoding/json"
	"testing"
)

func TestStrictTLSNestedStreamsPreserveServerAndPins(t *testing.T) {
	raw := []byte(`{"outbounds":[{"tag":"proxy","protocol":"vless","settings":{"address":"198.51.100.2","port":443,"id":"unchanged"},"streamSettings":{"security":"tls","tlsSettings":{"allowInsecure":true,"serverName":"kept.example","pinnedPeerCertSha256":["pin"]},"xhttpSettings":{"headers":{"tlsSettings":{"allowInsecure":true}},"extra":{"downloadSettings":{"security":"tls","tlsSettings":{"allowInsecure":true,"serverName":"download.example"}}}}}}]}`)
	var m M
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	changes := EnforceStrictTLS(m)
	if len(changes) != 1 || changes[0].Driver != StrictTLSDriver {
		t.Fatal(changes)
	}
	ob := m["outbounds"].([]any)[0].(M)
	if ob["settings"].(M)["address"] != "198.51.100.2" || ob["settings"].(M)["id"] != "unchanged" {
		t.Fatal("server changed")
	}
	s := ob["streamSettings"].(M)
	tls := s["tlsSettings"].(M)
	if tls["allowInsecure"] != false || tls["serverName"] != "kept.example" || tls["pinnedPeerCertSha256"].([]any)[0] != "pin" {
		t.Fatal("TLS policy damaged")
	}
	x := s["xhttpSettings"].(M)
	if x["headers"].(M)["tlsSettings"].(M)["allowInsecure"] != true {
		t.Fatal("HTTP data modified")
	}
	if x["extra"].(M)["downloadSettings"].(M)["tlsSettings"].(M)["allowInsecure"] != false {
		t.Fatal("download bypass retained")
	}
	if len(EnforceStrictTLS(m)) != 0 {
		t.Fatal("not idempotent")
	}
}
