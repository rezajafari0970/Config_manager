package panel

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/rezajafari0970/Config_manager/internal/egress"
)

func TestEgressAuditQueueUsesOneDueDefinition(t *testing.T) {
	s := testStore(t)
	newExportSource(t, s, "urgent-summary", sourceURI(761))
	j, err := s.ClaimEgress(context.Background())
	if err != nil || j == nil {
		t.Fatal(err)
	}
	now := egress.Now()
	if _, err = s.db.Exec("UPDATE egress_jobs SET token='',lease_until=0,ever_accepted=1,tested=?,valid_until=?,next_run=? WHERE hash=?", now, now+60000, now+180000, j.Hash); err != nil {
		t.Fatal(err)
	}
	app, err := NewServer(s, "http://127.0.0.1:18191")
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	egressModule{app}.state(rr, httptest.NewRequest("GET", "/api/egress?brief=1", nil))
	var body struct {
		Queue      map[string]int
		Accounting struct{ Queue map[string]int }
	}
	if rr.Code != 200 || json.Unmarshal(rr.Body.Bytes(), &body) != nil {
		t.Fatal(rr.Code, rr.Body.String())
	}
	if body.Queue["due"] != 1 || body.Queue["due"] != body.Accounting.Queue["due"] {
		t.Fatal(body)
	}
}

func TestEgressAuditTransientDNSRecoversAfterDeepAttempt(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "dns-recovery", sourceURI(762))
	j, err := s.ClaimEgress(ctx)
	if err != nil || j == nil {
		t.Fatal(err)
	}
	// A transient resolver fault can outlive both initial and deep budgets.
	if _, err = s.db.Exec("UPDATE egress_jobs SET state='UNKNOWN',reason='endpoint_dns_failed',tested=?,first_completed=1,deep_completed=1,retry_streak=3,token='',lease_until=0,next_run=0 WHERE hash=?", egress.Now(), j.Hash); err != nil {
		t.Fatal(err)
	}
	retry, err := s.ClaimEgressLane(ctx, "followup")
	if err != nil || retry == nil || retry.Hash != j.Hash || !retry.Deep {
		t.Fatal("temporary infrastructure fault abandoned", retry, err)
	}
	if err = s.CompleteEgress(ctx, egress.Completion{Job: *retry, Report: syntheticEgress(*retry)}); err != nil {
		t.Fatal(err)
	}
	var n int
	if err = s.db.QueryRow("SELECT count(*) FROM egress_eligible").Scan(&n); err != nil || n != 1 {
		t.Fatal("recovery did not require new eligible proof", n, err)
	}
}

func TestEgressAuditSnapshotScopesArchiveWithoutChangingGates(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "scope", sourceURI(763), sourceURI(764), sourceURI(765))
	for range 3 {
		j, err := s.ClaimEgress(ctx)
		if err != nil || j == nil {
			t.Fatal(err)
		}
		if err = s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: syntheticEgress(*j)}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.db.Exec(`WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i+1 FROM n WHERE i<3000)
 INSERT INTO egress_jobs(hash,kind,size,first_seen,last_seen) SELECT printf('%064x',500000+i),'audit-archive',1,1,1 FROM n;
 INSERT INTO egress_unique SELECT hash,hash,1 FROM egress_jobs WHERE kind='audit-archive';
 DELETE FROM memberships WHERE raw_hash=(SELECT hash FROM egress_jobs WHERE ever_accepted=1 ORDER BY hash LIMIT 1);`); err != nil {
		t.Fatal(err)
	}
	read := func(q databaseQuery, table string) []string {
		t.Helper()
		rows, err := q.QueryContext(ctx, "SELECT hash FROM "+table+" ORDER BY hash")
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		out := []string{}
		for rows.Next() {
			var h string
			if err = rows.Scan(&h); err != nil {
				t.Fatal(err)
			}
			out = append(out, h)
		}
		if err = rows.Err(); err != nil {
			t.Fatal(err)
		}
		return out
	}
	for _, mutation := range []string{"", "UPDATE egress_jobs SET valid_until=1 WHERE ever_accepted=1", "INSERT INTO egress_iran_guard(identity,state,first_seen) SELECT identity,'QUARANTINED',1 FROM egress_unique ORDER BY identity LIMIT 1"} {
		if mutation != "" {
			if _, err := s.db.Exec(mutation); err != nil {
				t.Fatal(err)
			}
		}
		q, err := freezeEgress(ctx, s.db)
		if err != nil {
			t.Fatal(err)
		}
		for _, table := range []string{"egress_inventory", "egress_work", "egress_eligible", "egress_export_eligible"} {
			if !reflect.DeepEqual(read(s.db, table), read(q, table)) {
				t.Fatal("scope/gate changed", table)
			}
		}
	}
	q, err := freezeEgress(ctx, s.db)
	if err != nil {
		t.Fatal(err)
	}
	r, ok := q.(egressSnapshotReader)
	if !ok {
		t.Fatal("snapshot reader")
	}
	rows, err := s.db.Query("EXPLAIN QUERY PLAN " + r.prefix + "SELECT count(*) FROM egress_inventory")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var a, b, c int
		var detail string
		if err = rows.Scan(&a, &b, &c, &detail); err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(detail, "SCAN j ") {
			t.Fatal("snapshot still scans archived jobs", detail)
		}
	}
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
}
