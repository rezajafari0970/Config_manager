// Package egress owns the new, finite-observation Ads investigation contract.
// Residential is the project label for a repeatedly observed dedicated Ads egress.
package egress

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/compat"
	"github.com/rezajafari0970/Config_manager/internal/convert"
	"github.com/rezajafari0970/Config_manager/internal/probe"

	"sort"
	"time"
)

const Version = "ads-egress.v5"

// Keep slots within the dedicated 32-address sandbox block.
const MaxWorkers = 31
const EvidenceTTL = int64(600000)

var Domains = []string{"googleads.g.doubleclick.net", "pubads.g.doubleclick.net", "securepubads.g.doubleclick.net"}
var Cores = map[string]string{
	"xray-26.10.31": "cc4b84d3069569f297a61f04b914f2bb37e9d202fd7b84cb50def1fc30fd40e1",
	"xray-26.11.9":  "e87bce586acdb9136aa5b1f1d9c8db9f1ea1d145b6d86368c7f6f662f9f6a66d",
	"xray-26.7.28":  "64d46afb80adea1bf97a0d467e83f4a9ac1ebd0995891e84bca3f1a1d1affb1d",
	"xray-26.5.9":   "b2c525e082cf2fef460499c88838d355f0b9bfb5a00bdb3eaa99b6af63825006",
}

func Hash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

type Variant struct {
	Name   string `json:"name"`
	Hash   string `json:"hash"`
	Config []byte `json:"config"`
}
type Plan struct {
	Driver   string    `json:"driver"`
	Code     string    `json:"code"`
	Complete bool      `json:"complete"`
	Variants []Variant `json:"variants"`
	Format   string    `json:"format"`
}

func Prepare(raw []byte) Plan {
	p := probe.Prepare(raw)
	for _, c := range p.Candidates {
		common, e := convert.Apply(c.Config)
		if e != nil {
			continue
		}
		original := c.Config
		if p.Kind == "json" && json.Valid(raw) {
			original = raw
		}
		complete := singlePath(original)
		return Plan{Driver: "xray", Code: "prepared", Complete: complete, Format: "xray", Variants: []Variant{{"raw", Hash(original), original}, {"template", Hash(common), common}}}
	}
	if n, ok := native(raw); ok {
		return n
	}
	return Plan{Driver: "none", Code: p.Code, Variants: []Variant{}}
}

// Inspect a private logical copy; raw and emitted bytes are never rewritten here.
func singlePath(raw []byte) bool {
	var root compat.M
	if json.Unmarshal(raw, &root) != nil {
		return false
	}
	if _, err := compat.Logical(root); err != nil {
		return false
	}
	b, err := json.Marshal(root)
	if err != nil {
		return false
	}
	var x struct {
		Outbounds []struct {
			Tag            string          `json:"tag"`
			Protocol       string          `json:"protocol"`
			ProxySettings  json.RawMessage `json:"proxySettings"`
			StreamSettings struct {
				Sockopt struct {
					DialerProxy string `json:"dialerProxy"`
				} `json:"sockopt"`
			} `json:"streamSettings"`
		} `json:"outbounds"`
		Routing struct {
			Balancers []json.RawMessage `json:"balancers"`
			Rules     []json.RawMessage `json:"rules"`
		} `json:"routing"`
	}
	if json.Unmarshal(b, &x) != nil || len(x.Routing.Balancers) > 0 || len(x.Outbounds) == 0 {
		return false
	}
	tags := map[string]string{}
	proxy := ""
	n := 0
	for _, o := range x.Outbounds {
		if o.Tag != "" {
			if _, exists := tags[o.Tag]; exists {
				return false
			}
			tags[o.Tag] = o.Protocol
		}
		if o.Protocol != "freedom" && o.Protocol != "blackhole" && o.Protocol != "dns" {
			n++
			proxy = o.Tag
		}
	}
	if n != 1 || x.Outbounds[0].Protocol == "freedom" || x.Outbounds[0].Protocol == "blackhole" || x.Outbounds[0].Protocol == "dns" {
		return false
	}
	for _, o := range x.Outbounds {
		if len(o.ProxySettings) > 0 && string(o.ProxySettings) != "{}" && string(o.ProxySettings) != "null" {
			return false
		}
		dial := o.StreamSettings.Sockopt.DialerProxy
		// DNS forwarded through the sole proxy is not a second egress path.
		if dial != "" && !(o.Protocol == "dns" && proxy != "" && dial == proxy) {
			return false
		}
	}
	for _, rawRule := range x.Routing.Rules {
		var rule struct {
			OutboundTag string `json:"outboundTag"`
			BalancerTag string `json:"balancerTag"`
		}
		if json.Unmarshal(rawRule, &rule) != nil {
			return false
		}
		if rule.BalancerTag != "" {
			return false
		}
		if rule.OutboundTag != "" {
			kind, ok := tags[rule.OutboundTag]
			if !ok || kind == "freedom" && !dnsBypass(rawRule) {
				return false
			}
		}
	}
	return true
}

