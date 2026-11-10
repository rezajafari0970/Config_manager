package residential

import (
	"net/netip"
	"slices"
)

func publicIP(s string) (netip.Addr, bool) {
	ip, e := netip.ParseAddr(s)
	return ip, e == nil && ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !ip.IsUnspecified()
}
func digestOK(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func recent(at, now int64, ttl int) bool { return at > 0 && at <= now+2000 && at+int64(ttl)*1000 > now }

func validObservation(o Observation, p Policy, t Trust, now int64) bool {
	ip, ok := publicIP(o.IP)
	if !ok || (o.Family == 4 && !ip.Is4()) || (o.Family == 6 && !ip.Is6()) || !recent(o.At, now, p.EvidenceTTLSeconds) || o.Outcome != "passed" {
		return false
	}
	c, ok := t.Capabilities[o.Capability]
	if !ok || c.Method != o.Method || c.Expires <= now || !digestOK(c.EvidenceSHA256) || !slices.Contains(c.Domains, o.Domain) || !slices.Contains(c.Transports, o.Transport) {
		return false
	}
	// A method plugin must check any extra protocol-specific attestation before
	// producing these fields. A nonce reflected on a different socket is invalid.
	return verifyAttestation(o, c) && o.PolicyHash == p.Digest() && digestOK(o.ConfigHash) && digestOK(o.RuntimeHash) && o.TLSVerified && o.OriginVerified && o.MeaningVerified && o.NonceBound && o.SameConnection && o.Connections == 1 && o.Redirects == 0 && o.ActualDomainRequest && len(o.ConnectionID) >= 16 && len(o.Nonce) >= 16 && digestOK(o.ResponseSHA256)
}

func Network(ip string, facts []IPFact, p Policy, t Trust, now int64) (string, int64) {
	groups := map[string]bool{}
	specialist := false
	class := ""
	country := ""
	var asn int64
	deadline := now + int64(p.EvidenceTTLSeconds)*1000
	for _, f := range facts {
		if f.IP != ip {
			continue
		}
		provider, ok := t.Providers[f.Source]
		if !ok || provider.Group == "" || !recent(f.At, now, p.IntelTTLSeconds) || f.Expires <= now || !digestOK(f.EvidenceSHA256) {
			continue
		}
		if !slices.Contains([]string{"Residential", "Mobile", "Datacenter"}, f.Class) {
			return "Unknown", 0
		}
		if class != "" && class != f.Class {
			return "Unknown", 0
		}
		if country != "" && f.Country != "" && country != f.Country || asn != 0 && f.ASN != 0 && asn != f.ASN {
			return "Unknown", 0
		}
		if f.Country != "" {
			country = f.Country
		}
		if f.ASN != 0 {
			asn = f.ASN
		}
		class = f.Class
		groups[provider.Group] = true
		specialist = specialist || provider.Specialist
		deadline = min(deadline, f.Expires, f.At+int64(p.IntelTTLSeconds)*1000)
	}
	if len(groups) < 2 || !specialist {
		return "Unknown", 0
	}
	return class, deadline
}

// Evaluate grants only the explicitly enumerated matrix; never all Google Ads.
// Matching control/Ads IPs do not disqualify an all-residential configuration.
func Evaluate(r Report, p Policy, t Trust, now int64) Decision {
	d := Decision{State: "UNKNOWN", Health: "UNKNOWN", Egress: "UNKNOWN", Network: "Unknown", Reason: "insufficient_evidence", Missing: []Scope{}}
	if r.Version != Version || r.PolicyHash != p.Digest() || !recent(r.Finished, now, p.EvidenceTTLSeconds) || r.Started > r.Finished || r.Started <= 0 {
		d.Reason = "stale_or_incompatible_report"
		return d
	}
	d.Egress, d.Network = evidenceDimensions(r, p, t, now)
	if !r.InfrastructureOK {
		d.Reason = "test_infrastructure_unavailable"
		return d
	}
	if len(r.Variants) != 2 {
		d.Reason = "raw_and_template_required"
		return d
	}
	variants := map[string]Variant{}
	for _, v := range r.Variants {
		if v.Name != "raw" && v.Name != "template" || variants[v.Name].Name != "" {
			d.Reason = "variant_mismatch"
			return d
		}
		variants[v.Name] = v
		if v.Core == "unsupported" {
			d.State = "UNSUPPORTED"
			d.Reason = "core_driver_unsupported"
			return d
		}
		if v.Core != "passed" || !digestOK(v.Hash) || !digestOK(v.RuntimeHash) {
			d.Reason = "core_evidence_missing"
			return d
		}
	}
	// Two administratively independent controls are required in each variant,
	// each wave. Their own egress IP does not become an Ads IP.
	controlOK := true
	for _, variant := range []string{"raw", "template"} {
		for wave := 1; wave <= p.Repeats; wave++ {
			groups := map[string]bool{}
			for _, o := range r.Controls {
				if o.Variant != variant || o.Wave != wave || o.Outcome != "passed" || !o.TLSVerified || !o.OriginVerified || !recent(o.At, now, p.EvidenceTTLSeconds) {
					continue
				}
				if o.Domain == "www.cloudflare.com" {
					groups["cloudflare"] = true
				}
				if o.Domain == "api.ipify.org" {
					groups["ipify"] = true
				}
			}
			if len(groups) < 2 {
				controlOK = false
			}
		}
	}
	if !controlOK {
		groups := map[string]bool{}
		for _, o := range r.InfrastructureControls {
			if o.Outcome == "passed" && o.TLSVerified && o.OriginVerified && recent(o.At, now, 60) {
				if o.Domain == "www.cloudflare.com" {
					groups["cloudflare"] = true
				}
				if o.Domain == "api.ipify.org" {
					groups["ipify"] = true
				}
			}
		}
		if len(groups) == 2 {
			for _, v := range variants {
				if v.Health == "UNHEALTHY" && repeatedControlFailure(r, v.Name, p) {
					d.Health = "UNHEALTHY"
					d.Reason = "repeated_connection_failure_with_fresh_controls"
					d.CredibleNegative = true
					return d
				}
			}
		}
		d.Reason = "independent_controls_incomplete"
		return d
	}
	d.Health = "HEALTHY"
	for _, v := range variants {
		if v.Health != "HEALTHY" {
			d.Health = v.Health
			d.Reason = "connection_not_healthy"
			d.CredibleNegative = false
			return d
		}
	}
	deadline := r.Started + int64(p.EvidenceTTLSeconds)*1000
	if deadline <= now {
		d.Reason = "expired_evidence"
		return d
	}
	nonResidential := false
	negativeClass := ""
	anyProven := false
	allPaths := true
	anyNetwork := false
	for _, variant := range []string{"raw", "template"} {
		v := variants[variant]
		paths := v.Paths
		if len(paths) == 0 {
			paths = []string{"unresolved"}
		}
		if !v.PathsComplete {
			allPaths = false
		}
		for _, path := range paths {
			for _, domain := range p.Domains {
				for _, family := range p.Families {
					for _, transport := range p.Transports {
						scope := Scope{variant, domain, family, transport, path}
						d.Required++
						waves := map[int]bool{}
						connections := map[string]bool{}
						ips := map[string]bool{}
						qualified := true
						for _, o := range r.Observations {
							if o.Scope != scope {
								continue
							}
							if o.ConfigHash != v.Hash || o.RuntimeHash != v.RuntimeHash || !validObservation(o, p, t, now) {
								qualified = false
								continue
							}
							anyProven = true
							deadline = min(deadline, o.At+int64(p.EvidenceTTLSeconds)*1000, t.Capabilities[o.Capability].Expires)
							class, until := Network(o.IP, r.IPFacts, p, t, now)
							if class != "Unknown" {
								anyNetwork = true
							}
							if class == "Mobile" || class == "Datacenter" {
								nonResidential = true
								if negativeClass == "" {
									negativeClass = class
								} else if negativeClass != class {
									negativeClass = "Unknown"
								}
							}
							if class != "Residential" {
								qualified = false
							} else {
								deadline = min(deadline, until)
							}
							if o.Wave < 1 || o.Wave > p.Repeats || connections[o.ConnectionID] {
								qualified = false
								continue
							}
							waves[o.Wave] = true
							connections[o.ConnectionID] = true
							ips[o.IP] = true
						}
						if len(ips) > 1 {
							d.RotationObserved = true
						}
						if qualified && len(waves) == p.Repeats {
							d.Covered++
						} else {
							d.Missing = append(d.Missing, scope)
						}
					}
				}
			}
		}
	}
	if anyProven {
		d.Egress = "OBSERVED"
	}
	if nonResidential {
		d.State = "NON_RESIDENTIAL"
		d.Network = negativeClass
		d.Reason = "verified_nonresidential_ads_sample"
		d.CredibleNegative = true
		return d
	}
	if d.Covered > 0 || anyProven || anyNetwork {
		d.State = "PARTIAL"
		d.Reason = "coverage_or_network_evidence_incomplete"
	}
	if d.Required > 0 && d.Covered == d.Required && allPaths && deadline > now {
		d.State = "ACCEPTED"
		d.Network = "Residential"
		d.Reason = "qualified_within_explicit_scope"
		d.Eligible = true
		d.ValidUntil = deadline
	} else if !allPaths {
		d.State = "PARTIAL"
		d.Reason = "selectable_paths_unproven"
	}
	return d
}

// These axes describe verified observations even when an independent health or
// coverage gate prevents acceptance. They never turn partial evidence into full
// publication or authorize deletion during an infrastructure outage.
func evidenceDimensions(r Report, p Policy, t Trust, now int64) (string, string) {
	observed := false
	class := ""
	conflict := false
	for _, o := range r.Observations {
		matched := false
		for _, v := range r.Variants {
			if v.Name == o.Variant && v.Hash == o.ConfigHash && v.RuntimeHash == o.RuntimeHash && slices.Contains(v.Paths, o.Path) {
				matched = true
			}
		}
		if !matched || !slices.Contains(p.Domains, o.Domain) || !slices.Contains(p.Families, o.Family) || !slices.Contains(p.Transports, o.Transport) || !validObservation(o, p, t, now) {
			continue
		}
		observed = true
		kind, _ := Network(o.IP, r.IPFacts, p, t, now)
		if kind == "Unknown" {
			conflict = true
		}
		if class != "" && class != kind {
			conflict = true
		}
		class = kind
	}
	if !observed {
		return "UNKNOWN", "Unknown"
	}
	if conflict || class == "" {
		class = "Unknown"
	}
	return "OBSERVED", class
}
func repeatedControlFailure(r Report, variant string, p Policy) bool {
	for wave := 1; wave <= p.Repeats; wave++ {
		domains := map[string]bool{}
		for _, o := range r.Controls {
			if o.Variant == variant && o.Wave == wave && o.Outcome == "failed" && !o.TLSVerified && (o.Domain == "www.cloudflare.com" || o.Domain == "api.ipify.org") {
				domains[o.Domain] = true
			}
		}
		if len(domains) != 2 {
			return false
		}
	}
	return true
}
