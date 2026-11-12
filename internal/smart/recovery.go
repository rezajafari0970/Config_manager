package smart

import (
	"bytes"
	"encoding/json"
	"errors"
)

// applyRecoveryEnvelope preserves the fast primary leastLoad pool. Only an empty
// primary selection dispatches once into a separately tagged, observed slow pool.
// Endpoint bootstrap has its own IP-only recovery pool and cannot recurse through
// a domain member. No unobserved or direct outbound is a fallback.
func applyRecoveryEnvelope(data []byte) ([]byte, error) {
	root, err := object(data)
	if err != nil {
		return nil, err
	}
	for _, pair := range [][2]string{{"smart-parallel-balancer", "smart-recovery"}, {"bootstrap-ip-balancer", "bootstrap-recovery"}} {
		if err := addObservedRecovery(root, pair[0], pair[1]); err != nil {
			return nil, err
		}
	}
	if err := applyPayloadHealth(root); err != nil {
		return nil, err
	}
	return encodeEnvelope(root)
}
func addObservedRecovery(root M, primaryTag, recoveryTag string) error {
	routing, ok := root["routing"].(M)
	if !ok {
		return errors.New("smart_invalid_recovery_structure")
	}
	balancers, ok := routing["balancers"].([]any)
	if !ok {
		return errors.New("smart_invalid_recovery_structure")
	}
	rules, ok := routing["rules"].([]any)
	if !ok {
		return errors.New("smart_invalid_recovery_structure")
	}
	nodes, ok := root["outbounds"].([]any)
	if !ok || recoveryTag == "" {
		return errors.New("smart_invalid_recovery_structure")
	}
	for _, value := range nodes {
		ob, ok := value.(M)
		if !ok {
			return errors.New("smart_invalid_recovery_structure")
		}
		if ob["tag"] == recoveryTag {
			return errors.New("smart_duplicate_recovery")
		}
	}
	for _, value := range balancers {
		b, ok := value.(M)
		if !ok {
			return errors.New("smart_invalid_recovery_structure")
		}
		if b["tag"] == recoveryTag+"-balancer" {
			return errors.New("smart_duplicate_recovery")
		}
	}
	for _, value := range rules {
		rule, ok := value.(M)
		if !ok {
			return errors.New("smart_invalid_recovery_structure")
		}
		if tags, ok := rule["inboundTag"].([]any); ok {
			for _, tag := range tags {
				if tag == recoveryTag+"-egress" {
					return errors.New("smart_duplicate_recovery")
				}
			}
		}
	}
	var primary M
	for _, value := range balancers {
		b := value.(M)
		if b["tag"] == primaryTag {
			primary = b
			break
		}
	}
	if primary == nil {
		return errors.New("smart_recovery_missing_primary")
	}
	selectors, ok := primary["selector"].([]any)
	if !ok || len(selectors) == 0 {
		return errors.New("smart_invalid_recovery_selector")
	}
	for _, value := range selectors {
		if tag, ok := value.(string); !ok || tag == "" {
			return errors.New("smart_invalid_recovery_selector")
		}
	}
	selector := append([]any{}, selectors...)
	primary["fallbackTag"] = recoveryTag
	routing["balancers"] = append(balancers, M{"tag": recoveryTag + "-balancer", "selector": selector,
		"strategy": M{"type": "leastLoad", "settings": M{"expected": 1, "maxRTT": PayloadProbeTimeout, "tolerance": 0.15}}, "fallbackTag": "block"})
	root["outbounds"] = append(root["outbounds"].([]any), M{"tag": recoveryTag, "protocol": "loopback", "settings": M{"inboundTag": recoveryTag + "-egress"}})
	routing["rules"] = append([]any{M{"type": "field", "inboundTag": []any{recoveryTag + "-egress"}, "network": "tcp,udp", "balancerTag": recoveryTag + "-balancer"}}, routing["rules"].([]any)...)
	return nil
}

func encodeEnvelope(root M) ([]byte, error) {
	allowed := map[string]bool{}
	for _, key := range []string{"remarks", "log", "dns", "inbounds", "outbounds", "routing", "burstObservatory", "policy", "stats", "metrics"} {
		allowed[key] = true
	}
	for key := range root {
		if !allowed[key] {
			return nil, errors.New("smart_unexpected_envelope_field")
		}
	}
	for key := range allowed {
		if _, ok := root[key]; !ok {
			return nil, errors.New("smart_missing_envelope_field")
		}
	}
	var ordered bytes.Buffer
	ordered.WriteByte('{')
	for i, key := range []string{"remarks", "log", "dns", "inbounds", "outbounds", "routing", "burstObservatory", "policy", "stats", "metrics"} {
		if i > 0 {
			ordered.WriteByte(',')
		}
		encoded, _ := json.Marshal(key)
		ordered.Write(encoded)
		ordered.WriteByte(':')
		if err := orderedJSON(&ordered, root[key]); err != nil {
			return nil, err
		}
	}
	ordered.WriteByte('}')
	return ordered.Bytes(), nil
}
