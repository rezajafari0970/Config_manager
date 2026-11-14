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
		if len(part.Evidence) == 0 && len(part.Samples) <= 256 && part.Started >= r.Started && part.Finished <= r.Finished {
			if x := iranRound(part, p, hash, now); x != nil {
				return x
			}
		}
	}
	return nil
}
func iranRound(r Report, p Plan, hash string, now int64) *IranObservation {
	if r.Version != Version || r.Hash != hash || r.Started <= 0 || r.Started > r.Finished || r.Started < now-300000 || r.Finished > now+2000 || r.ResourcePressure || Cores[r.Core] == "" || Cores[r.Core] != r.CoreSHA {
		return nil
	}
	// Only the observed variant needs to have run successfully. Failure of the
	// other variant, controls or payload tests cannot hide authenticated IR.
	valid := map[string]bool{}
	for _, v := range p.Variants {
		for _, c := range r.Checks {
			if c.Name == v.Name && c.Hash == v.Hash && c.Core && len(c.RuntimeHash) == 64 && (p.Driver != "xray" || c.RuntimeHash == v.Hash) {
				valid[v.Name] = true
			}
		}
	}
	for _, s := range r.Samples {
		if !valid[s.Variant] || !s.ActualAds || s.Geo.Country != "IR" || !validSource(s, r) {
			continue
		}
		// Validate this observation with the same independent provider policy as
		// publication; route completeness is deliberately not a country prerequisite.
		one := r
		one.Samples = []Sample{s}
		country := observedCountries(one, r.Finished+IranInterval, now)
		if country.Complete && len(country.Countries) == 1 && country.Countries[0] == "IR" {
			return &IranObservation{IranVersion, hash, s.Domain, s.Variant, s.At, country}
		}
	}
	return nil
}
