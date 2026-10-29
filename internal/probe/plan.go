// Package probe prepares version-pinned, separate test candidates. Preparation
// is not an Android acceptance, core validation or connection certificate.
package probe

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/rezajafari0970/Config_manager/internal/repair"
	"github.com/rezajafari0970/Config_manager/internal/target"
)

const Version = "xray-26.7.28.probe-plan.v2"

type M = map[string]any
type Candidate struct {
	Rule               string `json:"rule"`
	SHA256             string `json:"sha256"`
	Config             []byte `json:"config"`
	RequiresConnection bool   `json:"requires_connection"`
}
type Plan struct {
	Target         string      `json:"target"`
	Version        string      `json:"version"`
	OriginalSHA256 string      `json:"original_sha256"`
	Kind           string      `json:"kind"`
	Code           string      `json:"code"`
	AndroidImport  string      `json:"android_import"`
	Candidates     []Candidate `json:"candidates"`
}

func Prepare(raw []byte) (p Plan) {
	p = Plan{Target: target.Pinned().ID, Version: Version, OriginalSHA256: repair.Digest(raw), AndroidImport: "not_run", Candidates: []Candidate{}}
	defer func() {
		if recover() != nil {
			p.Code = "preparation_recovered"
			p.Candidates = nil
		}
	}()
	defer func() {
		if p.Code == "prepared" {
			p.Candidates = verifiedTLSAlternatives(p.Candidates)
		}
	}()
	if len(raw) > repair.MaxBytes {
		p.Code = "inspection_size_limit"
		return
	}
	if !utf8.Valid(raw) {
		p.Code = "invalid_utf8"
		return
	}
	s := strings.TrimSpace(string(raw))
	if strings.HasPrefix(s, "{") || strings.HasPrefix(s, "\ufeff{") {
		p.Kind = "json"
		d := (repair.JSONDriver{}).Propose(raw)
		if d.Status == "blocked" || d.Status == "unverified" {
			p.Code = "structural_unresolved"
			return
		}
		b := raw
		rule := "original-json"
		if len(d.Candidate) > 0 {
			b = d.Candidate
			rule = repair.RulesVersion
		}
		p.Candidates = append(p.Candidates, Candidate{Rule: rule, SHA256: repair.Digest(b), Config: bytes.Clone(b)})
		p.Code = "prepared"
		return
	}
	var ob M
	var err error
	switch {
	case strings.HasPrefix(strings.ToLower(s), "vmess://"):
		p.Kind = "vmess"
		ob, err = vmess(s[len("vmess://"):])
	case strings.HasPrefix(strings.ToLower(s), "[interface]"):
		p.Kind = "wireguard"
		ob, err = wireguardINI(s)
	default:
		u, e := url.Parse(s)
		if e != nil {
			p.Code = "invalid_uri"
			return
		}
		p.Kind = strings.ToLower(u.Scheme)
		switch p.Kind {
		case "socks", "socks5":
			ob, err = socksURI(u)
		case "wireguard", "wg":
			ob, err = wireguardURI(u)
		case "hysteria", "hy":
			err = errors.New("hysteria1_unsupported_by_target_core")
		case "json", "custom":
			payload := strings.SplitN(strings.TrimPrefix(s, u.Scheme+"://"), "#", 2)[0]
			b, e := url.PathUnescape(payload)
			if e == nil && !strings.HasPrefix(strings.TrimSpace(b), "{") {
				var decoded []byte
				decoded, e = decode(b)
				b = string(decoded)
			}
			if e != nil || !strings.HasPrefix(strings.TrimSpace(b), "{") {
				p.Code = "custom_json_envelope_invalid"
				return
			}
			nested := Prepare([]byte(b))
			p.Candidates = nested.Candidates
			p.Code = nested.Code
			return
		default:
			ob, err = uri(u)
		}
	}
	if err != nil {
		p.Code = err.Error()
		return
	}
	root := M{"remarks": label(raw, p.Kind), "log": M{"loglevel": "warning"}, "inbounds": []any{M{"tag": "socks", "listen": "127.0.0.1", "port": 10808, "protocol": "socks", "settings": M{"auth": "noauth", "udp": true}}}, "outbounds": []any{ob}, "routing": M{}}
	if dns, ok := ob["__dns"]; ok {
		root["dns"] = dns
		delete(ob, "__dns")
	}
	p.Candidates = append(p.Candidates, candidate(root, Version+".uri-json", false))
	// This is only a hypothesis, never a repair approval. A verified end-to-end
	// connection with certificate verification must gate its recommendation.
	stream, _ := ob["streamSettings"].(M)
	if p.Kind == "trojan" && stream["security"] == "none" {
		stream["security"] = "tls"
		settings := ob["settings"].(M)
		if stream["tlsSettings"] == nil {
			stream["tlsSettings"] = M{"serverName": settings["address"], "allowInsecure": false}
		}
		stream["tlsSettings"].(M)["allowInsecure"] = false
		p.Candidates = append(p.Candidates, candidate(root, Version+".trojan-tls-hypothesis", true))
	}
	p.Code = "prepared"
	return
}
func candidate(m M, rule string, connection bool) Candidate {
	b, _ := json.Marshal(m)
	return Candidate{rule, repair.Digest(b), b, connection}
}

