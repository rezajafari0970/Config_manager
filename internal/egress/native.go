package egress

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
)

const NativeID = "sing-box-1.14.4"
const NativeSHA = "1264361b5f8868c8768f7555d3e77e20118bf503c41ac08bb290a0d99056aa3a"

func init() { Cores[NativeID] = NativeSHA }
func native(raw []byte) (Plan, bool) {
	p := Plan{Driver: "sing-box", Code: "native_import_unsupported", Format: "sing-box", Variants: []Variant{}}
	var original map[string]any
	if json.Unmarshal(raw, &original) == nil {
		os, ok := original["outbounds"].([]any)
		if !ok || len(os) == 0 {
			return p, false
		}
		o, ok := os[0].(map[string]any)
		if !ok {
			return p, false
		}
		typ, _ := o["type"].(string)
		if typ != "tuic" && typ != "hysteria" && typ != "hysteria2" {
			return p, false
		}
		if len(os) != 1 {
			return p, true
		}
		for k := range original {
			if k != "outbounds" && k != "inbounds" && k != "log" && k != "dns" && k != "route" {
				return p, true
			}
		}
		if route, ok := original["route"].(map[string]any); ok {
			if rules, ok := route["rules"].([]any); ok && len(rules) > 0 {
				return p, true
			}
		}
		if o["detour"] != nil {
			return p, true
		}
	} else {
		u, e := url.Parse(strings.TrimSpace(string(raw)))
		if e != nil {
			return p, false
		}
		typ := u.Scheme
		if typ == "hy2" {
			typ = "hysteria2"
		}
		if typ != "tuic" && typ != "hysteria" && typ != "hysteria2" {
			return p, false
		}
		port, e := strconv.Atoi(u.Port())
		if e != nil || port < 1 || port > 65535 || u.Hostname() == "" {
			return p, true
		}
		q := u.Query()
		allowed := map[string]bool{"sni": true, "peer": true, "alpn": true, "insecure": true, "allowInsecure": true, "allow_insecure": true, "obfs": true, "obfs-password": true, "upmbps": true, "downmbps": true, "up_mbps": true, "down_mbps": true, "auth": true, "congestion_control": true, "udp_relay_mode": true, "pinSHA256": true}
		for k, vs := range q {
			if !allowed[k] || len(vs) != 1 {
				return p, true
			}
		}
		tls := map[string]any{"enabled": true}
		name := q.Get("sni")
		if name == "" {
			name = q.Get("peer")
		}
		if name != "" {
			tls["server_name"] = name
		}
		for _, k := range []string{"insecure", "allowInsecure", "allow_insecure"} {
			if v := q.Get(k); v != "" {
				if v == "1" || v == "true" {
					tls["insecure"] = true
				} else if v != "0" && v != "false" {
					return p, true
				}
			}
		}
		if q.Get("pinSHA256") != "" {
			return p, true
		} // No guessed pin representation.
		if v := q.Get("alpn"); v != "" {
			tls["alpn"] = strings.Split(v, ",")
		}
		o := map[string]any{"type": typ, "tag": "proxy", "server": u.Hostname(), "server_port": port, "tls": tls}
		auth := ""
		if u.User != nil {
			auth = u.User.Username()
			if v, ok := u.User.Password(); ok {
				auth += "\x00" + v
			}
		}
		if typ == "tuic" {
			parts := strings.SplitN(auth, "\x00", 2)
			if len(parts) != 2 {
				return p, true
			}
			o["uuid"] = parts[0]
			o["password"] = parts[1]
			for _, k := range []string{"congestion_control", "udp_relay_mode"} {
				if v := q.Get(k); v != "" {
					o[k] = v
				}
			}
		} else if typ == "hysteria2" {
			if strings.Contains(auth, "\x00") {
				auth = strings.Replace(auth, "\x00", ":", 1)
			}
			o["password"] = auth
			if v := q.Get("obfs"); v != "" {
				if v != "salamander" {
					return p, true
				}
				o["obfs"] = map[string]any{"type": "salamander", "password": q.Get("obfs-password")}
			}
		} else {
			if q.Get("auth") != "" {
				auth = q.Get("auth")
			}
			o["auth_str"] = auth
			for _, pair := range [][2]string{{"up_mbps", "upmbps"}, {"down_mbps", "downmbps"}} {
				v := q.Get(pair[0])
				if v == "" {
					v = q.Get(pair[1])
				}
				if v != "" {
					n, e := strconv.Atoi(v)
					if e != nil || n <= 0 {
						return p, true
					}
					o[pair[0]] = n
				}
			}
		}
		original = map[string]any{"log": map[string]any{"disabled": true}, "inbounds": []any{map[string]any{"type": "socks", "listen": "127.0.0.1", "listen_port": 10808}}, "outbounds": []any{o}, "route": map[string]any{"final": "proxy"}}
	}
	b, _ := json.Marshal(original)
	var common map[string]any
	json.Unmarshal(b, &common)
	o := common["outbounds"].([]any)[0].(map[string]any)
	o["tag"] = "@shenvpn"
	common["inbounds"] = []any{map[string]any{"type": "socks", "listen": "127.0.0.1", "listen_port": 10808}}
	common["route"] = map[string]any{"final": "@shenvpn"}
	common["log"] = map[string]any{"disabled": true}
	tpl, _ := json.Marshal(common)
	p.Complete = true
	p.Code = "prepared"
	p.Variants = []Variant{{"raw", Hash(b), b}, {"template", Hash(tpl), tpl}}
	return p, true
}
