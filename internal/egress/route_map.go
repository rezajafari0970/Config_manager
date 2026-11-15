package egress

import (
	"encoding/hex"
	"sort"
)

const RouteMapVersion = "ads-route-map.v2"

type RouteVariant struct {
	Variant      string `json:"variant"`
	State        string `json:"state"`
	Observed     int    `json:"observed_waves"`
	Paired       int    `json:"paired_waves"`
	Overlap      int    `json:"overlap_waves"`
	SourceCount  int    `json:"source_count"`
	HealthyWaves int    `json:"healthy_waves"`
}

type DomainRoute struct {
	Domain   string         `json:"domain"`
	Group    string         `json:"group"`
	State    string         `json:"state"`
	Variants []RouteVariant `json:"variants"`
}

type RouteMap struct {
	Version          string        `json:"version"`
	State            string        `json:"state"`
	Reason           string        `json:"reason"`
	Domains          []DomainRoute `json:"domains"`
	SplitDomains     []string      `json:"split_domains"`
	SharedDomains    []string      `json:"shared_domains"`
	Unresolved       []string      `json:"unresolved_domains"`
	Coverage         string        `json:"ads_coverage"`
	Health           string        `json:"general_health"`
	Until            int64         `json:"until"`
	IranObserved     bool          `json:"iran_observed"`
	PublicationReady bool          `json:"publication_ready"`
	Scope            string        `json:"scope"`
}

func routeHexBytes(s string, n int) bool {
	b, err := hex.DecodeString(s)
	return err == nil && len(b) == n
}

func routeBound(r Report, p Plan, hash string, now int64) string {
	if r.Version != Version || r.Hash != hash || r.Started <= 0 || r.Started > r.Finished || r.Finished > now+2000 || r.Started < now-300000 || r.Finished < now-EvidenceTTL {
		return "stale_or_unbound_receipt"
	}
	if r.ResourcePressure {
		return "local_resource_pressure"
	}
	if !p.Complete || len(p.Variants) != 2 || len(r.Checks) != 2 || p.Variants[0].Name != "raw" || p.Variants[1].Name != "template" {
		return "exact_variants_incomplete"
	}
	if r.Waves < 2 || r.Waves > 3 || len(r.Samples) > 256 || len(r.Evidence) > 4 || Cores[r.Core] == "" || Cores[r.Core] != r.CoreSHA {
		return "driver_receipt_invalid"
	}
	for i, v := range p.Variants {
		c := r.Checks[i]
		if Hash(v.Config) != v.Hash || c.Name != v.Name || c.Hash != v.Hash || !c.Core || !routeHexBytes(c.RuntimeHash, 32) || p.Driver == "xray" && c.RuntimeHash != v.Hash {
			return "variant_binding_failed"
		}
	}
	return ""
}

func routeValid(s Sample, r Report) bool {
	return routeHexBytes(s.Source, 32) && routeHexBytes(s.Certificate, 32) && routeHexBytes(s.Response, 32) && routeHexBytes(s.Nonce, 16) && s.Nonce == s.Connection && (s.Family == 4 || s.Family == 6) && s.TLS && s.SameStream && s.Echo && s.Transport == "tcp" && s.Wave >= 1 && s.Wave <= r.Waves && s.At >= r.Started && s.At <= r.Finished+2000
}

