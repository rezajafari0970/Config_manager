package repair

import (
	"bytes"
	"testing"
)

func TestCandidateRepairsObservedPenguinDefectsWithoutOtherChanges(t *testing.T) {
	raw := []byte(" {\"outbounds\":[{\"protocol\":\"vless\",\"settings\":{\"address\":\"192.0.2.1\",\"id\":\"00000000-0000-4000-8000-000000000001\",\"port\":443}},{\"protocol\":\"freedom\",\"settings\":[ ]}],\"stats\":[  ],\"inbounds\":[],\"routing\":{},\"opaque\":9007199254740993,\"secret\":\"keep [] }, , unchanged\"}\r\n")
	original := bytes.Clone(raw)
	d := (JSONDriver{}).Propose(raw)
	want := bytes.Replace(raw, []byte(`"settings":[ ]`), []byte(`"settings":{}`), 1)
	want = bytes.Replace(want, []byte(`"stats":[  ]`), []byte(`"stats":{}`), 1)
	if d.Status != "repaired" || len(d.Patches) != 2 || !bytes.Equal(d.Candidate, want) || !bytes.Equal(raw, original) {
		t.Fatal("unsafe repair", d)
	}
	again := (JSONDriver{}).Propose(d.Candidate)
	if again.Status != "unchanged" || len(again.Patches) != 0 || len(again.Candidate) != 0 {
		t.Fatal("not idempotent")
	}
	if d.CoreValidation != "not_run" || d.AndroidImport != "not_run" || d.Connection != "not_run" {
		t.Fatal("false evidence")
	}
}
func TestCandidateAmbiguityAndUnknownFields(t *testing.T) {
	for _, raw := range []string{`{"outbounds":[{"protocol":"vless","settings":[]}],"stats":[]}`, `{"outbounds":[{"protocol":"freedom","settings":[1]}]}`, `{"outbounds":[],"stats":[],"stats":{}}`, `{"outbounds":[{"protocol":"vless"}],"stats":[1]}`, `{"outbounds":[{"protocol":"hysteria","settings":{"version":1}}]}`, `{"outbounds":[`, `{"outbounds":[],}`, `{"outbounds":[{"protocol":"freedom","settings":{,}}]}`} {
		d := (JSONDriver{}).Propose([]byte(raw))
		if d.Status != "blocked" || len(d.Candidate) > 0 {
			t.Fatal("unsafe ambiguous candidate", raw, d)
		}
	}
	raw := []byte("{\"outbounds\":[{\"protocol\":\"vless\",\"settings\":{}}],\"unknown\":[  ],\"opaque\":\",}\",}\r\n")
	d := (JSONDriver{}).Propose(raw)
	if d.Status != "repaired" || !bytes.Contains(d.Candidate, []byte(`"unknown":[  ]`)) || !bytes.Contains(d.Candidate, []byte(`"opaque":",}"`)) || Analyze(d.Candidate).HasErrors() {
		t.Fatal(d)
	}
}

type badRepair struct{}

func (badRepair) Kind() string              { return "test" }
func (badRepair) Propose([]byte) Diagnostic { panic("test") }
func TestRepairPluginIsolationAndNoCredentialGuess(t *testing.T) {
	r := NewRegistry()
	r.Register(badRepair{})
	if r.Propose([]byte("x"), true, "test", "invalid").Status != "blocked" {
		t.Fatal("panic escaped")
	}
	if r.Propose([]byte("vless://missing"), true, "vless", "invalid").Status != "blocked" {
		t.Fatal("invented credential")
	}
	if r.Propose([]byte("[Interface]\nJc=4\n[Peer]\n"), true, "wireguard", "valid").Status != "unverified" {
		t.Fatal("discarded extension")
	}
}
func FuzzStructuralCandidate(f *testing.F) {
	f.Add([]byte(`{"stats":[],"outbounds":[{"protocol":"freedom","settings":[]}]}`))
	f.Fuzz(func(t *testing.T, b []byte) {
		orig := bytes.Clone(b)
		d := (JSONDriver{}).Propose(b)
		if !bytes.Equal(orig, b) {
			t.Fatal("changed input")
		}
		if d.Status == "repaired" {
			if strict(d.Candidate) != nil || Analyze(d.Candidate).HasErrors() {
				t.Fatal("bad candidate")
			}
			again := (JSONDriver{}).Propose(d.Candidate)
			if again.Status != "unchanged" {
				t.Fatal("not idempotent")
			}
		}
	})
}
