package panel

import (
	"context"
	"testing"
	"time"

	"github.com/rezajafari0970/Config_manager/internal/probe"
)

func TestSmartIdleAndFullPollsDoNotWaitForWriter(t *testing.T) {
	s, _ := smartFixture(t)
	ctx := context.Background()
	a, err := s.ClaimSmart(ctx)
	if err != nil || a == nil {
		t.Fatal(a, err)
	}
	b, err := s.ClaimSmart(ctx)
	if err != nil || b == nil {
		t.Fatal(b, err)
	}
	check := func(pipeline *SmartPipeline) {
		t.Helper()
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = tx.ExecContext(ctx, "UPDATE settings SET retention_seconds=retention_seconds"); err != nil {
			tx.Rollback()
			t.Fatal(err)
		}
		limited, cancel := context.WithTimeout(ctx, time.Second)
		j, err := pipeline.ClaimSmart(limited)
		cancel()
		tx.Rollback()
		if err != nil || j != nil {
			t.Fatal("idle/full poll waited for writer or claimed work", j, err)
		}
	}
	check(s.Smart())
	for _, j := range []*probe.Job{a, b} {
		if err = s.CompleteSmart(ctx, smartResult(j)); err != nil {
			t.Fatal(err)
		}
	}
	finishSmart(t, s)
	for _, pipeline := range []*SmartPipeline{s.Smart(), s.SplashSmart(), s.AdsSmart()} {
		check(pipeline)
	}
	newExportSource(t, s, "new-idle-input", sourceURI(43))
	if needed, err := s.Smart().smartClaimNeeded(ctx); err != nil || !needed {
		t.Fatal("new input hidden by idle preflight", needed, err)
	}
	j, err := s.ClaimSmart(ctx)
	if err != nil || j == nil {
		t.Fatal("new input never claimed", j, err)
	}
	if _, err = s.db.Exec("UPDATE smart_heads SET revision=revision-1 WHERE token!=''"); err != nil {
		t.Fatal(err)
	}
	if needed, err := s.Smart().smartClaimNeeded(ctx); err != nil || !needed {
		t.Fatal("invalid lease cleanup skipped", needed, err)
	}
}
