package schedule

import (
	"testing"
	"time"
)

func TestIndependentDeadlinesStayWithinBounds(t *testing.T) {
	now := time.Unix(0, 0)
	for i := 0; i < 1000; i++ {
		next, err := Next(now, 15*time.Minute, 20*time.Minute)
		if err != nil || next.Before(now.Add(15*time.Minute)) || next.After(now.Add(20*time.Minute)) {
			t.Fatalf("out of range: %v %v", next, err)
		}
	}
	next, err := Next(now, time.Second, time.Second)
	if err != nil || !next.Equal(now.Add(time.Second)) {
		t.Fatal(next, err)
	}
}
func TestRejectInvalidIntervals(t *testing.T) {
	for _, p := range [][2]time.Duration{{0, time.Second}, {-1, 2}, {20, 15}} {
		if _, err := Next(time.Now(), p[0], p[1]); err == nil {
			t.Fatal("accepted invalid bounds", p)
		}
	}
}
