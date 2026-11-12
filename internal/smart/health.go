package smart

import (
	"errors"
	"strconv"
	"strings"
)

// PayloadProbeBytes is deliberately larger than the 38-43 KiB truncation observed
// on the affected phone path. This is a download completion observation, not an
// upload test, content digest, HTTP status check, carrier detector or stream retry.
const PayloadProbeBytes = 65536
const PayloadProbeURL = "https://speed.cloudflare.com/__down?bytes=65536"
const PayloadProbeTimeout = "10s"

// applyPayloadHealth is shared by every Smart composer, including Ads. Bodies
// are read by both pinned official cores only with the exact GET method.
// With exactly 64 KiB responses, nominal steady-state body traffic is at most
// 64 KiB/s across the observed pool, excluding
// headers, TLS, retries and startup. Xray's immediate startup sweep is concurrent;
// JSON exposes no startup-concurrency limiter or response-body byte limit.
// Oversized responses exceed the nominal budget. Never call it a hard traffic cap.
func applyPayloadHealth(root M) error {
	observer, ok := root["burstObservatory"].(M)
	if !ok {
		return errors.New("smart_missing_observer")
	}
	selectors, ok := observer["subjectSelector"].([]any)
	if !ok || len(selectors) == 0 {
		return errors.New("smart_invalid_observer_selector")
	}
	for _, value := range selectors {
		if prefix, ok := value.(string); !ok || prefix == "" {
			return errors.New("smart_invalid_observer_selector")
		}
	}
	nodes, ok := root["outbounds"].([]any)
	if !ok {
		return errors.New("smart_missing_outbounds")
	}
	count := 0
	seen := map[string]bool{}
	for _, value := range nodes {
		ob, ok := value.(M)
		if !ok {
			return errors.New("smart_invalid_outbound")
		}
		tag, _ := ob["tag"].(string)
		if tag == "" || seen[tag] {
			return errors.New("smart_invalid_outbound_tag")
		}
		seen[tag] = true
		for _, value := range selectors {
			prefix := value.(string)
			if strings.HasPrefix(tag, prefix) {
				count++
				break
			}
		}
	}
	if count == 0 {
		return errors.New("smart_empty_observer")
	}
	interval := max(15, count)
	observer["pingConfig"] = M{
		"destination": PayloadProbeURL, "interval": strconv.Itoa(interval) + "s",
		"sampling": 2, "timeout": PayloadProbeTimeout, "httpMethod": "GET",
	}
	// No direct connectivity URL: a failed system-side check must not suppress
	// per-proxy failures or leak a probe outside the configured proxy path.
	return nil
}
