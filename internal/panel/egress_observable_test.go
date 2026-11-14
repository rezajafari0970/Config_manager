package panel

import (
	"fmt"
	"strings"
	"testing"

	"github.com/rezajafari0970/Config_manager/internal/convert"
	"github.com/rezajafari0970/Config_manager/internal/egress"
)

func TestV7UpgradePreservesUnaffectedIdentityVerdictsAndHistory(t *testing.T) {
	s := testStore(t)
	states := []struct {
		state, reason string
		accepted      int
	}{
		{"NO_SPLIT", "non_ads_uses_ads_egress", 0},
		{"UNSUPPORTED", "driver_unavailable", 0},
		{"UNKNOWN", "controls_incomplete", 0},
		{"RESIDENTIAL", "repeated_ads_exclusive_observation", 1},
		{"DELETED", "manual_delete", 0},
	}
	now := egress.Now()
	for i, x := range states {
		hash := fmt.Sprintf("%064x", i+1)
		_, err := s.db.Exec("INSERT INTO egress_jobs(hash,kind,size,first_seen,last_seen,state,reason,ever_accepted,tested,valid_until,attempts,first_completed,deep_completed,retry_streak,report,revision) VALUES(?,'fixture',1,1,1,?,?,?,?,?,3,1,1,2,?,7)", hash, x.state, x.reason, x.accepted, now, now+1000, `{"version":"ads-egress.v6","historical":true}`)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.db.Exec("INSERT INTO egress_events(hash,at,state,reason) VALUES(?,?,?,?)", hash, now, x.state, x.reason); err != nil {
			t.Fatal(err)
		}
	}
	var metrics string
	q := "SELECT json_array(since,baseline_inputs,registered,completed,first_assessments,definitive,lease_recoveries,unhealthy_deleted) FROM egress_metrics"
	if err := s.db.QueryRow(q).Scan(&metrics); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec("UPDATE egress_settings SET version=?", "ads-egress.v6|"+convert.Version); err != nil {
		t.Fatal(err)
	}
	if err := s.reconcileEgressVersion(); err != nil {
		t.Fatal(err)
	}
	for i, x := range states {
		var state, reason, report string
		var tested, revision, attempts, first, deep, retry int64
		err := s.db.QueryRow("SELECT state,reason,report,tested,revision,attempts,first_completed,deep_completed,retry_streak FROM egress_jobs WHERE hash=?", fmt.Sprintf("%064x", i+1)).Scan(&state, &reason, &report, &tested, &revision, &attempts, &first, &deep, &retry)
		if err != nil {
			t.Fatal(err)
		}
		if attempts != 3 || first != 1 || !strings.Contains(report, "historical") {
			t.Fatal("history erased", state, attempts, first, report)
		}
		if i < 2 || x.state == "DELETED" {
			if state != x.state || reason != x.reason || tested != now || revision != 7 || deep != 1 || retry != 2 {
				t.Fatal("unaffected verdict changed", state, reason, tested, revision, deep, retry)
			}
		} else if reason != "implementation_changed" || tested != 0 || revision != 8 || deep != 0 || retry != 0 {
			t.Fatal("affected receipt not fenced", state, reason, tested, revision, deep, retry)
		}
	}
	var after string
	s.db.QueryRow(q).Scan(&after)
	if after != metrics {
		t.Fatal("statistics reset", metrics, after)
	}
	var events int
	s.db.QueryRow("SELECT count(*) FROM egress_events").Scan(&events)
	if events != len(states) {
		t.Fatal("history erased", events)
	}
	if err := s.reconcileEgressVersion(); err != nil {
		t.Fatal(err)
	}
	var revision int
	s.db.QueryRow("SELECT revision FROM egress_jobs WHERE hash=?", fmt.Sprintf("%064x", 3)).Scan(&revision)
	if revision != 8 {
		t.Fatal("startup repeated invalidation", revision)
	}
}
