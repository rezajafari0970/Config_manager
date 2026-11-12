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
	return encodeEnvelope(root)
}
func addObservedRecovery(root M, primaryTag, recoveryTag string) error {
	routing := root["routing"].(M)
	balancers := routing["balancers"].([]any)
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
	selector := append([]any{}, primary["selector"].([]any)...)
	primary["fallbackTag"] = recoveryTag
	routing["balancers"] = append(balancers, M{"tag": recoveryTag + "-balancer", "selector": selector,
		"strategy": M{"type": "leastLoad", "settings": M{"expected": 1, "maxRTT": "5s", "tolerance": 0.5}}, "fallbackTag": "block"})
	root["outbounds"] = append(root["outbounds"].([]any), M{"tag": recoveryTag, "protocol": "loopback", "settings": M{"inboundTag": recoveryTag + "-egress"}})
	routing["rules"] = append([]any{M{"type": "field", "inboundTag": []any{recoveryTag + "-egress"}, "network": "tcp,udp", "balancerTag": recoveryTag + "-balancer"}}, routing["rules"].([]any)...)
	return nil
}

func encodeEnvelope(root M) ([]byte, error) {
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
