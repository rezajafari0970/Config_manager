package egress

import (
	_ "embed"
	"encoding/json"
	"sort"
)

const DeliveryVersion = "ads-delivery.v1"
const DeliveryMaxBytes = 524288

type DeliveryTarget struct {
	Profile string `json:"profile"`
	Domain  string `json:"domain"`
	Path    string `json:"path"`
	Marker  string `json:"marker"`
}

//go:embed delivery_targets.json
var deliveryJSON []byte
var deliveryTargets = func() []DeliveryTarget {
	var v []DeliveryTarget
	if json.Unmarshal(deliveryJSON, &v) != nil || len(v) != len(profiles) {
		panic("invalid Ads delivery registry")
	}
	seen := map[string]bool{}
	for _, t := range v {
		valid := false
		for _, p := range profiles {
			for _, d := range p.Domains {
				if t.Profile == p.ID && t.Domain == d {
					valid = true
				}
			}
		}
		if !valid || seen[t.Profile] || t.Path == "" || t.Path[0] != '/' || t.Marker == "" {
			panic("invalid Ads delivery target")
		}
		seen[t.Profile] = true
	}
	return v
}()

func DeliveryTargets() []DeliveryTarget { return append([]DeliveryTarget(nil), deliveryTargets...) }

type DeliveryAsset struct {
	Profile      string `json:"profile"`
	Domain       string `json:"domain"`
	Path         string `json:"path"`
	Nonce        string `json:"nonce"`
	HTTPStatus   int    `json:"http_status"`
	CurlExit     int    `json:"curl_exit"`
	TLSVerified  bool   `json:"tls_verified"`
	Complete     bool   `json:"complete"`
	ContentMatch bool   `json:"content_match"`
	Bytes        int    `json:"bytes"`
	BodySHA      string `json:"body_sha"`
	Code         string `json:"code"`
}
type DeliverySample struct {
	Version     string          `json:"version"`
	Variant     string          `json:"variant"`
	At          int64           `json:"at"`
	HostControl bool            `json:"host_control"`
	Assets      []DeliveryAsset `json:"assets"`
}
type DeliveryDecision struct {
	Status   string            `json:"status"`
	Profiles map[string]string `json:"profiles"`
	Scope    string            `json:"scope"`
}

func assetValid(a DeliveryAsset, t DeliveryTarget) bool {
	return a.Profile == t.Profile && a.Domain == t.Domain && a.Path == t.Path && routeHexBytes(a.Nonce, 16) && a.Bytes >= 0 && a.Bytes <= DeliveryMaxBytes
}
func assetAvailable(a DeliveryAsset) bool {
	return a.CurlExit == 0 && a.HTTPStatus == 200 && a.TLSVerified && a.Complete && a.ContentMatch && a.Bytes >= 1024 && routeHexBytes(a.BodySHA, 32)
}
func assetFailure(a DeliveryAsset) bool {
	if a.HTTPStatus == 429 || a.HTTPStatus >= 500 && a.HTTPStatus <= 599 {
		return false
	}
	// Verified JavaScript bytes followed by a deadline are incomplete progress,
	// not evidence that this config cannot deliver advertising.
	if a.CurlExit != 0 && a.HTTPStatus == 200 && a.TLSVerified && a.Bytes > 0 {
		return false
	}
	// A transport or explicit block response is failure evidence. Parser bugs,
	// rate limits, unsupported response formats and 5xx are uncertainty.
	if a.CurlExit == 0 && a.TLSVerified {
		return a.HTTPStatus == 403 || a.HTTPStatus == 451
	}
	return a.CurlExit == 7 || a.CurlExit == 28 || a.CurlExit == 35 || a.CurlExit == 52 || a.CurlExit == 56 || a.CurlExit == 60 || a.CurlExit == 97
}

// Delivery is a necessary asset-path check, never proof of ad fill or rendering.
// The host control only corroborates failures; it can never qualify a VPN.
func Delivery(r Report, p Plan, now int64) DeliveryDecision {
	d := DeliveryDecision{Status: "unknown", Profiles: map[string]string{}, Scope: "HTTPS Ads bootstrap assets in exact raw/template; ad fill and device rendering not measured"}
	for _, t := range deliveryTargets {
		d.Profiles[t.Profile] = "unknown"
	}
	if routeBound(r, p, r.Hash, now) != "" || len(r.Delivery) > 3 {
		return d
	}
	byVariant := map[string]map[string]DeliveryAsset{}
	nonces := map[string]bool{}
	for _, s := range r.Delivery {
		// Optional host corroboration expires at decision time, not report start.
		// Stale host data cannot erase independently valid VPN asset evidence.
		if s.HostControl && (s.Variant != "host" || s.At < max(now, r.Finished)-30000) {
			continue
		}
		if s.Version != DeliveryVersion || len(s.Assets) != len(deliveryTargets) || s.At > r.Finished+2000 {
			return d
		}
		if !s.HostControl && ((s.Variant != "raw" && s.Variant != "template") || s.At < r.Started) {
			return d
		}
		if byVariant[s.Variant] != nil {
			return d
		}
		m := map[string]DeliveryAsset{}
		for _, a := range s.Assets {
			valid := false
			for _, t := range deliveryTargets {
				if assetValid(a, t) {
					valid = true
				}
			}
			if !valid || nonces[a.Nonce] || m[a.Profile].Profile != "" {
				return d
			}
			nonces[a.Nonce] = true
			m[a.Profile] = a
		}
		byVariant[s.Variant] = m
	}
	good, bad := 0, 0
	for _, t := range deliveryTargets {
		a, b, h := byVariant["raw"][t.Profile], byVariant["template"][t.Profile], byVariant["host"][t.Profile]
		if assetAvailable(a) && assetAvailable(b) {
			d.Profiles[t.Profile] = "available"
			good++
		} else if assetFailure(a) && assetFailure(b) && assetAvailable(h) {
			d.Profiles[t.Profile] = "unavailable"
			bad++
		}
	}
	if good > 0 {
		d.Status = "available"
	} else if bad == len(deliveryTargets) {
		d.Status = "unavailable"
	}
	return d
}

func qualifyDelivery(d Decision, r Report, p Plan, now int64) Decision {
	d.Delivery = Delivery(r, p, now)
	if !d.Eligible {
		return d
	}
	ids, domains := []string{}, []string{}
	for _, id := range d.QualifiedProfiles {
		if d.Delivery.Profiles[id] != "available" {
			continue
		}
		ids = append(ids, id)
		for _, p := range profiles {
			if p.ID == id {
				domains = append(domains, p.Domains...)
			}
		}
	}
	if len(ids) == 0 {
		d.Eligible = false
		d.State = "UNKNOWN"
		d.Reason = "ads_delivery_unproven"
		d.Until = 0
	} else if len(ids) < len(d.QualifiedProfiles) {
		d.AdsCoverage = "PARTIAL"
		d.CoverageComplete = false
	}
	sort.Strings(domains)
	d.QualifiedProfiles = ids
	d.QualifiedDomains = domains
	return d
}