// ObserveRoutes is the shared per-domain evidence engine. The compiled registry
// selects measurement destinations, not a hidden all-or-nothing decision rule.
// Publication remains a separate deployment policy: discovery cannot authorize it.
func ObserveRoutes(r Report, p Plan, hash string, now int64, targets []Target) RouteMap {
	out := RouteMap{Version: RouteMapVersion, State: "UNKNOWN", Coverage: "UNKNOWN", Health: "unknown", Domains: []DomainRoute{}, SplitDomains: []string{}, SharedDomains: []string{}, Unresolved: []string{}, Scope: "Finite TCP/HTTP1.1 source observations; exact raw and template; two paired same-family control groups including a non-Ads Google hostname in two waves per separated domain; rotating Ads exits allowed; no physical-proxy, ISP, QUIC or all-domain proof; route map only; publication requires a separately verified named profile"}
	// Preserve an independent IR veto even when the second variant is missing.
	out.IranObserved = Iran(r, p, hash, now) != nil
	if reason := routeBound(r, p, hash, now); reason != "" {
		out.Reason = reason
		if out.IranObserved {
			out.State = "IR_QUARANTINE"
		}
		return out
	}
	out.Health = Health(r, p, now)
	registry := map[string]Target{}
	for _, t := range targets {
		registry[t.Domain] = t
	}
	// Reject every instance of a replayed connection, not just the second copy.
	connections := connectionCounts(r)
	buckets := map[string][]Sample{}
	earliest := r.Finished
	for _, s := range r.Samples {
		t, ok := registry[s.Domain]
		if !ok || (s.Variant != "raw" && s.Variant != "template") || !routeValid(s, r) || s.ActualAds != (t.Role == "ads") || connections[s.Connection] != 1 {
			continue
		}
		buckets[s.Variant+"|"+s.Domain] = append(buckets[s.Variant+"|"+s.Domain], s)
		if s.At < earliest {
			earliest = s.At
		}
	}
	for _, t := range targets {
		if t.Role != "ads" {
			continue
		}
		d := DomainRoute{Domain: t.Domain, Group: t.Group, State: "UNRESOLVED"}
		for _, v := range p.Variants {
			rows := buckets[v.Name+"|"+t.Domain]
			observed, paired, overlap, healthy := map[int]bool{}, map[int]bool{}, map[int]bool{}, map[int]bool{}
			sources := map[string]bool{}
			poolOverlap, unsharedObservation := false, false
			for _, a := range rows {
				observed[a.Wave], sources[a.Source] = true, true
				exchange := a.ResponseComplete && a.RequestBytes > 0 && a.RequestBytes <= 32768 && a.ResponseBytes > 0 && a.ResponseBytes <= 131072
				if exchange {
					healthy[a.Wave] = true
				}
				groups := map[string]bool{}
				shared := false
				for _, c := range targets {
					if c.Role != "control" {
						continue
					}
					controls := buckets[v.Name+"|"+c.Domain]
					for _, b := range controls {
						if b.Family != a.Family {
							continue
						}
						if b.Source == a.Source {
							poolOverlap = true
							if b.Wave == a.Wave {
								overlap[a.Wave] = true
								shared = true
							}
						} else if b.Wave == a.Wave && routeStable(controls, a.Family) {
							groups[c.Group] = true
						}
					}
				}
				if !shared {
					unsharedObservation = true
				}
				if groups["google-search"] && len(groups) >= 2 && exchange {
					paired[a.Wave] = true
				}
			}
			x := RouteVariant{Variant: v.Name, State: "UNRESOLVED", Observed: len(observed), Paired: len(paired), Overlap: len(overlap), SourceCount: len(sources), HealthyWaves: len(healthy)}
			switch {
			case len(overlap) >= 2 && !unsharedObservation:
				x.State = "SHARED"
			case poolOverlap:
				x.State = "MIXED"
			case len(paired) >= 2 && len(healthy) >= 2:
				x.State = "SEPARATE"
			}
			d.Variants = append(d.Variants, x)
		}
		if d.Variants[0].State == "SEPARATE" && d.Variants[1].State == "SEPARATE" {
			d.State = "SEPARATE"
			out.SplitDomains = append(out.SplitDomains, d.Domain)
		} else if d.Variants[0].State == "SHARED" && d.Variants[1].State == "SHARED" {
			d.State = "SHARED"
			out.SharedDomains = append(out.SharedDomains, d.Domain)
		} else {
			if d.Variants[0].State != d.Variants[1].State {
				d.State = "VARIANT_COVERAGE_DIFFERS"
			}
			out.Unresolved = append(out.Unresolved, d.Domain)
		}
		out.Domains = append(out.Domains, d)
	}
	for _, list := range [][]string{out.SplitDomains, out.SharedDomains, out.Unresolved} {
		sort.Strings(list)
	}
	switch {
	case len(out.SplitDomains) == len(out.Domains) && len(out.Domains) > 0:
		out.State, out.Coverage, out.Reason = "ADS_SPLIT_OBSERVED", "ALL_TESTED", "repeated_separation_for_each_tested_ads_domain"
	case len(out.SplitDomains) > 0:
		out.State, out.Coverage, out.Reason = "PARTIAL_ADS_SPLIT", "PARTIAL", "separate_ads_domains_with_shared_or_unresolved_remainder"
	case len(out.SharedDomains) == len(out.Domains) && len(out.Domains) > 0:
		out.State, out.Coverage, out.Reason = "SHARED_OBSERVED", "ALL_TESTED_SHARED", "all_tested_ads_domains_repeatedly_overlap_controls"
	default:
		out.Reason = "per_domain_observations_incomplete"
	}
	out.Until = earliest + EvidenceTTL
	if out.IranObserved {
		out.State, out.Reason = "IR_QUARANTINE", "authenticated_ads_iran_observation"
	}
	return out
}

func routeStable(rows []Sample, family int) bool {
	sources, waves := map[string]bool{}, map[int]bool{}
	for _, s := range rows {
		if s.Family == family {
			sources[s.Source], waves[s.Wave] = true, true
		}
	}
	return len(sources) == 1 && len(waves) >= 2
}

// Every copy of a repeated connection is invalid, including other attempts.
// The driver serializes each attempt only once; namespaces never cross jobs.
func connectionCounts(r Report) map[string]int {
	counts := map[string]int{}
	for _, s := range r.Samples {
		counts[s.Connection]++
	}
	for _, part := range r.Evidence {
		for _, s := range part.Samples {
			counts[s.Connection]++
		}
	}
	return counts
}
