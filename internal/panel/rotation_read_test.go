package panel

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRotationExistingRoundReadsWithoutWriterAndStillRechecks(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	source, _ := newExportSource(t, s, "read-round", sourceURI(41), sourceURI(42))
	for _, scope := range []string{"", source} {
		first, err := s.prepareScopedRotation(ctx, scope)
		if err != nil || first == nil {
			t.Fatal(first, err)
		}
		writer, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = writer.ExecContext(ctx, "UPDATE settings SET retention_seconds=retention_seconds"); err != nil {
			writer.Rollback()
			t.Fatal(err)
		}
		readCtx, cancel := context.WithTimeout(ctx, time.Second)
		peek, readErr := s.prepareScopedRotation(readCtx, scope)
		cancel()
		claimCtx, cancel := context.WithTimeout(ctx, 25*time.Millisecond)
		claimed, claimErr := s.claimRotation(claimCtx, first)
		cancel()
		writer.Rollback()
		if readErr != nil || peek == nil || peek.Conversion != first.Conversion || peek.Cycle != first.Cycle {
			t.Fatal("existing round waited for writer or changed member", peek, readErr)
		}
		if claimed || !errors.Is(claimErr, context.DeadlineExceeded) {
			t.Fatal("read proposal consumed a turn without writer recheck", claimed, claimErr)
		}
		if claimed, err = s.claimRotation(ctx, peek); err != nil || !claimed {
			t.Fatal("unconsumed proposal did not recover", claimed, err)
		}
	}
	peek, err := s.prepareScopedRotation(ctx, source)
	if err != nil || peek == nil {
		t.Fatal(peek, err)
	}
	if _, err = s.ManageConfigs(ctx, ConfigBatch{Action: "disable", IDs: []string{peek.Conversion.ID}}); err != nil {
		t.Fatal(err)
	}
	if claimed, err := s.claimRotation(ctx, peek); err != nil || claimed {
		t.Fatal("WAL proposal bypassed later revocation", claimed, err)
	}
	if next, err := s.nextSourceConversion(ctx, source); err != nil || next == nil || next.Conversion.ID == peek.Conversion.ID {
		t.Fatal("invalid first queue member blocked the round", next, err)
	}
}