// The pinned core removed allowInsecure=true. Try strict certificate validation,
// never invent a certificate pin or turn verification off. Publish only after
// a real connection; an expired/mismatched remote certificate remains unresolved.
func verifiedTLSAlternatives(in []Candidate) []Candidate {
	out := append([]Candidate(nil), in...)
	for _, c := range in {
		var m M
		d := json.NewDecoder(bytes.NewReader(c.Config))
		d.UseNumber()
		if d.Decode(&m) != nil {
			continue
		}
		obs, _ := m["outbounds"].([]any)
		changed := false
		for _, v := range obs {
			ob, ok := v.(map[string]any)
			if !ok {
				continue
			}
			s, _ := ob["streamSettings"].(map[string]any)
			tls, _ := s["tlsSettings"].(map[string]any)
			if tls["allowInsecure"] == true {
				tls["allowInsecure"] = false
				changed = true
			}
		}
		if changed {
			out = append(out, candidate(m, Version+".strict-tls-hypothesis", true))
		}
	}
	return out
}
func decode(s string) ([]byte, error) {
	for _, e := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		if b, err := e.DecodeString(s); err == nil {
			return b, nil
		}
	}
	return nil, errors.New("invalid_base64")
}
func str(m M, k string) string {
	if m[k] == nil {
		return ""
	}
	return fmt.Sprint(m[k])
}
func def(s, f string) string {
	if s == "" {
		return f
	}
	return s
}
func port(s string) (int, error) {
	n, e := strconv.Atoi(s)
	if e != nil || n < 1 || n > 65535 {
		return 0, errors.New("invalid_port")
	}
	return n, nil
}
func csv(s string) []string {
	var r []string
	for _, v := range strings.Split(s, ",") {
		if v = strings.TrimSpace(v); v != "" {
			r = append(r, v)
		}
	}
	return r
}
func strictObject(b []byte) (M, error) {
	if !utf8.Valid(b) {
		return nil, errors.New("invalid_utf8")
	}
	r := repair.Analyze(b)
	for _, f := range r.Findings {
		switch f.Code {
		case "duplicate_json_key", "invalid_json", "json_depth_limit", "trailing_json_data":
			return nil, errors.New(f.Code)
		}
	}
	var m M
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	if d.Decode(&m) != nil || m == nil {
		return nil, errors.New("invalid_json_object")
	}
	return m, nil
}

