package egress

import "sort"

type serviceBucket struct {
	samples   []Sample
	waves     map[int]bool
	blocked   map[int]bool
	attempted bool
}

func (b *serviceBucket) stableFamily(family int) bool {
	sources, waves := map[string]bool{}, map[int]bool{}
	for _, s := range b.samples {
		if s.Family == family {
			sources[s.Source] = true
			waves[s.Wave] = true
		}
	}
	return len(sources) == 1 && len(waves) >= 2
}

func (b *serviceBucket) unstable() bool {
	sources := map[int]map[string]bool{}
	for _, s := range b.samples {
		if sources[s.Family] == nil {
			sources[s.Family] = map[string]bool{}
		}
		sources[s.Family][s.Source] = true
	}
	for _, pool := range sources {
		if len(pool) > 1 {
			return true
		}
	}
	return false
}

func (b *serviceBucket) witnesses(s Sample) bool {
	if len(b.samples) == 0 && len(b.blocked) >= 2 {
		return b.blocked[s.Wave]
	}
	if !b.stableFamily(s.Family) {
		return false
	}
	for _, c := range b.samples {
		if c.Wave == s.Wave && c.Family == s.Family && c.Source != s.Source {
			return true
		}
	}
	return false
}

func evaluateServices(r Report, p Plan, variants []Variant, d Decision, now int64) Decision {
	buckets := map[string]*serviceBucket{}
	seen := map[string]bool{}
	ads := map[int]map[string]bool{}
	controls := map[string]bool{}
	earliest := r.Finished
	for _, v := range variants {
		for _, t := range registry {
			buckets[v.Name+"|"+t.Domain] = &serviceBucket{waves: map[int]bool{}, blocked: map[int]bool{}}
		}
	}
	for _, s := range r.Samples {
		t, ok := targetFor(s.Domain)
		b := buckets[s.Variant+"|"+s.Domain]
		if !ok || b == nil || s.Wave < 1 || s.Wave > r.Waves || s.At < r.Started || s.At > r.Finished+2000 {
			continue
		}
		b.attempted = true
		if !validSource(s, r) || s.ActualAds != (t.Role == "ads") || seen[s.Connection] {
			// Timeouts are not block proof. The exact bound configuration must contain
			// a locally provable blackhole for this destination.
			if t.Role == "control" && !s.TLS && (s.Code == "socks_connect" || s.Code == "connection_failed" || s.Code == "connection_timeout") {
				for _, v := range variants {
					if v.Name == s.Variant && localBlocked(v, t.Domain) {
						b.blocked[s.Wave] = true
						if s.At < earliest {
							earliest = s.At
						}
					}
				}
			}
			continue
		}
		seen[s.Connection] = true
		b.samples = append(b.samples, s)
		b.waves[s.Wave] = true
		if s.At < earliest {
			earliest = s.At
		}
		if t.Role == "ads" {
			if ads[s.Family] == nil {
				ads[s.Family] = map[string]bool{}
			}
			ads[s.Family][s.Source] = true
		} else {
			controls[s.Source] = true
		}
	}
	for _, pool := range ads {
		for x := range pool {
			d.Sources = append(d.Sources, x)
		}
	}
	for x := range controls {
		d.ControlSources = append(d.ControlSources, x)
	}
	sort.Strings(d.Sources)
	sort.Strings(d.ControlSources)

	adsComplete, requiredComplete, fullCoverage := true, true, true
	overlap, repeatedOverlap := false, false
	for _, v := range variants {
		for _, t := range registry {
			b := buckets[v.Name+"|"+t.Domain]
			status, n := "MISSING", len(b.waves)
			if t.Role == "ads" {
				if n >= 2 {
					status = "ADS_OBSERVED"
				} else {
					adsComplete = false
					d.Missing += 2 - n
					if n > 0 {
						status = "ADS_PARTIAL"
					}
				}
			} else if len(b.blocked) >= 2 && len(b.samples) == 0 {
				status = "LOCAL_BLOCK_CONFIRMED"
				n = len(b.blocked)
			} else {
				same, paired := map[int]bool{}, map[int]bool{}
				for _, s := range b.samples {
					for _, a := range registry {
						if a.Role != "ads" {
							continue
						}
						for _, x := range buckets[v.Name+"|"+a.Domain].samples {
							if x.Wave != s.Wave || x.Family != s.Family {
								continue
							}
							// A control need not match the address family of every other Ads
							// destination. Coverage is checked separately for EACH Ads destination.
							if b.stableFamily(s.Family) {
								paired[s.Wave] = true
							}
							if x.Source == s.Source {
								same[s.Wave] = true
							}
						}
					}
				}
				switch {
				case len(same) > 0 || hasPoolOverlap(b.samples, ads):
					status = "CONTROL_OVERLAP"
					overlap = true
					if len(same) >= 2 {
						repeatedOverlap = true
					}
				case b.unstable():
					status = "UNSTABLE_CONTROL"
				case n >= 2 && len(paired) >= 2:
					status = "DISTINCT_EGRESS"
				case n >= 2:
					status = "FAMILY_UNVERIFIED"
				case b.attempted:
					status = "ROUTE_UNOBSERVABLE"
				}
				if t.Required && status != "DISTINCT_EGRESS" {
					requiredComplete = false
					d.Missing += max(0, 2-n)
				}
			}
			if status != "ADS_OBSERVED" && status != "DISTINCT_EGRESS" {
				fullCoverage = false
			}
			d.Services = append(d.Services, ServiceDecision{v.Name, t.Domain, t.Group, status, n, t.Required})
		}
	}
	if len(d.Sources) > 0 {
		d.Route = "PARTIAL_DIFFERENCE"
	}
	// Any authenticated overlap vetoes publication, including supplementary
	// services. A repeated same-wave overlap is a conclusive non-exclusive result.
	if overlap {
		d.Route = "OVERLAP"
		d.Reason = "mixed_route_observations"
		if repeatedOverlap && adsComplete {
			d.State = "NO_SPLIT"
			d.Reason = "non_ads_uses_ads_egress"
			d.Negative = true
		}
		return d
	}
	if !adsComplete {
		d.Reason = "ads_peer_echo_incomplete"
		return d
	}
	if !requiredComplete {
		d.Reason = "controls_incomplete"
		return d
	}

	// Every Ads destination needs two coherent waves with stable, same-family
	// control witnesses from independent groups, including ordinary Google.
	// Explicit local blocks can witness separation; cross-family addresses,
	// reachability, missing observations and rotating pools cannot.
	for _, v := range variants {
		for _, a := range registry {
			if a.Role != "ads" {
				continue
			}
			witnessed := map[int]bool{}
			for _, s := range buckets[v.Name+"|"+a.Domain].samples {
				groups := map[string]bool{}
				for _, t := range registry {
					if t.Role == "control" && t.Required && buckets[v.Name+"|"+t.Domain].witnesses(s) {
						groups[t.Group] = true
					}
				}
				if groups["google-search"] && len(groups) >= 2 {
					witnessed[s.Wave] = true
				}
			}
			if len(witnessed) < 2 {
				d.Reason = "paired_control_coverage_incomplete"
				return d
			}
		}
	}
	if len(variants) != 2 {
		d.Reason = "template_evidence_missing"
		return d
	}
	d.Until = earliest + EvidenceTTL
	if d.Until <= now {
		d.Reason = "stale_or_unbound_report"
		return d
	}
	d.State = "RESIDENTIAL"
	d.Route = "DIFFERENT"
	d.Eligible = true
	d.CoverageComplete = fullCoverage
	d.EvidenceProfile = "ads-stable-controls.v1"
	d.Reason = "repeated_ads_split_observation"
	return d
}

func hasPoolOverlap(samples []Sample, pools map[int]map[string]bool) bool {
	for _, s := range samples {
		if pools[s.Family][s.Source] {
			return true
		}
	}
	return false
}