type Job struct {
	Hash     string `json:"hash"`
	Raw      []byte `json:"raw"`
	Kind     string `json:"kind"`
	Token    string `json:"token"`
	Revision int64  `json:"revision"`
	Issue    string `json:"issue"`
	Deep     bool   `json:"deep"`
	Attempt  int    `json:"attempt"`
}
type Sample struct {
	Geo                 CountryFact `json:"geo,omitempty"`
	Variant             string      `json:"variant"`
	Domain              string      `json:"domain"`
	Wave                int         `json:"wave"`
	At                  int64       `json:"at"`
	Source              string      `json:"source"`
	Family              int         `json:"family"`
	Transport           string      `json:"transport"`
	Nonce               string      `json:"nonce"`
	Connection          string      `json:"connection"`
	TLS                 bool        `json:"tls"`
	SameStream          bool        `json:"same_stream"`
	Echo                bool        `json:"echo"`
	ActualAds           bool        `json:"actual_ads"`
	Code                string      `json:"code"`
	Certificate         string      `json:"certificate"`
	Response            string      `json:"response"`
	FirstStatus         int         `json:"first_status"`
	EchoStatus          int         `json:"echo_status"`
	DiagnosticAuthority bool        `json:"diagnostic_authority"`
}
type Check struct {
	Name        string `json:"name"`
	Hash        string `json:"hash"`
	RuntimeHash string `json:"runtime_hash"`
	Core        bool   `json:"core"`
	Code        string `json:"code"`
}
type Attempt struct {
	Core   string             `json:"core"`
	Code   string             `json:"code"`
	MS     float64            `json:"ms"`
	Stages map[string]float64 `json:"stages,omitempty"`
}
type Report struct {
	Version        string    `json:"version"`
	Hash           string    `json:"hash"`
	Started        int64     `json:"started"`
	Finished       int64     `json:"finished"`
	Core           string    `json:"core"`
	CoreSHA        string    `json:"core_sha"`
	Infrastructure bool      `json:"infrastructure"`
	Code           string    `json:"code"`
	Checks         []Check   `json:"checks"`
	Samples        []Sample  `json:"samples"`
	Attempts       []Attempt `json:"attempts"`
	WallMS         float64   `json:"wall_ms"`
	Waves          int       `json:"waves"`
}
type Completion struct {
	Job    Job    `json:"job"`
	Report Report `json:"report"`
}
type Decision struct {
	State          string   `json:"state"`
	Reason         string   `json:"reason"`
	Route          string   `json:"route"`
	Scope          string   `json:"scope"`
	Sources        []string `json:"sources"`
	ControlSources []string `json:"control_sources"`
	Eligible       bool     `json:"eligible"`
	Negative       bool     `json:"negative"`
	Until          int64    `json:"until"`
	Missing        int      `json:"missing"`
}

