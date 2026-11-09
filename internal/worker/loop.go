// Package worker owns queue scheduling independently of transport, preparation
// and sandbox execution. No untrusted configuration is executed here.
package worker

import (
	"context"
	"github.com/rezajafari0970/Config_manager/internal/probe"
	"time"
)

type Queue interface {
	Claim(context.Context) (*probe.Job, error)
}
type Execute func(context.Context, int, probe.Job)

// RunSlot drains useful work immediately and backs off only on empty/error
// responses. The bounded delay avoids hot database polling without a global
// queue lock; each slot and each pipeline recovers independently.
func RunSlot(ctx context.Context, q Queue, execute Execute, slot int, minimum, maximum time.Duration) {
	if minimum <= 0 || maximum < minimum {
		panic("invalid worker polling interval")
	}
	delay := minimum
	for {
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		j, e := q.Claim(ctx)
		if ctx.Err() != nil {
			return
		}
		if e != nil || j == nil {
			delay = min(maximum, max(minimum, delay*2))
			continue
		}
		execute(ctx, slot, *j)
		delay = minimum
	}
}
