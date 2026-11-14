package panel

import (
	"context"
	"github.com/rezajafari0970/Config_manager/internal/collect"
	"github.com/rezajafari0970/Config_manager/internal/egress"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEgressHealthScheduleRetentionAndRouteLoss(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "lifecycle", sourceURI(230))
	j, e := s.ClaimEgress(ctx)
	if e != nil || j == nil {
		t.Fatal(e)
	}
	r := syntheticEgress(*j)
	if e = s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: r}); e != nil {
		t.Fatal(e)
	}
	var tested, next, until int64
	s.db.QueryRow("SELECT tested,next_run,valid_until FROM egress_jobs WHERE hash=?", j.Hash).Scan(&tested, &next, &until)
	if next-tested != 540000 || until > tested+600000 || until < next+59000 {
		t.Fatal(tested, next, until)
	}
	app, e := NewServer(s, "http://127.0.0.1:18191")
	if e != nil {
		t.Fatal(e)
	}
	m := egressModule{app}
	for _, x := range []struct {
		body   string
		status int
	}{{`{"revision":1,"health_minutes":5}`, 200}, {`{"revision":1,"health_minutes":20}`, 409}, {`{"revision":2,"health_minutes":0}`, 422}, {`{"revision":2,"health_minutes":61}`, 422}} {
		rr := httptest.NewRecorder()
		m.lifecycle(rr, httptest.NewRequest("POST", "/api/egress/lifecycle", strings.NewReader(x.body)))
		if rr.Code != x.status {
			t.Fatal(rr.Code, rr.Body.String())
		}
	}
	s.db.QueryRow("SELECT next_run FROM egress_jobs WHERE hash=?", j.Hash).Scan(&next)
	if next != tested+300000 {
		t.Fatal(next, tested)
	}
	// A healthy member changing route is withdrawn, never globally erased.
	for step := 0; step < 4; step++ {
		s.db.Exec("UPDATE egress_jobs SET next_run=0,last_failure=0,first_seen=1 WHERE hash=?", j.Hash)
		j, e = s.ClaimEgress(ctx)
		if e != nil || j == nil {
			t.Fatal(e)
		}
		r = syntheticEgress(*j)
		for i := range r.Samples {
			r.Samples[i].Source = egress.Hash([]byte("same-route"))
		}
		if e = s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: r}); e != nil {
			t.Fatal(e)
		}
		tx, _ := s.db.Begin()
		if e = s.expireEgress(ctx, tx, egress.Now()); e != nil {
			t.Fatal(e)
		}
		tx.Commit()
	}
	var state string
	var blocked int
	s.db.QueryRow("SELECT state FROM egress_jobs WHERE hash=?", j.Hash).Scan(&state)
	s.db.QueryRow("SELECT count(*) FROM input_exclusions WHERE hash=?", j.Hash).Scan(&blocked)
	if state == "DELETED" || blocked != 0 {
		t.Fatal(state, blocked)
	}
}
func TestEgressHealthCannotBeInventedFromRoute(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "payload", sourceURI(231))
	j, _ := s.ClaimEgress(ctx)
	r := syntheticEgress(*j)
	r.Health = nil
	for i := range r.Samples {
		r.Samples[i].ResponseComplete = false
	}
	if e := s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: r}); e != nil {
		t.Fatal(e)
	}
	var n int
	s.db.QueryRow("SELECT count(*) FROM egress_eligible").Scan(&n)
	if n != 0 {
		t.Fatal("route alone qualified payload")
	}
}
func TestEgressRevivalResetsFirstAssessment(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "one", sourceURI(232))
	j, _ := s.ClaimEgress(ctx)
	s.db.Exec("UPDATE egress_jobs SET state='DELETED',reason='retention_expired',tested=123,attempts=5,deep_completed=1,report='{}',token='old',lease_until=9999999999999 WHERE hash=?", j.Hash)
	newExportSource(t, s, "two", sourceURI(232))
	j, e := s.ClaimEgress(ctx)
	if e != nil || j == nil || j.Attempt != 1 || j.Deep {
		t.Fatal(j, e)
	}
}

func TestEgressStagedInputsSurviveFailedSourceProcessing(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	id, e := s.Save(ctx, "", 0, testSource())
	if e != nil {
		t.Fatal(e)
	}
	result := acquisitionResult(t, s, sourceURI(233), sourceURI(234))
	job := claimOne(t, s, id)
	if n, e := s.StageBatch(ctx, job, result.Items); e != nil || n != 2 {
		t.Fatal(n, e)
	}
	if e = s.Complete(ctx, job, collect.Result{State: "error", ErrorCode: "processing_timeout", Staged: true, HTTPStatus: 200}, time.Now()); e != nil {
		t.Fatal(e)
	}
	var members, queued int
	s.db.QueryRow("SELECT count(*) FROM memberships").Scan(&members)
	s.db.QueryRow("SELECT count(*) FROM egress_jobs WHERE state!='DELETED'").Scan(&queued)
	if members != 0 || queued != 2 {
		t.Fatal(members, queued)
	}
	for range 2 {
		j, e := s.ClaimEgress(ctx)
		if e != nil || j == nil || len(j.Raw) == 0 || j.Issue != "" {
			t.Fatal(j, e)
		}
	}
}
