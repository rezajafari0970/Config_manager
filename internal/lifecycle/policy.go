package lifecycle

type FetchOutcome string

const (
	Usable    FetchOutcome = "usable_generation"
	Empty     FetchOutcome = "empty_source"
	Invalid   FetchOutcome = "no_acceptable_candidates"
	Failed    FetchOutcome = "fetch_failed"
	Unchanged FetchOutcome = "unchanged"
)

type Decision struct {
	Replace            bool
	KeepPrevious       bool
	RequireFreshHealth bool
	Reason             string
}

// Decide separates source freshness from health. A fallback is never permanent
// approval. This pure policy is not yet a scheduler or persistent source store.
func Decide(outcome FetchOutcome, accepted int, jobGeneration, currentGeneration uint64) Decision {
	if jobGeneration != currentGeneration {
		return Decision{Reason: "stale_generation"}
	}
	switch outcome {
	case Usable:
		if accepted > 0 {
			return Decision{Replace: true, RequireFreshHealth: true, Reason: "publish_new_generation"}
		}
		return Decision{KeepPrevious: true, RequireFreshHealth: true, Reason: "zero_accepted"}
	case Empty, Invalid, Failed, Unchanged:
		return Decision{KeepPrevious: true, RequireFreshHealth: true, Reason: string(outcome)}
	default:
		return Decision{Reason: "unknown_outcome"}
	}
}