func uri(u *url.URL) (M, error) {
	kind := strings.ToLower(u.Scheme)
	if kind != "vless" && kind != "trojan" && kind != "ss" && kind != "hy2" && kind != "hysteria2" {
		return nil, errors.New("protocol_driver_pending")
	}
	if kind == "ss" && u.User == nil {
		b, e := decode(u.Host + u.Path)
		if e != nil {
			return nil, e
		}
		v, e := url.Parse("ss://" + string(b))
		if e != nil {
			return nil, errors.New("invalid_shadowsocks")
		}
		v.RawQuery = u.RawQuery
		u = v
	}
	if u.Hostname() == "" || u.User == nil {
		return nil, errors.New("missing_endpoint_or_auth")
	}
	if u.Path != "" && u.Path != "/" {
		return nil, errors.New("uri_path_ambiguous")
	}
	n, e := port(u.Port())
	if e != nil {
		return nil, e
	}
	q, e := url.ParseQuery(u.RawQuery)
	if e != nil {
		return nil, errors.New("invalid_query_encoding")
	}
	for _, values := range q {
		for _, v := range values {
			if !utf8.ValidString(v) {
				return nil, errors.New("invalid_utf8")
			}
		}
	}
	allowed := strings.Fields("type security encryption flow headerType host path sni fp alpn allowInsecure insecure allowinsecure pbk sid spx pqv ech vcn pcs serviceName authority mode extra xhttpMode xhttpExtra fm obfs obfs-password pinSHA256 mport")
	for k, values := range q {
		known := false
		for _, a := range allowed {
			if k == a {
				known = true
				break
			}
		}
		if !known {
			return nil, errors.New("unknown_query_parameter")
		}
		if len(values) != 1 {
			return nil, errors.New("ambiguous_query_parameter")
		}
	}
	credential := u.User.Username()
	if pass, ok := u.User.Password(); ok {
		credential += ":" + pass
	}
	if credential == "" {
		return nil, errors.New("missing_auth")
	}
	if !utf8.ValidString(credential) {
		return nil, errors.New("invalid_utf8")
	}
	settings := M{"address": u.Hostname(), "port": n}
	protocol := kind
	switch kind {
	case "vless":
		settings["id"] = credential
		settings["encryption"] = def(q.Get("encryption"), "none")
		settings["flow"] = q.Get("flow")
	case "trojan":
		settings["password"] = credential
		settings["flow"] = q.Get("flow")
		if q.Get("security") == "" {
			q.Set("security", "tls")
		}
	case "ss":
		if !strings.Contains(credential, ":") {
			b, e := decode(credential)
			if e != nil {
				return nil, e
			}
			if !utf8.Valid(b) {
				return nil, errors.New("invalid_utf8")
			}
			credential = string(b)
		}
		method, pass, ok := strings.Cut(credential, ":")
		if !ok || method == "" || pass == "" {
			return nil, errors.New("ambiguous_shadowsocks_auth")
		}
		protocol = "shadowsocks"
		settings["method"] = method
		settings["password"] = pass
	case "hy2", "hysteria2":
		protocol = "hysteria"
		settings["version"] = 2
		q.Set("type", "hysteria")
		q.Set("alpn", "h3")
		q.Set("security", "tls")
	}
	stream, e := transport(q, u.Hostname())
	if e != nil {
		return nil, e
	}
	if protocol == "hysteria" {
		stream["hysteriaSettings"] = M{"version": 2, "auth": credential}
		if q.Get("obfs") != "" && q.Get("obfs") != "salamander" {
			return nil, errors.New("unsupported_hysteria_obfs")
		}
		if q.Get("obfs-password") != "" {
			stream["finalmask"] = M{"udp": []any{M{"type": "salamander", "settings": M{"password": q.Get("obfs-password")}}}}
		}
		if q.Get("mport") != "" {
			if _, err := portRanges(q.Get("mport")); err != nil {
				return nil, err
			}
			fm, ok := stream["finalmask"].(M)
			if !ok {
				fm = M{}
				stream["finalmask"] = fm
			}
			fm["quicParams"] = M{"udpHop": M{"ports": q.Get("mport"), "interval": "30"}}
		}
		if q.Get("pinSHA256") != "" {
			stream["tlsSettings"].(M)["pinnedPeerCertSha256"] = q.Get("pinSHA256")
		}
	}
	return M{"tag": "proxy", "protocol": protocol, "settings": settings, "streamSettings": stream}, nil
}

