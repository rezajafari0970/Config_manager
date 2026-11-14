package blob

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

// A database writer can need a blob while the sweeper waits for that writer.
// The reference check must yield the publication lock, retain the file and retry.
func TestSweepBusyReferenceCannotStallPublicationOrDelete(t *testing.T) {
	for _, lateNil := range []bool{false, true} {
		t.Run(map[bool]string{false: "context_error", true: "late_nil_result"}[lateNil], func(t *testing.T) {
			v, err := New(t.TempDir(), bytes.Repeat([]byte{8}, 32))
			if err != nil {
				t.Fatal(err)
			}
			payload := []byte("still referenced by a writer")
			ref, err := v.Put(context.Background(), bytes.NewReader(payload))
			if err != nil {
				t.Fatal(err)
			}
			v.Release(ref.Hash)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			entered := make(chan struct{})
			done := make(chan error, 1)
			go func() {
				done <- v.Sweep(ctx, func(check context.Context, _ string) (bool, error) {
					close(entered)
					<-check.Done()
					if lateNil {
						return false, nil
					}
					return false, check.Err()
				})
			}()
			defer cancel()
			select {
			case <-entered:
			case <-time.After(5 * time.Second):
				t.Fatal("reference check did not start")
			}
			opened := make(chan error, 1)
			go func() {
				r, e := v.Open(ref.Hash)
				if e == nil {
					var b []byte
					b, e = io.ReadAll(r)
					r.Close()
					if e == nil && !bytes.Equal(b, payload) {
						e = errors.New("blob changed")
					}
				}
				opened <- e
			}()
			select {
			case err = <-opened:
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(5 * time.Second):
				cancel()
				<-done
				<-opened
				t.Fatal("reference wait monopolized publication lock")
			}
			if err = <-done; !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("expected bounded deferred cleanup, got %v", err)
			}
			r, err := v.Open(ref.Hash)
			if err != nil {
				t.Fatal("uncertain reference deleted", err)
			}
			r.Close()
		})
	}
}
