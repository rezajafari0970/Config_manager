package residential

import (
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/convert"
	"github.com/rezajafari0970/Config_manager/internal/driver"
	"github.com/rezajafari0970/Config_manager/internal/probe"
	"github.com/rezajafari0970/Config_manager/internal/repair"
	"strings"
)

type Execution struct {
	Name          string   `json:"name"`
	ImportRule    string   `json:"import_rule"`
	SHA256        string   `json:"sha256"`
	Config        []byte   `json:"config"`
	Paths         []string `json:"paths"`
	PathsComplete bool     `json:"paths_complete"`
}
type Plan struct {
	Driver     string      `json:"driver"`
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

func NewCoreRegistry() *CoreRegistry {
	r := &CoreRegistry{}
	r.Register(XrayDriver{})
	r.Register(NativeDriver{})
	return r
}
func (r *CoreRegistry) Select(raw []byte) Plan {
	p := r.Prepare("xray-26.7.28", raw)
	if len(p.Variants) == 2 {
		return p
	}
	n := r.Prepare(SingBoxID, raw)
	if n.Code != "native_protocol_unsupported" {
		return n
	}
	return p
}
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
	out := Plan{Version: Version, Driver: "xray", CoreSHA256: "64d46afb80adea1bf97a0d467e83f4a9ac1ebd0995891e84bca3f1a1d1affb1d", Code: "UNSUPPORTED", Variants: []Execution{}}
	p := probe.Prepare(raw)
	for _, c := range p.Candidates {
		if c.RequiresConnection {
			continue
		}
		rawConfig := c.Config
		importRule := "uri-import"
		if p.Kind == "json" {
			rawConfig = raw
			importRule = "original-json"
			// Apply only the shared deterministic structural repairs already admitted
			// by Prepare. Preserve the acquired bytes separately; never repair secrets,
			// protocols, endpoint values or TLS verification to obtain a connection.
			d := (repair.JSONDriver{}).Propose(raw)
			for _, patch := range d.Patches {
				if patch.RuleID != repair.RulesVersion+".custom-envelope" {
					rawConfig = c.Config
					importRule = repair.RulesVersion
					break
				}
			}
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
			out.Variants = append(out.Variants, Execution{Name: v.name, ImportRule: func() string {
				if v.name == "raw" {
					return importRule
				}
				return "common-template"
			}(), SHA256: Digest(v.b), Config: v.b, Paths: paths, PathsComplete: complete})
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
	// Xray sends unmatched traffic to the first outbound, not to the only
	// outbound counted as a proxy. A restrictive rule cannot fix a direct default.
	defaultProxy := false
	if len(obs) > 0 {
		first, _ := obs[0].(map[string]any)
		defaultProxy = first["tag"] == proxyTag && proxyTag != ""
	}
	for _, value := range rules {
		rule, ok := value.(map[string]any)
		if !ok {
			complete = false
			continue
		}
		tag, _ := rule["outboundTag"].(string)
		unconditional := tag == proxyTag && tag != ""
		for key, value := range rule {
			if key == "type" || key == "outboundTag" || key == "ruleTag" {
				continue
			}
			if key == "network" && (value == "tcp,udp" || value == "udp,tcp") {
				continue
			}
			unconditional = false
		}
		if unconditional {
			defaultProxy = true
		}
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
	if !defaultProxy {
		complete = false
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
