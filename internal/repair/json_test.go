package repair

import (
	"bytes"
	"strings"
	"testing"
)

func TestSurgicalRepairPreservesRawAndUnknownFields(t *testing.T) {
	raw := []byte("{\n \"remarks\":\"stats [] / فارسی\", \"stats\" : [  ], \"unknown\":{\"number\":9007199254740993}, \"inbounds\":[], \"outbounds\":[{\"protocol\":\"freedom\"}], \"routing\":{}\n}")
	out, patch, err := RepairStats(raw)
	if err != nil || len(patch) != 1 {
		t.Fatal(patch, err)
	}
	want := bytes.Replace(raw, []byte("\"stats\" : [  ]"), []byte("\"stats\" : {}"), 1)
	if !bytes.Equal(out, want) {
		t.Fatalf("non-local rewrite: %s", out)
	}
	if Analyze(out).HasErrors() || Analyze(out).Publishable {
		t.Fatal("wrong candidate status")
	}
	second, patch, err := RepairStats(out)
	if err != nil || len(patch) != 0 || !bytes.Equal(second, out) {
		t.Fatal("not idempotent")
	}
}
func TestHealthyDocumentStaysByteIdentical(t *testing.T) {
	raw := []byte(" {\"stats\": {},\"inbounds\":[],\"outbounds\":[{\"protocol\":\"freedom\"}],\"routing\":{}} \n")
	out, p, e := RepairStats(raw)
	if e != nil || len(p) > 0 || !bytes.Equal(raw, out) {
		t.Fatal("healthy input changed")
	}
}
func TestAmbiguousDocumentsCannotBeRepaired(t *testing.T) {
	for _, s := range []string{`{"stats":[],"stats":{}}`, `{"stats":[],"a":{"x":1,"x":2}}`, `{"stats":[1]}`, `{"stats":[]}{}`, `{"stats":`} {
		if _, _, err := RepairStats([]byte(s)); err == nil {
			t.Fatal("accepted ambiguity", s)
		}
	}
}
func TestDepthAndSizeLimits(t *testing.T) {
	for _, s := range []string{strings.Repeat("[", MaxDepth+2) + "0" + strings.Repeat("]", MaxDepth+2), strings.Repeat(" ", MaxBytes+1)} {
		if !Analyze([]byte(s)).HasErrors() {
			t.Fatal("missing limit")
		}
	}
}
func TestReportsMultipleIndependentProblems(t *testing.T) {
	r := Analyze([]byte(`{"stats":[],"policy":[],"inbounds":{},"outbounds":[{"protocol":"hysteria","settings":{"version":1}}],"routing":{}}`))
	if len(r.Findings) < 5 || r.Publishable {
		t.Fatalf("%+v", r)
	}
}
func TestImportEnvelopeIsIndependentFromCoreSchema(t *testing.T) {
	r := Analyze([]byte(`{"outbounds":[{"protocol":"freedom"}]}`))
	if len(r.Findings) != 2 || r.CoreValidation != "not_run" || r.AndroidImport != "not_run" {
		t.Fatalf("%+v", r)
	}
}
func FuzzStatsRepairPreservesUnaffectedBytes(f *testing.F) {
	f.Add([]byte(`{"stats":[],"inbounds":[],"outbounds":[{"protocol":"freedom"}],"routing":{}}`))
	f.Add([]byte(`{"stats":{},"opaque":9007199254740993}`))
	f.Fuzz(func(t *testing.T, b []byte) {
		out, p, err := RepairStats(b)
		if err != nil {
			return
		}
		if len(p) == 0 && !bytes.Equal(out, b) {
			t.Fatal("unmodified bytes changed")
		}
		if strict(out) != nil {
			t.Fatal("invalid repaired json")
		}
		again, q, e := RepairStats(out)
		if e != nil || len(q) != 0 || !bytes.Equal(out, again) {
			t.Fatal("repair not idempotent")
		}
	})
}