// Sources are job-local HMACs. The random key and plaintext IP never leave the helper.
func validSource(s Sample, r Report) bool {
	b, err := hex.DecodeString(s.Source)
	return err == nil && len(b) == 32 && (s.Family == 4 || s.Family == 6) && s.TLS && s.SameStream && s.Echo && s.Transport == "tcp" && len(s.Nonce) == 32 && s.Nonce == s.Connection && len(s.Certificate) == 64 && len(s.Response) == 64 && s.At >= r.Started && s.At <= r.Finished+2000
}
func controlGroup(domain string) string {
	switch domain {
	case "1.1.1.1", "www.cloudflare.com", "one.one.one.one":
		return "cloudflare"
	case "api.ipify.org", "api6.ipify.org":
		return "ipify"
	}
	return ""
}
func Evaluate(r Report, p Plan, hash string, now int64) Decision {
	d := Decision{State: "UNKNOWN", Reason: "incomplete_observations", Route: "UNKNOWN", Scope: "Project Residential = repeated Ads split egress; three DoubleClick domains; TCP; paired family; raw and template; authenticated peer report, not full hop topology or ISP class", Sources: []string{}, ControlSources: []string{}}
	if r.Version != Version || r.Hash != hash || r.Started <= 0 || r.Started > r.Finished || r.Finished > now+2000 || r.Started < now-300000 || r.Finished < now-EvidenceTTL {
		d.Reason = "stale_or_unbound_report"
		return d
	}
	if len(p.Variants) != 2 {
		d.State = "UNSUPPORTED"
		d.Reason = "driver_unavailable"
		return d
	}
	if r.Code == "network_retry_pending" {
		d.Reason = r.Code
		return d
	}
	rawOnly := r.Code == "raw_shared_egress" && len(r.Checks) == 1
	if (!rawOnly && len(r.Checks) != 2) || Cores[r.Core] == "" || Cores[r.Core] != r.CoreSHA {
		d.Reason = "core_evidence_missing"
		switch r.Code {
		case "direct_route_not_qualified", "direct_default_not_qualified", "endpoint_dns_failed", "ipv6_only_probe_pending", "core_rejected", "core_schema_mismatch", "missing_runtime_asset", "removed_allow_insecure", "unsupported_protocol_or_transport", "public_endpoint_requires_tls", "driver_subprocess_timeout", "driver_infrastructure_error", "runtime_start_failed":
			d.Reason = r.Code
		}
		return d
	}
	variants := p.Variants
	if rawOnly {
		variants = variants[:1]
		d.Scope += "; raw-only shared-egress rejection"
	}
	for i, v := range variants {
		c := r.Checks[i]
		if c.Name != v.Name || c.Hash != v.Hash || !c.Core || len(c.RuntimeHash) != 64 || p.Driver == "xray" && c.RuntimeHash != v.Hash {
			d.Reason = "core_or_variant_failed"
			return d
		}
	}
	if !p.Complete {
		d.Reason = "multiple_or_unresolved_paths"
		return d
	}
	if r.Waves < 2 || r.Waves > 3 {
		d.Reason = "invalid_wave_count"
		return d
	}
	adsFamilies := map[int]bool{}
	ads := map[string]bool{}
	controls := map[string]bool{}
	families := map[int]map[string]bool{}
	seen := map[string]bool{}
	repeats := map[string]int{}
	controlFailed := map[string]bool{}
	controlSuccess := 0
	healthMissing := 0
	ambiguous := false
	for _, variant := range variants {
		v := variant.Name
		for wave := 1; wave <= r.Waves; wave++ {
			paired := map[int]map[string]string{}
			for _, s := range r.Samples {
				group := controlGroup(s.Domain)
				if s.Variant != v || s.Wave != wave || group == "" {
					continue
				}
				if s.Code == "connection_timeout" || s.Code == "socks_connect" || s.Code == "connection_failed" {
					controlFailed[v+string(rune(wave))+group] = true
				}
				if !validSource(s, r) || seen[s.Connection] {
					continue
				}
				seen[s.Connection] = true
				controlSuccess++
				if paired[s.Family] == nil {
					paired[s.Family] = map[string]string{}
				}
				if old := paired[s.Family][group]; old != "" && old != s.Source {
					ambiguous = true
				}
				paired[s.Family][group] = s.Source
				if families[s.Family] == nil {
					families[s.Family] = map[string]bool{}
				}
				families[s.Family][s.Source] = true
				controls[s.Source] = true
			}
			healthy := false
			for _, m := range paired {
				if len(m) == 2 {
					healthy = true
				}
			}
			if !healthy {
				healthMissing++
			}
			for _, domain := range Domains {
				valid := false
				for _, s := range r.Samples {
					if s.Variant != v || s.Wave != wave || s.Domain != domain || !validSource(s, r) || !s.ActualAds || seen[s.Connection] {
						continue
					}
					seen[s.Connection] = true
					// Never ignore an observed conflicting Ads source merely because its controls failed.
					ads[s.Source] = true
					adsFamilies[s.Family] = true
					if len(paired[s.Family]) == 2 {
						valid = true
					}
				}
				if valid {
					repeats[v+"|"+domain]++
				} else {
					d.Missing++
				}
			}
		}
	}
	for x := range ads {
		d.Sources = append(d.Sources, x)
	}
	for x := range controls {
		d.ControlSources = append(d.ControlSources, x)
	}
	sort.Strings(d.Sources)
	sort.Strings(d.ControlSources)
	stable := len(controls) > 0 && !ambiguous
	for _, m := range families {
		if len(m) != 1 {
			stable = false
		}
	}
	for family := range adsFamilies {
		if len(families[family]) != 1 {
			stable = false
		}
	}
	if stable && len(ads) > 0 {
		d.Route = "DIFFERENT"
		for x := range ads {
			if controls[x] {
				d.Route = "OVERLAP"
			}
		}
	}
	repeated := true
	for _, variant := range variants {
		v := variant.Name
		for _, domain := range Domains {
			if repeats[v+"|"+domain] < 2 {
				repeated = false
			}
		}
	}
	if !repeated {
		d.Reason = "ads_peer_echo_incomplete"
		if healthMissing > 0 {
			d.Reason = "controls_incomplete"
		}
		d.Negative = r.Infrastructure && r.Waves == 3 && healthMissing == 6 && controlSuccess == 0 && len(controlFailed) >= 12 && len(ads) == 0
		if d.Route == "DIFFERENT" {
			d.Route = "PARTIAL_DIFFERENCE"
		}
		return d
	}
	if d.Route != "DIFFERENT" {
		d.Reason = "route_not_distinct"
		if stable && d.Route == "OVERLAP" {
			d.State = "NO_SPLIT"
			d.Negative = true
			for x := range ads {
				if !controls[x] {
					d.State = "UNKNOWN"
					d.Negative = false
					d.Reason = "mixed_route_observations"
				}
			}
		}
		return d
	}
	if rawOnly {
		d.Reason = "template_evidence_missing"
		return d
	}
	d.State = "RESIDENTIAL"
	d.Eligible = true
	d.Until = r.Finished + EvidenceTTL
	d.Reason = "repeated_ads_split_project_definition"
	return d
}
func Now() int64 { return time.Now().UnixMilli() }
