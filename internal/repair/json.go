// Package repair contains conservative, version-specific offline rules.
// A structurally acceptable candidate is NEVER a publication approval.
package repair

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strconv"

	"github.com/rezajafari0970/Config_manager/internal/driver"
	"github.com/rezajafari0970/Config_manager/internal/target"
)

const MaxBytes = 2 * 1024 * 1024
const MaxDepth = 64
const StatsRule = "xray-26.7.28.stats-empty-array-to-object.v1"

type Report struct {
	Target         string           `json:"target"`
	SHA256         string           `json:"sha256"`
	Findings       []driver.Finding `json:"findings"`
	CoreValidation string           `json:"core_validation"`
	AndroidImport  string           `json:"android_import"`
	Connection     string           `json:"connection"`
	Publishable    bool             `json:"publishable"`
}

func Digest(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func (r Report) HasErrors() bool {
	for _, f := range r.Findings {
		if f.Severity == "error" {
			return true
		}
	}
	return false
}
func (r *Report) add(code, path string) {
	r.Findings = append(r.Findings, driver.Finding{Code: code, Path: path, Severity: "error"})
}

// Walk tokens first to reject duplicate keys at ALL depths and ambiguous documents.
func checkValue(d *json.Decoder, depth int) error {
	if depth > MaxDepth {
		return errors.New("json_depth_limit")
	}
	t, err := d.Token()
	if err != nil {
		return errors.New("invalid_json")
	}
	if c, ok := t.(json.Delim); ok {
		switch c {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				k, err := d.Token()
				if err != nil {
					return errors.New("invalid_json")
				}
				key, ok := k.(string)
				if !ok {
					return errors.New("invalid_json")
				}
				if seen[key] {
					return errors.New("duplicate_json_key")
				}
				seen[key] = true
				if err := checkValue(d, depth+1); err != nil {
					return err
				}
			}
		case '[':
			for d.More() {
				if err := checkValue(d, depth+1); err != nil {
					return err
				}
			}
		default:
			return errors.New("invalid_json")
		}
		end, err := d.Token()
		if err != nil || (c == '{' && end != json.Delim('}')) || (c == '[' && end != json.Delim(']')) {
			return errors.New("invalid_json")
		}
	}
	return nil
}
func strict(b []byte) error {
	if len(b) > MaxBytes {
		return errors.New("input_too_large")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	if err := checkValue(d, 0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return errors.New("trailing_json_data")
	}
	return nil
}
func object(b json.RawMessage) bool { b = bytes.TrimSpace(b); return len(b) > 1 && b[0] == '{' }
func array(b json.RawMessage) bool  { b = bytes.TrimSpace(b); return len(b) > 1 && b[0] == '[' }

func Analyze(b []byte) Report {
	r := Report{Target: target.Pinned().ID, SHA256: Digest(b), Findings: []driver.Finding{}, CoreValidation: "not_run", AndroidImport: "not_run", Connection: "not_run"}
	if err := strict(b); err != nil {
		r.add(err.Error(), "")
		return r
	}
	if !object(b) {
		r.add("expected_single_custom_object", "")
		return r
	}
	var root map[string]json.RawMessage
	_ = json.Unmarshal(b, &root)
	for _, key := range []string{"inbounds", "outbounds", "routing"} {
		if _, ok := root[key]; !ok {
			r.add("v2rayng_custom_import_required_field", "/"+key)
		}
	}
	for _, key := range []string{"stats", "log", "policy", "dns", "routing"} {
		if v, ok := root[key]; ok && !object(v) {
			r.add("expected_object", "/"+key)
		}
	}
	for _, key := range []string{"inbounds", "outbounds"} {
		if v, ok := root[key]; ok && !array(v) {
			r.add("expected_array", "/"+key)
		}
	}
	if object(root["dns"]) {
		var dns map[string]json.RawMessage
		_ = json.Unmarshal(root["dns"], &dns)
		if v, ok := dns["hosts"]; ok && !object(v) {
			r.add("expected_object", "/dns/hosts")
		}
	}
	if object(root["routing"]) {
		var routing map[string]json.RawMessage
		_ = json.Unmarshal(root["routing"], &routing)
		for _, k := range []string{"rules", "balancers"} {
			if v, ok := routing[k]; ok && !array(v) {
				r.add("expected_array", "/routing/"+k)
			}
		}
	}
	if array(root["outbounds"]) {
		var obs []json.RawMessage
		_ = json.Unmarshal(root["outbounds"], &obs)
		if len(obs) == 0 {
			r.add("missing_outbound", "/outbounds")
		}
		for i, ob := range obs {
			p := "/outbounds/" + strconv.Itoa(i)
			if !object(ob) {
				r.add("expected_object", p)
				continue
			}
			var v map[string]json.RawMessage
			_ = json.Unmarshal(ob, &v)
			var protocol string
			if json.Unmarshal(v["protocol"], &protocol) != nil || protocol == "" {
				r.add("missing_protocol", p+"/protocol")
			}
			if s, ok := v["settings"]; ok && !object(s) {
				r.add("expected_object", p+"/settings")
			}
			if protocol == "hysteria" && object(v["settings"]) {
				var s map[string]json.RawMessage
				_ = json.Unmarshal(v["settings"], &s)
				var version int
				_ = json.Unmarshal(s["version"], &version)
				if version != 2 {
					r.add("hysteria_version_unsupported", p+"/settings/version")
				}
				var address string
				if json.Unmarshal(s["address"], &address) != nil || address == "" {
					r.add("missing_hysteria_address", p+"/settings/address")
				}
			}
		}
	}
	return r
}

// RepairStats changes ONLY the exact top-level empty stats array span.
// Unknown fields, strings, key order and all other bytes remain untouched.
// Output is a candidate requiring core, Android and network validation.
func RepairStats(b []byte) ([]byte, []driver.Patch, error) {
	if err := strict(b); err != nil {
		return nil, nil, err
	}
	if !object(b) {
		return nil, nil, errors.New("expected_single_custom_object")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	_, _ = d.Token()
	for d.More() {
		key, err := d.Token()
		if err != nil {
			return nil, nil, err
		}
		var raw json.RawMessage
		if err := d.Decode(&raw); err != nil {
			return nil, nil, err
		}
		if key != "stats" || !array(raw) {
			continue
		}
		var vals []json.RawMessage
		_ = json.Unmarshal(raw, &vals)
		if len(vals) != 0 {
			return nil, nil, errors.New("nonempty_stats_array_not_repairable")
		}
		end := int(d.InputOffset())
		start := end - len(raw)
		out := make([]byte, 0, len(b))
		out = append(out, b[:start]...)
		out = append(out, '{', '}')
		out = append(out, b[end:]...)
		return out, []driver.Patch{{RuleID: StatsRule, Path: "/stats", BeforeSHA256: Digest(b), AfterSHA256: Digest(out)}}, nil
	}
	return bytes.Clone(b), []driver.Patch{}, nil
}
