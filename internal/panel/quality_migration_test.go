package panel

import (
	"context"
	"github.com/rezajafari0970/Config_manager/internal/smart"
	"strings"
	"testing"
)

func TestQualityUpgradeRetainsRawAndFencesOldCacheAndLease(t *testing.T) {
	s, _ := smartFixture(t)
	ctx := context.Background()
	finishSmart(t, s)
	before, _ := s.SmartSettings(ctx)
	var members, raws int
	s.db.QueryRow("SELECT count(*) FROM memberships").Scan(&members)
	s.db.QueryRow("SELECT count(*) FROM conversion_results").Scan(&raws)
	rows, _ := querySmart(ctx, s.db, true, 0)
	tx, e := s.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"smart_inputs", "splash_smart_inputs", "smart_ready", "splash_smart_ready", "ads_smart_ready"} {
		var sql string
		if e = tx.QueryRow("SELECT sql FROM sqlite_master WHERE name=?", name).Scan(&sql); e != nil {
			t.Fatal(e)
		}
		sql = strings.NewReplacer("https_payload_verified", "https_multi_site_verified", smart.Version, "xray-26.11.9.smart.v13", smart.AdsVersion, "xray-26.11.9.smart-ads.v6").Replace(sql)
		if _, e = tx.Exec("DROP VIEW " + name + ";" + sql); e != nil {
			t.Fatal(e)
		}
	}
	for _, sql := range []string{"UPDATE conversion_results SET code='https_multi_site_verified'", "UPDATE conversion_jobs SET token='old-short-probe',lease_until=9999999999999", "UPDATE smart_cache SET version='xray-26.11.9.smart.v13'", "PRAGMA user_version=25"} {
		if _, e = tx.Exec(sql); e != nil {
			t.Fatal(e)
		}
	}
	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 2; i++ {
		if e = s.migrateV26(); e != nil {
			t.Fatal(e)
		}
	}
	var n int
	s.db.QueryRow("SELECT count(*) FROM smart_inputs").Scan(&n)
	if n != 0 {
		t.Fatal("short proof promoted")
	}
	s.db.QueryRow("SELECT count(*) FROM conversion_jobs WHERE token!='' OR lease_until!=0").Scan(&n)
	if n != 0 {
		t.Fatal("old worker lease survived")
	}
	s.db.QueryRow("SELECT count(*) FROM memberships").Scan(&n)
	if n != members {
		t.Fatal("raw memberships changed")
	}
	s.db.QueryRow("SELECT count(*) FROM conversion_results").Scan(&n)
	if n != raws {
		t.Fatal("historical results removed")
	}
	after, _ := s.SmartSettings(ctx)
	if before != after {
		t.Fatal("user settings changed")
	}
	for _, r := range rows {
		if _, e = s.smartBytes(ctx, r); e == nil {
			t.Fatal("old cache exported")
		}
	}
}
func TestPayloadReceiptCannotBeAssertedByCodeAlone(t *testing.T) {
	s := testStore(t)
	_, j := addConversion(t, s, conversionSample)
	r := conversionResult(j, 0, true)
	delete(r.Evidence, "payload_upload")
	if e := s.CompleteProbe(context.Background(), r); e == nil {
		t.Fatal("unearned payload proof")
	}
	r = conversionResult(j, 0, true)
	if e := s.CompleteProbe(context.Background(), r); e != nil {
		t.Fatal(e)
	}
	job, e := s.ClaimSmart(context.Background())
	if e != nil || job == nil {
		t.Fatal(e)
	}
	sr := smartResult(job)
	delete(sr.Evidence, "payload_recheck")
	if e = s.CompleteSmart(context.Background(), sr); e == nil {
		t.Fatal("Smart accepted incomplete final transfer")
	}
}
