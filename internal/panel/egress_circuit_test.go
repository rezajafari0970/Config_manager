package panel

import (
	"context"
	"github.com/rezajafari0970/Config_manager/internal/egress"
	"testing"
)

func TestEgressGenericOutageCannotOpenCircuitOrDeleteAds(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "guard", sourceURI(235), sourceURI(236), sourceURI(237))
	hashes := []string{}
	for range 3 {
		j, e := s.ClaimEgress(ctx)
		if e != nil || j == nil {
			t.Fatal(e)
		}
		if e = s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: syntheticEgress(*j)}); e != nil {
			t.Fatal(e)
		}
		hashes = append(hashes, j.Hash)
	}
	fail := func(hash string) {
		t.Helper()
		// Deterministically exercise spacing without making the suite sleep.
		s.db.Exec("UPDATE egress_jobs SET next_run=0,last_failure=0 WHERE hash=?", hash)
		j, e := s.ClaimEgress(ctx)
		if e != nil || j == nil || j.Hash != hash {
			t.Fatal(j, e)
		}
		r := syntheticEgress(*j)
		host := r.Health[0]
		host.HostControl = true
		host.Variant = "host"
		host.Providers = append([]egress.HealthProvider(nil), host.Providers...)
		for i := range r.Health {
			for k := range r.Health[i].Providers {
				r.Health[i].Providers[k].Download.CurlExit = 28
				r.Health[i].Providers[k].Upload.CurlExit = 28
			}
		}
		r.Health = append(r.Health, host)
		if e = s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: r}); e != nil {
			t.Fatal(e)
		}
	}
	for _, h := range hashes {
		fail(h)
	}
	var opened int64
	s.db.QueryRow("SELECT circuit_until FROM egress_settings").Scan(&opened)
	if opened != 0 {
		t.Fatal("generic outage polluted Ads circuit")
	}
	// Additional simultaneous failures must not restart the five-minute window.
	for _, h := range hashes {
		fail(h)
	}
	var again int64
	s.db.QueryRow("SELECT circuit_until FROM egress_settings").Scan(&again)
	if again != opened {
		t.Fatal("guard renewed indefinitely", opened, again)
	}
	// Simulate the end of that window. The cooldown allows confirmed revocation.
	s.db.Exec("UPDATE egress_settings SET circuit_until=?", egress.Now()-1)
	for _, h := range hashes {
		fail(h)
	}
	var deleted int
	s.db.QueryRow("SELECT count(*) FROM egress_jobs WHERE state='DELETED' AND reason='confirmed_unhealthy'").Scan(&deleted)
	if deleted != 0 {
		t.Fatal("generic failures deleted valid Ads", deleted)
	}
}
