package smart

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestBootstrapUsesOnlyIndependentIPEntries(t *testing.T) {
	nodes := []any{
		M{"tag": "node-a", "protocol": "vless", "settings": M{"address": "192.0.2.1"}, "streamSettings": M{"tlsSettings": M{"serverName": "never-sni.test"}, "wsSettings": M{"host": "never-host.test"}}},
		M{"tag": "node-b", "protocol": "trojan", "settings": M{"servers": []any{M{"address": "Endpoint.Test."}}}},
		M{"tag": "node-c", "protocol": "vless", "settings": M{"address": "192.0.2.2"}, "proxySettings": M{"tag": "node-b"}},
		M{"tag": "node-d", "protocol": "vless", "settings": M{"address": "192.0.2.3"}, "streamSettings": M{"xhttpSettings": M{"extra": M{"downloadSettings": M{"address": "Download.Test"}}}}},
		M{"tag": "node-e", "protocol": "vless", "settings": M{"vnext": []any{M{"address": "2001:db8::1"}}}, "streamSettings": M{"sockopt": M{"dialerProxy": "node-a", "tcpFastOpen": true}}},
	}
	source := M{"outbounds": nodes, "routing": M{"balancers": []any{M{"tag": "pool", "selector": []string{"node-"}}}, "rules": []any{M{"network": "tcp,udp", "balancerTag": "pool"}}}}
	raw, _ := json.Marshal(source)
	before := bytes.Clone(raw)
	b, e := Compose([]Member{{Body: raw}})
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(before, raw) {
		t.Fatal("raw input mutated")
	}
	root, _ := object(b)
	pool := root["routing"].(M)["balancers"].([]any)[1].(M)["selector"]
	if !reflect.DeepEqual(pool, []any{"proxy-01", "proxy-05"}) {
		t.Fatal("DNS-dependent bootstrap", pool)
	}
	for _, server := range root["dns"].(M)["servers"].([]any)[:2] {
		if !reflect.DeepEqual(server.(M)["domains"], []any{"full:download.test", "full:endpoint.test"}) {
			t.Fatal(server)
		}
	}
	out := root["outbounds"].([]any)
	for _, value := range out[1:6] {
		socket := value.(M)["streamSettings"].(M)["sockopt"].(M)
		if socket["domainStrategy"] != "UseIPv4" || socket["tcpKeepAliveIdle"] != json.Number("45") || socket["tcpKeepAliveInterval"] != json.Number("15") {
			t.Fatal(socket)
		}
	}
	socket := out[5].(M)["streamSettings"].(M)["sockopt"].(M)
	if socket["tcpFastOpen"] != true || socket["dialerProxy"] != "proxy-01" {
		t.Fatal("socket dependency changed", socket)
	}
	if out[0].(M)["tag"] != "block" {
		t.Fatal("implicit fallback must block")
	}
}

func TestBootstrapEndpointSchemas(t *testing.T) {
	cases := []struct {
		settings, stream M
		want             []string
	}{
		{M{"address": "compact.test"}, nil, []string{"compact.test"}},
		{M{"vnext": []any{M{"address": "vnext.test"}}, "servers": []any{M{"address": "servers.test"}}}, nil, []string{"vnext.test", "servers.test"}},
		{M{"peers": []any{M{"endpoint": "[2001:db8::1]:443"}, M{"endpoint": "wg.test:443"}}}, nil, []string{"2001:db8::1", "wg.test"}},
		{M{"address": "192.0.2.1"}, M{"xhttpSettings": M{"downloadSettings": M{"address": "download.test", "xhttpSettings": M{"extra": M{"downloadSettings": M{"address": "nested.test"}}}}}, "extension": M{"downloadSettings": M{"address": "never-extension.test"}}}, []string{"192.0.2.1", "download.test", "nested.test"}},
	}
	for _, c := range cases {
		if got := endpointAddresses(M{"settings": c.settings, "streamSettings": c.stream}); !reflect.DeepEqual(got, c.want) {
			t.Fatal(got, c.want)
		}
	}
}

func TestBootstrapNoIPFailsExplicitlyAndMalformedTagsDoNotPanic(t *testing.T) {
	m := member(t, 1, "A")
	m.Body = bytes.ReplaceAll(m.Body, []byte("192.0.2.1"), []byte("endpoint.test"))
	if _, e := Compose([]Member{m}); e == nil || e.Error() != "smart_bootstrap_no_ip" {
		t.Fatal(e)
	}
	for _, tag := range []any{nil, 1, ""} {
		root := M{"outbounds": []any{M{"tag": tag, "protocol": "vless"}}, "routing": M{"balancers": []any{M{"tag": "pool", "selector": []string{"proxy"}}}, "rules": []any{M{"network": "tcp,udp", "balancerTag": "pool"}}}}
		body, _ := json.Marshal(root)
		if _, e := Compose([]Member{{Body: body}}); e == nil {
			t.Fatal("invalid tag accepted")
		}
	}
}

func TestBootstrapSelectorsCannotMatchLongerDomainTags(t *testing.T) {
	for _, count := range []int{5, 7, 10, 15, 50, 100, 101, 150} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			group := []Member{}
			for n := 1; n <= count; n++ {
				m := member(t, n, "A")
				if n == count {
					m.Body = bytes.ReplaceAll(m.Body, []byte(fmt.Sprintf("192.0.2.%d", n)), []byte("endpoint.test"))
				}
				group = append(group, m)
			}
			b, e := Compose(group)
			if e != nil {
				t.Fatal(e)
			}
			root, _ := object(b)
			out := root["outbounds"].([]any)
			domainTag := out[count].(M)["tag"].(string)
			selector := root["routing"].(M)["balancers"].([]any)[1].(M)["selector"].([]any)
			if len(selector) != count-1 {
				t.Fatal(len(selector))
			}
			for _, v := range selector {
				prefix := v.(string)
				if len(prefix) != len(domainTag) || strings.HasPrefix(domainTag, prefix) {
					t.Fatal("prefix collision", prefix, domainTag)
				}
			}
		})
	}
}
