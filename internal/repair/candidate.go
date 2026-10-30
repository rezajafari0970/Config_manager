package repair

import (
	"bytes"
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/rezajafari0970/Config_manager/internal/driver"
	"github.com/rezajafari0970/Config_manager/internal/target"
)

const RulesVersion = "xray-26.7.28.structural.v3"

type Diagnostic struct {
	Version         string           `json:"version"`
	Target          string           `json:"target"`
	Status          string           `json:"status"`
	OriginalSHA256  string           `json:"original_sha256"`
	CandidateSHA256 string           `json:"candidate_sha256,omitempty"`
	Patches         []driver.Patch   `json:"patches"`
	Findings        []driver.Finding `json:"findings"`
	CoreValidation  string           `json:"core_validation"`
	AndroidImport   string           `json:"android_import"`
	Connection      string           `json:"connection"`
	Candidate       []byte           `json:"-"`
}

// CandidateDriver is an offline, compiled extension boundary. Drivers never
// overwrite the original and may not perform network or core execution.
type CandidateDriver interface {
	Kind() string
	Propose([]byte) Diagnostic
}
type Registry struct{ drivers map[string]CandidateDriver }

func NewRegistry() *Registry {
	r := &Registry{drivers: map[string]CandidateDriver{}}
	r.Register(JSONDriver{})
	for _, kind := range []string{"vless", "trojan", "ss", "hy2", "hysteria2", "socks", "socks5"} {
		r.Register(URIUserinfoDriver{Protocol: kind})
	}
	return r
}
func (r *Registry) Register(d CandidateDriver) { r.drivers[d.Kind()] = d }
func baseDiagnostic(raw []byte) Diagnostic {
	return Diagnostic{Version: RulesVersion, Target: target.Pinned().ID, Status: "unchanged", OriginalSHA256: Digest(raw), Patches: []driver.Patch{}, Findings: []driver.Finding{}, CoreValidation: "not_run", AndroidImport: "not_run", Connection: "not_run"}
}
func (r *Registry) Propose(raw []byte, complete bool, kind, quality string) (d Diagnostic) {
	d = baseDiagnostic(raw)
	defer func() {
		if recover() != nil {
			d = baseDiagnostic(raw)
			d.Status = "blocked"
			d.Findings = append(d.Findings, driver.Finding{Code: "repair_driver_recovered", Severity: "error"})
		}
	}()
	if !complete || len(raw) > MaxBytes {
		d.Status = "unverified"
		d.Findings = append(d.Findings, driver.Finding{Code: "inspection_size_limit", Severity: "warning"})
		return
	}
	if impl := r.drivers[kind]; impl != nil {
		result := impl.Propose(raw)
		if result.Status != "unchanged" || quality != "invalid" {
			return result
		}
	}
	if quality == "invalid" {
		d.Status = "blocked"
		code := "no_unambiguous_repair"
		if kind == "ss" {
			code = "shadowsocks_credentials_or_scheme_ambiguous"
		}
		if kind == "wireguard" {
			code = "wireguard_incomplete_sections"
		}
		d.Findings = append(d.Findings, driver.Finding{Code: code, Severity: "error"})
	} else if quality != "valid" {
		d.Status = "unverified"
	}
	// Amnezia extensions must not be silently removed or called compatible WireGuard.
	if kind == "wireguard" && bytes.HasPrefix(bytes.TrimSpace(raw), []byte("[")) {
		for _, line := range strings.Split(string(raw), "\n") {
			k, _, ok := strings.Cut(line, "=")
			if ok {
				switch strings.ToLower(strings.TrimSpace(k)) {
				case "jc", "jmin", "jmax", "s1", "s2", "s3", "s4", "h1", "h2", "h3", "h4":
					d.Status = "unverified"
					d.Findings = []driver.Finding{{Code: "wireguard_extension_requires_supported_core", Severity: "warning"}}
					return
				}
			}
		}
	}
	return
}

type JSONDriver struct{}

func (JSONDriver) Kind() string { return "json" }

type span struct {
	start, end int
	path       string
	raw        []byte
}

func spans(b []byte, offset int, path string, all *[]span) {
	trim := bytes.TrimSpace(b)
	offset += bytes.Index(b, trim)
	*all = append(*all, span{offset, offset + len(trim), path, trim})
	if len(trim) == 0 || (trim[0] != '{' && trim[0] != '[') {
		return
	}
	dec := json.NewDecoder(bytes.NewReader(trim))
	dec.UseNumber()
	_, _ = dec.Token()
	i := 0
	for dec.More() {
		var key string
		if trim[0] == '{' {
			t, _ := dec.Token()
			key = t.(string)
		} else {
			key = strconv.Itoa(i)
			i++
		}
		var child json.RawMessage
		_ = dec.Decode(&child)
		end := int(dec.InputOffset())
		spans(child, offset+end-len(child), path+"/"+strings.ReplaceAll(strings.ReplaceAll(key, "~", "~0"), "/", "~1"), all)
	}
}

