package panel

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func TestSingleExportSurvivesSaturatedWorkerReaders(t *testing.T) {
	s := testStore(t)
	seedRotation(t, s, 51, 2)
	ctx := context.Background()
	first, err := s.prepareRotation(ctx)
	if err != nil || first == nil {
		t.Fatal(first, err)
	}
	var held []*sql.Conn
	defer func() {
		for _, conn := range held {
			conn.Close()
		}
	}()
	for i := 0; i < s.workerReader().Stats().MaxOpenConnections; i++ {
		conn, err := s.workerReader().Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		held = append(held, conn)
	}
	if _, err = held[0].ExecContext(ctx, "UPDATE settings SET retention_seconds=retention_seconds"); err == nil {
		t.Fatal("background preflight connection can mutate data")
	}
	limited, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	got, err := s.NextConversion(limited)
	if err != nil || got == nil || got.Conversion.CandidateHash != first.Conversion.CandidateHash || len(got.Bytes) == 0 {
		t.Fatal("background reader saturation blocked valid single output", got, err)
	}
}
