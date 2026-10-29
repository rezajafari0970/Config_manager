package collect

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// InspectConfig only labels the received bytes. It never repairs, reserializes,
// executes or claims Android/core/network compatibility.
func InspectConfig(raw []byte, complete bool, kind string) string {
	if !complete {
		return "unknown"
	}
	b := bytes.TrimSpace(bytes.TrimPrefix(raw, []byte{0xef, 0xbb, 0xbf}))
	switch kind {
	case "json":
		if !json.Valid(b) {
			return "invalid"
		}
		if _, e := documentKeys(b); e != nil {
			return "invalid"
		}
		var root map[string]json.RawMessage
		if json.Unmarshal(b, &root) != nil {
			return "invalid"
		}
		for _, key := range []string{"stats", "dns", "policy", "routing", "log"} {
			if v, ok := root[key]; ok && (len(bytes.TrimSpace(v)) == 0 || bytes.TrimSpace(v)[0] != '{') {
				return "invalid"
			}
		}
		if value, ok := root["outbounds"]; ok {
			var outbounds []map[string]json.RawMessage
			if json.Unmarshal(value, &outbounds) != nil || len(outbounds) == 0 {
				return "invalid"
			}
			proxy := false
			for _, o := range outbounds {
				var protocol string
				if json.Unmarshal(o["protocol"], &protocol) != nil || protocol == "" {
					return "invalid"
				}
				if protocol != "freedom" && protocol != "blackhole" && protocol != "dns" {
					proxy = true
				}
				for _, key := range []string{"settings", "streamSettings"} {
					if value, ok := o[key]; ok && (!bytes.HasPrefix(bytes.TrimSpace(value), []byte("{"))) {
						return "invalid"
					}
				}
			}
			if !proxy {
				return "invalid"
			}
			return "valid"
		}
		return "unknown"
	case "vmess":
		_, payload, ok := strings.Cut(string(b), "://")
		if !ok {
			return "invalid"
		}
		payload, _, _ = strings.Cut(payload, "#")
		decoded, e := decodeBase64(payload)
		if e != nil {
			return "invalid"
		}
		var v map[string]json.RawMessage
		if json.Unmarshal(decoded, &v) != nil {
			return "invalid"
		}
		for _, key := range []string{"add", "id", "port"} {
			if len(v[key]) == 0 || string(v[key]) == `""` || string(v[key]) == "null" {
				return "invalid"
			}
		}
		if _, e = documentKeys(decoded); e != nil {
			return "invalid"
		}
		return "valid"
	case "ss":
		if _, e := ssIdentity(string(b)); e != nil {
			return "invalid"
		}
		return "valid"
	case "wireguard":
		if bytes.HasPrefix(bytes.ToLower(b), []byte("[interface]")) {
			v := strings.ToLower(string(b))
			for _, key := range []string{"[peer]", "privatekey", "publickey", "endpoint"} {
				if !strings.Contains(v, key) {
					return "invalid"
				}
			}
			return "valid"
		}
	}
	switch kind {
	case "vless", "trojan", "hysteria", "hysteria2", "hy2", "wireguard", "wg", "socks", "socks5":
		u, e := url.Parse(string(b))
		if e != nil || u.Hostname() == "" {
			return "invalid"
		}
		if _, e = url.ParseQuery(u.RawQuery); e != nil {
			return "invalid"
		}
		if port := u.Port(); port != "" {
			n, e := strconv.Atoi(port)
			if e != nil || n < 1 || n > 65535 {
				return "invalid"
			}
		}
		if (kind == "vless" || kind == "trojan") && (u.User == nil || u.User.Username() == "") {
			return "invalid"
		}
		return "valid"
	}
	return "unknown"
}

func IsContentError(err error) bool {
	if err == nil {
		return false
	}
	var syntax *json.SyntaxError
	if errors.As(err, &syntax) {
		return true
	}
	switch err.Error() {
	case "invalid_json", "ambiguous_json", "invalid_json_string":
		return true
	}
	return false
}

func ContentState(path string, valid, invalid, unknown int) string {
	if invalid > 0 {
		if valid+unknown > 0 {
			return "partial"
		}
		return "invalid"
	}
	if valid > 0 {
		return "config"
	}
	if unknown > 0 {
		return "unverified"
	}
	f, e := os.Open(path)
	if e != nil {
		return "unknown"
	}
	defer f.Close()
	r := bufio.NewReader(f)
	prefix := make([]byte, 0, 4096)
	for len(prefix) < 4096 {
		b, e := r.ReadByte()
		if e == io.EOF {
			break
		}
		if e != nil {
			return "unknown"
		}
		if len(prefix) == 0 && (b == ' ' || b == '\t' || b == '\r' || b == '\n') {
			continue
		}
		prefix = append(prefix, b)
	}
	b := bytes.TrimSpace(bytes.TrimPrefix(prefix, []byte{0xef, 0xbb, 0xbf}))
	if len(b) == 0 || bytes.Equal(b, []byte("[]")) || bytes.Equal(b, []byte("null")) {
		return "empty"
	}
	lower := strings.ToLower(string(b))
	if strings.HasPrefix(lower, "<") || strings.Contains(lower, `"error"`) || strings.Contains(lower, "no valid config") || strings.Contains(lower, "no config") || strings.Contains(lower, "error:") {
		return "error_response"
	}
	if b[0] == '{' || b[0] == '[' {
		if !json.Valid(b) {
			return "invalid"
		}
	}
	return "unrecognized"
}

type InspectionRegistry struct{ drivers map[string]InspectionDriver }
type basicInspector struct{ kind string }

func (d basicInspector) Kind() string { return d.kind }
func (d basicInspector) Inspect(raw []byte, complete bool) string {
	return InspectConfig(raw, complete, d.kind)
}
func NewInspectionRegistry() *InspectionRegistry {
	r := &InspectionRegistry{drivers: map[string]InspectionDriver{}}
	for _, kind := range []string{"json", "vless", "vmess", "trojan", "ss", "hysteria", "hysteria2", "hy2", "wireguard", "wg", "socks", "socks5"} {
		r.Register(basicInspector{kind})
	}
	return r
}
func (r *InspectionRegistry) Register(driver InspectionDriver) { r.drivers[driver.Kind()] = driver }
func (r *InspectionRegistry) Inspect(raw []byte, complete bool, kind string) (quality string) {
	quality = "unknown"
	defer func() {
		if recover() != nil {
			quality = "unknown"
		}
	}()
	if d := r.drivers[kind]; d != nil {
		quality = d.Inspect(raw, complete)
	}
	if quality != "valid" && quality != "invalid" {
		quality = "unknown"
	}
	return
}
