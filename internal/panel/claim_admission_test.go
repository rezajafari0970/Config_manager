package panel

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestClaimAdmissionKeepsSingleExportAndWriterAvailable(t *testing.T) {
	s := testStore(t)
	seedRotation(t, s, 61, 2)
	ctx := context.Background()
	// Prepare a real output round before simulating one slow background claim.
	if next, err := s.prepareRotation(ctx); err != nil || next == nil {
		t.Fatal(next, err)
	}
	release, err := s.admitClaim(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	claims := map[string]func(context.Context) error{
		"ordinary": func(ctx context.Context) error { _, err := s.ClaimProbe(ctx); return err },
		"initial":  func(ctx context.Context) error { _, err := s.ClaimEgressLane(ctx, "initial"); return err },
		"followup": func(ctx context.Context) error { _, err := s.ClaimEgressLane(ctx, "followup"); return err },
		"smart":    func(ctx context.Context) error { _, err := s.Smart().ClaimSmart(ctx); return err },
		"splash":   func(ctx context.Context) error { _, err := s.SplashSmart().ClaimSmart(ctx); return err },
		"ads":      func(ctx context.Context) error { _, err := s.AdsSmart().ClaimSmart(ctx); return err },
	}
	before := s.db.Stats()
	for name, claim := range claims {
		t.Run(name, func(t *testing.T) {
			limited, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
			defer cancel()
			if err := claim(limited); !errors.Is(err, context.DeadlineExceeded) {
				t.Fatal("claim bypassed shared admission", err)
			}
		})
	}
	if after := s.db.Stats(); after.WaitCount != before.WaitCount || after.InUse != 0 {
		t.Fatal("waiting claims reached writer", before, after)
	}
	limited, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := s.Ping(limited); err != nil {
		t.Fatal("database watchdog blocked behind admission", err)
	}
	if next, err := s.NextConversion(limited); err != nil || next == nil || len(next.Bytes) == 0 {
		t.Fatal("single export blocked behind admission", next, err)
	}
	release()
	// Cancellation must not leak admission or invalidate normal lease recovery.
	if _, err := s.ClaimProbe(limited); err != nil {
		t.Fatal("ordinary claim did not recover", err)
	}
	if _, err := s.ClaimEgressLane(limited, "initial"); err != nil {
		t.Fatal("egress claim did not recover", err)
	}
}
