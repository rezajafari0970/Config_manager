package smart

import (
	"encoding/json"
	"strconv"
	"testing"
	"time"
)

func TestPayloadHealthBudgetAndStrictRecovery(t *testing.T) {
	for _, count := range []int{5, 7, 10, 15, 50, 100} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			members := []Member{}
			for i := 1; i <= count; i++ {
				members = append(members, member(t, i, "source"))
			}
			b, e := Compose(members)
			if e != nil {
				t.Fatal(e)
			}
			root, _ := object(b)
			observer := root["burstObservatory"].(M)
			ping := observer["pingConfig"].(M)
			interval, e := time.ParseDuration(ping["interval"].(string))
			if e != nil {
				t.Fatal(e)
			}
			if ping["httpMethod"] != "GET" || ping["destination"] != PayloadProbeURL ||
				ping["timeout"] != "10s" || ping["sampling"] != json.Number("2") {
				t.Fatal("payload gate missing", ping)
			}
			if _, ok := ping["connectivity"]; ok {
				t.Fatal("direct connectivity bypass")
			}
			if interval < 15*time.Second || float64(count*PayloadProbeBytes)/interval.Seconds() > 65536 {
				t.Fatal("steady-state payload budget exceeded", count, interval)
			}
			for _, v := range root["routing"].(M)["balancers"].([]any) {
				pool := v.(M)
				strategy := pool["strategy"].(M)
				settings := strategy["settings"].(M)
				if strategy["type"] != "leastLoad" || settings["tolerance"] != json.Number("0.15") {
					t.Fatal("failed transfers admitted", pool)
				}
				if pool["fallbackTag"] == "block" && settings["maxRTT"] != "10s" {
					t.Fatal("slow complete paths lost", pool)
				}
			}
			validateCore(t, b)
		})
	}
}
func TestAdsPayloadBudgetCountsFinalObservedScopes(t *testing.T) {
	members := []Member{}
	for i := 1; i <= 15; i++ {
		members = append(members, member(t, i, "main"))
	}
	b, e := ComposeAds(members, member(t, 16, "ads"))
	if e != nil {
		t.Fatal(e)
	}
	root, _ := object(b)
	observer := root["burstObservatory"].(M)
	ping := observer["pingConfig"].(M)
	if ping["interval"] != "16s" || ping["destination"] != PayloadProbeURL {
		t.Fatal("Ads scope absent from budget", ping)
	}
	for _, v := range observer["subjectSelector"].([]any) {
		if v == "member-" || v == "smart-" {
			t.Fatal("helper observation widened")
		}
	}
	validateCore(t, b)
}

func TestPayloadHealthRejectsMalformedSelectorsAndDuplicateTags(t *testing.T) {
	b, err := Compose([]Member{member(t, 1, "main")})
	if err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(M){
		func(r M) { r["burstObservatory"].(M)["subjectSelector"] = []any{"proxy-", 7} },
		func(r M) { r["outbounds"] = append(r["outbounds"].([]any), r["outbounds"].([]any)[0]) },
	} {
		root, _ := object(b)
		mutate(root)
		if applyPayloadHealth(root) == nil {
			t.Fatal("invalid observer structure accepted")
		}
	}
}
func TestRecoveryRejectsDuplicateApplicationAndOpenEnvelope(t *testing.T) {
	b, err := Compose([]Member{member(t, 1, "main")})
	if err != nil {
		t.Fatal(err)
	}
	root, _ := object(b)
	if addObservedRecovery(root, "smart-parallel-balancer", "smart-recovery") == nil {
		t.Fatal("duplicate recovery accepted")
	}
	if addObservedRecovery(M{}, "main", "recovery") == nil {
		t.Fatal("malformed recovery accepted")
	}
	root["unrecognized"] = true
	if _, err = encodeEnvelope(root); err == nil {
		t.Fatal("unknown field silently lost")
	}
}
