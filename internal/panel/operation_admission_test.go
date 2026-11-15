package panel

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/rezajafari0970/Config_manager/internal/collect"
)

func waitAdmission(t *testing.T, a *operationAdmission, foreground, background int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		a.mu.Lock()
		ok := len(a.foreground) == foreground && len(a.background) == background
		a.mu.Unlock()
		if ok {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("operation did not enter the expected admission queue")
}

func TestOperationAdmissionPriorityAndBoundedFairness(t *testing.T) {
	var a operationAdmission
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	release, err := a.acquire(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	type grant struct {
		id      int
		release func()
		err     error
	}
	grants := make(chan grant, 10)
	queue := func(id int, foreground bool) {
		go func() { r, e := a.acquire(ctx, foreground); grants <- grant{id, r, e} }()
	}
	queue(0, false)
	waitAdmission(t, &a, 0, 1)
	for i := 1; i <= foregroundBurst+1; i++ {
		queue(i, true)
		waitAdmission(t, &a, i, 1)
	}
	release()
	want := []int{1, 2, 3, 4, 5, 6, 7, 8, 0, 9}
	for _, id := range want {
		select {
		case got := <-grants:
			if got.err != nil || got.id != id {
				t.Fatalf("grant=%d err=%v want=%d", got.id, got.err, id)
			}
			got.release()
			got.release() // terminal tx paths may release twice
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
}

func TestOperationAdmissionCancellationRacesDoNotLeak(t *testing.T) {
	var a operationAdmission
	for i := 0; i < 300; i++ {
		release, err := a.acquire(context.Background(), false)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() {
			r, e := a.acquire(ctx, true)
			if e == nil {
				r()
			}
			done <- e
		}()
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); cancel() }()
		go func() { defer wg.Done(); release() }()
		wg.Wait()
		select {
		case err = <-done:
			if err != nil && !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("canceled acquisition leaked")
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := a.acquire(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	r()
}

func TestCollectorAdmissionCancellationLeavesWriterFree(t *testing.T) {
	s := testStore(t)
	release, err := s.operations.acquire(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	before := s.db.Stats().WaitCount
	calls := []func(context.Context) error{
		func(c context.Context) error { _, e := s.Claim(c, time.Now(), 1, time.Minute); return e },
		func(c context.Context) error { return s.Complete(c, collect.Job{}, collect.Result{}, time.Now()) },
		func(c context.Context) error { return s.Expire(c, time.Now()) },
		func(c context.Context) error { _, e := s.Stage(c, collect.Job{}, collect.Item{}); return e },
		func(c context.Context) error { return s.Abandon(c, collect.Job{}) },
	}
	for _, call := range calls {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		err = call(ctx)
		cancel()
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal(err)
		}
	}
	if s.db.Stats().WaitCount != before {
		t.Fatal("queued collector reached writer")
	}
	release()
	if _, err = s.Stage(context.Background(), collect.Job{}, collect.Item{}); !errors.Is(err, ErrConflict) {
		t.Fatal("Stage facade nested admission", err)
	}
}

type slowAdmissionResponse struct {
	*httptest.ResponseRecorder
	entered chan struct{}
	resume  chan struct{}
}

func (w *slowAdmissionResponse) WriteHeader(code int) {
	close(w.entered)
	<-w.resume
	w.ResponseRecorder.WriteHeader(code)
}

func TestPrivateWorkerAdmissionEndsBeforeResponseWrite(t *testing.T) {
	s := testStore(t)
	w := &slowAdmissionResponse{ResponseRecorder: httptest.NewRecorder(), entered: make(chan struct{}), resume: make(chan struct{})}
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.admitWorker(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })).ServeHTTP(w, httptest.NewRequest("POST", "/claim", nil))
	}()
	defer func() { close(w.resume); <-done }()
	select {
	case <-w.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("worker never wrote")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := s.Ping(ctx); err != nil {
		t.Fatal("slow worker response held writer admission", err)
	}
}

func TestOutputTransactionsReleaseAdmission(t *testing.T) {
	s := testStore(t)
	seedRotation(t, s, 62, 2)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for i := 0; i < 4; i++ {
		v, err := s.NextConversion(ctx)
		if err != nil || v == nil || len(v.Bytes) == 0 {
			t.Fatal("rotation leaked admission", err)
		}
	}
	for _, p := range []*SmartPipeline{s.Smart(), s.SplashSmart(), s.AdsSmart()} {
		tx, err := p.outputTransaction(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if err = tx.Commit(); err != nil {
			t.Fatal(err)
		}
		_ = tx.Rollback()
		if err = s.Ping(ctx); err != nil {
			t.Fatal("scoped transaction leaked admission", err)
		}
	}
}
