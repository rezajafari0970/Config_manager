package panel

import (
	"context"
	"github.com/rezajafari0970/Config_manager/internal/smart"
	"reflect"
	"strings"
	"testing"
)

func TestCarrierHealthUpgradeFencesOnlySmartAndPreservesInventory(t *testing.T) {
	s, _ := smartFixture(t)
	ctx := context.Background()
	finishSmart(t, s)
	beforeSettings, _ := s.SmartSettings(ctx)
	beforeRows, _ := querySmart(ctx, s.db, true, 0)
	counts := map[string]int{}
	for _, table := range []string{"sources", "memberships", "conversion_results", "conversion_jobs"} {
		var n int
		if e := s.db.QueryRow("SELECT count(*) FROM " + table).Scan(&n); e != nil {
			t.Fatal(e)
		}
		counts[table] = n
	}
	tx, e := s.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"smart_inputs", "splash_smart_inputs", "smart_ready", "splash_smart_ready", "ads_smart_ready"} {
		var q string
		if e = tx.QueryRow("SELECT sql FROM sqlite_master WHERE name=?", name).Scan(&q); e != nil {
			t.Fatal(e)
		}
		q = strings.NewReplacer(smart.Version, "xray-26.11.9.smart.v14", smart.AdsVersion, "xray-26.11.9.smart-ads.v7").Replace(q)
		if _, e = tx.Exec("DROP VIEW " + name + ";" + q); e != nil {
			t.Fatal(e)
		}
	}
	for _, q := range []string{
		"UPDATE smart_heads SET version='xray-26.11.9.smart.v14',token='old-smart-lease',lease_until=9999999999999",
		"UPDATE smart_cache SET version='xray-26.11.9.smart.v14'",
		"UPDATE conversion_jobs SET token='preserve-ordinary-lease',lease_until=9999999999999",
		"CREATE VIEW carrier_unrelated AS SELECT 'xray-26.11.9.smart.v14' AS marker",
		"PRAGMA user_version=26",
	} {
		if _, e = tx.Exec(q); e != nil {
			t.Fatal(e)
		}
	}
	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}
	for i := 0; i < 2; i++ {
		if e = s.migrateV27(); e != nil {
			t.Fatal(e)
		}
	}
	var marker string
	if e = s.db.QueryRow("SELECT marker FROM carrier_unrelated").Scan(&marker); e != nil || marker != "xray-26.11.9.smart.v14" {
		t.Fatal("unrelated view changed", e)
	}
	for table, want := range counts {
		var got int
		s.db.QueryRow("SELECT count(*) FROM " + table).Scan(&got)
		if got != want {
			t.Fatal("inventory changed", table, got, want)
		}
	}
	var n int
	s.db.QueryRow("SELECT count(*) FROM smart_heads WHERE token!='' OR lease_until!=0").Scan(&n)
	if n != 0 {
		t.Fatal("old Smart lease survived")
	}
	s.db.QueryRow("SELECT count(*) FROM conversion_jobs WHERE token='preserve-ordinary-lease' AND lease_until=9999999999999").Scan(&n)
	if n != counts["conversion_jobs"] {
		t.Fatal("ordinary leases changed")
	}
	afterSettings, _ := s.SmartSettings(ctx)
	if !reflect.DeepEqual(beforeSettings, afterSettings) {
		t.Fatal("settings changed")
	}
	for _, r := range beforeRows {
		if _, e = s.smartBytes(ctx, r); e == nil {
			t.Fatal("old Smart cache exported")
		}
	}
	s.db.QueryRow("PRAGMA user_version").Scan(&n)
	if n != 27 {
		t.Fatal("migration version", n)
	}
}
