package panel

import (
	"context"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/egress"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestEgressCountersCommitOnceSurviveCleanupAndRestart(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "counters", sourceURI(241), sourceURI(242))
	j, _ := s.ClaimEgress(ctx)
	c := egress.Completion{Job: *j, Report: syntheticEgress(*j)}
	if err := s.CompleteEgress(ctx, c); err != nil {
		t.Fatal(err)
	}
	if err := s.CompleteEgress(ctx, c); err != ErrConflict {
		t.Fatal("duplicate receipt accepted", err)
	}
	next, _ := s.ClaimEgress(ctx)
	if _, err := s.db.Exec("UPDATE egress_jobs SET lease_until=1 WHERE hash=?", next.Hash); err != nil {
		t.Fatal(err)
	}
	recovered, err := s.ClaimEgress(ctx)
	if err != nil || recovered == nil || recovered.Hash != next.Hash || recovered.Token == next.Token {
		t.Fatal("lease not recovered", err)
	}
	if err = s.CompleteEgress(ctx, egress.Completion{Job: *recovered, Report: syntheticEgress(*recovered)}); err != nil {
		t.Fatal(err)
	}
	s.db.Exec("UPDATE egress_jobs SET next_run=0 WHERE hash=?", j.Hash)
	j, _ = s.ClaimEgress(ctx)
	if err = s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: syntheticEgress(*j)}); err != nil {
		t.Fatal(err)
	}
	m, err := egressMetricState(ctx, s.db)
	if err != nil || m.Registered != 2 || m.Completed != 3 || m.FirstAssessments != 2 || m.Definitive != 3 || m.LeaseRecoveries != 1 {
		t.Fatal(m, err)
	}
	app, _ := NewServer(s, "http://127.0.0.1:18191")
	rr := httptest.NewRecorder()
	egressModule{app}.state(rr, httptest.NewRequest("GET", "/api/egress", nil))
	var response struct {
		Throughput struct {
			Completed int `json:"completed_10m"`
			Unique    int `json:"unique_10m"`
		} `json:"throughput"`
	}
	if rr.Code != 200 || json.Unmarshal(rr.Body.Bytes(), &response) != nil || response.Throughput.Completed != 3 || response.Throughput.Unique != 2 {
		t.Fatal(rr.Code, rr.Body.String())
	}
	s.db.Exec("UPDATE egress_events SET at=1")
	tx, _ := s.db.Begin()
	if err = s.expireEgress(ctx, tx, egress.Now()); err != nil {
		t.Fatal(err)
	}
	tx.Commit()
	path := filepath.Join(filepath.Dir(s.vault.Directory()), "panel.db")
	key := s.key
	s.Close()
	reopened, err := Open(path, key)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	after, err := egressMetricState(ctx, reopened.db)
	if err != nil || m != after {
		t.Fatal("durable metrics changed", m, after, err)
	}
}

func TestEgressSnapshotFreezesViewExpiryAndCountryCounts(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "clock", sourceURI(243))
	j, _ := s.ClaimEgress(ctx)
	if err := s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: syntheticEgress(*j)}); err != nil {
		t.Fatal(err)
	}
	now := egress.Now()
	s.db.Exec("UPDATE egress_jobs SET valid_until=?", now-1)
	s.db.Exec("UPDATE egress_health SET valid_until=?", now-1)
	s.db.Exec("UPDATE egress_geo_results SET valid_until=?", now-1)
	ctx = context.WithValue(ctx, readTimeKey{}, now-1000)
	q, err := freezeEgress(ctx, s.db)
	if err != nil {
		t.Fatal(err)
	}
	info, err := countryState(ctx, q)
	if err != nil || info["known"] != 1 || info["published"] != 1 {
		t.Fatal(info, err)
	}
	var n int
	s.db.QueryRow("SELECT count(*) FROM egress_export_eligible").Scan(&n)
	if n != 0 {
		t.Fatal("fixture not expired")
	}
}

func TestEgressBackoffAndLocalPressurePreserveAcceptedInput(t *testing.T) {
	if egressRetryDelay("ab", "UNKNOWN", "unknown", false, 1, 900000, true) != 30000 {
		t.Fatal("first deep retry delayed")
	}
	previous := int64(0)
	for streak := 1; streak <= 20; streak++ {
		delay := egressRetryDelay("ab", "UNKNOWN", "unknown", false, streak, 900000, false)
		if delay < previous || delay > 3600000 {
			t.Fatal(streak, delay)
		}
		previous = delay
	}
	if egressRetryDelay("ab", "RESIDENTIAL", "healthy", true, 0, 900000, false) != 900000 || egressRetryDelay("ab", "QUARANTINED", "unhealthy", true, 99, 900000, false) != 30000 {
		t.Fatal("health schedule changed")
	}
	if egressRetryDelay("ab", "QUARANTINED", "healthy", true, 1, 480000, true) != 480000 {
		t.Fatal("healthy Ads unknown-route retry storm")
	}
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "pressure", sourceURI(244))
	j, _ := s.ClaimEgress(ctx)
	if err := s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: syntheticEgress(*j)}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		s.db.Exec("UPDATE egress_jobs SET next_run=0,last_failure=0 WHERE hash=?", j.Hash)
		j, _ = s.ClaimEgress(ctx)
		r := syntheticEgress(*j)
		r.ResourcePressure = true
		if err := s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: r}); err != nil {
			t.Fatal(err)
		}
	}
	var state string
	var failures int
	s.db.QueryRow("SELECT state,failures FROM egress_jobs WHERE hash=?", j.Hash).Scan(&state, &failures)
	if state != "QUARANTINED" || failures != 0 {
		t.Fatal("host pressure deleted config", state, failures)
	}
	tx, _ := s.db.Begin()
	if err := s.revokeEgress(ctx, tx, j.Hash, "manual_delete", egress.Now()); err != nil {
		t.Fatal(err)
	}
	tx.Commit()
	tx, _ = s.db.Begin()
	if err := s.manageEgress(ctx, tx, "enable", true, egress.Now()); err != nil {
		t.Fatal(err)
	}
	tx.Commit()
	s.db.QueryRow("SELECT state FROM egress_jobs WHERE hash=?", j.Hash).Scan(&state)
	if state != "DELETED" {
		t.Fatal("enable resurrected deleted config")
	}
}
