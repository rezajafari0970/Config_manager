package panel

import (
	"context"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/egress"
	"net/http/httptest"
	"testing"
)

func TestEgressBriefPollingAndTargetedEvidence(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "brief", sourceURI(246), sourceURI(247))
	hash := ""
	for range 2 {
		j, _ := s.ClaimEgress(ctx)
		hash = j.Hash
		if e := s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: syntheticEgress(*j)}); e != nil {
			t.Fatal(e)
		}
	}
	app, _ := NewServer(s, "http://127.0.0.1:18191")
	m := egressModule{app}
	request := func(path string) (int, int, map[string]any) {
		rr := httptest.NewRecorder()
		m.state(rr, httptest.NewRequest("GET", path, nil))
		var body map[string]any
		json.Unmarshal(rr.Body.Bytes(), &body)
		return rr.Code, rr.Body.Len(), body
	}
	code, full, _ := request("/api/egress")
	if code != 200 {
		t.Fatal(code)
	}
	code, brief, body := request("/api/egress?brief=1")
	if code != 200 || brief >= full/2 {
		t.Fatal(code, brief, full)
	}
	for _, v := range body["entries"].([]any) {
		report := v.(map[string]any)["report"].(map[string]any)
		if _, ok := report["samples"]; ok {
			t.Fatal("poll leaked heavy evidence")
		}
	}
	code, _, body = request("/api/egress?view=diagnostics&hash=" + hash)
	if code != 200 || len(body["entries"].([]any)) != 1 {
		t.Fatal(code, body)
	}
	entry := body["entries"].([]any)[0].(map[string]any)
	if entry["hash"] != hash || len(entry["report"].(map[string]any)["samples"].([]any)) == 0 {
		t.Fatal("targeted proof missing")
	}
	code, _, _ = request("/api/egress?hash=invalid")
	if code != 422 {
		t.Fatal(code)
	}
	for _, file := range []string{"configs.json", "configs.jsonl", "next.json", "raw.txt"} {
		rr := exportRequest(t, app.Handler(), "HEAD", app.origin+"/egress-subscription/"+s.digest("egress-export-v2", "all")+"/"+file)
		if rr.Code != 200 || rr.Header().Get("X-Country-Policy") != "any-observed-country-selected; 24h; GeoIP-estimate" {
			t.Fatal(file, rr.Code, rr.Header())
		}
	}
}
