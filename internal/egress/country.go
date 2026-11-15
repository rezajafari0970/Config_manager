package egress

import (
	"encoding/hex"
	"sort"
	"strings"
)

const CountryVersion = "ads-country.v2"
const CountryWindow = int64(86400000)
const CountryDataAge = int64(45 * 86400000)

// Explicit ISO 3166-1 alpha-2 allowlist, plus the widely used XK code.
const countryCodes = "AD AE AF AG AI AL AM AO AQ AR AS AT AU AW AX AZ BA BB BD BE BF BG BH BI BJ BL BM BN BO BQ BR BS BT BV BW BY BZ CA CC CD CF CG CH CI CK CL CM CN CO CR CU CV CW CX CY CZ DE DJ DK DM DO DZ EC EE EG EH ER ES ET FI FJ FK FM FO FR GA GB GD GE GF GG GH GI GL GM GN GP GQ GR GS GT GU GW GY HK HM HN HR HT HU ID IE IL IM IN IO IQ IR IS IT JE JM JO JP KE KG KH KI KM KN KP KR KW KY KZ LA LB LC LI LK LR LS LT LU LV LY MA MC MD ME MF MG MH MK ML MM MN MO MP MQ MR MS MT MU MV MW MX MY MZ NA NC NE NF NG NI NL NO NP NR NU NZ OM PA PE PF PG PH PK PL PM PN PR PS PT PW PY QA RE RO RS RU RW SA SB SC SD SE SG SH SI SJ SK SL SM SN SO SR SS ST SV SX SY SZ TC TD TF TG TH TJ TK TL TM TN TO TR TT TV TW TZ UA UG UM US UY UZ VA VC VE VG VI VN VU WF WS XK YE YT ZA ZM ZW"

func CountryCodes() []string { return strings.Fields(countryCodes) }
func ValidCountry(code string) bool {
	return len(code) == 2 && strings.Contains(" "+countryCodes+" ", " "+code+" ")
}

type CountryProvider struct {
	Lineage    string `json:"lineage"`
	Kind       string `json:"kind"`
	CheckedAt  int64  `json:"checked_at,omitempty"`
	Name       string `json:"name"`
	Dataset    string `json:"dataset"`
	ReleasedAt int64  `json:"released_at"`
	Country    string `json:"country"`
}
type CountryFact struct {
	Version   string            `json:"version"`
	Status    string            `json:"status"`
	Country   string            `json:"country"`
	At        int64             `json:"at"`
	Providers []CountryProvider `json:"providers"`
}
type CountryResult struct {
	ProfileKey string            `json:"profile_key,omitempty"`
	Complete   bool              `json:"complete"`
	Status     string            `json:"status"`
	Countries  []string          `json:"countries"`
	Providers  []CountryProvider `json:"providers"`
	ValidUntil int64             `json:"valid_until"`
}

// This supplements, never replaces, the independently validated route decision.
// Every authenticated observation in the qualified profile needs fresh consensus.
// The independent Iran veto also inspects Ads outside that profile.
func Countries(r Report, route Decision, now int64) CountryResult {
	out := CountryResult{Status: "country_incomplete", Countries: []string{}, Providers: []CountryProvider{}}
	if route.CountryConflict {
		out.Status = "country_iran_conflict"
		return out
	}
	if !route.Eligible {
		out.Status = "route_unqualified"
		return out
	}
	// Safety still inspects ALL measured Ads, including shared/nonqualifying ones.
	for _, s := range r.Samples {
		if !s.ActualAds || !validSource(s, r) {
			continue
		}
		one := r
		one.Samples = []Sample{s}
		check := observedCountries(one, route.Until, now)
		if check.Status == "country_iran_conflict" || check.Complete && len(check.Countries) == 1 && check.Countries[0] == "IR" {
			out.Status = "country_iran_conflict"
			return out
		}
	}
	// A second route-qualified profile with unresolved GeoIP cannot suppress a
	// fully located profile. Keep every observation within each qualifying profile.
	qualified := map[string]bool{}
	for _, id := range route.QualifiedProfiles {
		qualified[id] = true
	}
	countries := map[string]bool{}
	providers := map[string]CountryProvider{}
	keys := []string{}
	until := route.Until
	for _, profile := range profiles {
		if !qualified[profile.ID] {
			continue
		}
		members := map[string]bool{}
		for _, name := range profile.Domains {
			members[name] = true
		}
		filtered := r
		filtered.Samples = nil
		for _, sample := range r.Samples {
			if members[sample.Domain] {
				filtered.Samples = append(filtered.Samples, sample)
			}
		}
		g := observedCountries(filtered, route.Until, now)
		if !g.Complete {
			continue
		}
		keys = append(keys, profile.ID)
		until = min(until, g.ValidUntil)
		for _, country := range g.Countries {
			countries[country] = true
		}
		for _, provider := range g.Providers {
			providers[provider.Name+":"+provider.Dataset+":"+provider.Country] = provider
		}
	}
	if len(keys) == 0 {
		return out
	}
	sort.Strings(keys)
	out.ProfileKey = strings.Join(keys, ",")
	for country := range countries {
		out.Countries = append(out.Countries, country)
	}
	sort.Strings(out.Countries)
	providerKeys := []string{}
	for k := range providers {
		providerKeys = append(providerKeys, k)
	}
	sort.Strings(providerKeys)
	for _, k := range providerKeys {
		out.Providers = append(out.Providers, providers[k])
	}
	out.Complete = true
	out.Status = "located"
	out.ValidUntil = until

	return out
}