// stripTrailingCommas never touches quoted text and only removes a comma
// immediately preceding an array/object close. Strict parsing still gates it.
func stripTrailingCommas(b []byte) []byte {
	out := make([]byte, 0, len(b))
	quoted, escaped := false, false
	for i, c := range b {
		if quoted {
			out = append(out, c)
			if escaped {
				escaped = false
			} else if c == '\\' {
				escaped = true
			} else if c == '"' {
				quoted = false
			}
			continue
		}
		if c == '"' {
			quoted = true
		}
		if c == ',' {
			j := i + 1
			for j < len(b) && bytes.ContainsRune([]byte(" \t\r\n"), rune(b[j])) {
				j++
			}
			if j < len(b) && (b[j] == '}' || b[j] == ']') {
				k := len(out) - 1
				for k >= 0 && bytes.ContainsRune([]byte(" \t\r\n"), rune(out[k])) {
					k--
				}
				if k >= 0 && !bytes.ContainsRune([]byte("{[:,"), rune(out[k])) {
					continue
				}
			}
		}
		out = append(out, c)
	}
	return out
}
func (JSONDriver) Propose(raw []byte) Diagnostic {
	d := baseDiagnostic(raw)
	work := bytes.Clone(raw)
	if len(work) > MaxBytes {
		d.Status = "unverified"
		return d
	}
	if !json.Valid(work) {
		clean := stripTrailingCommas(work)
		if bytes.HasPrefix(clean, []byte{0xef, 0xbb, 0xbf}) {
			clean = clean[3:]
		}
		if strict(clean) == nil && !bytes.Equal(clean, work) {
			d.Patches = append(d.Patches, driver.Patch{RuleID: RulesVersion + ".json-syntax", Path: "", BeforeSHA256: Digest(work), AfterSHA256: Digest(clean)})
			work = clean
		}
	}
	if err := strict(work); err != nil {
		d.Status = "blocked"
		d.Findings = []driver.Finding{{Code: err.Error(), Severity: "error"}}
		return d
	}
	if !object(work) {
		d.Status = "blocked"
		d.Findings = []driver.Finding{{Code: "expected_single_custom_object", Severity: "error"}}
		return d
	}
	var root map[string]json.RawMessage
	_ = json.Unmarshal(work, &root)
	if !array(root["outbounds"]) {
		d.Status = "blocked"
		d.Findings = Analyze(work).Findings
		return d
	}
	allowed := map[string]bool{"/stats": true, "/log": true, "/dns": true, "/policy": true, "/routing": true, "/dns/hosts": true, "/policy/levels": true, "/policy/system": true}
	var obs []map[string]json.RawMessage
	_ = json.Unmarshal(root["outbounds"], &obs)
	for i, ob := range obs {
		var p string
		_ = json.Unmarshal(ob["protocol"], &p)
		if p == "freedom" || p == "blackhole" || p == "dns" {
			allowed["/outbounds/"+strconv.Itoa(i)+"/settings"] = true
		}
	}
	var all []span
	spans(work, 0, "", &all)
	var changes []span
	for _, s := range all {
		if !allowed[s.path] || !array(s.raw) {
			continue
		}
		var values []json.RawMessage
		_ = json.Unmarshal(s.raw, &values)
		if len(values) == 0 {
			changes = append(changes, s)
		}
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].start > changes[j].start })
	for _, s := range changes {
		before := Digest(work)
		out := append([]byte{}, work[:s.start]...)
		out = append(out, '{', '}')
		out = append(out, work[s.end:]...)
		work = out
		d.Patches = append(d.Patches, driver.Patch{RuleID: RulesVersion + ".empty-object", Path: s.path, BeforeSHA256: before, AfterSHA256: Digest(work)})
	}
	// Absent inbound/routing containers have empty semantics in the pinned core;
	// add only these two client import markers, never invent an outbound or secret.
	if len(obs) > 0 {
		add := ""
		if _, ok := root["inbounds"]; !ok {
			add += `,"inbounds":[]`
		}
		if _, ok := root["routing"]; !ok {
			add += `,"routing":{}`
		}
		if add != "" {
			end := bytes.LastIndexByte(work, '}')
			before := Digest(work)
			out := append([]byte{}, work[:end]...)
			out = append(out, add...)
			out = append(out, work[end:]...)
			work = out
			d.Patches = append(d.Patches, driver.Patch{RuleID: RulesVersion + ".custom-envelope", Path: "", BeforeSHA256: before, AfterSHA256: Digest(work)})
		}
	}
	report := Analyze(work)
	d.Findings = report.Findings
	if report.HasErrors() {
		d.Status = "blocked"
		return d
	}
	if !bytes.Equal(work, raw) {
		d.Status = "repaired"
		d.Candidate = work
		d.CandidateSHA256 = Digest(work)
	}
	return d
}
