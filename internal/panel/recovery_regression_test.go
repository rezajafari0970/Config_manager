package panel

import (
	"context"
	"github.com/rezajafari0970/Config_manager/internal/collect"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRejectedResponseCannotAdvanceValidators(t *testing.T) {
	for _, state := range []string{"error", "empty", "invalid"} {
		t.Run(state, func(t *testing.T) {
			s := testStore(t)
			ctx := context.Background()
			id, _ := s.Save(ctx, "", 0, testSource())
			first := fixtureResult(t, s, sourceURI(1))
			j := claimOne(t, s, id)
			if e := s.Complete(ctx, j, first, time.Now()); e != nil {
				t.Fatal(e)
			}
			j = claimOne(t, s, id)
			if j.ETag == "" {
				t.Fatal("accepted validator missing")
			}
			bad := collect.Result{State: state, ErrorCode: "body_read_failed", HTTPStatus: 200, ETag: "new-unaccepted", Modified: "new-date"}
			if e := s.Complete(ctx, j, bad, time.Now()); e != nil {
				t.Fatal(e)
			}
			var next int64
			var reason string
			if e := s.db.QueryRow("SELECT next_run,refresh_reason FROM sources WHERE id=?", id).Scan(&next, &reason); e != nil {
				t.Fatal(e)
			}
			if reason != "incomplete_response" || next > time.Now().Add(6*time.Second).UnixNano() {
				t.Fatal("recovery not scheduled")
			}
			j = claimOne(t, s, id)
			if j.ETag != "" || j.Modified != "" {
				t.Fatal("failed body's validators escaped")
			}
			rows, e := s.Outputs(ctx, "", 0, true)
			if e != nil || len(rows) != 1 {
				t.Fatal("retained generation lost", e)
			}
		})
	}
}

func TestLargerRetentionNeverExtendsExistingDeadlines(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	id, _ := s.Save(ctx, "", 0, testSource())
	if e := s.Complete(ctx, claimOne(t, s, id), fixtureResult(t, s, sourceURI(1)), time.Now()); e != nil {
		t.Fatal(e)
	}
	v, _ := s.Settings(ctx)
	old := map[string]int64{}
	for _, table := range []string{"artifacts", "responses"} {
		var expiry int64
		s.db.QueryRow("SELECT expires FROM " + table + " LIMIT 1").Scan(&expiry)
		old[table] = expiry
	}
	v.RetentionSeconds *= 2
	if e := s.SaveSettings(ctx, v); e != nil {
		t.Fatal(e)
	}
	for table, expiry := range old {
		var got int64
		s.db.QueryRow("SELECT expires FROM " + table + " LIMIT 1").Scan(&got)
		if got != expiry {
			t.Fatal("deadline extended", table)
		}
	}
	if e := s.Complete(ctx, claimOne(t, s, id), fixtureResult(t, s, sourceURI(2)), time.Now()); e != nil {
		t.Fatal(e)
	}
	var maxLife int64
	s.db.QueryRow("SELECT max(expires-created) FROM artifacts").Scan(&maxLife)
	if maxLife != v.RetentionSeconds*1000 {
		t.Fatal("new retention not applied to new input")
	}
}

func TestAbandonedClaimsCannotRevokeReplacements(t *testing.T) {
	s, _ := smartFixture(t)
	ctx := context.Background()
	p := s.Smart()
	j, e := p.ClaimSmart(ctx)
	if e != nil || j == nil {
		t.Fatal(e)
	}
	if e = p.abortSmart(ctx, j.Hash, j.Token, j.Revision); e != nil {
		t.Fatal(e)
	}
	var token string
	s.db.QueryRow("SELECT token FROM smart_heads WHERE id=?", j.Hash).Scan(&token)
	if token != "" {
		t.Fatal("abandoned lease retained")
	}
	s.db.Exec("UPDATE smart_heads SET token='replacement',state='running' WHERE id=?", j.Hash)
	p.abortSmart(ctx, j.Hash, j.Token, j.Revision)
	s.db.QueryRow("SELECT token FROM smart_heads WHERE id=?", j.Hash).Scan(&token)
	if token != "replacement" {
		t.Fatal("new lease revoked")
	}
	id, _ := s.Save(ctx, "", 0, testSource())
	cj := claimOne(t, s, id)
	if e = s.Abandon(ctx, cj); e != nil {
		t.Fatal(e)
	}
	replacement := claimOne(t, s, id)
	s.Abandon(ctx, cj)
	s.db.QueryRow("SELECT lease_token FROM sources WHERE id=?", id).Scan(&token)
	if token != replacement.Token {
		t.Fatal("new collection lease revoked")
	}
}

func TestSmartUnreadableCacheCleanupFailureIsBounded(t *testing.T) {
	s := testStore(t)
	newExportSource(t, s, "One", sourceURI(1))
	finishSmart(t, s)
	ctx := context.Background()
	rows, _ := querySmart(ctx, s.db, true, 0)
	v := rows[0]
	if e := os.WriteFile(filepath.Join(s.vault.Directory(), v.Hash+".blob"), []byte("corrupt"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := s.db.Exec("CREATE TRIGGER synthetic_no_cleanup BEFORE DELETE ON smart_cache BEGIN SELECT RAISE(FAIL,'synthetic_unavailable'); END"); e != nil {
		t.Fatal(e)
	}
	bounded, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	_, _, _, e := s.smartNext(bounded)
	if e == nil || bounded.Err() != nil || !strings.Contains(e.Error(), "synthetic_unavailable") {
		t.Fatal("spin or hidden cleanup failure", e)
	}
	var count int
	s.db.QueryRow("SELECT count(*) FROM smart_rotation_queue").Scan(&count)
	if count != 1 {
		t.Fatal("failed delivery consumed turn")
	}
	s.db.Exec("DROP TRIGGER synthetic_no_cleanup")
	// A subsequent request has its own deadline. Intervening SQLite assertions
	// must not consume the timeout used to prove that the rotation lock released.
	nextRequest, nextCancel := context.WithTimeout(ctx, time.Second)
	defer nextCancel()
	if _, _, _, e = s.smartNext(nextRequest); e != nil {
		t.Fatal("rotation lock was not released", e)
	}
}

func TestFullSmartExportRevalidatesSpoolProvenance(t *testing.T) {
	s, _ := smartFixture(t)
	finishSmart(t, s)
	ctx := context.Background()
	p := s.Smart()
	rows, _ := querySmart(ctx, s.db, true, 0)
	if _, e := p.smartBytes(ctx, rows[0]); e != nil {
		t.Fatal(e)
	} // An entry has already been spooled.
	if _, e := s.db.Exec("UPDATE smart_cache SET expires=? WHERE id=?", time.Now().Add(-time.Second).UnixMilli(), rows[0].ID); e != nil {
		t.Fatal(e)
	}
	if valid, e := p.validateSmartExport(ctx, rows); e != nil || valid {
		t.Fatal("expired spool authorized", valid, e)
	}
	app, _ := NewServer(s, "http://127.0.0.1:18191")
	s.db.Exec("UPDATE sources SET enabled=0")
	r := exportRequest(t, app.Handler(), "GET", app.smartURL(false))
	if r.Code != 503 || r.Header().Get("X-Config-Count") != "0" {
		t.Fatal("empty full export reported success", r.Code)
	}
}
