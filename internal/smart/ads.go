package smart

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/rezajafari0970/Config_manager/internal/probe"
	"github.com/rezajafari0970/Config_manager/internal/repair"
)

const AdsVersion = "xray-26.7.28.smart-ads.v2"

func PrepareAds(b []byte) probe.Plan {
	p := Prepare(b)
	p.Version = AdsVersion
	for i := range p.Candidates {
		p.Candidates[i].Rule = AdsVersion
	}
	return p
}

// ComposeAds keeps one whole Splash input as the exit and the ordinary inputs
// as its carrier. Compose remains the single owner of the common Smart envelope.
// A balanced Splash input is preserved as a pool, never flattened into main.
func ComposeAds(main []Member, ads Member) ([]byte, error) {
	if len(main) == 0 {
		return nil, errors.New("smart_ads_main_unavailable")
	}
	mainCount := 0
	for _, m := range main {
		root, e := object(m.Body)
		if e != nil {
			return nil, e
		}
		nodes, ok := root["outbounds"].([]any)
		if !ok {
			return nil, errors.New("smart_missing_outbounds")
		}
		entries, e := entryTags(root, nodes)
		if e != nil {
			return nil, e
		}
		mainCount += len(entries)
	}
	combined := append(append([]Member{}, main...), ads)
	body, e := Compose(combined)
	if e != nil {
		return nil, e
	}
	root, e := object(body)
	if e != nil {
		return nil, e
	}
	nodes := root["outbounds"].([]any)
	mapping := map[string]string{}
	adsTags := map[string]bool{}
	entries := []string{}
	adsPrefix := fmt.Sprintf("member-%04d-", len(combined))
	for _, v := range nodes {
		ob := v.(M)
		tag := ob["tag"].(string)
		mapping[tag] = tag
		if strings.HasPrefix(tag, "proxy-") {
			n, _ := strconv.Atoi(strings.TrimPrefix(tag, "proxy-"))
			if n > mainCount {
				entries = append(entries, tag)
				adsTags[tag] = true
			}
		} else if strings.HasPrefix(tag, adsPrefix) {
			mapping[tag] = "ads-member-" + strings.TrimPrefix(tag, adsPrefix)
			adsTags[tag] = true
		}
	}
	if len(entries) == 0 {
		return nil, errors.New("smart_ads_missing_entry")
	}
	for i, tag := range entries {
		if len(entries) == 1 {
			mapping[tag] = "ads-01"
		} else {
			mapping[tag] = fmt.Sprintf("ads-pool-%04d", i+1)
		}
	}
	routing := root["routing"].(M)
	balancers := routing["balancers"].([]any)
	bootstrap := balancers[1].(M)
	allowed := []any{}
	for _, tag := range bootstrap["selector"].([]any) {
		if !adsTags[tag.(string)] {
			allowed = append(allowed, tag)
		}
	}
	if len(allowed) == 0 {
		return nil, errors.New("smart_bootstrap_no_ip")
	}
	bootstrap["selector"] = allowed
	byTag := map[string]M{}
	mainNodes, adsNodes := []any{}, []any{}
	for _, v := range nodes {
		ob := v.(M)
		old := ob["tag"].(string)
		ob["tag"] = mapping[old]
		if e = rewriteReferences(ob, mapping, ""); e != nil {
			return nil, e
		}
		byTag[mapping[old]] = ob
		if adsTags[old] {
			adsNodes = append(adsNodes, ob)
		} else {
			mainNodes = append(mainNodes, ob)
		}
	}
	// All live Ads dependency paths must terminate at the main dispatcher.
	// Unsupported custom dialers fail explicitly; helpers never become an exit.
	visited := map[string]bool{}
	visiting := map[string]bool{}
	var chain func(string) error
	chain = func(tag string) error {
		if visited[tag] {
			return nil
		}
		if visiting[tag] {
			return errors.New("smart_dependency_cycle")
		}
		visiting[tag] = true
		defer delete(visiting, tag)
		ob := byTag[tag]
		if ob == nil {
			return errors.New("smart_missing_dependency")
		}
		switch ob["protocol"] {
		case "vless", "vmess", "trojan", "shadowsocks", "socks", "http", "hysteria":
		default:
			return errors.New("smart_ads_unsupported_transport")
		}
		refs := outboundReferences(ob)
		for _, ref := range refs {
			if e := chain(ref); e != nil {
				return e
			}
		}
		stream := ob["streamSettings"].(M)
		socket := stream["sockopt"].(M)
		proxy, _ := ob["proxySettings"].(M)
		proxyTag, _ := proxy["tag"].(string)
		dialer, _ := socket["dialerProxy"].(string)
		if proxyTag != "" && dialer != "" {
			return errors.New("smart_ads_ambiguous_chain")
		}
		if proxyTag == "" && dialer == "" {
			socket["dialerProxy"] = "main-chain-dispatch"
			dialer = "main-chain-dispatch"
		}
		nested := false
		walkDownloadStreams(stream, func(download M) {
			nested = true
			sock := download["sockopt"].(M)
			if ref, _ := sock["dialerProxy"].(string); ref == "" {
				sock["dialerProxy"] = dialer
			}
		})
		if nested && proxyTag != "" {
			return errors.New("smart_ads_ambiguous_chain")
		}
		visited[tag] = true
		return nil
	}
	for _, tag := range entries {
		if e = chain(mapping[tag]); e != nil {
			return nil, e
		}
	}
	for i, v := range mainNodes {
		if v.(M)["tag"] == allowed[0] {
			mainNodes[0], mainNodes[i] = mainNodes[i], mainNodes[0]
			break
		}
	}
	out := append(mainNodes, adsNodes...)
	out = append(out, M{"tag": "main-chain-dispatch", "protocol": "loopback", "settings": M{"inboundTag": "main-chain-dial"}})
	rules := routing["rules"].([]any)
	extra := []any{M{"type": "field", "inboundTag": []any{"main-chain-dial"}, "balancerTag": "smart-parallel-balancer"}}
	subjects := []any{"proxy-", "ads-01"}
	if len(entries) > 1 {
		out = append(out, M{"tag": "ads-01", "protocol": "loopback", "settings": M{"inboundTag": "ads-pool-dial"}})
		extra = append(extra, M{"type": "field", "inboundTag": []any{"ads-pool-dial"}, "balancerTag": "ads-exit-balancer"})
		routing["balancers"] = append(balancers, M{"tag": "ads-exit-balancer", "selector": []any{"ads-pool-"}, "strategy": M{"type": "leastLoad", "settings": M{"expected": json.Number("3"), "maxRTT": "2s", "tolerance": json.Number("0.15")}}, "fallbackTag": "block"})
		subjects = []any{"proxy-", "ads-pool-"}
	}
	extra = append(extra, M{"type": "field", "domain": []any{"geosite:google@ads", "domain:browserleaks.com", "domain:doubleclick.net", "domain:googleadservices.com", "domain:googlesyndication.com", "domain:admob.com", "domain:app-measurement.com", "domain:googletagservices.com", "domain:googletagmanager.com"}, "network": "tcp,udp", "outboundTag": "ads-01"})
	routing["rules"] = append(append(append([]any{}, rules[:1]...), extra...), rules[1:]...)
	root["remarks"] = "@shenvpn-smart-ads"
	root["burstObservatory"].(M)["subjectSelector"] = subjects
	root["outbounds"] = out
	// Preserve the existing root-field order as well as all unchanged globals.
	var result bytes.Buffer
	result.WriteByte('{')
	for i, key := range []string{"remarks", "log", "dns", "inbounds", "outbounds", "routing", "burstObservatory", "policy", "stats", "metrics"} {
		if i > 0 {
			result.WriteByte(',')
		}
		k, _ := json.Marshal(key)
		result.Write(k)
		result.WriteByte(':')
		if e = orderedJSON(&result, root[key]); e != nil {
			return nil, e
		}
	}
	result.WriteByte('}')
	if result.Len() > repair.MaxBytes {
		return nil, errors.New("smart_profile_size_limit")
	}
	return result.Bytes(), nil
}
