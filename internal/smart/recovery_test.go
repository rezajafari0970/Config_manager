package smart

import (
	"reflect"
	"testing"
)

func TestObservedRecoveryScopesAndFailClosedGraph(t *testing.T) {
	b, e := Compose([]Member{member(t, 1, "A"), member(t, 2, "B")})
	if e != nil {
		t.Fatal(e)
	}
	root, _ := object(b)
	route := root["routing"].(M)
	pools := map[string]M{}
	outs := map[string]M{}
	for _, v := range route["balancers"].([]any) {
		p := v.(M)
		pools[p["tag"].(string)] = p
	}
	for _, v := range root["outbounds"].([]any) {
		o := v.(M)
		outs[o["tag"].(string)] = o
	}
	for _, pair := range [][2]string{{"smart-parallel-balancer", "smart-recovery"}, {"bootstrap-ip-balancer", "bootstrap-recovery"}} {
		primary := pools[pair[0]]
		slow := pools[pair[1]+"-balancer"]
		if primary["fallbackTag"] != pair[1] || slow["fallbackTag"] != "block" || outs["block"]["protocol"] != "blackhole" {
			t.Fatal("fallback escape")
		}
		if !reflect.DeepEqual(primary["selector"], slow["selector"]) {
			t.Fatal("fallback widened pool")
		}
		if outs[pair[1]]["protocol"] != "loopback" {
			t.Fatal("recovery became direct")
		}
		tag := outs[pair[1]]["settings"].(M)["inboundTag"]
		found := false
		for _, v := range route["rules"].([]any) {
			r := v.(M)
			in, _ := r["inboundTag"].([]any)
			if len(in) == 1 && in[0] == tag {
				found = r["balancerTag"] == pair[1]+"-balancer"
				break
			}
		}
		if !found {
			t.Fatal("missing scoped recovery route")
		}
	}
	if root["outbounds"].([]any)[0].(M)["protocol"] == "loopback" {
		t.Fatal("native ping lost real default")
	}
}
