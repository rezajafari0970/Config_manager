package panel

import (
	"context"
	"sync"
)

// Workers share one database admission slot, independently of their running
// network jobs. Queue before the expensive WAL preflight, rather than competing
// for CPU/readers and joining the sole writer's queue at once. Exports, lease
// renewals and completions never wait for this slot. The database remains the
// authority for capacity, ownership and eligibility, including across Stores.
func (s *Store) admitClaim(ctx context.Context) (func(), error) {
	s.claimOnce.Do(func() { s.claimSlot = make(chan struct{}, 1) })
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	select {
	case s.claimSlot <- struct{}{}:
		var once sync.Once
		return func() { once.Do(func() { <-s.claimSlot }) }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
