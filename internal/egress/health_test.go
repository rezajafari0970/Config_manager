package egress

import "testing"

func TestPayloadHealthRequiresBoundReceiptsAndIndependentOutageControl(t *testing.T) {
	now := Now()
	p := Plan{Driver: "xray", Variants: []Variant{{Name: "raw", Hash: Hash([]byte("raw"))}, {Name: "template", Hash: Hash([]byte("template"))}}}
	r := Report{Started: now, Finished: now, Core: "xray-26.11.9", CoreSHA: Cores["xray-26.11.9"]}
	tr := Transfer{Bytes: HealthBytes, CurlExit: 0, HTTPStatus: 200, TLSVerified: true}
	for _, v := range p.Variants {
		r.Checks = append(r.Checks, Check{Name: v.Name, Hash: v.Hash, RuntimeHash: v.Hash, Core: true})
		r.Health = append(r.Health, HealthSample{Version: HealthVersion, Variant: v.Name, At: now, Providers: []HealthProvider{{Provider: "cloudflare", Download: tr, Upload: tr}, {Provider: "httpbingo", Download: tr, Upload: tr}}})
	}
	if Health(r, p, now) != "healthy" {
		t.Fatal("complete payload")
	}
	r.Checks[0].Hash = Hash([]byte("other"))
	if Health(r, p, now) != "unknown" {
		t.Fatal("wrong bytes")
	}
	r.Checks[0].Hash = p.Variants[0].Hash
	host := r.Health[0]
	host.Variant = "host"
	host.HostControl = true
	host.Providers = append([]HealthProvider(nil), host.Providers...)
	for i := range r.Health {
		for j := range r.Health[i].Providers {
			r.Health[i].Providers[j].Upload.CurlExit = 28
		}
	}
	if Health(r, p, now) != "unknown" {
		t.Fatal("timeout without outage control")
	}
	r.Health = append(r.Health, host)
	if Health(r, p, now) != "unhealthy" {
		t.Fatal("confirmed transport failure")
	}
	r.Health[2].Providers[0].Upload.Bytes--
	if Health(r, p, now) != "unknown" {
		t.Fatal("incomplete host control")
	}
}
