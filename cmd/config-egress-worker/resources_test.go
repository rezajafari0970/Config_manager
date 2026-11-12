package main

import (
	"testing"
	"time"
)

func TestResourceAdmissionAndGradualRecovery(t *testing.T) {
	for _, c := range []struct {
		s    resourceSample
		want int
	}{{resourceSample{2 * 1024 * 1024, 10, 0}, 10}, {resourceSample{400 * 1024, 0, 0}, 0}, {resourceSample{800 * 1024, 0, 0}, 2}, {resourceSample{1200 * 1024, 0, 0}, 4}, {resourceSample{2 * 1024 * 1024, 85, 0}, 5}, {resourceSample{2 * 1024 * 1024, 95, 0}, 2}, {resourceSample{2 * 1024 * 1024, 10, 26}, 0}, {resourceSample{-1, 0, 0}, 2}} {
		if n := resourceCapacity(10, c.s); n != c.want {
			t.Fatal(c, n)
		}
	}
	now := time.Now()
	g := resourceGovernor{current: 10, changed: now}
	if g.update(2, now) != 2 || g.update(10, now.Add(time.Second)) != 2 || g.update(10, now.Add(10*time.Second)) != 3 {
		t.Fatal(g)
	}
	if pressure([]byte("some avg10=81.25 avg60=50\nfull avg10=3.0\n"), "some") != 81.25 {
		t.Fatal("PSI parser")
	}
}
