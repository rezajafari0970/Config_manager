package collect

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rezajafari0970/Config_manager/internal/blob"
)

type pipelineFetch struct{ body io.ReadCloser }

func (d pipelineFetch) Name() string { return "fixture" }
func (d pipelineFetch) Fetch(ctx context.Context, j Job) (Response, error) {
	return Response{Status: 200, Body: d.body}, nil
}

type pipelineRepo struct {
	Repository
	result         Result
	batches, items int
	delay          time.Duration
	cancel         context.CancelFunc
	fail           bool
	vault          *blob.Vault
}

func (r *pipelineRepo) StageBatch(ctx context.Context, j Job, items []Item) (int, error) {
	r.batches++
	if r.cancel != nil {
		r.cancel()
	}
	if r.delay > 0 {
		select {
		case <-time.After(r.delay):
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if r.fail {
		return 0, errors.New("synthetic storage failure")
	}
	if len(items) > 32 {
		panic("unbounded batch")
	}
	// Pins must survive a sweep before the transaction commits.
	if err := r.vault.Sweep(ctx, func(context.Context, string) (bool, error) { return false, nil }); err != nil {
		return 0, err
	}
	for _, item := range items {
		v, e := r.vault.Open(item.Hash)
		if e != nil {
			panic(e)
		}
		_, e = io.Copy(io.Discard, v)
		v.Close()
		if e != nil {
			panic(e)
		}
	}
	r.items += len(items)
	return len(items), nil
}
func (r *pipelineRepo) Complete(_ context.Context, _ Job, result Result, _ time.Time) error {
	r.result = result
	return nil
}
func pipelineFixture(t *testing.T, count int) (*Engine, *pipelineRepo) {
	t.Helper()
	v, e := blob.New(t.TempDir(), bytes.Repeat([]byte{8}, 32))
	if e != nil {
		t.Fatal(e)
	}
	var body strings.Builder
	for i := 0; i < count; i++ {
		fmt.Fprintf(&body, "vless://00000000-0000-4000-8000-%012d@192.0.2.1:443?security=tls#fixture\n", i)
	}
	repo := &pipelineRepo{vault: v}
	return NewEngine(repo, v, pipelineFetch{io.NopCloser(strings.NewReader(body.String()))}), repo
}
func TestAcquisitionProcessingHasIndependentBudget(t *testing.T) {
	for _, count := range []int{62, 97, 186} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			e, r := pipelineFixture(t, count)
			r.delay = 150 * time.Millisecond
			// Network completes quickly, local staging lasts longer than its fetch budget.
			e.runJobWithBudgets(context.Background(), Job{}, 100*time.Millisecond, 10*time.Second)
			if r.result.State != "updated" || r.result.Extracted != count || r.items != count || r.batches != (count+31)/32 {
				t.Fatalf("lost healthy response: %+v batches=%d items=%d", r.result, r.batches, r.items)
			}
		})
	}
}
func TestAcquisitionProcessingFailureNeverPublishesPartialBatch(t *testing.T) {
	for _, mode := range []string{"deadline", "cancel", "storage"} {
		t.Run(mode, func(t *testing.T) {
			e, r := pipelineFixture(t, 40)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			budget := time.Second
			switch mode {
			case "deadline":
				r.delay = time.Second
				budget = 10 * time.Millisecond
			case "cancel":
				r.cancel = cancel
			case "storage":
				r.fail = true
			}
			e.runJobWithBudgets(ctx, Job{}, time.Second, budget)
			if r.result.State == "updated" {
				t.Fatal("partial data published")
			}
			if mode == "deadline" && r.result.ErrorCode != "processing_timeout" {
				t.Fatal(r.result.ErrorCode)
			}
			if mode == "cancel" && r.result.ErrorCode != "cancelled" {
				t.Fatal(r.result.ErrorCode)
			}
			if err := e.vault.Sweep(context.Background(), func(context.Context, string) (bool, error) { return false, nil }); err != nil {
				t.Fatal(err)
			}
			entries, _ := os.ReadDir(e.vault.Directory())
			if len(entries) != 0 {
				t.Fatal("pins leaked after failure", len(entries))
			}
		})
	}
}

type brokenBody struct{ sent bool }

func (b *brokenBody) Read(p []byte) (int, error) {
	if b.sent {
		return 0, io.ErrUnexpectedEOF
	}
	b.sent = true
	return copy(p, []byte("vless://synthetic@192.0.2.1:443\n")), io.ErrUnexpectedEOF
}
func (b *brokenBody) Close() error { return nil }
func TestAcquisitionTruncatedBodyNeverReachesParser(t *testing.T) {
	e, r := pipelineFixture(t, 1)
	e.fetch = pipelineFetch{&brokenBody{}}
	e.runJob(context.Background(), Job{})
	if r.result.State != "error" || r.result.ErrorCode != "incomplete_body" || r.batches != 0 || r.result.Raw.Hash != "" {
		t.Fatalf("truncated response accepted: %+v", r.result)
	}
}

type deadlineBody struct{ ctx context.Context }

func (b deadlineBody) Read([]byte) (int, error) { <-b.ctx.Done(); return 0, b.ctx.Err() }
func (b deadlineBody) Close() error             { return nil }

type slowFetch struct{}

func (slowFetch) Name() string { return "slow" }
func (slowFetch) Fetch(ctx context.Context, j Job) (Response, error) {
	return Response{Status: 200, Body: deadlineBody{ctx}}, nil
}
func TestAcquisitionFetchBudgetIncludesBody(t *testing.T) {
	e, r := pipelineFixture(t, 1)
	e.fetch = slowFetch{}
	e.runJobWithBudgets(context.Background(), Job{}, 10*time.Millisecond, time.Second)
	if r.result.ErrorCode != "fetch_timeout" || r.batches != 0 {
		t.Fatal(r.result)
	}
}
