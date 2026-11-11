package panel

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rezajafari0970/Config_manager/internal/probe"
)

// Build a real schema11 fixture, not merely a forged version marker on schema12.
func dropSplashForMigration(t *testing.T, s *Store) {
	t.Helper()
	dropAdsForMigration(t, s)
	rows, e := s.db.Query("SELECT name FROM sqlite_master WHERE type='view' AND name GLOB 'splash_smart_*'")
	if e != nil {
		t.Fatal(e)
	}
	var names []string
	for rows.Next() {
		var n string
		rows.Scan(&n)
		names = append(names, n)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		t.Fatal(e)
	}
	for _, n := range names {
		if _, e = s.db.Exec("DROP VIEW " + n); e != nil {
			t.Fatal(e)
		}
	}
	for _, n := range []string{"cache_members", "members", "rotation_queue", "cache", "usage", "source_usage", "worker", "settings", "rotation_state", "heads"} {
		if _, e = s.db.Exec("DROP TABLE splash_smart_" + n); e != nil {
			t.Fatal(e)
		}
	}
	if _, e = s.db.Exec("PRAGMA user_version=11"); e != nil {
		t.Fatal(e)
	}
	if e = s.migrateV11(); e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec("DROP INDEX sources_splash;ALTER TABLE sources DROP COLUMN splash;ALTER TABLE smart_heads DROP COLUMN inventory"); e != nil {
		t.Fatal(e)
	}
}
func markSplash(t *testing.T, s *Store, id string, value bool) {
	t.Helper()
	v, e := s.Get(context.Background(), id)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.SetSplash(context.Background(), id, v.Revision, value); e != nil {
		t.Fatal(e)
	}
}
func finishPipeline(t *testing.T, p *SmartPipeline) {
	t.Helper()
	for n := 0; n < 100; n++ {
		j, e := p.ClaimSmart(context.Background())
		if e != nil {
			t.Fatal(e)
		}
		if j == nil {
			return
		}
		if j.Issue != "" {
			t.Fatal(j.Issue)
		}
		if e = p.CompleteSmart(context.Background(), smartResult(j)); e != nil {
			t.Fatal(e)
		}
	}
	t.Fatal("pipeline did not quiesce")
}
func pipelinePool(t *testing.T, p *SmartPipeline) []string {
	t.Helper()
	m, e := smartInventory(context.Background(), p.db)
	if e != nil {
		t.Fatal(e)
	}
	var ids []string
	for _, v := range m {
		ids = append(ids, v.ID)
	}
	return ids
}
func TestSplashIndependentSettingsCacheRotationAndHTTP(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	a, _ := newExportSource(t, s, "Ordinary", sourceURI(1), sourceURI(2), sourceURI(3))
	b, _ := newExportSource(t, s, "Splash", sourceURI(11), sourceURI(12))
	markSplash(t, s, b, true)
	before, _ := s.Conversions(ctx, 0, true)
	normal, sp := s.Smart(), s.SplashSmart()
	v, _ := sp.SmartSettings(ctx)
	v.Count = 1
	if e := sp.SaveSmartSettings(ctx, v); e != nil {
		t.Fatal(e)
	}
	finishPipeline(t, normal)
	finishPipeline(t, sp)
	nr, e := querySmart(ctx, normal.db, true, 0)
	if e != nil || len(nr) != 3 {
		t.Fatal(e, nr)
	}
	sr, e := querySmart(ctx, sp.db, true, 0)
	if e != nil || len(sr) != 2 {
		t.Fatal(e, sr)
	}
	for _, r := range nr {
		if !r.Available || r.Count != 3 {
			t.Fatal(r)
		}
	}
	for _, r := range sr {
		if !r.Available || r.Count != 1 {
			t.Fatal(r)
		}
	}
	for _, p := range []*SmartPipeline{normal, sp} {
		var bad int
		want := a
		if p.splash {
			want = b
		}
		if e = p.db.QueryRowContext(ctx, "SELECT count(*) FROM smart_cache_members WHERE source!=?", want).Scan(&bad); e != nil || bad != 0 {
			t.Fatal("cross cohort member", bad, e)
		}
	}
	// Independent durable rotations and status, not just different API labels.
	for i := 0; i < 3; i++ {
		if v, _, _, e := sp.smartNext(ctx); e != nil || v == nil {
			t.Fatal(e)
		}
	}
	var cycle int
	s.db.QueryRow("SELECT cycle FROM smart_rotation_state").Scan(&cycle)
	if cycle != 0 {
		t.Fatal("normal round consumed")
	}
	if e = sp.SetSmartWorker(ctx, probe.WorkerStatus{Status: "running"}); e != nil {
		t.Fatal(e)
	}
	w, _ := normal.SmartWorker(ctx)
	if w.Status != "offline" {
		t.Fatal("worker shared")
	}
	app, e := NewServer(s, "http://127.0.0.1:18191")
	if e != nil {
		t.Fatal(e)
	}
	h := app.Handler()
	for _, p := range []*SmartPipeline{normal, sp} {
		for _, single := range []bool{false, true} {
			r := exportRequest(t, h, "GET", app.pipelineURL(p, single))
			if r.Code != 200 {
				t.Fatal(r.Code, r.Body.String())
			}
			for _, line := range exportObjects(t, r.Body.Bytes()) {
				var j map[string]any
				if json.Unmarshal(line, &j) != nil {
					t.Fatal("bad output")
				}
				body := string(line)
				if p.splash {
					if strings.Contains(body, "192.0.2.1\"") || strings.Contains(body, "192.0.2.2\"") {
						t.Fatal("normal in splash")
					}
				} else if strings.Contains(body, "192.0.2.11") || strings.Contains(body, "192.0.2.12") {
					t.Fatal("splash in normal")
				}
			}
		}
	}
	wrong := strings.Replace(app.smartURL(false), "/smart-subscription/", "/smart-splash-subscription/", 1)
	if r := exportRequest(t, h, "GET", wrong); r.Code != 404 {
		t.Fatal("cross-scope token accepted")
	}
	for _, path := range []string{"/smart-splash/claim", "/smart-splash/complete", "/smart-splash/status"} {
		req := httptest.NewRequest("POST", app.origin+path, strings.NewReader("{}"))
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code == 200 {
			t.Fatal("private route exposed")
		}
	}
	if r := exportRequest(t, h, "GET", app.origin+"/smart-splash/json/"+nr[0].ID); r.Code != 404 {
		t.Fatal("normal canonical accessible via splash")
	}
	savedNormal, _ := normal.SmartSettings(ctx)
	v, _ = sp.SmartSettings(ctx)
	v.Enabled = false
	sp.SaveSmartSettings(ctx, v)
	if b, e := normal.smartBytes(ctx, nr[0]); e != nil || len(b) == 0 {
		t.Fatal("splash settings revoked normal", e)
	}
	if _, e = sp.smartBytes(ctx, sr[0]); e == nil {
		t.Fatal("disabled splash available")
	}
	gotNormal, _ := normal.SmartSettings(ctx)
	if gotNormal != savedNormal {
		t.Fatal("settings coupled")
	}
	after, _ := s.Conversions(ctx, 0, true)
	if len(after) != len(before) {
		t.Fatal("ordinary output changed")
	}
	path := filepath.Join(filepath.Dir(s.vault.Directory()), "panel.db")
	other, e := Open(path, s.key)
	if e != nil {
		t.Fatal(e)
	}
	defer other.Close()
	sv, _ := other.Get(ctx, b)
	if !sv.Splash {
		t.Fatal("flag lost on restart")
	}
	ov, _ := other.SplashSmart().SmartSettings(ctx)
	if ov.Enabled || ov.Count != 1 {
		t.Fatal("splash settings lost")
	}
}
func TestSplashDuplicatePriorityAndInFlightRevocation(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	a, _ := newExportSource(t, s, "Ordinary", sourceURI(1), sourceURI(2))
	b, _ := newExportSource(t, s, "DuplicateSplash", sourceURI(1))
	finishSmart(t, s)
	old, _ := querySmart(ctx, s.db, true, 0)
	markSplash(t, s, b, true)
	if len(pipelinePool(t, s.Smart())) != 1 || len(pipelinePool(t, s.SplashSmart())) != 1 {
		t.Fatal("duplicate classification failed")
	}
	for _, v := range old {
		if _, e := s.smartBytes(ctx, v); e == nil {
			t.Fatal("old cache leaked duplicate Splash")
		}
	}
	finishSmart(t, s)
	j, e := s.SplashSmart().ClaimSmart(ctx)
	if e != nil || j == nil {
		t.Fatal(e)
	}
	if e = s.CompleteSmart(ctx, smartResult(j)); e == nil {
		t.Fatal("cross-scope completion accepted")
	}
	markSplash(t, s, b, false)
	if e = s.SplashSmart().RenewSmart(ctx, *j); e == nil {
		t.Fatal("changed category renewed")
	}
	if e = s.SplashSmart().CompleteSmart(ctx, smartResult(j)); e == nil {
		t.Fatal("changed category completed")
	}
	if len(pipelinePool(t, s.SplashSmart())) != 0 {
		t.Fatal("ordinary input remained in splash")
	}
	// The legacy edit API omits Splash: it must preserve a classified source.
	markSplash(t, s, a, true)
	v, _ := s.Get(ctx, a)
	if _, e = s.Save(ctx, a, v.Revision, v.SourceInput); e != nil {
		t.Fatal(e)
	}
	v, _ = s.Get(ctx, a)
	if !v.Splash {
		t.Fatal("legacy edit cleared flag")
	}
	if e = s.SetSplash(ctx, a, v.Revision-1, false); e == nil {
		t.Fatal("stale toggle accepted")
	}
}
func TestSplashPublishedProofRevocationAndGarbageCollection(t *testing.T) {
	for _, action := range []string{"disable", "delete", "proof", "expiry", "toggle"} {
		t.Run(action, func(t *testing.T) {
			s := testStore(t)
			ctx := context.Background()
			id, _ := newExportSource(t, s, "Splash", sourceURI(1))
			markSplash(t, s, id, true)
			p := s.SplashSmart()
			finishPipeline(t, p)
			rows, _ := querySmart(ctx, p.db, true, 0)
			v := rows[0]
			if used, e := s.Referenced(ctx, v.Hash); e != nil || !used {
				t.Fatal("splash blob not pinned", e)
			}
			switch action {
			case "disable":
				s.db.Exec("UPDATE sources SET enabled=0 WHERE id=?", id)
			case "delete":
				s.db.Exec("DELETE FROM sources WHERE id=?", id)
			case "proof":
				s.db.Exec("UPDATE conversion_results SET connection='failed'")
			case "expiry":
				s.db.Exec("UPDATE artifacts SET expires=1")
			case "toggle":
				markSplash(t, s, id, false)
			}
			if _, e := p.smartBytes(ctx, v); e == nil {
				t.Fatal("revoked splash available")
			}
		})
	}
}
func TestSmartBlockedBootstrapRetriesOnPoolChange(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "Domain", strings.Replace(sourceURI(1), "192.0.2.1", "server.example", 1))
	j, e := s.ClaimSmart(ctx)
	if e != nil || j == nil || j.Issue != "smart_bootstrap_no_ip" {
		t.Fatal("expected domain-only failure", e, j)
	}
	r := smartResult(j)
	r.Core = "blocked"
	r.Connection = "not_run"
	r.Candidate = nil
	r.Code = j.Issue
	if e = s.CompleteSmart(ctx, r); e != nil {
		t.Fatal(e)
	}
	if next, e := s.ClaimSmart(ctx); e != nil || next != nil {
		t.Fatal("unchanged pool ignored backoff", e)
	}
	newExportSource(t, s, "IndependentIP", sourceURI(2))
	finishSmart(t, s)
	rows, e := querySmart(ctx, s.db, true, 0)
	if e != nil || len(rows) != 2 || !rows[0].Available || !rows[1].Available {
		t.Fatal("improved pool not retried immediately", e, rows)
	}
}
func TestSplashMigrationPreservesNormalDataAndRounds(t *testing.T) {
	s, _ := smartFixture(t)
	finishSmart(t, s)
	ctx := context.Background()
	s.smartNext(ctx)
	before, _ := querySmart(ctx, s.db, true, 0)
	settings, _ := s.SmartSettings(ctx)
	var cycle, remaining int
	s.db.QueryRow("SELECT cycle,(SELECT count(*) FROM smart_rotation_queue) FROM smart_rotation_state").Scan(&cycle, &remaining)
	dropSplashForMigration(t, s)
	if e := s.migrateV12(); e != nil {
		t.Fatal(e)
	}
	after, _ := querySmart(ctx, s.db, true, 0)
	if len(after) != len(before) {
		t.Fatal("normal cache lost")
	}
	for i, v := range before {
		if v.Hash != after[i].Hash || !after[i].Available {
			t.Fatal("normal cache rewritten")
		}
	}
	got, _ := s.SmartSettings(ctx)
	if got != settings {
		t.Fatal("normal settings reset")
	}
	var c, n int
	s.db.QueryRow("SELECT cycle,(SELECT count(*) FROM smart_rotation_queue) FROM smart_rotation_state").Scan(&c, &n)
	if c != cycle || n != remaining {
		t.Fatal("normal round reset")
	}
	if len(pipelinePool(t, s.SplashSmart())) != 0 {
		t.Fatal("ordinary data imported into splash")
	}
	if e := s.migrateV12(); e != nil {
		t.Fatal("repeated migration", e)
	}
}

