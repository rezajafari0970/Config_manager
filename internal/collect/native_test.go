package collect

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestNativeProtocolsEnterAllSubscriptionFormatsExactly(t *testing.T) {
	raw := []byte("tuic://00000000-0000-4000-8000-000000000001:synthetic@192.0.2.1:443?sni=example.com#unchanged\r\n")
	for _, body := range [][]byte{raw, []byte(base64.StdEncoding.EncodeToString(raw))} {
		values, kinds, e := extractTest(t, body)
		if e != nil || len(values) != 1 || kinds[0] != "tuic" || !bytes.Equal(values[0], raw) {
			t.Fatal("native acquisition lost bytes", e, kinds)
		}
	}
}
