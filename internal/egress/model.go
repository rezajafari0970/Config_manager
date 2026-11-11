// Package egress owns the new, finite-observation Ads investigation contract.
// A peer-reported egress and a provider's network class are separate assertions.
package egress

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/convert"
	"github.com/rezajafari0970/Config_manager/internal/probe"
	"net/netip"
	"sort"
	"time"
)

const Version = "ads-egress.v2"
const EvidenceTTL = int64(600000)

var Domains = []string{"googleads.g.doubleclick.net", "pubads.g.doubleclick.net", "securepubads.g.doubleclick.net"}
var Cores = map[string]string{
	"xray-26.11.9": "e87bce586acdb9136aa5b1f1d9c8db9f1ea1d145b6d86368c7f6f662f9f6a66d",
	"xray-26.7.28": "64d46afb80adea1bf97a0d467e83f4a9ac1ebd0995891e84bca3f1a1d1affb1d",
	"xray-26.5.9":  "b2c525e082cf2fef460499c88838d355f0b9bfb5a00bdb3eaa99b6af63825006",
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
func singlePath(raw []byte) bool {
	var x struct {
		Outbounds []struct {
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
		} `json:"routing"`
	}
	if json.Unmarshal(raw, &x) != nil || len(x.Routing.Balancers) > 0 {
		return false
	}
	n := 0
	for _, o := range x.Outbounds {
		if o.Protocol != "freedom" && o.Protocol != "blackhole" && o.Protocol != "dns" {
			n++
		}
		if len(o.ProxySettings) > 0 && string(o.ProxySettings) != "{}" || o.StreamSettings.Sockopt.DialerProxy != "" {
			return false
		}
	}
	// An unrestricted direct-first input is not an investigated proxy path.
	return n == 1 && len(x.Outbounds) > 0 && x.Outbounds[0].Protocol != "freedom"
}

type Job struct {
	Hash     string `json:"hash"`
	Raw      []byte `json:"raw"`
	Kind     string `json:"kind"`
	Token    string `json:"token"`
	Revision int64  `json:"revision"`
	Issue    string `json:"issue"`
	Deep     bool   `json:"deep"`
}
type Sample struct {
	Variant             string `json:"variant"`
	Domain              string `json:"domain"`
	Wave                int    `json:"wave"`
	At                  int64  `json:"at"`
	IP                  string `json:"ip"`
	Family              int    `json:"family"`
	Transport           string `json:"transport"`
	Nonce               string `json:"nonce"`
	Connection          string `json:"connection"`
	TLS                 bool   `json:"tls"`
	SameStream          bool   `json:"same_stream"`
	Echo                bool   `json:"echo"`
	ActualAds           bool   `json:"actual_ads"`
	Code                string `json:"code"`
	Certificate         string `json:"certificate"`
	Response            string `json:"response"`
	FirstStatus         int    `json:"first_status"`
	EchoStatus          int    `json:"echo_status"`
	DiagnosticAuthority bool   `json:"diagnostic_authority"`
}
type Fact struct {
	IP       string `json:"ip"`
	Provider string `json:"provider"`
	Class    string `json:"class"`
	ASN      string `json:"asn"`
	Country  string `json:"country"`
	At       int64  `json:"at"`
	Expires  int64  `json:"expires"`
	Digest   string `json:"digest"`
	Code     string `json:"code"`
}
type Check struct {
	Name        string `json:"name"`
	Hash        string `json:"hash"`
	RuntimeHash string `json:"runtime_hash"`
	Core        bool   `json:"core"`
	Code        string `json:"code"`
}
type Attempt struct {
	Core string  `json:"core"`
	Code string  `json:"code"`
	MS   float64 `json:"ms"`
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
	Facts          []Fact    `json:"facts"`
	Attempts       []Attempt `json:"attempts"`
	WallMS         float64   `json:"wall_ms"`
}
type Completion struct {
	Job    Job    `json:"job"`
	Report Report `json:"report"`
}
type Decision struct {
	State      string   `json:"state"`
	Reason     string   `json:"reason"`
	Route      string   `json:"route"`
	Network    string   `json:"network"`
	Scope      string   `json:"scope"`
	IPs        []string `json:"ips"`
	ControlIPs []string `json:"control_ips"`
	Eligible   bool     `json:"eligible"`
	Negative   bool     `json:"negative"`
	Until      int64    `json:"until"`
	Missing    int      `json:"missing"`
}

func canonical(ip string) (netip.Addr, bool) {
	a, e := netip.ParseAddr(ip)
	return a.Unmap(), e == nil && a.IsGlobalUnicast() && !a.IsPrivate() && !a.IsLoopback() && !a.IsLinkLocalUnicast()
}
func controlGroup(domain string) string {
	switch domain {
	case "www.cloudflare.com", "one.one.one.one":
		return "cloudflare"
	case "api.ipify.org", "api6.ipify.org":
		return "ipify"
	}
	return ""
}
func Evaluate(r Report, p Plan, hash string, now int64) Decision {
	d := Decision{State: "UNKNOWN", Reason: "incomplete_observations", Route: "UNKNOWN", Network: "Unknown", Scope: "three explicit DoubleClick domains; TCP; observed IP family; raw and template; peer-reported", IPs: []string{}, ControlIPs: []string{}}
	if r.Version != Version || r.Hash != hash || r.Started <= 0 || r.Started > r.Finished || r.Finished > now+2000 || r.Started < now-300000 || r.Finished < now-EvidenceTTL {
		d.Reason = "stale_or_unbound_report"
		return d
	}
	if len(p.Variants) != 2 {
		d.State = "UNSUPPORTED"
		d.Reason = "driver_unavailable"
		return d
	}
	if len(r.Checks) != 2 || Cores[r.Core] == "" || Cores[r.Core] != r.CoreSHA {
		d.Reason = "core_evidence_missing"
		return d
	}
	for i, v := range p.Variants {
		c := r.Checks[i]
		if c.Name != v.Name || c.Hash != v.Hash || !c.Core || len(c.RuntimeHash) != 64 {
			d.Reason = "core_or_variant_failed"
			return d
		}
	}
	if !p.Complete {
		d.Reason = "multiple_or_unresolved_paths"
		return d
	}
	ips := map[string]bool{}
	controls := map[string]bool{}
	seen := map[string]bool{}
	healthMissing := 0
	adsMissing := 0
	controlSuccess := 0
	controlFailed := map[string]bool{}
	for _, v := range []string{"raw", "template"} {
		for wave := 1; wave <= 3; wave++ {
			paired := map[int]map[string]string{}
			for _, s := range r.Samples {
				if s.Variant != v || s.Wave != wave || controlGroup(s.Domain) == "" {
					continue
				}
				if s.Code == "connection_timeout" || s.Code == "socks_connect" || s.Code == "connection_failed" {
					controlFailed[v+string(rune(wave))+s.Domain] = true
				}
				a, ok := canonical(s.IP)
				if !ok || !s.TLS || !s.Echo || s.At < r.Started || s.At > r.Finished+2000 || s.Transport != "tcp" || (s.Family == 4 && !a.Is4()) || (s.Family == 6 && !a.Is6()) || s.Family != 4 && s.Family != 6 {
					continue
				}
				controlSuccess++
				if paired[s.Family] == nil {
					paired[s.Family] = map[string]string{}
				}
				paired[s.Family][controlGroup(s.Domain)] = a.String()
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
					if s.Variant != v || s.Wave != wave || s.Domain != domain {
						continue
					}
					a, ok := canonical(s.IP)
					m := paired[s.Family]
					if !ok || !s.TLS || !s.SameStream || !s.Echo || !s.ActualAds || s.Transport != "tcp" || len(s.Nonce) != 32 || s.Nonce != s.Connection || seen[s.Connection] || len(s.Certificate) != 64 || len(s.Response) != 64 || s.At < r.Started || s.At > r.Finished+2000 || len(m) != 2 {
						continue
					}
					if (s.Family == 4 && !a.Is4()) || (s.Family == 6 && !a.Is6()) {
						continue
					}
					seen[s.Connection] = true
					ips[a.String()] = true
					for _, ip := range m {
						controls[ip] = true
					}
					valid = true
					break
				}
				if !valid {
					adsMissing++
				}
			}
		}
	}
	d.Missing = adsMissing
	if healthMissing > 0 {
		d.Reason = "controls_incomplete"
		d.Negative = r.Infrastructure && healthMissing == 6 && controlSuccess == 0 && len(controlFailed) >= 12 && len(ips) == 0
		return d
	}
	if adsMissing > 0 {
		d.Reason = "ads_peer_echo_incomplete"
		return d
	}
	for ip := range ips {
		d.IPs = append(d.IPs, ip)
	}
	for ip := range controls {
		d.ControlIPs = append(d.ControlIPs, ip)
	}
	sort.Strings(d.IPs)
	sort.Strings(d.ControlIPs)
	if len(controls) == 1 {
		d.Route = "DIFFERENT"
		for ip := range ips {
			if controls[ip] {
				d.Route = "OVERLAP"
			}
		}
	}
	allRes := true
	allNonResidential := true
	until := r.Finished + EvidenceTTL
	for ip := range ips {
		class := "Unknown"
		for _, f := range r.Facts {
			a, ok := canonical(f.IP)
			if ok && a.String() == ip && f.Provider == "proxycheck-v3" && len(f.Digest) == 64 && f.At <= now+2000 && f.Expires > now && f.At > now-3600000 {
				class = f.Class
				if f.Expires < until {
					until = f.Expires
				}
			}
		}
		allRes = allRes && class == "Residential"
		allNonResidential = allNonResidential && (class == "Hosting" || class == "Wireless" || class == "Business")
	}
	if allRes {
		d.State = "RESIDENTIAL"
		d.Network = "Residential"
		d.Eligible = true
		d.Until = until
		d.Reason = "authenticated_ads_peer_report_and_provider_class"
	} else if allNonResidential {
		d.State = "NON_RESIDENTIAL"
		d.Network = "NonResidential"
		d.Negative = true
		d.Reason = "provider_non_residential"
	} else {
		d.Reason = "classification_missing_or_mixed"
	}
	return d
}
func Now() int64 { return time.Now().UnixMilli() }
