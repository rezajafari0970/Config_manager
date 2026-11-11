package collect

import "time"

const CardinalityAdmissionVersion = "source-cardinality.v1"

// SourceAdmissionDriver is a closed acquisition policy extension point. It
// decides whether a new snapshot can replace membership, never network health.
type SourceAdmissionDriver interface {
	Name() string
	Evaluate(AdmissionSample, AdmissionState) AdmissionDecision
}
type AdmissionSample struct {
	ExistingUnique, CandidateUnique int
	Now                             time.Time
}
type AdmissionState struct {
	Driver                          string
	BaselineUnique, PendingAttempts int
	LastCounted                     time.Time
}
type AdmissionDecision struct {
	Accept         bool
	Code           string
	RetryAfter     time.Duration
	State          AdmissionState
	PreviousUnique int
}
type CardinalityAdmission struct{}

func (CardinalityAdmission) Name() string { return CardinalityAdmissionVersion }
func (d CardinalityAdmission) Evaluate(sample AdmissionSample, state AdmissionState) AdmissionDecision {
	if state.Driver != d.Name() {
		state = AdmissionState{Driver: d.Name(), BaselineUnique: sample.ExistingUnique}
	}
	baseline := min(state.BaselineUnique, sample.ExistingUnique)
	if state.BaselineUnique == 0 {
		baseline = sample.ExistingUnique
	}
	decision := AdmissionDecision{Accept: true, Code: "accepted", State: state, PreviousUnique: baseline}
	// A small/single rotating source is never treated as a truncated large batch.
	// Expired or explicitly deleted memberships cannot be kept alive by this gate.
	if baseline < 16 || sample.CandidateUnique > baseline/4 {
		decision.State = AdmissionState{Driver: d.Name(), BaselineUnique: sample.CandidateUnique}
		return decision
	}
	if state.LastCounted.IsZero() || sample.Now.Sub(state.LastCounted) > 2*time.Minute || sample.Now.Before(state.LastCounted) {
		state.PendingAttempts = 0
		state.LastCounted = time.Time{}
	}
	if state.LastCounted.IsZero() || sample.Now.Sub(state.LastCounted) >= 2*time.Second {
		state.PendingAttempts++
		state.LastCounted = sample.Now
	}
	decision.State = state
	if state.PendingAttempts >= 3 {
		decision.Code = "drop_repeated"
		decision.State = AdmissionState{Driver: d.Name(), BaselineUnique: sample.CandidateUnique}
		return decision
	}
	decision.Accept = false
	decision.Code = "drop_retry"
	decision.RetryAfter = 15 * time.Second
	return decision
}
