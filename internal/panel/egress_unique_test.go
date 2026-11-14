package panel

import (
	"context"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/egress"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEgressIdentityCachesAliasesAndIgnoresArchive(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	base := strings.Split(sourceURI(241), "#")[0]
	newExportSource(t, s, "alias-a", base+"#one")
	newExportSource(t, s, "alias-b", base+"#two")
	if _, err := s.db.Exec("INSERT INTO egress_jobs(hash,kind,size,first_seen,last_seen) VALUES(?,'fixture',1,1,1)", strings.Repeat("a", 64)); err != nil {
		t.Fatal(err)
	}
	j, err := s.ClaimEgress(ctx)
	if err != nil || j == nil {
		t.Fatal(j, err)
	}
	second, err := s.ClaimEgress(ctx)
	if err != nil || second != nil {
		t.Fatal("duplicate/archive claim", second, err)
	}
	r := syntheticEgress(*j)
	for i := range r.Samples {
		r.Samples[i].Source = egress.Hash([]byte("1.1.1.1"))
	}
	if err = s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: r}); err != nil {
		t.Fatal(err)
	}
	var state string
	s.db.QueryRow("SELECT state FROM egress_jobs WHERE hash=?", j.Hash).Scan(&state)
	if state != "NO_SPLIT" {
		t.Fatal(state)
	}
	s.db.Exec("UPDATE egress_jobs SET next_run=0")
	second, err = s.ClaimEgress(ctx)
	if err != nil || second != nil {
		t.Fatal("negative re-tested", second, err)
	}
	// Reappearance with another cosmetic label must keep the same exact receipt.
	s.db.Exec("DELETE FROM memberships")
	newExportSource(t, s, "alias-c", base+"#three")
	second, err = s.ClaimEgress(ctx)
	if err != nil || second != nil {
		t.Fatal("reappearance reset cache", second, err)
	}
	var attempts int
	s.db.QueryRow("SELECT attempts FROM egress_jobs WHERE hash=?", j.Hash).Scan(&attempts)
	if attempts != 1 {
		t.Fatal(attempts)
	}
	newExportSource(t, s, "changed", sourceURI(242))
	changed, err := s.ClaimEgress(ctx)
	if err != nil || changed == nil || changed.Hash == j.Hash {
		t.Fatal("changed connection not scheduled", changed, err)
	}
}

func TestEgressIdentityBoundedUnknownAndCurrentScopeFence(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "unknown", sourceURI(243))
	j, _ := s.ClaimEgress(ctx)
	r := syntheticEgress(*j)
	r.Samples = nil
	r.Health = nil
	if err := s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: r}); err != nil {
		t.Fatal(err)
	}
	s.db.Exec("UPDATE egress_jobs SET next_run=0")
	follow, err := s.ClaimEgress(ctx)
	if err != nil || follow == nil || !follow.Deep {
		t.Fatal(follow, err)
	}
	r = syntheticEgress(*follow)
	r.Samples = nil
	r.Health = nil
	if err = s.CompleteEgress(ctx, egress.Completion{Job: *follow, Report: r}); err != nil {
		t.Fatal(err)
	}
	s.db.Exec("UPDATE egress_jobs SET next_run=0")
	end, err := s.ClaimEgress(ctx)
	if err != nil || end != nil {
		t.Fatal("unbounded unknown retry", end, err)
	}
	newExportSource(t, s, "removed", sourceURI(244))
	pending, _ := s.ClaimEgress(ctx)
	report := syntheticEgress(*pending)
	s.db.Exec("DELETE FROM memberships WHERE raw_hash=?", pending.Hash)
	if err = s.CompleteEgress(ctx, egress.Completion{Job: *pending, Report: report}); err != ErrConflict {
		t.Fatal("out-of-scope proof committed", err)
	}
}

func TestEgressIdentityPublicationAndAccountingAgree(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	base := strings.Split(sourceURI(245), "#")[0]
	newExportSource(t, s, "pub-a", base+"#a")
	newExportSource(t, s, "pub-b", base+"#b")
	j, _ := s.ClaimEgress(ctx)
	if err := s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: syntheticEgress(*j)}); err != nil {
		t.Fatal(err)
	}
	app, _ := NewServer(s, "http://127.0.0.1:18191")
	rr := httptest.NewRecorder()
	egressModule{app}.state(rr, httptest.NewRequest("GET", "/api/egress?brief=1", nil))
	var body struct {
		Total, Eligible int
		Entries         []json.RawMessage
		Accounting      struct {
			Current     int            `json:"current_identities"`
			Publication map[string]int `json:"publication"`
		} `json:"accounting"`
	}
	if rr.Code != 200 || json.Unmarshal(rr.Body.Bytes(), &body) != nil {
		t.Fatal(rr.Code, rr.Body.String())
	}
	if body.Total != 1 || body.Eligible != 1 || len(body.Entries) != 1 || body.Accounting.Current != 1 || body.Accounting.Publication["published"] != 1 || body.Accounting.Publication["identities"] != 1 {
		t.Fatal(body)
	}
	url := app.origin + "/egress-subscription/" + s.digest("egress-export-v2", "all") + "/"
	for _, f := range []string{"configs.json", "configs.jsonl", "raw.txt", "next.json"} {
		rr = exportRequest(t, app.Handler(), "GET", url+f)
		if rr.Code != 200 || rr.Header().Get("X-Config-Count") != "1" {
			t.Fatal(f, rr.Code, rr.Header())
		}
	}
	// Verified retained identities survive disappearance; never create a duplicate.
	s.db.Exec("DELETE FROM memberships")
	q, _ := freezeEgress(ctx, s.db)
	a, err := egressAccounting(ctx, q)
	if err != nil || a["retained_outside_current"] != 1 || a["live_versions"] != 1 {
		t.Fatal(a, err)
	}
}
