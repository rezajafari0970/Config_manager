package collect

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/rezajafari0970/Config_manager/internal/blob"
)

type Engine struct {
	repo        Repository
	vault       *blob.Vault
	fetch       FetchDriver
	formats     *Registry
	mu          sync.Mutex
	active      map[string]context.CancelFunc
	wake        chan struct{}
	wg          sync.WaitGroup
	lastTick    atomic.Int64
	running     atomic.Bool
	cleaning    atomic.Bool
	diskBlocked atomic.Bool
}

func NewEngine(repo Repository, vault *blob.Vault, driver FetchDriver) *Engine {
	return &Engine{repo: repo, vault: vault, fetch: driver, formats: NewRegistry(), active: map[string]context.CancelFunc{}, wake: make(chan struct{}, 1)}
}
func (e *Engine) RegisterFormat(driver FormatDriver) { e.formats.Register(driver) }
func (e *Engine) Wake(id string, cancel bool) {
	if cancel {
		e.mu.Lock()
		if stop := e.active[id]; stop != nil {
			stop()
		}
		e.mu.Unlock()
	}
	select {
	case e.wake <- struct{}{}:
	default:
	}
}
func (e *Engine) Status() (bool, int, int64) {
	e.mu.Lock()
	n := len(e.active)
	e.mu.Unlock()
	return e.running.Load(), n, e.lastTick.Load()
}
func (e *Engine) Run(ctx context.Context) error {
	if err := e.repo.Recover(ctx); err != nil {
		return err
	}
	e.running.Store(true)
	defer e.running.Store(false)
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	cleanup := time.NewTicker(5 * time.Second)
	defer cleanup.Stop()
	defer func() {
		e.mu.Lock()
		for _, cancel := range e.active {
			cancel()
		}
		e.mu.Unlock()
		e.wg.Wait()
	}()
	for {
		e.dispatch(ctx)
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
		case <-e.wake:
		case <-cleanup.C:
			if e.cleaning.CompareAndSwap(false, true) {
				e.wg.Add(1)
				go func() {
					defer e.wg.Done()
					defer e.cleaning.Store(false)
					cleanctx, cancel := context.WithTimeout(ctx, 20*time.Second)
					defer cancel()
					if err := e.repo.Expire(cleanctx, time.Now()); err != nil {
						log.Print("collector expiry will retry")
					}
					if err := e.vault.Sweep(cleanctx, e.repo.Referenced); err != nil && ctx.Err() == nil {
						log.Print("collector file cleanup will retry")
					}
				}()
			}
		}
	}
}
func (e *Engine) dispatch(ctx context.Context) {
	e.lastTick.Store(time.Now().UnixMilli())
	settings, err := e.repo.Settings(ctx)
	if err != nil {
		log.Print("collector scheduling will retry")
		return
	}
	// Fair bounded admission, independent per-source client/deadline/cancellation.
	// Capacity is adaptive; it does not limit the number of stored sources/configs.
	capacity := settings.Concurrency
	physical := runtime.NumCPU() * 8
	if physical < 4 {
		physical = 4
	}
	if capacity > physical {
		capacity = physical
	}
	e.mu.Lock()
	available := capacity - len(e.active)
	e.mu.Unlock()
	if available <= 0 {
		return
	}
	var disk syscall.Statfs_t
	if syscall.Statfs(e.vault.Directory(), &disk) == nil && disk.Bavail*uint64(disk.Bsize) < 256*1024*1024 {
		e.diskBlocked.Store(true)
		return
	}
	e.diskBlocked.Store(false)
	jobs, err := e.repo.Claim(ctx, time.Now(), available, time.Duration(settings.TimeoutSeconds)*time.Second+30*time.Second)
	if err != nil {
		log.Print("collector claim will retry")
		return
	}
	for _, job := range jobs {
		job := job
		e.mu.Lock()
		if _, exists := e.active[job.ID]; exists {
			e.mu.Unlock()
			continue
		}
		jobctx, cancel := context.WithTimeout(ctx, time.Duration(settings.TimeoutSeconds)*time.Second)
		e.active[job.ID] = cancel
		e.mu.Unlock()
		e.wg.Add(1)
		go func() {
			defer e.wg.Done()
			defer cancel()
			defer func() { e.mu.Lock(); delete(e.active, job.ID); e.mu.Unlock(); e.Wake("", false) }()
			e.runJob(jobctx, job)
		}()
	}
}
func (e *Engine) runJob(ctx context.Context, job Job) {
	started := time.Now()
	result := Result{State: "error"}
	pins := []string{}
	defer func() {
		if recover() != nil {
			result = Result{State: "error", ErrorCode: "worker_recovered"}
		}
		result.DurationMS = time.Since(started).Milliseconds()
		commitctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := e.repo.Complete(commitctx, job, result, time.Now()); err != nil {
			log.Print("collector result rejected or pending lease recovery")
		}
		for _, hash := range pins {
			e.vault.Release(hash)
		}
	}()
	response, err := e.fetch.Fetch(ctx, job)
	if err != nil {
		result.ErrorCode = errorCode(err)
		return
	}
	defer response.Body.Close()
	result.HTTPStatus = response.Status
	result.ETag = response.ETag
	result.Modified = response.Modified
	if response.Status == 304 {
		result.State = "unchanged"
		return
	}
	if response.Status < 200 || response.Status >= 300 {
		result.ErrorCode = "http_error"
		return
	}
	spool, err := os.CreateTemp("", "config-fetch-")
	if err != nil {
		result.ErrorCode = "storage_error"
		return
	}
	defer os.Remove(spool.Name())
	defer spool.Close()
	raw, err := e.vault.Put(ctx, io.TeeReader(&contextReader{ctx, response.Body}, spool))
	if err != nil {
		result.ErrorCode = errorCode(err)
		return
	}
	pins = append(pins, raw.Hash)
	result.Raw = raw
	if err = spool.Close(); err != nil {
		result.ErrorCode = "storage_error"
		return
	}
	unique := 0
	result.Staged = true
	err = e.formats.Extract(ctx, spool.Name(), func(ctx context.Context, r io.Reader, kind string) error {
		sample := &sampleWriter{}
		ref, err := e.vault.Put(ctx, io.TeeReader(r, sample))
		if err != nil {
			return err
		}
		defer e.vault.Release(ref.Hash)
		identity := Fingerprint(sample.data, ref.Size <= int64(len(sample.data)), kind, ref.Hash)
		result.Extracted++
		added, err := e.repo.Stage(ctx, job, Item{Ref: ref, Identity: identity, Kind: kind})
		if added {
			unique++
		}
		return err
	})
	if err != nil {
		result.Items = nil
		result.ErrorCode = errorCode(err)
		return
	}
	result.State = "updated"
	if unique == 0 {
		result.State = "empty"
	}
}

type sampleWriter struct{ data []byte }

func (w *sampleWriter) Write(p []byte) (int, error) {
	n := len(p)
	remaining := 1024*1024 - len(w.data)
	if remaining > 0 {
		if len(p) > remaining {
			p = p[:remaining]
		}
		w.data = append(w.data, p...)
	}
	return n, nil
}
func errorCode(err error) string {
	if errors.Is(err, ErrBlocked) {
		return "blocked_destination"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	if errors.Is(err, context.Canceled) {
		return "cancelled"
	}
	var dns *net.DNSError
	if errors.As(err, &dns) {
		return "dns_error"
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return "timeout"
	}
	return "fetch_or_parse_error"
}

func (e *Engine) BlockedReason() string {
	if e.diskBlocked.Load() {
		return "storage_low"
	}
	return ""
}
