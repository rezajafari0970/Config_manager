package residential

import (
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/convert"
	"github.com/rezajafari0970/Config_manager/internal/driver"
	"github.com/rezajafari0970/Config_manager/internal/probe"
	"strings"
)

type Execution struct {
	Name          string   `json:"name"`
	SHA256        string   `json:"sha256"`
	Config        []byte   `json:"config"`
	Paths         []string `json:"paths"`
	PathsComplete bool     `json:"paths_complete"`
}
type Plan struct {
	Version    string      `json:"version"`
	CoreSHA256 string      `json:"core_sha256"`
	Code       string      `json:"code"`
	Variants   []Execution `json:"variants"`
}
type CoreDriver interface {
	ID() string
	Prepare([]byte) Plan
}
type CoreRegistry struct{ drivers driver.Registry[CoreDriver] }

func NewCoreRegistry() *CoreRegistry { r := &CoreRegistry{}; r.Register(XrayDriver{}); return r }
func (r *CoreRegistry) Register(d CoreDriver) {
	if d != nil {
		r.drivers.Put(d.ID(), d)
	}
}
func (r *CoreRegistry) Prepare(id string, raw []byte) Plan {
	if d := r.drivers.Get(id); d != nil {
		return d.Prepare(raw)
	}
	return Plan{Version: Version, Code: "core_driver_unsupported"}
}

type XrayDriver struct{}

func (XrayDriver) ID() string { return "xray-26.7.28" }
func (XrayDriver) Prepare(raw []byte) Plan {
	out := Plan{Version: Version, CoreSHA256: probe.CoreSHA256, Code: "UNSUPPORTED", Variants: []Execution{}}
	p := probe.Prepare(raw)
	for _, c := range p.Candidates {
		if c.RequiresConnection {
			continue
		}
		rawConfig := c.Config
		// A JSON original is executed byte-for-byte, even if the conversion parser
		// produced a formatting/structural repair for the separate template variant.
		// Its own pinned-core pass and observations remain mandatory.
		if p.Kind == "json" {
			if !json.Valid(raw) {
				out.Code = "raw_json_invalid"
				return out
			}
			rawConfig = raw
		}
		common, e := convert.Apply(c.Config)
		if e != nil {
			out.Code = "common_template_unsupported"
			return out
		}
		for _, v := range []struct {
			name string
			b    []byte
		}{{"raw", rawConfig}, {"template", common}} {
			paths, complete := executionPaths(v.b)
			out.Variants = append(out.Variants, Execution{Name: v.name, SHA256: Digest(v.b), Config: v.b, Paths: paths, PathsComplete: complete})
		}
		out.Code = "prepared"
		return out
	}
	if p.Code != "" {
		out.Code = p.Code
	}
	return out
}

// Until a route-instrumentation plugin can prove all effective choices, retain
// multi-path inputs as PARTIAL. Repeated random balancer choices are not a proof
// of complete coverage. Names are indices, never arbitrary credential-like tags.
func executionPaths(b []byte) ([]string, bool) {
	var m map[string]any
	if json.Unmarshal(b, &m) != nil {
		return []string{"unresolved"}, false
	}
	obs, _ := m["outbounds"].([]any)
	count := 0
	proxyTag := ""
	complete := true
	for _, value := range obs {
		ob, ok := value.(map[string]any)
		if !ok {
			return []string{"unresolved"}, false
		}
		switch ob["protocol"] {
		case "freedom", "blackhole", "dns":
		case "loopback":
			complete = false
		default:
			count++
			proxyTag, _ = ob["tag"].(string)
		}
	}
	routing, _ := m["routing"].(map[string]any)
	if balances, ok := routing["balancers"].([]any); ok && len(balances) > 0 {
		complete = false
	}
	rules, _ := routing["rules"].([]any)
	for _, value := range rules {
		rule, ok := value.(map[string]any)
		if !ok {
			complete = false
			continue
		}
		tag, _ := rule["outboundTag"].(string)
		if rule["balancerTag"] != nil {
			complete = false
		}
		if tag != "" && tag != proxyTag && rule["port"] != "53" {
			complete = false
		}
		for _, key := range []string{"domain", "ip"} {
			if values, ok := rule[key].([]any); ok {
				for _, value := range values {
					if text, ok := value.(string); ok && (strings.HasPrefix(text, "geosite:") || strings.HasPrefix(text, "geoip:") || strings.HasPrefix(text, "ext:")) {
						complete = false
					}
				}
			}
		}
	}
	for _, key := range []string{"api", "reverse", "fakedns"} {
		if m[key] != nil {
			complete = false
		}
	}
	if count != 1 {
		complete = false
	}
	if !complete {
		return []string{"unresolved-selectable-paths"}, false
	}
	return []string{"single-proxy-route"}, true
}
