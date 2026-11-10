package residential

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"strconv"
)

// SigningPayload is the versioned wire contract for an independently authorized
// server/egress observer. All values are strings, keys are lexicographically
// sorted by encoding/json. A signature cannot turn unobserved routing into
// evidence: the operator must trust the observer's measurement and scope.
func SigningPayload(o Observation) []byte {
	fields := map[string]string{
		"audience": "shen.residential.flow.v1", "variant": o.Variant, "domain": o.Domain,
		"family": strconv.Itoa(o.Family), "transport": o.Transport, "path": o.Path,
		"at": strconv.FormatInt(o.At, 10), "wave": strconv.Itoa(o.Wave), "connection_id": o.ConnectionID,
		"nonce": o.Nonce, "ip": o.IP, "method": o.Method, "capability": o.Capability,
		"config_hash": o.ConfigHash, "runtime_hash": o.RuntimeHash, "policy_hash": o.PolicyHash,
		"tls_verified": strconv.FormatBool(o.TLSVerified), "origin_verified": strconv.FormatBool(o.OriginVerified),
		"meaning_verified": strconv.FormatBool(o.MeaningVerified), "nonce_bound": strconv.FormatBool(o.NonceBound),
		"same_connection": strconv.FormatBool(o.SameConnection), "connections": strconv.Itoa(o.Connections),
		"redirects": strconv.Itoa(o.Redirects), "actual_domain_request": strconv.FormatBool(o.ActualDomainRequest),
		"response_sha256": o.ResponseSHA256, "outcome": o.Outcome,
	}
	b, _ := json.Marshal(fields)
	return b
}
func verifyAttestation(o Observation, c Capability) bool {
	if c.Method != "server-flow-ed25519-v1" || o.Method != c.Method {
		return false
	}
	public, e := base64.StdEncoding.DecodeString(c.PublicKey)
	if e != nil || len(public) != ed25519.PublicKeySize {
		return false
	}
	signature, e := base64.StdEncoding.DecodeString(o.Attestation)
	if e != nil || len(signature) != ed25519.SignatureSize {
		return false
	}
	return ed25519.Verify(public, SigningPayload(o), signature)
}
