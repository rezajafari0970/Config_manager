package smart

import (
	"bytes"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/convert"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestAdsCompositionAndPinnedCore(t *testing.T) {
	for _, count := range []int{1, 2, 5, 7, 10, 15, 50, 100} {
		main := []Member{}
		for i := 1; i <= count; i++ {
			main = append(main, member(t, i, "main"))
		}
		ads := member(t, 200, "splash")
		before := bytes.Clone(ads.Body)
		b, e := ComposeAds(main, ads)
		if e != nil {
			t.Fatal(e)
		}
		if !bytes.Equal(ads.Body, before) {
			t.Fatal("input changed")
		}
		root, _ := object(b)
		baseBody, _ := Compose(main)
		baseRoot, _ := object(baseBody)
		for _, k := range []string{"log", "dns", "inbounds", "policy", "stats", "metrics"} {
			if !reflect.DeepEqual(root[k], baseRoot[k]) {
				t.Fatal("changed global", k)
			}
		}
		byTag := map[string]M{}
		for _, v := range root["outbounds"].([]any) {
			ob := v.(M)
			tag := ob["tag"].(string)
			if byTag[tag] != nil {
				t.Fatal("duplicate tag")
			}
			byTag[tag] = ob
		}
		ad := byTag["ads-01"]
		if ad["streamSettings"].(M)["sockopt"].(M)["dialerProxy"] != "main-chain-dispatch" {
			t.Fatal("ads not chained")
		}
		original, _ := object(ads.Body)
		want := original["outbounds"].([]any)[0].(M)
		if !reflect.DeepEqual(ad["settings"], want["settings"]) {
			t.Fatal("changed ads credentials")
		}
		routing := root["routing"].(M)
		rules := routing["rules"].([]any)
		if len(rules) != 6 || rules[1].(M)["balancerTag"] != "smart-parallel-balancer" || rules[2].(M)["outboundTag"] != "ads-01" {
			t.Fatal("wrong route precedence")
		}
		for _, p := range routing["balancers"].([]any) {
			for _, tag := range p.(M)["selector"].([]any) {
				if strings.HasPrefix(tag.(string), "ads-") {
					t.Fatal("ads leaked into main/bootstrap")
				}
			}
		}
		for i := 0; i < 10; i++ {
			shuffled, e := Shuffle(b)
			if e != nil {
				t.Fatal(e)
			}
			var v M
			json.Unmarshal(shuffled, &v)
			if !strings.HasPrefix(v["outbounds"].([]any)[0].(M)["tag"].(string), "proxy-") {
				t.Fatal("bad RealPing default")
			}
		}
		validateCore(t, b)
	}
}

func TestAdsWholeBalancedInputAndDependencyPreservation(t *testing.T) {
	main := []Member{member(t, 1, "main")}
	ads := member(t, 100, "splash")
	source, _ := object(ads.Body)
	nodes := source["outbounds"].([]any)
	first := nodes[0].(M)
	first["tag"] = "node-a"
	secondRoot, _ := object(member(t, 101, "splash").Body)
	second := secondRoot["outbounds"].([]any)[0].(M)
	second["tag"] = "node-b"
	source["outbounds"] = []any{first, second}
	source["routing"] = M{"rules": []any{M{"type": "field", "network": "tcp,udp", "balancerTag": "exit"}}, "balancers": []any{M{"tag": "exit", "selector": []any{"node-"}}}}
	ads.Body, _ = json.Marshal(source)
	body, e := ComposeAds(main, ads)
	if e != nil {
		t.Fatal(e)
	}
	root, _ := object(body)
	if len(root["routing"].(M)["balancers"].([]any)) != 3 {
		t.Fatal("balanced input lost")
	}
	count := 0
	for _, v := range root["outbounds"].([]any) {
		ob := v.(M)
		if strings.HasPrefix(ob["tag"].(string), "ads-pool-") {
			count++
			if ob["streamSettings"].(M)["sockopt"].(M)["dialerProxy"] != "main-chain-dispatch" {
				t.Fatal("unprotected pool entry")
			}
		}
	}
	if count != 2 {
		t.Fatal(count)
	}
	validateCore(t, body)
	// Existing chain remains, and only its terminal transport gains the carrier.
	source["routing"] = M{"rules": []any{M{"type": "field", "network": "tcp,udp", "outboundTag": "node-a"}}}
	first["streamSettings"].(M)["sockopt"].(M)["dialerProxy"] = "node-b"
	ads.Body, _ = json.Marshal(source)
	body, e = ComposeAds(main, ads)
	if e != nil {
		t.Fatal(e)
	}
	root, _ = object(body)
	var entry, dependency M
	for _, v := range root["outbounds"].([]any) {
		ob := v.(M)
		if ob["tag"] == "ads-01" {
			entry = ob
		}
		if strings.HasPrefix(ob["tag"].(string), "ads-member-") && ob["protocol"] == "vless" {
			dependency = ob
		}
	}
	if entry["streamSettings"].(M)["sockopt"].(M)["dialerProxy"] != dependency["tag"] || dependency["streamSettings"].(M)["sockopt"].(M)["dialerProxy"] != "main-chain-dispatch" {
		t.Fatal("chain overwritten")
	}
	validateCore(t, body)
}

func TestAdsFailsClosedWithoutIndependentMain(t *testing.T) {
	ads := member(t, 2, "splash")
	if _, e := ComposeAds(nil, ads); e == nil {
		t.Fatal("empty main accepted")
	}
	main := member(t, 1, "main")
	main.Body = bytes.ReplaceAll(main.Body, []byte("192.0.2.1"), []byte("endpoint.example"))
	if _, e := ComposeAds([]Member{main}, ads); e == nil || e.Error() != "smart_bootstrap_no_ip" {
		t.Fatal("ads supplied bootstrap", e)
	}
}

func TestAdsControlledChainTraffic(t *testing.T) {
	if os.Getenv("XRAY_TRAFFIC_TEST") != "1" {
		t.Skip("explicit isolated traffic fixture")
	}
	core, e := filepath.Abs(os.Getenv("XRAY_TEST_BINARY"))
	if e != nil {
		t.Fatal(e)
	}
	makeMember := func(port string) Member {
		p := convert.Prepare([]byte("vless://11111111-1111-4111-8111-111111111111@127.0.0.1:" + port + "?security=tls&sni=localhost&flow=xtls-rprx-vision-udp443"))
		if len(p.Candidates) == 0 {
			t.Fatal(p.Code)
		}
		return Member{Body: p.Candidates[0].Config}
	}
	body, e := ComposeAds([]Member{makeMember("24443")}, makeMember("24448"))
	if e != nil {
		t.Fatal(e)
	}
	payload, _ := json.Marshal(map[string]any{"core": core, "config": json.RawMessage(body)})
	cmd := exec.Command("python3", "../../scripts/ads-traffic-test.py")
	cmd.Stdin = bytes.NewReader(payload)
	out, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatalf("Ads chain: %v\n%s", e, out)
	}
	var receipt struct{ Passed bool }
	if json.Unmarshal(out, &receipt) != nil || !receipt.Passed {
		t.Fatal(string(out))
	}
	t.Log(string(out))
}
