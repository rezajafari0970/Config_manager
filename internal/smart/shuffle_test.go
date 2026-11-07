package smart

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

func TestShufflePreservesCanonicalValuesAndIPDefault(t *testing.T) {
	for _, count := range []int{1, 2, 5, 7, 10, 15, 50, 100} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			members := []Member{}
			for i := 1; i <= count; i++ {
				m := member(t, i, "A")
				if i == 1 && count > 1 {
					m.Body = bytes.ReplaceAll(m.Body, []byte("192.0.2.1"), []byte("endpoint.test"))
				}
				members = append(members, m)
			}
			canonical, err := Compose(members)
			if err != nil {
				t.Fatal(err)
			}
			original := bytes.Clone(canonical)
			var expected map[string]json.RawMessage
			_ = json.Unmarshal(canonical, &expected)
			nodes := outboundBytes(t, expected["outbounds"])
			root, _ := object(canonical)
			eligible := map[string]bool{}
			for _, tag := range root["routing"].(M)["balancers"].([]any)[1].(M)["selector"].([]any) {
				eligible[tag.(string)] = true
			}
			layouts, firstTags := map[string]bool{}, map[string]bool{}
			for trial := 0; trial < 24; trial++ {
				b, err := Shuffle(canonical)
				if err != nil {
					t.Fatal(err)
				}
				layouts[string(b)] = true
				var got map[string]json.RawMessage
				_ = json.Unmarshal(b, &got)
				if !reflect.DeepEqual(outboundBytes(t, got["outbounds"]), nodes) {
					t.Fatal("outbound values changed")
				}
				got["outbounds"] = expected["outbounds"]
				if !reflect.DeepEqual(got, expected) || !bytes.Equal(original, canonical) {
					t.Fatal("global fields or cache mutated")
				}
				parsed, _ := object(b)
				first := parsed["outbounds"].([]any)[0].(M)["tag"].(string)
				if !eligible[first] {
					t.Fatal("RealPing default depends on DNS", first)
				}
				firstTags[first] = true
				if trial == 0 {
					validateCore(t, b)
				}
			}
			if count > 2 && (len(layouts) < 2 || len(firstTags) < 2) {
				t.Fatal("order was not randomized")
			}
		})
	}
}
func outboundBytes(t *testing.T, raw []byte) map[string]string {
	t.Helper()
	var nodes []json.RawMessage
	if err := json.Unmarshal(raw, &nodes); err != nil {
		t.Fatal(err)
	}
	result := map[string]string{}
	for _, node := range nodes {
		var value struct {
			Tag string `json:"tag"`
		}
		_ = json.Unmarshal(node, &value)
		if _, ok := result[value.Tag]; ok {
			t.Fatal("duplicate tag")
		}
		result[value.Tag] = string(node)
	}
	return result
}
