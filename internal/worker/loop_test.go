package worker

import (
	"context"
	"errors"
	"github.com/rezajafari0970/Config_manager/internal/probe"
	"testing"
	"time"
)

type queueFunc func(context.Context) (*probe.Job, error)

func (f queueFunc) Claim(ctx context.Context) (*probe.Job, error) { return f(ctx) }

func TestSlotRecoversFromQueueFailuresAndStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	calls, runs := 0, 0
	q := queueFunc(func(context.Context) (*probe.Job, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("offline")
		}
		if calls == 2 {
			return nil, nil
		}
		return &probe.Job{Hash: "fixture"}, nil
	})
	RunSlot(ctx, q, func(_ context.Context, slot int, j probe.Job) {
		if slot != 3 || j.Hash != "fixture" {
			t.Fatal("wrong slot/job")
		}
		runs++
		cancel()
	}, 3, time.Millisecond, 4*time.Millisecond)
	if calls != 3 || runs != 1 {
		t.Fatal("queue recovery", calls, runs)
	}
}

func TestIdleSlotDoesNotHotPoll(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 65*time.Millisecond)
	defer cancel()
	calls := 0
	RunSlot(ctx, queueFunc(func(context.Context) (*probe.Job, error) { calls++; return nil, nil }), func(context.Context, int, probe.Job) { t.Fatal("executed empty job") }, 1, time.Millisecond, 20*time.Millisecond)
	if calls > 8 {
		t.Fatal("unbounded idle polling", calls)
	}
}
