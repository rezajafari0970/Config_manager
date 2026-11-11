package collect

import (
	"testing"
	"time"
)

func TestCardinalityAdmission(t *testing.T) {
	d := CardinalityAdmission{}
	now := time.Unix(1000, 0)
	state := AdmissionState{Driver: d.Name(), BaselineUnique: 148}
	first := d.Evaluate(AdmissionSample{148, 2, now}, state)
	if first.Accept || first.RetryAfter != 15*time.Second || first.State.PendingAttempts != 1 {
		t.Fatal(first)
	}
	rapid := d.Evaluate(AdmissionSample{148, 2, now.Add(time.Second)}, first.State)
	if rapid.Accept || rapid.State.PendingAttempts != 1 {
		t.Fatal(rapid)
	}
	recovered := d.Evaluate(AdmissionSample{148, 146, now.Add(15 * time.Second)}, first.State)
	if !recovered.Accept || recovered.State.PendingAttempts != 0 || recovered.State.BaselineUnique != 146 {
		t.Fatal(recovered)
	}
	second := d.Evaluate(AdmissionSample{148, 2, now.Add(15 * time.Second)}, first.State)
	third := d.Evaluate(AdmissionSample{148, 2, now.Add(30 * time.Second)}, second.State)
	if second.Accept || !third.Accept || third.Code != "drop_repeated" || third.State.BaselineUnique != 2 {
		t.Fatal(second, third)
	}
	for _, at := range []time.Time{now.Add(3 * time.Minute), now.Add(-time.Second)} {
		got := d.Evaluate(AdmissionSample{148, 2, at}, second.State)
		if got.Accept || got.State.PendingAttempts != 1 {
			t.Fatal("expired/backwards counter", got)
		}
	}
	for _, sample := range []AdmissionSample{{15, 1, now}, {148, 38, now}, {2, 2, now}, {0, 2, now}} {
		if got := d.Evaluate(sample, state); !got.Accept {
			t.Fatal("small/deleted inventory held", got)
		}
	}
	stale := state
	stale.Driver = "old"
	stale.PendingAttempts = 2
	if got := d.Evaluate(AdmissionSample{148, 2, now}, stale); got.Accept || got.State.PendingAttempts != 1 {
		t.Fatal("version reset", got)
	}
}
