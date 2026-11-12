package egress

import (
	"encoding/hex"
	"sort"
	"strings"
)

const CountryVersion = "ads-country.v1"
const CountryWindow = int64(86400000)
const CountryDataAge = int64(45 * 86400000)

// Explicit ISO 3166-1 alpha-2 allowlist, plus the widely used XK code.
const countryCodes = "AD AE AF AG AI AL AM AO AQ AR AS AT AU AW AX AZ BA BB BD BE BF BG BH BI BJ BL BM BN BO BQ BR BS BT BV BW BY BZ CA CC CD CF CG CH CI CK CL CM CN CO CR CU CV CW CX CY CZ DE DJ DK DM DO DZ EC EE EG EH ER ES ET FI FJ FK FM FO FR GA GB GD GE GF GG GH GI GL GM GN GP GQ GR GS GT GU GW GY HK HM HN HR HT HU ID IE IL IM IN IO IQ IR IS IT JE JM JO JP KE KG KH KI KM KN KP KR KW KY KZ LA LB LC LI LK LR LS LT LU LV LY MA MC MD ME MF MG MH MK ML MM MN MO MP MQ MR MS MT MU MV MW MX MY MZ NA NC NE NF NG NI NL NO NP NR NU NZ OM PA PE PF PG PH PK PL PM PN PR PS PT PW PY QA RE RO RS RU RW SA SB SC SD SE SG SH SI SJ SK SL SM SN SO SR SS ST SV SX SY SZ TC TD TF TG TH TJ TK TL TM TN TO TR TT TV TW TZ UA UG UM US UY UZ VA VC VE VG VI VN VU WF WS XK YE YT ZA ZM ZW"

func CountryCodes() []string { return strings.Fields(countryCodes) }
func ValidCountry(code string) bool {
	return len(code) == 2 && strings.Contains(" "+countryCodes+" ", " "+code+" ")
}

type CountryProvider struct {
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
	Complete   bool              `json:"complete"`
	Status     string            `json:"status"`
	Countries  []string          `json:"countries"`
	Providers  []CountryProvider `json:"providers"`
	ValidUntil int64             `json:"valid_until"`
}

// This supplements, never replaces, the independently validated route decision.
// Every authenticated Ads observation must carry fresh, agreeing provider facts.
func Countries(r Report, route Decision, now int64) CountryResult {
	out := CountryResult{Status: "country_incomplete", Countries: []string{}, Providers: []CountryProvider{}}
	if !route.Eligible {
		out.Status = "route_unqualified"
		return out
	}
	countries := map[string]bool{}
	sources := map[string]string{}
	providers := map[string]CountryProvider{}
	until := route.Until
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
		if g.Version != CountryVersion || g.Status != "located" || !ValidCountry(g.Country) || g.At < r.Started || g.At > r.Finished+2000 || g.At > now+2000 || len(g.Providers) < 1 || len(g.Providers) > 4 {
			return out
		}
		seen := map[string]bool{}
		for _, p := range g.Providers {
			digest, e := hex.DecodeString(p.Dataset)
			if p.Name == "" || len(p.Name) > 48 || seen[p.Name] || e != nil || len(digest) != 32 || p.Country != g.Country || p.ReleasedAt <= 0 || p.ReleasedAt > g.At+86400000 || now-p.ReleasedAt >= CountryDataAge {
				return out
			}
			seen[p.Name] = true
			if expires := p.ReleasedAt + CountryDataAge; expires < until {
				until = expires
			}
			providers[p.Name+":"+p.Dataset+":"+p.Country] = p
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
