package egress

import "sort"

func evaluateServices(r Report, p Plan, variants []Variant, d Decision, now int64) Decision {
	type bucket struct {
		samples   []Sample
		waves     map[int]bool
		blocked   map[int]bool
		attempted bool
	}
	buckets := map[string]*bucket{}
	seen := map[string]bool{}
	ads := map[int]map[string]bool{}
	controls := map[string]bool{}
	earliest := r.Finished
	for _, v := range variants {
		for _, t := range registry {
			buckets[v.Name+"|"+t.Domain] = &bucket{waves: map[int]bool{}, blocked: map[int]bool{}}
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
			// Blackhole proof comes from the bound exact configuration, never from a
			// timeout, a 403, a provider claim, or a sandbox firewall denial alone.
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
	adsComplete, controlsComplete := true, true
	overlap, repeatedOverlap := false, false
	for _, v := range variants {
		for _, t := range registry {
			b := buckets[v.Name+"|"+t.Domain]
			status := "MISSING"
			n := len(b.waves)
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
				same := map[int]bool{}
				familyMissing := false
				paired := map[int]bool{}
				for _, s := range b.samples {
					if len(ads[s.Family]) == 0 {
						familyMissing = true
						continue
					}
					if ads[s.Family][s.Source] {
						for _, a := range registry {
							if a.Role == "ads" {
								for _, x := range buckets[v.Name+"|"+a.Domain].samples {
									if x.Wave == s.Wave && x.Family == s.Family && x.Source == s.Source {
										same[s.Wave] = true
									}
								}
							}
						}
						overlap = true
					}
					all := true
					for _, a := range registry {
						if a.Role == "ads" {
							ab := buckets[v.Name+"|"+a.Domain]
							found := false
							for _, x := range ab.samples {
								if x.Wave == s.Wave && x.Family == s.Family {
									found = true
								}
							}
							if !found {
								all = false
							}
						}
					}
					if all {
						paired[s.Wave] = true
					}
				}
				switch {
				case len(same) > 0 || hasPoolOverlap(b.samples, ads):
					status = "CONTROL_OVERLAP"
					controlsComplete = false
					if len(same) >= 2 {
						repeatedOverlap = true
					}
				case familyMissing:
					status = "FAMILY_UNVERIFIED"
					controlsComplete = false
				case n >= 2 && len(paired) >= 2:
					status = "DISTINCT_EGRESS"
				default:
					controlsComplete = false
					d.Missing += max(0, 2-n)
					if b.attempted {
						status = "ROUTE_UNOBSERVABLE"
					}
				}
			}
			d.Services = append(d.Services, ServiceDecision{v.Name, t.Domain, t.Group, status, n, t.Required})
		}
	}
	if len(d.Sources) > 0 {
		d.Route = "PARTIAL_DIFFERENCE"
	}
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
	if !controlsComplete {
		d.Reason = "controls_incomplete"
		return d
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
	d.CoverageComplete = true
	d.Reason = "repeated_ads_exclusive_observation"
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
