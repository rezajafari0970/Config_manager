package probe

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"
)

// VMess share metadata is not VMess wire configuration. This closed, versioned
// adapter follows the target client's DTO without silently discarding unknown
// active extensions. Exact received bytes are retained by the caller.
const shareCompatibilityVersion = "share-compat.v1"

func normalizeVMessShare(m M) ([]string, error) {
	rules := []string{}
	for k, v := range m {
		switch k {
		case "name", "test_name":
			if _, ok := v.(string); !ok {
				return nil, errors.New("invalid_vmess_metadata")
			}
			delete(m, k)
			rules = appendRule(rules, "metadata")
		case "batch", "echConfigList", "echServerKeys", "fakehost_domain", "interval", "length", "mux", "muxConcurrency", "packets", "xhttp_extra":
			// Only the exact empty-string placeholders are inert. false/0/null or a
			// populated option must not be guessed away.
			if v != "" {
				return nil, errors.New("vmess_extension_requires_driver")
			}
			delete(m, k)
			rules = appendRule(rules, "empty-placeholders")
		case "skip-cert-verify":
			skip, err := shareBool(v)
			if err != nil {
				return nil, err
			}
			if old, exists := m["insecure"]; exists && old != "" {
				other, e := shareBool(old)
				if e != nil {
					return nil, e
				}
				if other != skip {
					// Both instructions can be satisfied only by retaining certificate
					// verification. Never let an alias relax an explicit strict setting.
					skip = false
					rules = appendRule(rules, "tls-verification-strict-conflict")
				}
			}
			// The alias has no effect without TLS. With TLS it enters the existing
			// strict-certificate candidate path; it never enables insecure publication.
			if str(m, "tls") == "tls" {
				if skip {
					m["insecure"] = "1"
				} else {
					m["insecure"] = "0"
				}
			}
			delete(m, k)
			rules = appendRule(rules, "tls-verification-alias")
		}
	}
	return rules, nil
}

func shareBool(v any) (bool, error) {
	switch x := v.(type) {
	case bool:
		return x, nil
	case string:
		switch x {
		case "1", "true":
			return true, nil
		case "0", "false":
			return false, nil
		}
	case json.Number:
		if x == "1" {
			return true, nil
		}
		if x == "0" {
			return false, nil
		}
	}
	return false, errors.New("invalid_tls_verification_flag")
}

func appendRule(rules []string, rule string) []string {
	for _, r := range rules {
		if r == rule {
			return rules
		}
	}
	return append(rules, rule)
}

// Known subscription annotations are retained in raw input but never treated
// as network settings or fetched. All other unknown query keys remain errors.
func stripShareAnnotations(q url.Values) ([]string, error) {
	rules := []string{}
	for _, key := range []string{"x-sub", "Telegram"} {
		values, ok := q[key]
		if !ok {
			continue
		}
		if len(values) != 1 {
			return nil, errors.New("ambiguous_query_parameter")
		}
		value := values[0]
		if len(value) > 2048 || strings.ContainsAny(value, "\x00\r\n") {
			return nil, errors.New("invalid_share_metadata")
		}
		delete(q, key)
		rules = appendRule(rules, "subscription-metadata")
	}
	return rules, nil
}

func shareFieldAllowed(key, fields string) bool {
	for _, field := range strings.Fields(fields) {
		if key == field {
			return true
		}
	}
	return false
}
func slicesContain(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
