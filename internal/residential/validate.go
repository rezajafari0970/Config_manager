package residential

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"net"
	"strings"
)

func identifier(s string) bool {
	if len(s) < 1 || len(s) > 128 {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("._-", c)) {
			return false
		}
	}
	return true
}
func domain(s string) bool {
	return identifier(s) && strings.Contains(s, ".") && net.ParseIP(s) == nil && s == strings.ToLower(s) && !strings.HasPrefix(s, ".") && !strings.HasSuffix(s, ".") && !strings.Contains(s, "..")
}
func (p Policy) Validate() error {
	invalid := errors.New("invalid_residential_policy")
	if p.RouteComparisonVersion != "" && p.RouteComparisonVersion != RouteComparisonVersion || p.Version != Version || !identifier(p.DomainsVersion) || len(p.Domains) < 1 || len(p.Domains) > 64 || p.Repeats < 2 || p.Repeats > 8 || p.Concurrency < 1 || p.Concurrency > 4 || p.TimeoutSeconds < 1 || p.TimeoutSeconds > 20 || p.RetestSeconds < 30 || p.RetestSeconds > 1800 || p.EvidenceTTLSeconds < p.RetestSeconds || p.EvidenceTTLSeconds > 3600 || p.IntelTTLSeconds < 30 || p.IntelTTLSeconds > 3600 || p.ConfirmationAttempts < 3 || p.ConfirmationAttempts > 8 || p.ConfirmationSeconds < 30 || p.ConfirmationSeconds > 600 {
		return invalid
	}
	seen := map[string]bool{}
	for _, d := range p.Domains {
		if !domain(d) || seen[d] {
			return invalid
		}
		seen[d] = true
	}
	if len(p.Families) < 1 || len(p.Families) > 2 || len(p.Transports) < 1 || len(p.Transports) > 3 {
		return invalid
	}
	seen = map[string]bool{}
	for _, f := range p.Families {
		k := string(rune(f))
		if f != 4 && f != 6 || seen[k] {
			return invalid
		}
		seen[k] = true
	}
	seen = map[string]bool{}
	for _, transport := range p.Transports {
		if transport != "tcp" && transport != "udp" && transport != "quic" || seen[transport] {
			return invalid
		}
		seen[transport] = true
	}
	return nil
}
func (t Trust) Validate() error {
	invalid := errors.New("invalid_residential_trust")
	if t.Version != Version || len(t.Providers) > 8 || len(t.Capabilities) > 64 {
		return invalid
	}
	for id, p := range t.Providers {
		if !identifier(id) || !identifier(p.Group) {
			return invalid
		}
	}
	for id, c := range t.Capabilities {
		key, e := base64.StdEncoding.DecodeString(c.PublicKey)
		if e != nil || len(key) != ed25519.PublicKeySize || c.Method != "server-flow-ed25519-v1" {
			return invalid
		}
		if !identifier(id) || !identifier(c.Method) || c.Expires <= 0 || !digestOK(c.EvidenceSHA256) || len(c.Domains) < 1 || len(c.Domains) > 64 || len(c.Transports) < 1 || len(c.Transports) > 3 {
			return invalid
		}
		for _, d := range c.Domains {
			if !domain(d) {
				return invalid
			}
		}
	}
	return nil
}
