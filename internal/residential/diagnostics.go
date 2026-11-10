package residential

import "sort"

// RouteDiagnostic is an observation summary, never an acceptance/delete vote.
// Candidate IPs remain labelled unverified even when geography corroborates.
type RouteDiagnostic struct {
	Scope
	Status         string   `json:"status"`
	Samples        int      `json:"samples"`
	Waves          int      `json:"waves"`
	Compared       int      `json:"compared"`
	Different      int      `json:"different"`
	Same           int      `json:"same"`
	ASNChanges     int      `json:"asn_changes"`
	CountryChanges int      `json:"country_changes"`
	Rotation       bool     `json:"rotation"`
	IPs            []string `json:"ips"`
	Metadata       []IPFact `json:"metadata,omitempty"`
}

func controlGroup(domain string) string {
	switch domain {
	case "www.cloudflare.com":
		return "cloudflare"
	case "api.ipify.org", "api64.ipify.org":
		return "ipify"
	}
	return ""
}

func metadataGroup(source string, t Trust) string {
	switch source {
	case "metadata-ipinfo":
		return "ipinfo"
	case "metadata-ipwhois":
		return "ipwhois"
	}
	return t.Providers[source].Group
}

func corroboratedMetadata(ip string, facts []IPFact, p Policy, t Trust, now int64) (string, int64, []IPFact) {
	groups := map[string]bool{}
	country := ""
	var asn int64
	conflict := false
	selected := []IPFact{}
	for _, f := range facts {
		group := metadataGroup(f.Source, t)
		if f.IP != ip || group == "" || !recent(f.At, now, p.IntelTTLSeconds) || f.Expires <= now || !digestOK(f.EvidenceSHA256) {
			continue
		}
		f.MAC = ""
		selected = append(selected, f)
		if f.Country == "" || f.ASN == 0 {
			continue
		}
		if country != "" && country != f.Country || asn != 0 && asn != f.ASN {
			conflict = true
		}
		country = f.Country
		asn = f.ASN
		groups[group] = true
	}
	if conflict || len(groups) < 2 {
		return "", 0, selected
	}
	return country, asn, selected
}

// Caller authenticates IPFacts before use. Compare the same family, transport,
// variant and wave against BOTH independent controls. A different IP can be
// rotation; matching IPs may still be residential. Neither grants acceptance.
func RouteDiagnostics(r Report, p Policy, t Trust, now int64) []RouteDiagnostic {
	groups := map[string]*RouteDiagnostic{}
	waves := map[string]map[int]bool{}
	ips := map[string]map[string]bool{}
	for _, o := range r.Observations {
		ip, ok := publicIP(o.IP)
		if !ok || !recent(o.At, now, p.EvidenceTTLSeconds) || !o.TLSVerified || !o.ActualDomainRequest || !o.SameConnection || !o.NonceBound || o.Connections != 1 || o.Redirects != 0 || !digestOK(o.ResponseSHA256) || o.Wave < 1 || o.Wave > p.Repeats {
			continue
		}
		if o.Family != 4 && o.Family != 6 || o.Family == 4 && !ip.Is4() || o.Family == 6 && !ip.Is6() {
			continue
		}
		key := o.Scope.Key()
		d := groups[key]
		if d == nil {
			d = &RouteDiagnostic{Scope: o.Scope, Status: "ATTRIBUTED", IPs: []string{}}
			groups[key] = d
			waves[key] = map[int]bool{}
			ips[key] = map[string]bool{}
		}
		if !validObservation(o, p, t, now) {
			d.Status = "CANDIDATE_ONLY"
		}
		d.Samples++
		waves[key][o.Wave] = true
		ips[key][o.IP] = true
		controls := map[string]Observation{}
		for _, c := range r.Controls {
			group := controlGroup(c.Domain)
			ci, valid := publicIP(c.IP)
			if group == "" || !valid || ci.Is4() != ip.Is4() || c.Variant != o.Variant || c.Transport != o.Transport || c.Wave != o.Wave || c.Outcome != "passed" || !c.TLSVerified || !c.OriginVerified || !recent(c.At, now, p.EvidenceTTLSeconds) {
				continue
			}
			controls[group] = c
		}
		if len(controls) != 2 {
			continue
		}
		d.Compared++
		different := true
		asnDifferent := true
		countryDifferent := true
		country, asn, _ := corroboratedMetadata(o.IP, r.IPFacts, p, t, now)
		for _, c := range controls {
			if c.IP == o.IP {
				different = false
			}
			cc, ca, _ := corroboratedMetadata(c.IP, r.IPFacts, p, t, now)
			if asn == 0 || ca == 0 || asn == ca {
				asnDifferent = false
			}
			if country == "" || cc == "" || country == cc {
				countryDifferent = false
			}
		}
		if different {
			d.Different++
		} else {
			d.Same++
		}
		if asnDifferent {
			d.ASNChanges++
		}
		if countryDifferent {
			d.CountryChanges++
		}
	}
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	result := make([]RouteDiagnostic, 0, len(keys))
	for _, key := range keys {
		d := groups[key]
		d.Waves = len(waves[key])
		for ip := range ips[key] {
			d.IPs = append(d.IPs, ip)
			_, _, facts := corroboratedMetadata(ip, r.IPFacts, p, t, now)
			d.Metadata = append(d.Metadata, facts...)
		}
		sort.Strings(d.IPs)
		d.Rotation = len(d.IPs) > 1
		result = append(result, *d)
	}
	return result
}
