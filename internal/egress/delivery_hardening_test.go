package egress

import "testing"

func deliveryFailureFixture() (Report, Plan, int64) {
	r, p, now := fixture()
	host := cloneReceipt(r).Delivery[0]
	host.HostControl = true
	host.Variant = "host"
	for i := range host.Assets {
		host.Assets[i].Nonce = Hash([]byte("hardening-host" + host.Assets[i].Profile))[:32]
	}
	for i := range r.Delivery {
		for k := range r.Delivery[i].Assets {
			a := &r.Delivery[i].Assets[k]
			a.CurlExit = 28
			a.HTTPStatus = 0
			a.TLSVerified = false
			a.Complete = false
			a.ContentMatch = false
			a.Bytes = 0
			a.BodySHA = ""
		}
	}
	r.Delivery = append(r.Delivery, host)
	return r, p, now
}
func TestDeliveryUncertainHTTPNeverBecomesDestructiveThroughCurlError(t *testing.T) {
	for _, status := range []int{429, 500, 503, 599, 200} {
		for _, exit := range []int{0, 28, 56} {
			r, p, now := deliveryFailureFixture()
			for i := 0; i < 2; i++ {
				for k := range r.Delivery[i].Assets {
					a := &r.Delivery[i].Assets[k]
					a.HTTPStatus = status
					a.CurlExit = exit
					a.TLSVerified = true
					a.Bytes = 50
					a.Complete = exit == 0
				}
			}
			if d := Delivery(r, p, now); d.Status != "unknown" {
				t.Fatal(status, exit, d)
			}
		}
	}
}
func TestDeliveryHostFreshnessIsCheckedWhenUsed(t *testing.T) {
	r, p, now := deliveryFailureFixture()
	r.Started = now - 20000
	r.Delivery[2].At = r.Started - 29000
	if d := Delivery(r, p, now); d.Status != "unknown" {
		t.Fatal("49s old host control corroborated destructive result", d)
	}
	r.Delivery[2].At = now - 29000
	if d := Delivery(r, p, now); d.Status != "unavailable" {
		t.Fatal("fresh corroboration lost", d)
	}
	r, p, now = fixture()
	host := cloneReceipt(r).Delivery[0]
	host.HostControl = true
	host.Variant = "host"
	host.At = now - 40000
	r.Delivery = append(r.Delivery, host)
	if d := Delivery(r, p, now); d.Status != "available" {
		t.Fatal("stale optional host evidence erased valid VPN assets", d)
	}
}