func TestSplashCandidateAliasCannotCrossPools(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "NormalAlias", sourceURI(1))
	id, _ := newExportSource(t, s, "SplashAlias", sourceURI(2))
	markSplash(t, s, id, true)
	var hash string
	if e := s.db.QueryRow("SELECT r.candidate_hash FROM memberships m JOIN conversion_results r ON r.hash=m.raw_hash WHERE m.source_id=?", id).Scan(&hash); e != nil {
		t.Fatal(e)
	}
	// Different canonical identities can still produce identical converted bytes.
	if _, e := s.db.Exec("UPDATE conversion_results SET candidate_hash=?", hash); e != nil {
		t.Fatal(e)
	}
	if len(pipelinePool(t, s.Smart())) != 0 || len(pipelinePool(t, s.SplashSmart())) != 1 {
		t.Fatal("candidate alias escaped Splash reservation")
	}
	finishPipeline(t, s.SplashSmart())
	if j, e := s.ClaimSmart(ctx); e != nil || j != nil {
		t.Fatal("ordinary alias generated a job", e)
	}
}
func TestSplashAndNormalClaimConcurrentlyWithIndependentSlots(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "NormalConcurrent", sourceURI(1), sourceURI(2), sourceURI(3))
	id, _ := newExportSource(t, s, "SplashConcurrent", sourceURI(11), sourceURI(12), sourceURI(13))
	markSplash(t, s, id, true)
	type result struct {
		p *SmartPipeline
		j *probe.Job
		e error
	}
	ch := make(chan result, 6)
	for _, p := range []*SmartPipeline{s.Smart(), s.SplashSmart()} {
		for i := 0; i < 3; i++ {
			go func(p *SmartPipeline) { j, e := p.ClaimSmart(ctx); ch <- result{p, j, e} }(p)
		}
	}
	jobs := []result{}
	counts := map[bool]int{}
	for i := 0; i < 6; i++ {
		r := <-ch
		if r.e != nil {
			t.Fatal(r.e)
		}
		if r.j != nil {
			counts[r.p.splash]++
			jobs = append(jobs, r)
		}
	}
	if counts[false] != 2 || counts[true] != 2 {
		t.Fatal("shared or unbounded slots", counts)
	}
	v, _ := s.SmartSettings(ctx)
	v.Count = 1
	if e := s.SaveSmartSettings(ctx, v); e != nil {
		t.Fatal(e)
	}
	for _, r := range jobs {
		e := r.p.CompleteSmart(ctx, smartResult(r.j))
		if r.p.splash && e != nil {
			t.Fatal("ordinary setting revoked Splash", e)
		}
		if !r.p.splash && e == nil {
			t.Fatal("stale normal claim accepted")
		}
	}
}