// Country consensus independent of the route classifier, shared by the
// publication policy and the Iran deny guard.
func observedCountries(r Report, until, now int64) CountryResult {
	out := CountryResult{Status: "country_incomplete", Countries: []string{}, Providers: []CountryProvider{}}
	countries := map[string]bool{}
	sources := map[string]string{}
	providers := map[string]CountryProvider{}
	count := 0
	for _, s := range r.Samples {
		if !s.ActualAds || !validSource(s, r) {
			continue
		}
		domain := false
		for _, d := range Domains {
			if s.Domain == d {
				domain = true
			}
		}
		if !domain || (s.Variant != "raw" && s.Variant != "template") {
			continue
		}
		g := s.Geo
		if g.Version != CountryVersion || g.Status != "located" || !ValidCountry(g.Country) || g.At < r.Started || g.At > r.Finished+2000 || g.At > now+2000 || len(g.Providers) < 2 || len(g.Providers) > 8 {
			return out
		}
		seen := map[string]bool{}
		lineages := map[string]string{}
		for _, p := range g.Providers {
			digest, err := hex.DecodeString(p.Dataset)
			if p.Name == "" || len(p.Name) > 48 || seen[p.Name] || p.Lineage == "" || len(p.Lineage) > 48 || err != nil || len(digest) != 32 || !ValidCountry(p.Country) {
				return out
			}
			seen[p.Name] = true
			expiry := int64(0)
			if p.Kind == "database" {
				if p.ReleasedAt <= 0 || p.ReleasedAt > g.At+86400000 {
					return out
				}
				expiry = p.ReleasedAt + CountryDataAge
			} else if p.Kind == "live" && p.Name == "geojs" && p.Lineage == "maxmind" {
				if p.CheckedAt <= 0 || p.CheckedAt > g.At+2000 || p.ReleasedAt != 0 {
					return out
				}
				expiry = p.CheckedAt + CountryWindow
			} else {
				return out
			}
			if expiry <= now {
				return out
			}
			if expiry < until {
				until = expiry
			}
			if old, ok := lineages[p.Lineage]; ok && old != p.Country {
				lineages[p.Lineage] = "conflict"
			} else if !ok {
				lineages[p.Lineage] = p.Country
			}
			providers[p.Name+":"+p.Dataset+":"+p.Country] = p
		}
		votes, total := 0, 0
		for _, country := range lineages {
			if country != "conflict" {
				total++
				if country == g.Country {
					votes++
				}
			}
		}
		// A fresh minority IR observation blocks publication without being promoted
		// to a confirmed IR observation for the sticky deny guard.
		for _, country := range lineages {
			if country == "IR" && g.Country != "IR" {
				out.Status = "country_iran_conflict"
				return out
			}
		}
		if votes < 2 || votes*2 <= total {
			out.Status = "country_conflict"
			return out
		}

		if previous := sources[s.Source]; previous != "" && previous != g.Country {
			out.Status = "country_conflict"
			return out
		}
		sources[s.Source] = g.Country
		countries[g.Country] = true
		count++
	}
	if count == 0 || until <= now {
		return out
	}
	for c := range countries {
		out.Countries = append(out.Countries, c)
	}
	sort.Strings(out.Countries)
	keys := []string{}
	for k := range providers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out.Providers = append(out.Providers, providers[k])
	}
	out.Complete = true
	out.Status = "located"
	out.ValidUntil = until
	return out
}
