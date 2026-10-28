package lifecycle

import "testing"

func TestNoEmptyFetchWipesPreviousGeneration(t *testing.T) {
	for _, o := range []FetchOutcome{Empty, Invalid, Failed, Unchanged} {
		d := Decide(o, 0, 2, 2)
		if d.Replace || !d.KeepPrevious || !d.RequireFreshHealth {
			t.Fatal(o, d)
		}
	}
	d := Decide(Usable, 4, 2, 2)
	if !d.Replace || d.KeepPrevious || !d.RequireFreshHealth {
		t.Fatal(d)
	}
}
func TestOldJobCannotResurrectDeletedConfiguration(t *testing.T) {
	d := Decide(Usable, 4, 1, 2)
	if d.Replace || d.KeepPrevious || d.Reason != "stale_generation" {
		t.Fatal(d)
	}
}
func TestUnknownOutcomeDoesNotApproveFallback(t *testing.T) {
	d := Decide("unknown", 2, 2, 2)
	if d.Replace || d.KeepPrevious {
		t.Fatal(d)
	}
}
