package residential

import (
	"bytes"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/repair"
	"net"
	"net/url"
	"strconv"
	"strings"
)

const SingBoxID = "sing-box-1.14.4"
const SingBoxSHA256 = "1264361b5f8868c8768f7555d3e77e20118bf503c41ac08bb290a0d99056aa3a"

// NativeDriver retains protocols that cannot be represented as Xray outbounds.
// Its common envelope is native sing-box JSON; it is never advertised as Xray
// import compatible. Unknown URI extensions and complex JSON routes stay explicit.
type NativeDriver struct{}

func (NativeDriver) ID() string { return SingBoxID }
func (NativeDriver) Prepare(raw []byte) Plan {
	p := Plan{Version: Version, Driver: "sing-box", CoreSHA256: SingBoxSHA256, Code: "native_protocol_unsupported", Variants: []Execution{}}
	var config map[string]any
	input := bytes.TrimSpace(raw)
	if bytes.HasPrefix(input, []byte("{")) {
		if repair.ValidateJSON(input) != nil {
			p.Code = "native_json_ambiguous"
			return p
		}
		d := json.NewDecoder(bytes.NewReader(input))
		d.UseNumber()
		if d.Decode(&config) != nil {
			return p
		}
		nativeFound := false
		if obs, ok := config["outbounds"].([]any); ok {
			for _, v := range obs {
				if ob, ok := v.(map[string]any); ok {
					if ob["type"] == "tuic" || ob["type"] == "hysteria" || ob["type"] == "hysteria2" {
						nativeFound = true
					}
				}
			}
		}
		if !nativeFound {
			return p
		}
		for k := range config {
			if k != "log" && k != "dns" && k != "inbounds" && k != "outbounds" && k != "route" {
				p.Code = "native_extension_unsupported"
				return p
			}
		}
	} else {
		u, e := url.Parse(string(input))
		if e != nil || u.Hostname() == "" || u.Path != "" || (u.Scheme != "tuic" && u.Scheme != "hysteria" && u.Scheme != "hy") {
			return p
		}
		port, e := strconv.Atoi(u.Port())
		if e != nil || port < 1 || port > 65535 {
			p.Code = "native_port_invalid"
			return p
		}
		q, e := url.ParseQuery(u.RawQuery)
		if e != nil {
			return p
		}
		allowed := " sni peer alpn insecure allow_insecure "
		if u.Scheme == "tuic" {
			allowed += " congestion_control udp_relay_mode "
		} else {
			allowed += " auth upmbps downmbps obfs protocol "
		}
		for k, v := range q {
			if len(v) != 1 || !strings.Contains(allowed, " "+k+" ") {
				p.Code = "native_extension_unsupported"
				return p
			}
		}
		for _, k := range []string{"insecure", "allow_insecure"} {
			if q.Get(k) != "" && q.Get(k) != "0" && q.Get(k) != "false" {
				p.Code = "native_tls_verification_required"
				return p
			}
		}
		if q.Get("sni") != "" && q.Get("peer") != "" && q.Get("sni") != q.Get("peer") {
			p.Code = "native_tls_name_ambiguous"
			return p
		}
		sni := q.Get("sni")
		if sni == "" {
			sni = q.Get("peer")
		}
		if sni == "" {
			sni = u.Hostname()
		}
		tls := map[string]any{"enabled": true, "server_name": sni}
		if q.Get("alpn") != "" {
			tls["alpn"] = strings.Split(q.Get("alpn"), ",")
		}
		ob := map[string]any{"type": u.Scheme, "tag": "proxy", "server": u.Hostname(), "server_port": port, "tls": tls}
		if u.Scheme == "tuic" {
			if u.User == nil {
				p.Code = "native_auth_required"
				return p
			}
			password, ok := u.User.Password()
			if !ok || u.User.Username() == "" {
				p.Code = "native_auth_required"
				return p
			}
			ob["uuid"], ob["password"] = u.User.Username(), password
			for _, k := range []string{"congestion_control", "udp_relay_mode"} {
				if q.Get(k) != "" {
					ob[k] = q.Get(k)
				}
			}
		} else {
			ob["type"] = "hysteria"
			auth := q.Get("auth")
			if u.User != nil {
				if _, ok := u.User.Password(); ok {
					p.Code = "native_auth_ambiguous"
					return p
				}
				if auth != "" && auth != u.User.Username() {
					p.Code = "native_auth_ambiguous"
					return p
				}
				auth = u.User.Username()
			}
			if auth == "" {
				p.Code = "native_auth_required"
				return p
			}
			ob["auth_str"] = auth
			if q.Get("protocol") != "" && q.Get("protocol") != "udp" {
				p.Code = "native_transport_unsupported"
				return p
			}
			for _, pair := range [][2]string{{"upmbps", "up_mbps"}, {"downmbps", "down_mbps"}} {
				n, e := strconv.Atoi(q.Get(pair[0]))
				if e != nil || n < 1 {
					p.Code = "native_bandwidth_required"
					return p
				}
				ob[pair[1]] = n
			}
			if q.Get("obfs") != "" {
				ob["obfs"] = q.Get("obfs")
			}
		}
		config = map[string]any{"log": map[string]any{"disabled": true}, "inbounds": []any{map[string]any{"type": "socks", "tag": "socks", "listen": "127.0.0.1", "listen_port": 10808}}, "outbounds": []any{ob}, "route": map[string]any{"final": "proxy"}}
		input, _ = json.Marshal(config)
	}
	obs, ok := config["outbounds"].([]any)
	if !ok || len(obs) != 1 {
		p.Code = "native_selectable_paths_unsupported"
		return p
	}
	ob, ok := obs[0].(map[string]any)
	if !ok {
		return p
	}
	if ob["type"] != "tuic" && ob["type"] != "hysteria" && ob["type"] != "hysteria2" {
		return p
	}
	if ob["detour"] != nil || ob["domain_resolver"] != nil {
		p.Code = "native_dialer_unsupported"
		return p
	}
	tls, _ := ob["tls"].(map[string]any)
	if tls["enabled"] != true || tls["insecure"] == true {
		p.Code = "native_tls_verification_required"
		return p
	}
	route, _ := config["route"].(map[string]any)
	for k, v := range route {
		if k != "final" || v != ob["tag"] {
			p.Code = "native_selectable_paths_unsupported"
			return p
		}
	}
	server, _ := ob["server"].(string)
	complete := net.ParseIP(server) != nil
	// Hostname endpoints can be screened with supervisor-pinned resolution, but
	// do not certify a portable DNS bootstrap until a native DNS driver proves it.
	paths := []string{"single-proxy-route"}
	if !complete {
		paths = []string{"native-bootstrap-unproven"}
	}
	commonOB := map[string]any{}
	for k, v := range ob {
		commonOB[k] = v
	}
	commonOB["tag"] = "@shenvpn"
	common := map[string]any{"log": map[string]any{"disabled": true}, "inbounds": []any{map[string]any{"type": "socks", "tag": "socks", "listen": "127.0.0.1", "listen_port": 10808}}, "outbounds": []any{commonOB}, "route": map[string]any{"final": "@shenvpn"}}
	b, _ := json.Marshal(common)
	p.Variants = []Execution{{Name: "raw", ImportRule: "native-original-or-uri-v1", SHA256: Digest(input), Config: input, Paths: paths, PathsComplete: complete}, {Name: "template", ImportRule: "native-common-v1", SHA256: Digest(b), Config: b, Paths: paths, PathsComplete: complete}}
	p.Code = "prepared"
	return p
}
