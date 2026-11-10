package residential

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// IPFactMAC authenticates the trusted supervisor's HTTPS intelligence receipt
// at the owner boundary. It does not make the upstream provider infallible;
// specialist capability, independent agreement and freshness remain mandatory.
// The sandboxed core has neither this key nor the private completion socket.
func IPFactMAC(f IPFact, key []byte) string {
	f.MAC = ""
	body, _ := json.Marshal(f)
	h := hmac.New(sha256.New, key)
	h.Write([]byte("shen.residential.ip-intelligence.v1\x00"))
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}
func AuthenticatedFacts(facts []IPFact, key []byte) []IPFact {
	out := []IPFact{}
	for _, f := range facts {
		if hmac.Equal([]byte(f.MAC), []byte(IPFactMAC(f, key))) {
			out = append(out, f)
		}
	}
	return out
}
