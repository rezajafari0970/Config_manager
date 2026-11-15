package egress

import (
	_ "embed"
	"encoding/json"
	"sort"
)

const ProfileVersion = "ads-profiles.v1"

// Profiles are operator-reviewed, compiled policy, never supplied by a receipt.
// Every member must independently qualify in BOTH exact variants. A partial
// profile cannot qualify by combining different successful members/variants.
type Profile struct {
	ID      string   `json:"id"`
	Domains []string `json:"domains"`
}
type ProfileDecision struct {
	ID      string   `json:"id"`
	State   string   `json:"state"`
	Domains []string `json:"domains"`
}

//go:embed profiles.json
var profileJSON []byte
var profiles = func() []Profile {
	var values []Profile
	if json.Unmarshal(profileJSON, &values) != nil || len(values) == 0 {
		panic("invalid Ads profiles")
	}
	ids := map[string]bool{}
	for _, p := range values {
		if p.ID == "" || ids[p.ID] || len(p.Domains) < 2 {
			panic("invalid Ads profile")
		}
		ids[p.ID] = true
		seen := map[string]bool{}
		for _, name := range p.Domains {
			t, ok := targetFor(name)
			if !ok || t.Role != "ads" || seen[name] {
				panic("invalid Ads profile member")
			}
			seen[name] = true
		}
	}
	return values
}()

func Profiles() []Profile {
	out := make([]Profile, len(profiles))
	for i, p := range profiles {
		out[i] = Profile{p.ID, append([]string(nil), p.Domains...)}
	}
	return out
}

// Deterministic compiled policy identity fences cached receipts after registry
// or profile changes even if the evaluator implementation version is unchanged.
func PolicyDigest() string {
	b, _ := json.Marshal(struct {
		Version  string
		Profiles []Profile
		Targets  []Target
		Delivery []DeliveryTarget
	}{ProfileVersion, Profiles(), Targets(), DeliveryTargets()})
	return Hash(b)
}

func evaluateProfiles(r Report, p Plan, hash string, now int64) Decision {
	m := ObserveRoutes(r, p, hash, now, Targets())
	d := Decision{State: "UNKNOWN", Reason: m.Reason, Route: "UNKNOWN", Scope: m.Scope,
		CountryConflict: IranCountryConflict(r, p, hash, now), EvidenceProfile: ProfileVersion, PolicyDigest: PolicyDigest(), AdsCoverage: m.Coverage, Domains: m.Domains,
		Sources: []string{}, ControlSources: []string{}, QualifiedProfiles: []string{}, QualifiedDomains: []string{}}
	if reason := routeBound(r, p, hash, now); reason != "" {
		if len(p.Variants) == 2 && (p.Variants[0].Name != "raw" || p.Variants[1].Name != "template") {
			d.Reason = "variant_binding_invalid"
		}
		if len(r.Checks) != 2 {
			switch r.Code {
			case "driver_session_failed", "direct_route_not_qualified", "direct_default_not_qualified", "endpoint_dns_failed", "ipv6_only_probe_pending", "core_rejected", "core_schema_mismatch", "missing_runtime_asset", "removed_allow_insecure", "unsupported_protocol_or_transport", "public_endpoint_requires_tls", "driver_subprocess_timeout", "driver_subprocess_failed", "driver_infrastructure_error", "runtime_start_failed", "network_retry_pending":
				d.Reason = r.Code
			}
		}
		return d
	}
	byDomain := map[string]DomainRoute{}
	for _, x := range m.Domains {
		byDomain[x.Domain] = x
	}
	qualified := map[string]bool{}
	for _, profile := range profiles {
		state, allShared := "SEPARATE", true
		for _, name := range profile.Domains {
			status := byDomain[name].State
			if status != "SEPARATE" {
				state = "UNRESOLVED"
			}
			if status != "SHARED" {
				allShared = false
			}
		}
		if allShared {
			state = "SHARED"
		}
		d.Profiles = append(d.Profiles, ProfileDecision{profile.ID, state, append([]string(nil), profile.Domains...)})
		if state == "SEPARATE" {
			d.QualifiedProfiles = append(d.QualifiedProfiles, profile.ID)
			for _, name := range profile.Domains {
				qualified[name] = true
			}
		}
	}
	for name := range qualified {
		d.QualifiedDomains = append(d.QualifiedDomains, name)
	}
	sort.Strings(d.QualifiedProfiles)
	sort.Strings(d.QualifiedDomains)
	connections := connectionCounts(r)
	sources, controls := map[string]bool{}, map[string]bool{}
	full := m.Coverage == "ALL_TESTED"
	for _, v := range p.Variants {
		for _, t := range registry {
			waves, attempted := map[int]bool{}, false
			for _, s := range r.Samples {
				if s.Variant != v.Name || s.Domain != t.Domain {
					continue
				}
				attempted = true
				if !routeValid(s, r) || connections[s.Connection] != 1 || s.ActualAds != (t.Role == "ads") {
					continue
				}
				waves[s.Wave] = true
				if t.Role == "ads" {
					sources[s.Source] = true
				} else {
					controls[s.Source] = true
				}
			}
			status := "MISSING"
			if t.Role == "ads" {
				for _, x := range byDomain[t.Domain].Variants {
					if x.Variant == v.Name {
						status = x.State
						d.Missing += max(0, 2-x.Paired)
					}
				}
			} else if len(waves) >= 2 {
				status = "CONTROL_OBSERVED"
			} else if attempted {
				status = "ROUTE_UNOBSERVABLE"
			}
			if t.Role == "control" && status != "CONTROL_OBSERVED" {
				full = false
			}
			d.Services = append(d.Services, ServiceDecision{Variant: v.Name, Domain: t.Domain, Group: t.Group, Status: status, Observations: len(waves), Required: t.Required})
		}
	}
	for s := range sources {
		d.Sources = append(d.Sources, s)
	}
	for s := range controls {
		d.ControlSources = append(d.ControlSources, s)
	}
	sort.Strings(d.Sources)
	sort.Strings(d.ControlSources)
	if len(d.Sources) > 0 {
		d.Route = "PARTIAL_DIFFERENCE"
	}
	if m.IranObserved || d.CountryConflict {
		d.State = "QUARANTINED"
		d.Reason = "ads_iran_quarantined"
		if !m.IranObserved {
			d.Reason = "ads_iran_conflict"
		}
		return d
	}
	if len(d.QualifiedProfiles) > 0 && m.Until > now {
		d.State = "RESIDENTIAL"
		d.Reason = "verified_ads_profile"
		d.Route = "DIFFERENT"
		d.Eligible = true
		d.Until = m.Until
		d.CoverageComplete = full
		return d
	}
	if m.State == "SHARED_OBSERVED" {
		d.State = "NO_SPLIT"
		d.Reason = "all_tested_ads_share_control"
		d.Route = "OVERLAP"
		d.Negative = true
	} else {
		d.Reason = "ads_profile_evidence_incomplete"
	}
	return d
}
