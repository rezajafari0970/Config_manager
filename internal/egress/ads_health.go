package egress

// AdsHealth measures completed bidirectional application exchanges on the actual
// observed Ads streams. It is NOT a 64KiB upload or a speed certificate. Generic
// payload results remain separately visible and cannot delete Ads-only inputs.
func AdsHealth(r Report, p Plan, now int64) string {
	if r.Version != Version || r.ResourcePressure || r.Started <= 0 || r.Finished < r.Started || r.Started < now-300000 || r.Finished > now+2000 || Cores[r.Core] == "" || Cores[r.Core] != r.CoreSHA || len(p.Variants) != 2 || len(r.Checks) != 2 {
		return "unknown"
	}
	for i, v := range p.Variants {
		c := r.Checks[i]
		if c.Name != v.Name || c.Hash != v.Hash || !c.Core || len(c.RuntimeHash) != 64 || (p.Driver == "xray" && c.RuntimeHash != v.Hash) {
			return "unknown"
		}
		for _, domain := range Domains {
			waves := map[int]bool{}
			seen := map[string]bool{}
			for _, s := range r.Samples {
				if s.Variant == v.Name && s.Domain == domain && s.ActualAds && s.Wave >= 1 && s.Wave <= r.Waves && validSource(s, r) && s.ResponseComplete && s.RequestBytes > 0 && s.RequestBytes <= 32768 && s.ResponseBytes > 0 && s.ResponseBytes <= 131072 && !seen[s.Connection] {
					seen[s.Connection] = true
					waves[s.Wave] = true
				}
			}
			if len(waves) < 2 {
				return "unknown"
			}
		}
	}
	return "healthy"
}
