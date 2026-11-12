package main

import "testing"

func TestInvestigationLanesAtEveryPositiveCapacity(t *testing.T) {
	for capacity := 1; capacity <= 31; capacity++ {
		first, follow := 0, 0
		for round := uint64(0); round < 5; round++ {
			for slot := 1; slot <= capacity; slot++ {
				if investigationLane(slot, capacity, round) == "followup" {
					follow++
				} else {
					first++
				}
			}
		}
		if first == 0 || follow == 0 {
			t.Fatal(capacity, first, follow)
		}
		if capacity == 1 && (first != 4 || follow != 1) {
			t.Fatal(first, follow)
		}
	}
}
