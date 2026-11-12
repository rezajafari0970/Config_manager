package main

// Reserve follow-up capacity even under resource pressure. A single surviving
// slot rotates every fifth completed claim; idle lanes can steal in the store.
func investigationLane(slot, capacity int, claims uint64) string {
	if capacity == 1 {
		if claims%5 == 4 {
			return "followup"
		}
		return "initial"
	}
	if capacity > 1 && slot > capacity-max(1, (capacity+4)/5) {
		return "followup"
	}
	return "initial"
}
