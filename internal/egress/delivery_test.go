package egress

import (
	"encoding/json"
	"testing"
)

func cloneReceipt(r Report) Report {
	b, _ := json.Marshal(r)
	var out Report
	_ = json.Unmarshal(b, &out)
	return out
}
func TestDeliveryIsMandatoryAndNeverMeansRendering(t *testing.T) {
	r, p, now := fixture()
	if d := Evaluate(r, p, r.Hash, now); !d.Eligible || d.Delivery.Scope == "" {
		t.Fatal(d)
	}
	for _, mutate := range []func(*Report){
		func(r *Report) { r.Delivery = nil },
		func(r *Report) { r.Delivery = r.Delivery[:1] },
		func(r *Report) {
			for i := range r.Delivery {
				for k := range r.Delivery[i].Assets {
					r.Delivery[i].Assets[k].TLSVerified = false
				}
			}
		},
		func(r *Report) {
			for i := range r.Delivery {
				for k := range r.Delivery[i].Assets {
					r.Delivery[i].Assets[k].ContentMatch = false
				}
			}
		},
		func(r *Report) { r.Delivery[1].Assets[0].Nonce = r.Delivery[0].Assets[0].Nonce },
		func(r *Report) { r.Delivery[0].At = now - 400000 },
	} {
		rr := cloneReceipt(r)
		mutate(&rr)
		if d := Evaluate(rr, p, rr.Hash, now); d.Eligible {
			t.Fatal("unproved delivery published", d)
		}
	}
}
func TestDeliveryHealthyProfileSurvivesOtherAssetFailure(t *testing.T) {
	r, p, now := fixture()
	for i := range r.Delivery {
		r.Delivery[i].Assets[1].HTTPStatus = 403
	}
	d := Evaluate(r, p, r.Hash, now)
	if !d.Eligible || len(d.QualifiedProfiles) != 1 || d.QualifiedProfiles[0] != "doubleclick.v1" || d.AdsCoverage != "PARTIAL" {
		t.Fatal(d)
	}
}
func TestDeliveryNegativeNeedsExactVariantsAndFreshIndependentControl(t *testing.T) {
	r, p, now := fixture()
	host := cloneReceipt(r).Delivery[0]
	host.HostControl = true
	host.Variant = "host"
	for i := range host.Assets {
		host.Assets[i].Nonce = Hash([]byte("host" + host.Assets[i].Profile))[:32]
	}
	for i := range r.Delivery {
		for k := range r.Delivery[i].Assets {
			r.Delivery[i].Assets[k].CurlExit = 28
			r.Delivery[i].Assets[k].HTTPStatus = 0
			r.Delivery[i].Assets[k].TLSVerified = false
			r.Delivery[i].Assets[k].Complete = false
			r.Delivery[i].Assets[k].ContentMatch = false
			r.Delivery[i].Assets[k].Bytes = 0
			r.Delivery[i].Assets[k].BodySHA = ""
		}
	}
	if AdsHealth(r, p, now) == "unhealthy" {
		t.Fatal("outage alone became negative")
	}
	r.Delivery = append(r.Delivery, host)
	if AdsHealth(r, p, now) != "unhealthy" {
		t.Fatal("corroborated negative lost")
	}
	for i := range r.Delivery[:2] {
		for k := range r.Delivery[i].Assets {
			r.Delivery[i].Assets[k].CurlExit = 97
		}
	}
	if AdsHealth(r, p, now) != "unhealthy" {
		t.Fatal("verified SOCKS refusal not recognized")
	}
	for _, mutate := range []func(*Report){func(r *Report) { r.Delivery[2].At = r.Started - 30001 }, func(r *Report) { r.Delivery[2].Assets[0].CurlExit = 28 }, func(r *Report) { r.Checks = r.Checks[:1] }, func(r *Report) { r.ResourcePressure = true }} {
		x := cloneReceipt(r)
		mutate(&x)
		if AdsHealth(x, p, now) == "unhealthy" {
			t.Fatal("unreliable negative")
		}
	}
}

func TestPartialVerifiedAssetDownloadIsUnknownNotUnhealthy(t *testing.T) {
	r, p, now := fixture()
	host := cloneReceipt(r).Delivery[0]
	host.HostControl = true
	host.Variant = "host"
	for i := range host.Assets {
		host.Assets[i].Nonce = Hash([]byte("host" + host.Assets[i].Profile))[:32]
	}
	for i := range r.Delivery {
		for k := range r.Delivery[i].Assets {
			r.Delivery[i].Assets[k].CurlExit = 28
			r.Delivery[i].Assets[k].Complete = false
			r.Delivery[i].Assets[k].Bytes = 37000
		}
	}
	r.Delivery = append(r.Delivery, host)
	if AdsHealth(r, p, now) == "unhealthy" {
		t.Fatal("valid partial JavaScript became destructive failure")
	}
	if d := Evaluate(r, p, r.Hash, now); d.Eligible || d.Negative || d.Reason != "ads_delivery_unproven" {
		t.Fatal(d)
	}
}
