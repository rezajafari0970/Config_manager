package panel

import (
	"context"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/egress"
	"net/http/httptest"
	"testing"
)

func TestEgressAccountingPartitionsAndOutputUnits(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "accounting", sourceURI(251), sourceURI(252), sourceURI(253), sourceURI(254), sourceURI(255), sourceURI(256))
	hashes := []string{}
	jobs := map[string]egress.Job{}
	for range 6 {
		j, err := s.ClaimEgress(ctx)
		if err != nil || j == nil {
			t.Fatal(err)
		}
		hashes = append(hashes, j.Hash)
		jobs[j.Hash] = *j
	}
	now := egress.Now()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := s.db.Exec(q, args...); err != nil {
			t.Fatal(err)
		}
	}
	// Two raw versions can legitimately emit the same template. Keep both rows.
	for _, h := range hashes[:2] {
		j := jobs[h]
		if err := s.CompleteEgress(ctx, egress.Completion{Job: j, Report: syntheticEgress(j)}); err != nil {
			t.Fatal(err)
		}
	}
	exec("UPDATE egress_jobs SET candidate_hash=(SELECT candidate_hash FROM egress_jobs WHERE hash=?) WHERE hash=?", hashes[0], hashes[1])
	exec("UPDATE egress_jobs SET paused=1 WHERE hash=?", hashes[2])
	exec("UPDATE egress_jobs SET token='',lease_until=0,next_run=0 WHERE hash=?", hashes[3])
	exec("UPDATE egress_jobs SET token='',lease_until=0,next_run=?,tested=1 WHERE hash=?", now+60000, hashes[4])
	exec("DELETE FROM memberships WHERE raw_hash=?", hashes[4])
	// Sixth is actively leased: never count it again as due.
	frozen := context.WithValue(ctx, readTimeKey{}, now)
	q, err := freezeEgress(frozen, s.db)
	if err != nil {
		t.Fatal(err)
	}
	a, err := egressAccounting(frozen, q)
	if err != nil {
		t.Fatal(err)
	}
	queue := a["queue"].(map[string]int)
	pub := a["publication"].(map[string]int)
	if a["live_versions"] != 5 || a["current_versions"] != 5 || a["archive_versions"] != 0 || a["awaiting_first"] != 3 || a["assessed_versions"] != 2 {
		t.Fatal(a)
	}
	if queue["paused"] != 1 || queue["running"] != 1 || queue["due"] != 1 || queue["scheduled"] != 2 {
		t.Fatal(queue)
	}
	if pub["published"] != 2 || pub["distinct_templates"] != 1 || pub["identities"] != 2 || pub["awaiting_proof"] != 0 {
		t.Fatal(pub)
	}
	// Expiry is computed at the same frozen clock, even before maintenance runs.
	exec("UPDATE egress_jobs SET valid_until=? WHERE hash=?", now-1, hashes[0])
	a, err = egressAccounting(frozen, q)
	if err != nil {
		t.Fatal(err)
	}
	pub = a["publication"].(map[string]int)
	if pub["eligible"] != 1 || pub["awaiting_proof"] != 1 || pub["published"] != 1 {
		t.Fatal(pub)
	}
	exec("UPDATE egress_country_policy SET allowed='[]'")
	a, err = egressAccounting(frozen, q)
	if err != nil {
		t.Fatal(err)
	}
	pub = a["publication"].(map[string]int)
	if pub["published"] != 0 || pub["country_filtered"] != 1 || pub["retained"] != 2 {
		t.Fatal(pub)
	}
}

func TestEgressDueAcceptedStartsBeforeLastMinute(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "lead", sourceURI(257), sourceURI(258))
	j, _ := s.ClaimEgress(ctx)
	if err := s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: syntheticEgress(*j)}); err != nil {
		t.Fatal(err)
	}
	// A slow probe is deliberately due four minutes before proof expiry.
	if _, err := s.db.Exec("UPDATE egress_jobs SET next_run=0,valid_until=? WHERE hash=?", egress.Now()+240000, j.Hash); err != nil {
		t.Fatal(err)
	}
	renewal, err := s.ClaimEgressLane(ctx, "initial")
	if err != nil || renewal == nil || renewal.Hash != j.Hash {
		t.Fatal("renewal lead ignored", renewal, err)
	}
	fresh, err := s.ClaimEgressLane(ctx, "initial")
	if err != nil || fresh == nil || fresh.Hash == j.Hash {
		t.Fatal("fresh work or lease isolation lost", fresh, err)
	}
}

func TestEgressThroughputCountsDistinctConclusiveInputs(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "attempts", sourceURI(259))
	for range 3 {
		s.db.Exec("UPDATE egress_jobs SET next_run=0")
		j, _ := s.ClaimEgress(ctx)
		if err := s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: syntheticEgress(*j)}); err != nil {
			t.Fatal(err)
		}
	}
	app, _ := NewServer(s, "http://127.0.0.1:18191")
	rr := httptest.NewRecorder()
	egressModule{app}.state(rr, httptest.NewRequest("GET", "/api/egress?brief=1", nil))
	var v struct {
		Throughput map[string]int `json:"throughput"`
	}
	if rr.Code != 200 || json.Unmarshal(rr.Body.Bytes(), &v) != nil {
		t.Fatal(rr.Code, rr.Body.String())
	}
	if v.Throughput["completed_10m"] != 3 || v.Throughput["definitive_10m"] != 3 || v.Throughput["unique_definitive_10m"] != 1 {
		t.Fatal(v)
	}
}
