package egress

// AdsHealth is completed request/response on a whole named profile, not bulk
// throughput. Generic upload/download remains a diagnostic, never an Ads veto.
func AdsHealth(r Report, p Plan, now int64) string {
	if Delivery(r, p, now).Status == "unavailable" {
		return "unhealthy"
	}
	if routeBound(r, p, r.Hash, now) != "" {
		return "unknown"
	}
	for _, profile := range profiles {
		if healthyDomains(r, profile.Domains) && Delivery(r, p, now).Profiles[profile.ID] == "available" {
			return "healthy"
		}
	}
	return "unknown"
}
func AdsHealthForDecision(r Report, p Plan, d Decision, now int64) string {
	if !d.Eligible {
		return AdsHealth(r, p, now)
	}
	if routeBound(r, p, r.Hash, now) != "" || len(d.QualifiedDomains) == 0 || !healthyDomains(r, d.QualifiedDomains) {
		return "unknown"
	}
	return "healthy"
}
func healthyDomains(r Report, domains []string) bool {
	connections := connectionCounts(r)
	for _, variant := range []string{"raw", "template"} {
		for _, domain := range domains {
			waves := map[int]bool{}
			for _, s := range r.Samples {
				if s.Variant == variant && s.Domain == domain && s.ActualAds && routeValid(s, r) && connections[s.Connection] == 1 && s.ResponseComplete && s.RequestBytes > 0 && s.RequestBytes <= 32768 && s.ResponseBytes > 0 && s.ResponseBytes <= 131072 {
					waves[s.Wave] = true
				}
			}
			if len(waves) < 2 {
				return false
			}
		}
	}
	return true
}
