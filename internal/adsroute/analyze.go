// Package adsroute maps observed Ads routes without the legacy global veto.
// It consumes authenticated driver receipts; it does not infer a physical proxy
// topology, ISP class, or all-domain coverage from a finite source observation.
package adsroute

import (
	"encoding/hex"
	"sort"

	"github.com/rezajafari0970/Config_manager/internal/egress"
)

const Version = "ads-route-map.v1"

type VariantResult struct {
	Variant      string `json:"variant"`
	State        string `json:"state"`
	Observed     int    `json:"observed_waves"`
	Paired       int    `json:"paired_waves"`
	Overlap      int    `json:"overlap_waves"`
	SourceCount  int    `json:"source_count"`
	HealthyWaves int    `json:"healthy_waves"`
}

type DomainResult struct {
	Domain   string          `json:"domain"`
	Group    string          `json:"group"`
	State    string          `json:"state"`
	Variants []VariantResult `json:"variants"`
}

type Result struct {
	Version          string         `json:"version"`
	State            string         `json:"state"`
	Reason           string         `json:"reason"`
	Domains          []DomainResult `json:"domains"`
	SplitDomains     []string       `json:"split_domains"`
	SharedDomains    []string       `json:"shared_domains"`
	Unresolved       []string       `json:"unresolved_domains"`
	Coverage         string         `json:"ads_coverage"`
	Health           string         `json:"general_health"`
	Until            int64          `json:"until"`
	IranObserved     bool           `json:"iran_observed"`
	PublicationReady bool           `json:"publication_ready"`
	Scope            string         `json:"scope"`
}

func hexBytes(s string, n int) bool {
	b, err := hex.DecodeString(s)
	return err == nil && len(b) == n
}

func bound(r egress.Report, p egress.Plan, hash string, now int64) string {
	if r.Version != egress.Version || r.Hash != hash || r.Started <= 0 || r.Started > r.Finished || r.Finished > now+2000 || r.Started < now-300000 || r.Finished < now-egress.EvidenceTTL {
		return "stale_or_unbound_receipt"
	}
	if r.ResourcePressure {
		return "local_resource_pressure"
	}
	if !p.Complete || len(p.Variants) != 2 || len(r.Checks) != 2 || p.Variants[0].Name != "raw" || p.Variants[1].Name != "template" {
		return "exact_variants_incomplete"
	}
	if r.Waves < 2 || r.Waves > 3 || len(r.Samples) > 256 || egress.Cores[r.Core] == "" || egress.Cores[r.Core] != r.CoreSHA {
		return "driver_receipt_invalid"
	}
	for i, v := range p.Variants {
		c := r.Checks[i]
		if egress.Hash(v.Config) != v.Hash || c.Name != v.Name || c.Hash != v.Hash || !c.Core || !hexBytes(c.RuntimeHash, 32) || p.Driver == "xray" && c.RuntimeHash != v.Hash {
			return "variant_binding_failed"
		}
	}
	return ""
}

func valid(s egress.Sample, r egress.Report) bool {
	return hexBytes(s.Source, 32) && hexBytes(s.Certificate, 32) && hexBytes(s.Response, 32) && hexBytes(s.Nonce, 16) && s.Nonce == s.Connection && (s.Family == 4 || s.Family == 6) && s.TLS && s.SameStream && s.Echo && s.Transport == "tcp" && s.Wave >= 1 && s.Wave <= r.Waves && s.At >= r.Started && s.At <= r.Finished+2000
}

// Analyze is independent of egress.Evaluate and its cached verdicts. The registry
// selects measurement destinations, not a hidden all-or-nothing decision rule.
// Publication remains a separate deployment policy: discovery cannot authorize it.
func Analyze(r egress.Report, p egress.Plan, hash string, now int64) Result {
	out := Result{Version: Version, State: "UNKNOWN", Coverage: "UNKNOWN", Health: "unknown", Domains: []DomainResult{}, SplitDomains: []string{}, SharedDomains: []string{}, Unresolved: []string{}, Scope: "Finite TCP/HTTP1.1 source observations; exact raw and template; two paired same-family control groups including a non-Ads Google hostname in two waves per separated domain; rotating Ads exits allowed; no physical-proxy, ISP, QUIC or all-domain proof; discovery only, no publication authorization"}
	// Preserve an independent IR veto even when the second variant is missing.
	out.IranObserved = egress.Iran(r, p, hash, now) != nil
	if reason := bound(r, p, hash, now); reason != "" {
		out.Reason = reason
		if out.IranObserved {
			out.State = "IR_QUARANTINE"
		}
		return out
	}
	out.Health = egress.Health(r, p, now)
	targets := Targets()
	registry := map[string]egress.Target{}
	for _, t := range targets {
		registry[t.Domain] = t
	}
	// Reject every instance of a replayed connection, not just the second copy.
	connections := map[string]int{}
	for _, s := range r.Samples {
		connections[s.Connection]++
	}
	buckets := map[string][]egress.Sample{}
	earliest := r.Finished
	for _, s := range r.Samples {
		t, ok := registry[s.Domain]
		if !ok || (s.Variant != "raw" && s.Variant != "template") || !valid(s, r) || s.ActualAds != (t.Role == "ads") || connections[s.Connection] != 1 {
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
		d := DomainResult{Domain: t.Domain, Group: t.Group, State: "UNRESOLVED"}
		for _, v := range p.Variants {
			rows := buckets[v.Name+"|"+t.Domain]
			observed, paired, overlap, healthy := map[int]bool{}, map[int]bool{}, map[int]bool{}, map[int]bool{}
			sources := map[string]bool{}
			poolOverlap := false
			for _, a := range rows {
				observed[a.Wave], sources[a.Source] = true, true
				if a.ResponseComplete && a.RequestBytes > 0 && a.RequestBytes <= 32768 && a.ResponseBytes > 0 && a.ResponseBytes <= 131072 {
					healthy[a.Wave] = true
				}
				groups := map[string]bool{}
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
							}
						} else if b.Wave == a.Wave && stable(controls, a.Family) {
							groups[c.Group] = true
						}
					}
				}
				if groups["google-search"] && len(groups) >= 2 {
					paired[a.Wave] = true
				}
			}
			x := VariantResult{Variant: v.Name, State: "UNRESOLVED", Observed: len(observed), Paired: len(paired), Overlap: len(overlap), SourceCount: len(sources), HealthyWaves: len(healthy)}
			switch {
			case len(overlap) >= 2:
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
	out.Until = earliest + egress.EvidenceTTL
	if out.IranObserved {
		out.State, out.Reason = "IR_QUARANTINE", "authenticated_ads_iran_observation"
	}
	return out
}

func stable(rows []egress.Sample, family int) bool {
	sources, waves := map[string]bool{}, map[int]bool{}
	for _, s := range rows {
		if s.Family == family {
			sources[s.Source], waves[s.Wave] = true, true
		}
	}
	return len(sources) == 1 && len(waves) >= 2
}
