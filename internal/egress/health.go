package egress

const HealthVersion = "ads-health.v1"
const HealthBytes = 65536

type Transfer struct {
	Bytes       int64 `json:"bytes"`
	CurlExit    int   `json:"curl_exit"`
	HTTPStatus  int   `json:"http_status"`
	TLSVerified bool  `json:"tls_verified"`
	MS          int64 `json:"ms"`
}
type HealthProvider struct {
	Provider string   `json:"provider"`
	Download Transfer `json:"download"`
	Upload   Transfer `json:"upload"`
}
type HealthSample struct {
	Version     string           `json:"version"`
	Variant     string           `json:"variant"`
	At          int64            `json:"at"`
	HostControl bool             `json:"host_control"`
	Providers   []HealthProvider `json:"providers"`
}

func transferHealthy(t Transfer) bool {
	return t.Bytes == HealthBytes && t.CurlExit == 0 && t.HTTPStatus == 200 && t.TLSVerified
}
func healthProviders(h HealthSample) (int, bool) {
	if len(h.Providers) != 2 {
		return 0, false
	}
	seen := map[string]bool{}
	passes := 0
	for _, p := range h.Providers {
		if (p.Provider != "cloudflare" && p.Provider != "httpbingo") || seen[p.Provider] {
			return 0, false
		}
		seen[p.Provider] = true
		if transferHealthy(p.Download) && transferHealthy(p.Upload) {
			passes++
		}
	}
	return passes, true
}

// An incomplete route observation is never a negative health verdict. Credible
// negatives require BOTH payload targets failing in BOTH exact runtimes while
// the same two targets succeed outside the tunnel. Repetition is a store policy.
func Health(r Report, p Plan, now int64) string {
	if r.Started <= 0 || r.Started > r.Finished || now-r.Finished > 300000 || r.Finished > now+2000 || len(r.Health) > 3 || len(p.Variants) != 2 || len(r.Checks) != 2 || Cores[r.Core] == "" || Cores[r.Core] != r.CoreSHA {
		return "unknown"
	}
	for i, v := range p.Variants {
		c := r.Checks[i]
		if c.Name != v.Name || c.Hash != v.Hash || !c.Core || len(c.RuntimeHash) != 64 || p.Driver == "xray" && c.RuntimeHash != v.Hash {
			return "unknown"
		}
	}
	seen := map[string]bool{}
	good, bad, host := 0, 0, false
	for _, h := range r.Health {
		if h.Version != HealthVersion || h.At < r.Started || h.At > r.Finished+2000 {
			return "unknown"
		}
		n, valid := healthProviders(h)
		if !valid {
			return "unknown"
		}
		if h.HostControl {
			if h.Variant != "host" || seen["host"] {
				return "unknown"
			}
			seen["host"] = true
			host = n == 2
			continue
		}
		if (h.Variant != "raw" && h.Variant != "template") || seen[h.Variant] {
			return "unknown"
		}
		seen[h.Variant] = true
		if n > 0 {
			good++
		} else {
			bad++
		}
	}
	if good == 2 {
		return "healthy"
	}
	if bad == 2 && host {
		return "unhealthy"
	}
	return "unknown"
}