func transport(q url.Values, server string) (M, error) {
	network := def(q.Get("type"), "tcp")
	s := M{"network": network}
	host, path := q.Get("host"), def(q.Get("path"), "/")
	sniHost := host
	switch network {
	case "tcp", "raw":
		header := def(q.Get("headerType"), "none")
		h := M{"type": header}
		if header != "none" && header != "http" {
			return nil, errors.New("unsupported_tcp_header")
		}
		if header == "http" {
			h["request"] = M{"version": "1.1", "method": "GET", "path": csv(path), "headers": M{"Host": csv(host), "User-Agent": []string{"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.6478.122 Mobile Safari/537.36"}, "Accept-Encoding": []string{"gzip, deflate"}, "Connection": []string{"keep-alive"}, "Pragma": "no-cache"}}
			if hs := csv(host); len(hs) > 0 {
				sniHost = hs[0]
			}
		}
		s["tcpSettings"] = M{"header": h}
	case "ws":
		s["wsSettings"] = M{"host": host, "path": path}
	case "httpupgrade":
		s["httpupgradeSettings"] = M{"host": host, "path": path}
	case "grpc":
		s["grpcSettings"] = M{"serviceName": q.Get("serviceName"), "authority": q.Get("authority"), "multiMode": q.Get("mode") == "multi", "idle_timeout": 60, "health_check_timeout": 20}
		sniHost = q.Get("authority")
	case "xhttp":
		x := M{"host": host, "path": path, "mode": def(q.Get("xhttpMode"), q.Get("mode"))}
		extra := def(q.Get("xhttpExtra"), q.Get("extra"))
		if extra != "" {
			v, e := strictObject([]byte(extra))
			if e != nil {
				return nil, errors.New("invalid_xhttp_extra")
			}
			x["extra"] = v
		}
		s["xhttpSettings"] = x
	case "h2", "http":
		s["network"] = "h2"
		s["httpSettings"] = M{"host": csv(host), "path": path}
		if hs := csv(host); len(hs) > 0 {
			sniHost = hs[0]
		}
	case "hysteria":
	default:
		return nil, errors.New("transport_driver_pending")
	}
	sec := def(q.Get("security"), "none")
	if sec != "none" && sec != "tls" && sec != "reality" {
		return nil, errors.New("unsupported_security")
	}
	s["security"] = sec
	name := q.Get("sni")
	if name == "" {
		if sniHost != "" && net.ParseIP(sniHost) == nil {
			name = sniHost
		} else if net.ParseIP(server) == nil {
			name = server
		} else {
			name = sniHost
		}
	}
	tls := M{"allowInsecure": false}
	if name != "" {
		tls["serverName"] = name
	}
	for _, k := range []string{"insecure", "allowInsecure", "allowinsecure"} {
		if q.Get(k) == "1" || q.Get(k) == "true" {
			tls["allowInsecure"] = true
		}
	}
	for from, to := range map[string]string{"fp": "fingerprint", "ech": "echConfigList", "vcn": "verifyPeerCertByName", "pcs": "pinnedPeerCertSha256", "pbk": "publicKey", "sid": "shortId", "spx": "spiderX", "pqv": "mldsa65Verify"} {
		if q.Get(from) != "" {
			tls[to] = q.Get(from)
		}
	}
	if q.Get("pcs") != "" {
		tls["allowInsecure"] = false
	}
	if q.Get("alpn") != "" {
		tls["alpn"] = csv(q.Get("alpn"))
	}
	if sec == "tls" || sec == "none" {
		s["tlsSettings"] = tls
	} else {
		s["realitySettings"] = tls
	}
	if q.Get("fm") != "" {
		v, e := strictObject([]byte(q.Get("fm")))
		if e != nil {
			return nil, errors.New("invalid_finalmask")
		}
		s["finalmask"] = v
	}
	return s, nil
}

func vmess(s string) (M, error) {
	b, e := decode(s)
	if e != nil {
		return nil, e
	}
	m, e := strictObject(b)
	if e != nil {
		return nil, e
	}
	for k := range m {
		if !strings.Contains(" v ps add port id aid scy net type host path tls sni fp alpn insecure vcn pcs ", " "+k+" ") {
			return nil, errors.New("unknown_vmess_field")
		}
	}
	n, e := port(str(m, "port"))
	if e != nil {
		return nil, e
	}
	if str(m, "add") == "" || str(m, "id") == "" {
		return nil, errors.New("missing_endpoint_or_auth")
	}
	if aid := str(m, "aid"); aid != "" && aid != "0" {
		return nil, errors.New("legacy_vmess_alterid_unsupported")
	}
	q := url.Values{}
	for from, to := range map[string]string{"net": "type", "type": "headerType", "host": "host", "path": "path", "tls": "security", "sni": "sni", "fp": "fp", "alpn": "alpn", "insecure": "insecure", "vcn": "vcn", "pcs": "pcs"} {
		q.Set(to, str(m, from))
	}
	if q.Get("type") == "grpc" {
		q.Set("mode", str(m, "type"))
		q.Set("serviceName", str(m, "path"))
		q.Set("authority", str(m, "host"))
	}
	stream, e := transport(q, str(m, "add"))
	if e != nil {
		return nil, e
	}
	return M{"tag": "proxy", "protocol": "vmess", "settings": M{"address": str(m, "add"), "port": n, "id": str(m, "id"), "security": def(str(m, "scy"), "auto")}, "streamSettings": stream}, nil
}
