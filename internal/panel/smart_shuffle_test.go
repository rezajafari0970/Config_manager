package panel

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/repair"
	"reflect"
	"sort"
	"testing"
)

func smartSemantic(t *testing.T, body []byte) string {
	t.Helper()
	var root map[string]json.RawMessage
	if e := json.Unmarshal(body, &root); e != nil {
		t.Fatal(e)
	}
	var nodes []json.RawMessage
	_ = json.Unmarshal(root["outbounds"], &nodes)
	sort.Slice(nodes, func(i, j int) bool { return string(nodes[i]) < string(nodes[j]) })
	root["outbounds"], _ = json.Marshal(nodes)
	b, _ := json.Marshal(root)
	return string(b)
}
func TestSmartPerRequestShuffleKeepsCacheAndRoundOwnership(t *testing.T) {
	s, _ := smartFixture(t)
	finishSmart(t, s)
	ctx := context.Background()
	rows, _ := querySmart(ctx, s.db, true, 0)
	canonical, hashes := map[string]bool{}, map[string]string{}
	for _, row := range rows {
		b, e := s.smartBytes(ctx, row)
		if e != nil {
			t.Fatal(e)
		}
		canonical[smartSemantic(t, b)] = true
		hashes[row.ID] = repair.Digest(b)
	}
	app, _ := NewServer(s, "http://127.0.0.1:18191")
	h := app.Handler()
	orders, first := map[string]bool{}, map[string]bool{}
	for n := 0; n < 16; n++ {
		r := exportRequest(t, h, "GET", app.smartURL(false))
		if r.Code != 200 || r.Header().Get("Cache-Control") != "no-store" || r.Header().Get("X-Content-SHA256") != repair.Digest(r.Body.Bytes()) {
			t.Fatal("invalid full response")
		}
		lines := exportObjects(t, r.Body.Bytes())
		seen := map[string]bool{}
		order := ""
		for i, line := range lines {
			key := smartSemantic(t, line)
			if !canonical[key] || seen[key] {
				t.Fatal("changed or duplicated profile")
			}
			seen[key] = true
			order += key
			if i == 0 {
				first[key] = true
			}
		}
		if !reflect.DeepEqual(seen, canonical) {
			t.Fatal("missing profiles")
		}
		orders[order] = true
	}
	if len(orders) < 2 || len(first) < 2 {
		t.Fatal("full export order is fixed")
	}
	turns := map[string]bool{}
	for range rows {
		r := exportRequest(t, h, "GET", app.smartURL(true))
		body := bytes.TrimSpace(r.Body.Bytes())
		id := r.Header().Get("X-Smart-First-ID")
		if r.Code != 200 || turns[id] || !canonical[smartSemantic(t, body)] {
			t.Fatal("round ownership changed")
		}
		turns[id] = true
		if r.Header().Get("X-Content-SHA256") != repair.Digest(body) || r.Header().Get("X-Smart-Cache-SHA256") != hashes[id] {
			t.Fatal("response/cache hash conflated")
		}
	}
	for _, row := range rows {
		b, e := s.smartBytes(ctx, row)
		if e != nil || repair.Digest(b) != hashes[row.ID] {
			t.Fatal("cache mutated", e)
		}
		r := exportRequest(t, h, "GET", app.origin+"/smart/json/"+row.ID)
		if r.Code != 200 || !bytes.Equal(r.Body.Bytes(), b) {
			t.Fatal("individual cached API changed")
		}
	}
}
