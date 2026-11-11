package panel

import (
	"context"
	"testing"
	"time"
)

func TestResidentialFirstAssessmentPrecedesUnqualifiedRetry(t *testing.T) {
	s := testStore(t)
	resPolicy(t, s)
	ctx := context.Background()
	_, old := addConversion(t, s, conversionSample)
	// A source with an old retry must not monopolize workers while another
	// source's never-assessed inputs survive normal TTL in the independent queue.
	_, fresh := addConversion(t, s, `{"outbounds":[{"protocol":"trojan","tag":"proxy","settings":{"servers":[{"address":"198.51.100.9","port":443,"password":"synthetic"}]},"streamSettings":{"security":"tls","tlsSettings":{"serverName":"example.invalid"}}}]}`)
	_, e := s.db.Exec("UPDATE residential_records SET tested=?,next_run=1 WHERE hash=?", time.Now().UnixMilli(), old.Hash)
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.db.Exec("UPDATE residential_source_fairness SET last_claim=0")
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.db.Exec("INSERT INTO residential_source_fairness SELECT source,? FROM residential_owners WHERE hash=? ON CONFLICT(source) DO UPDATE SET last_claim=excluded.last_claim", time.Now().UnixMilli(), fresh.Hash)
	if e != nil {
		t.Fatal(e)
	}
	j, e := s.ClaimResidential(ctx)
	if e != nil || j == nil || j.Hash != fresh.Hash {
		t.Fatal("unqualified retry starved first assessment", e, j)
	}
}
