package panel

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/rezajafari0970/Config_manager/internal/collect"
)

func TestStageBatchAtomicDedupFenceAndFailureRetainsInventory(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	id, _ := s.Save(ctx, "", 0, testSource())
	old := acquisitionResult(t, s, "vless://old@vpn.example:443")
	if err := s.Complete(ctx, claimOne(t, s, id), old, time.Now()); err != nil {
		t.Fatal(err)
	}
	result := acquisitionResult(t, s, "vless://one@vpn.example:443", "vless://two@vpn.example:443")
	j := claimOne(t, s, id)
	n, err := s.StageBatch(ctx, j, append(result.Items, result.Items...))
	if err != nil || n != 2 {
		t.Fatal(n, err)
	}
	n, err = s.StageBatch(ctx, j, result.Items)
	if err != nil || n != 0 {
		t.Fatal("cross batch dedup", n, err)
	}
	got, _ := s.Get(ctx, id)
	if got.ConfigCount != 1 {
		t.Fatal("staging published early")
	}
	// A processing failure must discard staging, not replace the old generation.
	if err = s.Complete(ctx, j, collect.Result{State: "error", ErrorCode: "processing_timeout", Staged: true, HTTPStatus: 200}, time.Now()); err != nil {
		t.Fatal(err)
	}
	got, _ = s.Get(ctx, id)
	if got.ConfigCount != 1 || got.ErrorCode != "processing_timeout" {
		t.Fatal(got)
	}
	var staged int
	s.db.QueryRow("SELECT count(*) FROM staged").Scan(&staged)
	if staged != 0 {
		t.Fatal("failed staging retained")
	}
	j = claimOne(t, s, id)
	// Failure on the second insertion must roll back the first.
	_, err = s.db.Exec("CREATE TRIGGER fail_batch BEFORE INSERT ON staged WHEN NEW.identity='" + result.Items[1].Identity + "' BEGIN SELECT RAISE(ABORT,'synthetic');END")
	if err != nil {
		t.Fatal(err)
	}
	if n, err = s.StageBatch(ctx, j, result.Items); err == nil || n != 0 {
		t.Fatal("partial transaction accepted")
	}
	s.db.QueryRow("SELECT count(*) FROM staged").Scan(&staged)
	if staged != 0 {
		t.Fatal("batch rollback lost")
	}
	s.db.Exec("DROP TRIGGER fail_batch")
	source, _ := s.Get(ctx, id)
	if _, err = s.Save(ctx, id, source.Revision, source.SourceInput); err != nil {
		t.Fatal(err)
	}
	if _, err = s.StageBatch(ctx, j, result.Items); !errors.Is(err, ErrConflict) {
		t.Fatal("stale batch survived edit", err)
	}
}

type batchAcquisitionFetch struct{ body string }

func (batchAcquisitionFetch) Name() string { return "synthetic-acquisition" }
func (f batchAcquisitionFetch) Fetch(context.Context, collect.Job) (collect.Response, error) {
	return collect.Response{Status: 200, Body: io.NopCloser(strings.NewReader(f.body))}, nil
}

type slowBatchStore struct{ *Store }

func (s slowBatchStore) StageBatch(ctx context.Context, j collect.Job, items []collect.Item) (int, error) {
	select {
	case <-time.After(175 * time.Millisecond):
	case <-ctx.Done():
		return 0, ctx.Err()
	}
	return s.Store.StageBatch(ctx, j, items)
}
func TestCollectorLargeFastResponseSurvivesSlowStorage(t *testing.T) {
	s := testStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := s.SaveSettings(ctx, collect.Settings{RetentionSeconds: 3600, Concurrency: 2, TimeoutSeconds: 1}); err != nil {
		t.Fatal(err)
	}
	id, _ := s.Save(ctx, "", 0, testSource())
	var b strings.Builder
	for i := 0; i < 186; i++ {
		fmt.Fprintf(&b, "vless://00000000-0000-4000-8000-%012d@192.0.2.1:443?security=tls\n", i)
	}
	e := collect.NewEngine(slowBatchStore{s}, s.vault, batchAcquisitionFetch{b.String()})
	done := make(chan error, 1)
	go func() { done <- e.Run(ctx) }()
	defer func() {
		cancel()
		if err := <-done; err != nil {
			t.Error(err)
		}
	}()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		source, err := s.Get(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if source.FetchCount > 0 {
			if source.State != "updated" || source.ConfigCount != 186 || source.ReceiptUnique != 186 {
				t.Fatalf("large healthy input lost: %+v", source)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("collector never completed")
}
