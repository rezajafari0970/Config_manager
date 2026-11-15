package egress

// Iran is a deny signal, independent of split-route qualification and payload
// health. Missing observations never manufacture a country or a clean result.
const IranVersion = "ads-iran.v2"
const IranInterval = int64(600000)
const IranSpacing = int64(60000)

type IranObservation struct {
	Version string        `json:"version"`
	Hash    string        `json:"hash"`
	Domain  string        `json:"domain"`
	Variant string        `json:"variant"`
	At      int64         `json:"at"`
	Country CountryResult `json:"country"`
}

func Iran(r Report, p Plan, hash string, now int64) *IranObservation {
	if len(r.Evidence) > 4 {
		return nil
	}
	if x := iranRound(r, p, hash, now); x != nil {
		return x
	}
	for _, part := range r.Evidence {
		part, ok := resolveAttemptSamples(r, part)
		if !ok {
			continue
		}
		if len(part.Evidence) == 0 && len(part.Samples) <= 256 && part.Started >= r.Started && part.Finished <= r.Finished {
			if x := iranRound(part, p, hash, now); x != nil {
				return x
			}
		}
	}
	return nil
}

// Minority-IR conflicts veto publication without becoming confirmed guard hits.
func IranCountryConflict(r Report, p Plan, hash string, now int64) bool {
	if len(r.Evidence) > 4 {
		return false
	}
	if iranSignal(r, p, hash, now, true) != nil {
		return true
	}
	for _, part := range r.Evidence {
		part, ok := resolveAttemptSamples(r, part)
		if !ok {
			continue
		}
		if len(part.Evidence) == 0 && len(part.Samples) <= 256 && part.Started >= r.Started && part.Finished <= r.Finished && iranSignal(part, p, hash, now, true) != nil {
			return true
		}
	}
	return false
}
func iranRound(r Report, p Plan, hash string, now int64) *IranObservation {
	return iranSignal(r, p, hash, now, false)
}
func iranSignal(r Report, p Plan, hash string, now int64, conflict bool) *IranObservation {
	if r.Version != Version || r.Hash != hash || r.Started <= 0 || r.Started > r.Finished || r.Started < now-300000 || r.Finished > now+2000 || r.ResourcePressure || Cores[r.Core] == "" || Cores[r.Core] != r.CoreSHA {
		return nil
	}
	// Only the observed variant needs to have run successfully. Failure of the
	// other variant, controls or payload tests cannot hide authenticated IR.
	valid := map[string]bool{}
	for _, v := range p.Variants {
		for _, c := range r.Checks {
			if c.Name == v.Name && Hash(v.Config) == v.Hash && c.Hash == v.Hash && c.Core && routeHexBytes(c.RuntimeHash, 32) && (p.Driver != "xray" || c.RuntimeHash == v.Hash) {
				valid[v.Name] = true
			}
		}
	}
	for _, s := range r.Samples {
		if !valid[s.Variant] || !s.ActualAds || (!conflict && s.Geo.Country != "IR") || !validSource(s, r) {
			continue
		}
		// Validate this observation with the same independent provider policy as
		// publication; route completeness is deliberately not a country prerequisite.
		one := r
		one.Samples = []Sample{s}
		country := observedCountries(one, r.Finished+IranInterval, now)
		if conflict && country.Status == "country_iran_conflict" || !conflict && country.Complete && len(country.Countries) == 1 && country.Countries[0] == "IR" {
			return &IranObservation{IranVersion, hash, s.Domain, s.Variant, s.At, country}
		}
	}
	return nil
}

// Samples are serialized once. A selected-attempt reference may reuse them only
// with exactly matching core/check bindings; its pressure flag stays independent.
func resolveAttemptSamples(r, part Report) (Report, bool) {
	if !part.SelectedSamples {
		return part, true
	}
	if len(part.Samples) != 0 || part.Core != r.Core || part.CoreSHA != r.CoreSHA || part.Hash != r.Hash || part.Version != r.Version || len(part.Checks) != len(r.Checks) {
		return part, false
	}
	for i, c := range part.Checks {
		if c != r.Checks[i] {
			return part, false
		}
	}
	part.Samples = r.Samples
	return part, true
}
