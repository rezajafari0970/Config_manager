package panel

import (
	"encoding/json"
	"testing"
)

func TestScopedStateKeepsSummaryWithoutUnrelatedPayloads(t *testing.T) {
	s, _ := smartFixture(t)
	app, e := NewServer(s, "http://127.0.0.1:18191")
	if e != nil {
		t.Fatal(e)
	}
	h := app.Handler()
	for _, view := range []string{"", "input", "summary", "invalid"} {
		r := exportRequest(t, h, "GET", app.origin+"/api/state?view="+view)
		if view == "invalid" {
			if r.Code != 422 {
				t.Fatal(r.Code)
			}
			continue
		}
		var v struct {
			Sources     []Source                           `json:"sources"`
			Outputs     []Artifact                         `json:"outputs"`
			Conversions map[string]SourceConversionSummary `json:"conversions"`
			Stats       map[string]int                     `json:"stats"`
		}
		if e = json.Unmarshal(r.Body.Bytes(), &v); e != nil || r.Code != 200 {
			t.Fatal("state", e, r.Code)
		}
		if v.Stats["sources"] != 3 {
			t.Fatal("summary lost sources", v.Stats)
		}
		if view == "summary" && (len(v.Sources) != 0 || len(v.Outputs) != 0 || len(v.Conversions) != 0) {
			t.Fatal("summary performed unrelated feature work")
		}
		if view == "input" && (len(v.Sources) != 3 || len(v.Outputs) != 0 || len(v.Conversions) != 3) {
			t.Fatal("input ownership summaries changed")
		}
		if view == "" && len(v.Outputs) != 7 {
			t.Fatal("legacy API changed")
		}
	}
}
