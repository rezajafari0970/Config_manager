package smart

import (
	"bytes"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"strings"
)

// Shuffle permutes only proxy objects in an already validated Smart profile.
// Tags, references, helpers and all connection bytes remain attached to their
// original objects. The independent IP pool supplies the default for libv2ray's
// route-less RealPing. Canonical cache bytes and their proof are never rewritten.
func Shuffle(body []byte) ([]byte, error) {
	var root struct {
		Outbounds json.RawMessage `json:"outbounds"`
		Routing   struct {
			Balancers []struct {
				Tag      string   `json:"tag"`
				Selector []string `json:"selector"`
			} `json:"balancers"`
		} `json:"routing"`
	}
	if json.Unmarshal(body, &root) != nil {
		return nil, errors.New("smart_invalid_profile")
	}
	allowed := map[string]bool{}
	for _, pool := range root.Routing.Balancers {
		if pool.Tag == "bootstrap-ip-balancer" {
			for _, tag := range pool.Selector {
				allowed[tag] = true
			}
		}
	}
	var nodes []json.RawMessage
	if json.Unmarshal(root.Outbounds, &nodes) != nil {
		return nil, errors.New("smart_invalid_profile")
	}
	slots := []int{}
	for i, node := range nodes {
		var ob struct {
			Tag string `json:"tag"`
		}
		if json.Unmarshal(node, &ob) != nil {
			return nil, errors.New("smart_invalid_outbound")
		}
		if strings.HasPrefix(ob.Tag, "proxy-") {
			slots = append(slots, i)
		}
	}
	if len(slots) == 0 || slots[0] != 0 {
		return nil, errors.New("smart_invalid_default")
	}
	rand.Shuffle(len(slots), func(i, j int) { nodes[slots[i]], nodes[slots[j]] = nodes[slots[j]], nodes[slots[i]] })
	chosen := -1
	for _, i := range slots {
		var ob struct {
			Tag string `json:"tag"`
		}
		_ = json.Unmarshal(nodes[i], &ob)
		if allowed[ob.Tag] {
			chosen = i
			break
		}
	}
	if chosen < 0 {
		return nil, errors.New("smart_bootstrap_no_ip")
	}
	nodes[0], nodes[chosen] = nodes[chosen], nodes[0]
	// Avoid re-encoding copied objects (large numbers, extension fields and
	// null/absent values). The root comes only from our canonical composer.
	var out bytes.Buffer
	out.WriteByte('[')
	for i, node := range nodes {
		if i > 0 {
			out.WriteByte(',')
		}
		out.Write(node)
	}
	out.WriteByte(']')
	return bytes.Replace(body, root.Outbounds, out.Bytes(), 1), nil
}
