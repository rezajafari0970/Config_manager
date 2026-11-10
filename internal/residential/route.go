package residential

import (
	"slices"
	"sort"
)

// RouteComparison is a scoped observation, independent of the access-network
// class. It never promotes a candidate response or ASN metadata into Ads proof.
type RouteComparison struct {
	Version    string  `json:"version"`
	State      string  `json:"state"`
	Reason     string  `json:"reason"`
	Different  []Scope `json:"different"`
	Same       []Scope `json:"same"`
	ValidUntil int64   `json:"valid_until"`
}

func Evaluate(r Report, p Policy, t Trust, now int64) Decision {
	d := evaluateResidential(r, p, t, now)
	d.AdsRoute = compareAdsRoutes(r, p, t, now)
	if !d.Eligible && d.Health == "HEALTHY" && d.AdsRoute.State == "DIFFERENT" {
		d.State, d.Reason = "ADS_DIFFERENT", "repeated_attributed_ads_difference"
		d.Eligible, d.CredibleNegative = true, false
		d.ValidUntil = d.AdsRoute.ValidUntil
	}
	return d
}

func compareAdsRoutes(r Report, p Policy, t Trust, now int64) RouteComparison {
	out := RouteComparison{Version: RouteComparisonVersion, State: "UNKNOWN", Reason: "attributed_paired_samples_required", Different: []Scope{}, Same: []Scope{}}
	if p.RouteComparisonVersion != RouteComparisonVersion || p.Repeats < 3 || r.Version != Version || r.PolicyHash != p.Digest() || !r.InfrastructureOK || r.Started <= 0 || r.Started > r.Finished || r.Finished > now+2000 || r.Started+int64(p.EvidenceTTLSeconds)*1000 <= now || len(r.Variants) != 2 {
		return out
	}
	variants := map[string]Variant{}
	for _, v := range r.Variants {
		if (v.Name != "raw" && v.Name != "template") || variants[v.Name].Name != "" || v.Core != "passed" || v.Health != "HEALTHY" || !v.PathsComplete || len(v.Paths) != 1 || !digestOK(v.Hash) || !digestOK(v.RuntimeHash) {
			return out
		}
		variants[v.Name] = v
	}
	type measured struct {
		scope     Scope
		different bool
		until     int64
	}
	paired := map[string]map[string]measured{}
	for _, v := range r.Variants {
		for _, domain := range p.Domains {
			for _, family := range p.Families {
				for _, transport := range p.Transports {
					scope := Scope{v.Name, domain, family, transport, v.Paths[0]}
					ads := map[string]bool{}
					controls := map[string]bool{}
					waves := map[int]bool{}
					connections := map[string]bool{}
					nonces := map[string]bool{}
					until := r.Started + int64(p.EvidenceTTLSeconds)*1000
					complete := true
					for wave := 1; wave <= p.Repeats; wave++ {
						groups := map[string]bool{}
						found := false
						for _, o := range r.Observations {
							if o.Scope != scope || o.Wave != wave {
								continue
							}
							found = true
							if !validObservation(o, p, t, now) || o.ConfigHash != v.Hash || o.RuntimeHash != v.RuntimeHash || o.At < r.Started || o.At > r.Finished+2000 || connections[o.ConnectionID] || nonces[o.Nonce] {
								complete = false
								continue
							}
							connections[o.ConnectionID] = true
							nonces[o.Nonce] = true
							observedIP, _ := publicIP(o.IP)
							ads[observedIP.String()] = true
							waves[wave] = true
							until = min(until, o.At+int64(p.EvidenceTTLSeconds)*1000, t.Capabilities[o.Capability].Expires)
						}
						if !found {
							complete = false
						}
						for _, c := range r.Controls {
							if c.Variant != v.Name || c.Path != scope.Path || c.Family != family || c.Transport != transport || c.Wave != wave {
								continue
							}
							group := controlGroup(c.Domain)
							ip, valid := publicIP(c.IP)
							if group == "" || !valid || (family == 4) != ip.Is4() || c.Outcome != "passed" || !c.TLSVerified || !c.OriginVerified || !c.SameConnection || c.Connections != 1 || c.Redirects != 0 || !c.ActualDomainRequest || !recent(c.At, now, p.EvidenceTTLSeconds) || c.At < r.Started || c.At > r.Finished+2000 || len(c.ConnectionID) < 16 || !digestOK(c.ResponseSHA256) || connections[c.ConnectionID] {
								complete = false
								continue
							}
							connections[c.ConnectionID] = true
							groups[group] = true
							controls[ip.String()] = true
							until = min(until, c.At+int64(p.EvidenceTTLSeconds)*1000)
						}
						if len(groups) < 2 {
							complete = false
						}
					}
					// A rotating or destination-dependent control pool cannot exclude
					// ordinary egress rotation. Keep it unknown even when ASNs differ.
					if !complete || len(waves) != p.Repeats || len(controls) != 1 || len(ads) == 0 || until <= now {
						continue
					}
					same := true
					different := true
					for ip := range ads {
						if controls[ip] {
							different = false
						} else {
							same = false
						}
					}
					if !same && !different {
						continue
					}
					key := Scope{"", domain, family, transport, ""}.Key()
					if paired[key] == nil {
						paired[key] = map[string]measured{}
					}
					paired[key][v.Name] = measured{scope, different, until}
				}
			}
		}
	}
	keys := make([]string, 0, len(paired))
	for k := range paired {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		a, b := paired[k]["raw"], paired[k]["template"]
		if a.scope.Variant == "" || b.scope.Variant == "" || a.different != b.different {
			continue
		}
		if a.different {
			out.Different = append(out.Different, a.scope, b.scope)
		} else {
			out.Same = append(out.Same, a.scope, b.scope)
		}
		if out.ValidUntil == 0 {
			out.ValidUntil = min(a.until, b.until)
		} else {
			out.ValidUntil = min(out.ValidUntil, a.until, b.until)
		}
	}
	if len(out.Different) > 0 {
		out.State, out.Reason = "DIFFERENT", "repeated_difference_with_stable_independent_controls"
	} else if len(out.Same) > 0 {
		out.State, out.Reason = "SAME", "repeated_same_egress_with_stable_independent_controls"
	}
	return out
}

// Only fresh loss of the SAME previously accepted scopes can authorize a
// negative vote. Missing methods/intelligence, rotation and outages cannot.
func LostQualification(previous, current Decision) bool {
	if current.Eligible {
		return false
	}
	if current.Health == "UNHEALTHY" {
		return current.CredibleNegative
	}
	if previous.State != "ADS_DIFFERENT" {
		return current.CredibleNegative
	}
	if current.Health != "HEALTHY" || current.AdsRoute.State != "SAME" || len(previous.AdsRoute.Different) == 0 {
		return false
	}
	for _, scope := range previous.AdsRoute.Different {
		if !slices.Contains(current.AdsRoute.Same, scope) {
			return false
		}
	}
	return true
}
