package probe

import (
	"encoding/json"
	"errors"
	"github.com/rezajafari0970/Config_manager/internal/repair"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// These adapters preserve supplied authentication and endpoint semantics. Core
// validation and a real connection, rather than successful mapping, approve them.
func socksURI(u *url.URL) (M, error) {
	if u.Hostname() == "" {
		return nil, errors.New("missing_endpoint")
	}
	n, e := port(u.Port())
	if e != nil {
		return nil, e
	}
	if u.RawQuery != "" || u.Path != "" && u.Path != "/" {
		return nil, errors.New("socks_extension_requires_review")
	}
	s := M{"address": u.Hostname(), "port": n}
	if u.User != nil {
		user := u.User.Username()
		pass, ok := u.User.Password()
		if !ok {
			b, e := decode(user)
			if e != nil {
				return nil, errors.New("socks_auth_invalid")
			}
			user, pass, ok = strings.Cut(string(b), ":")
			if !ok {
				return nil, errors.New("socks_auth_invalid")
			}
		}
		if user != "" {
			s["user"] = user
			s["pass"] = pass
		} else if pass != "" {
			return nil, errors.New("socks_auth_ambiguous")
		}
	}
	return M{"tag": "proxy", "protocol": "socks", "settings": s}, nil
}
func wireguardURI(u *url.URL) (M, error) {
	if u.User == nil || u.Hostname() == "" || u.Path != "" && u.Path != "/" {
		return nil, errors.New("wireguard_endpoint_or_key_missing")
	}
	n, e := port(u.Port())
	if e != nil {
		return nil, e
	}
	q, e := url.ParseQuery(u.RawQuery)
	if e != nil {
		return nil, errors.New("invalid_query_encoding")
	}
	for k, v := range q {
		if len(v) != 1 {
			return nil, errors.New("ambiguous_query_parameter")
		}
		if !strings.Contains(" address publickey presharedkey mtu reserved keepalive allowedips ", " "+k+" ") {
			return nil, errors.New("wireguard_extension_unsupported")
		}
	}
	key := u.User.Username()
	if p, ok := u.User.Password(); ok {
		key += ":" + p
	}
	peer := map[string]string{"publickey": q.Get("publickey"), "presharedkey": q.Get("presharedkey"), "endpoint": net.JoinHostPort(u.Hostname(), strconv.Itoa(n)), "persistentkeepalive": q.Get("keepalive"), "allowedips": q.Get("allowedips")}
	return wireguard(map[string]string{"privatekey": key, "address": q.Get("address"), "mtu": q.Get("mtu"), "reserved": q.Get("reserved")}, []map[string]string{peer})
}
func wireguardINI(raw string) (M, error) {
	iface := map[string]string{}
	peers := []map[string]string{}
	var section map[string]string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		switch strings.ToLower(line) {
		case "[interface]":
			if section != nil {
				return nil, errors.New("wireguard_duplicate_interface")
			}
			section = iface
			continue
		case "[peer]":
			section = map[string]string{}
			peers = append(peers, section)
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		k = strings.ToLower(strings.TrimSpace(k))
		v = strings.TrimSpace(v)
		if !ok || section == nil || v == "" {
			return nil, errors.New("wireguard_ini_invalid")
		}
		if _, ok = section[k]; ok {
			return nil, errors.New("wireguard_duplicate_key")
		}
		section[k] = v
	}
	return wireguard(iface, peers)
}
func wireguard(iface map[string]string, peers []map[string]string) (M, error) {
	for k := range iface {
		if !strings.Contains(" privatekey address mtu reserved dns ", " "+k+" ") {
			return nil, errors.New("wireguard_extension_unsupported")
		}
	}
	if iface["privatekey"] == "" || len(peers) == 0 {
		return nil, errors.New("wireguard_key_or_peer_missing")
	}
	// An interface DNS override changes routing semantics and cannot be dropped.
	dns := csv(iface["dns"])
	for _, v := range dns {
		if net.ParseIP(v) == nil {
			return nil, errors.New("wireguard_dns_search_domain_unsupported")
		}
	}
	addresses := csv(iface["address"])
	if len(addresses) == 0 {
		addresses = []string{"172.16.0.2/32"}
	}
	s := M{"secretKey": iface["privatekey"], "address": addresses, "noKernelTun": true}
	ps := []any{}
	if iface["mtu"] != "" {
		n, e := strconv.Atoi(iface["mtu"])
		if e != nil || n < 576 || n > 65535 {
			return nil, errors.New("wireguard_mtu_invalid")
		}
		s["mtu"] = n
	}
	if iface["reserved"] != "" {
		values := csv(iface["reserved"])
		if len(values) != 3 {
			return nil, errors.New("wireguard_reserved_invalid")
		}
		r := []int{}
		for _, v := range values {
			n, e := strconv.Atoi(v)
			if e != nil || n < 0 || n > 255 {
				return nil, errors.New("wireguard_reserved_invalid")
			}
			r = append(r, n)
		}
		s["reserved"] = r
	}
	for _, p := range peers {
		for k := range p {
			if !strings.Contains(" publickey presharedkey endpoint persistentkeepalive allowedips ", " "+k+" ") {
				return nil, errors.New("wireguard_extension_unsupported")
			}
		}
		h, po, e := net.SplitHostPort(p["endpoint"])
		if e != nil || h == "" {
			return nil, errors.New("wireguard_endpoint_invalid")
		}
		if _, e = port(po); e != nil {
			return nil, e
		}
		if p["publickey"] == "" {
			return nil, errors.New("wireguard_public_key_missing")
		}
		peer := M{"publicKey": p["publickey"], "endpoint": p["endpoint"]}
		if p["presharedkey"] != "" {
			peer["preSharedKey"] = p["presharedkey"]
		}
		if p["persistentkeepalive"] != "" {
			n, e := strconv.Atoi(p["persistentkeepalive"])
			if e != nil || n < 0 || n > 65535 {
				return nil, errors.New("wireguard_keepalive_invalid")
			}
			peer["keepAlive"] = n
		}
		if p["allowedips"] != "" {
			peer["allowedIPs"] = csv(p["allowedips"])
		}
		ps = append(ps, peer)
	}
	s["peers"] = ps
	out := M{"tag": "proxy", "protocol": "wireguard", "settings": s}
	if len(dns) > 0 {
		out["__dns"] = M{"servers": dns}
	}
	return out, nil
}

func label(raw []byte, kind string) string {
	u, e := url.Parse(strings.TrimSpace(string(raw)))
	if e == nil && u.Fragment != "" {
		return u.Fragment
	}
	if kind == "vmess" {
		b, e := decode(strings.TrimPrefix(strings.TrimSpace(string(raw)), "vmess://"))
		if e == nil {
			var m M
			if json.Unmarshal(b, &m) == nil {
				if v, ok := m["ps"].(string); ok && v != "" {
					return v
				}
			}
		}
	}
	return strings.ToUpper(kind) + " · " + repair.Digest(raw)[:10]
}

func portRanges(s string) ([]string, error) {
	out := csv(s)
	if len(out) == 0 {
		return nil, errors.New("hysteria_port_hopping_invalid")
	}
	for _, v := range out {
		a, b, rangeOK := strings.Cut(v, "-")
		lo, e := port(strings.TrimSpace(a))
		if e != nil {
			return nil, errors.New("hysteria_port_hopping_invalid")
		}
		if rangeOK {
			hi, e := port(strings.TrimSpace(b))
			if e != nil || hi < lo {
				return nil, errors.New("hysteria_port_hopping_invalid")
			}
		}
	}
	return out, nil
}
