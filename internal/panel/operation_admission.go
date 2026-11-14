package panel

import (
	"context"
	"database/sql"
	"net/http"
	"sync"
)

// Schedule database operations before they enter database/sql's unordered
// connection wait queue. Foreground turns are bounded so a sustained export
// stream cannot starve lease renewal, completion, or collection.
type operationAdmission struct {
	mu                     sync.Mutex
	active                 bool
	burst                  int
	foreground, background []*operationWaiter
}

type operationWaiter struct {
	ready   chan struct{}
	granted bool
}

const foregroundBurst = 8

func (a *operationAdmission) acquire(ctx context.Context, foreground bool) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	w := &operationWaiter{ready: make(chan struct{})}
	a.mu.Lock()
	if foreground {
		a.foreground = append(a.foreground, w)
	} else {
		a.background = append(a.background, w)
	}
	a.advance()
	a.mu.Unlock()
	select {
	case <-w.ready:
		var once sync.Once
		return func() { once.Do(func() { a.mu.Lock(); a.active = false; a.advance(); a.mu.Unlock() }) }, nil
	case <-ctx.Done():
		a.mu.Lock()
		if w.granted {
			a.active = false
		} else {
			queue := &a.background
			if foreground {
				queue = &a.foreground
			}
			for i, item := range *queue {
				if item == w {
					*queue = append((*queue)[:i], (*queue)[i+1:]...)
					break
				}
			}
		}
		a.advance()
		a.mu.Unlock()
		return nil, ctx.Err()
	}
}

// Called with mu held; each grant belongs to exactly one acquire/release pair.
func (a *operationAdmission) advance() {
	if a.active {
		return
	}
	var w *operationWaiter
	if len(a.foreground) > 0 && (len(a.background) == 0 || a.burst < foregroundBurst) {
		w, a.foreground = a.foreground[0], a.foreground[1:]
		a.burst++
	} else if len(a.background) > 0 {
		w, a.background = a.background[0], a.background[1:]
		a.burst = 0
	} else {
		a.burst = 0
		return
	}
	a.active, w.granted = true, true
	close(w.ready)
}

// The gate ends with the transaction, before blob I/O or response streaming.
type outputTx struct {
	*sql.Tx
	release func()
}

func (t *outputTx) Commit() error   { defer t.release(); return t.Tx.Commit() }
func (t *outputTx) Rollback() error { defer t.release(); return t.Tx.Rollback() }
func (s *Store) outputTransaction(ctx context.Context) (*outputTx, error) {
	release, err := s.operations.acquire(ctx, true)
	if err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		release()
		return nil, err
	}
	return &outputTx{Tx: tx, release: release}, nil
}

type admittedResponse struct {
	http.ResponseWriter
	release func()
}

func (w admittedResponse) WriteHeader(status int)      { w.release(); w.ResponseWriter.WriteHeader(status) }
func (w admittedResponse) Write(b []byte) (int, error) { w.release(); return w.ResponseWriter.Write(b) }

// These handlers are private local worker RPCs. Serialize their database work,
// but release before the first response byte (which may contain a large job).
func (s *Store) admitWorker(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		release, err := s.operations.acquire(r.Context(), false)
		if err != nil {
			failure(w, err)
			return
		}
		defer release()
		next.ServeHTTP(admittedResponse{ResponseWriter: w, release: release}, r)
	})
}
