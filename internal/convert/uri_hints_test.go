package convert

import (
	"bytes"
	"testing"
)

func TestRealityMLKEMHintPreservesNativeConfig(t *testing.T) {
	base := fixtures()["vless-reality"]
	want := Prepare([]byte(base))
	if want.Code != "prepared" {
		t.Fatal(want.Code)
	}
	for _, flag := range []string{"true", "1"} {
		raw := []byte(base + "&support-x25519mlkem768=" + flag)
		before := bytes.Clone(raw)
		got := Prepare(raw)
		if got.Code != "prepared" || len(got.Candidates) != 1 {
			t.Fatal(got.Code)
		}
		if !bytes.Equal(got.Candidates[0].Config, want.Candidates[0].Config) || !bytes.Equal(raw, before) {
			t.Fatal("native credentials/fingerprint/transport or raw source changed")
		}
	}
	for query, code := range map[string]string{
		"&support-x25519mlkem768=false":                             "reality_mlkem_disable_unsupported",
		"&support-x25519mlkem768=0":                                 "reality_mlkem_disable_unsupported",
		"&support-x25519mlkem768=":                                  "reality_mlkem_hint_invalid",
		"&support-x25519mlkem768=yes":                               "reality_mlkem_hint_invalid",
		"&support-x25519mlkem768=true&support-x25519mlkem768=false": "ambiguous_query_parameter",
		"&support-x25519mlkem768=true&unknown-option=true":          "unknown_query_parameter",
	} {
		p := Prepare([]byte(base + query))
		if p.Code != code || len(p.Candidates) != 0 {
			t.Fatalf("%s: %s", query, p.Code)
		}
	}
	for _, base := range []string{fixtures()["vless-tls"], fixtures()["trojan"]} {
		p := Prepare([]byte(base + "&support-x25519mlkem768=true"))
		if p.Code != "reality_mlkem_hint_incompatible" || len(p.Candidates) != 0 {
			t.Fatal(p.Code)
		}
	}
}
