package panel

import (
	"context"
	"errors"
	"github.com/rezajafari0970/Config_manager/internal/convert"
	"github.com/rezajafari0970/Config_manager/internal/smart"
	"io"
	"strings"
	"testing"
)

func TestRepairUpgradeFencesEveryExportScope(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	newExportSource(t, s, "Main", sourceURI(1), sourceURI(2))
	sp, _ := newExportSource(t, s, "Splash", sourceURI(11))
	markSplash(t, s, sp, true)
	finishSmart(t, s)
	finishPipeline(t, s.SplashSmart())
	finishAds(t, s.AdsSmart())
	app, err := NewServer(s, "http://127.0.0.1:18191")
	if err != nil {
		t.Fatal(err)
	}
	scopes := []*SmartPipeline{s.Smart(), s.SplashSmart(), s.AdsSmart()}
	cached := make([][]SmartEntry, len(scopes))
	for i, p := range scopes {
		cached[i], err = querySmart(ctx, p.db, true, 0)
		if err != nil || len(cached[i]) == 0 || !cached[i][0].Available {
			t.Fatal("fixture", err)
		}
	}
	var rawHash string
	if err = s.db.QueryRow("SELECT hash FROM conversion_results LIMIT 1").Scan(&rawHash); err != nil {
		t.Fatal(err)
	}
	raw, err := s.vault.Open(rawHash)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := io.ReadAll(raw)
	raw.Close()
	settings, err := s.SmartSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// Reproduce an installed schema23, including persisted old SQL predicates.
	tx, err := s.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	inverse := strings.NewReplacer(convert.Version, "xray-26.11.9.template.v7", smart.Version, "xray-26.11.9.smart.v11", smart.AdsVersion, "xray-26.11.9.smart-ads.v4")
	for _, name := range []string{"smart_inputs", "splash_smart_inputs", "smart_ready", "splash_smart_ready", "ads_smart_ready"} {
		var q string
		if err = tx.QueryRow("SELECT sql FROM sqlite_master WHERE name=?", name).Scan(&q); err != nil {
			t.Fatal(err)
		}
		if _, err = tx.Exec("DROP VIEW " + name + ";" + inverse.Replace(q)); err != nil {
			t.Fatal(err)
		}
	}
	for _, q := range []string{
		"UPDATE conversion_results SET version='xray-26.11.9.template.v7'",
		"UPDATE conversion_jobs SET version='xray-26.11.9.template.v7',token='old-worker',lease_until=9999999999999",
		"UPDATE smart_cache SET version='xray-26.11.9.smart.v11'",
		"UPDATE splash_smart_cache SET version='xray-26.11.9.smart.v11'",
		"UPDATE ads_smart_cache SET version='xray-26.11.9.smart-ads.v4'",
		"PRAGMA user_version=23",
	} {
		if _, err = tx.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err = s.migrateV24(); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := s.Conversions(ctx, 0, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if row.Available {
			t.Fatal("old ordinary proof exported")
		}
	}
	for i, p := range scopes {
		for _, row := range cached[i] {
			if _, err = p.smartBytes(ctx, row); err == nil {
				t.Fatal("old smart cache survived")
			}
		}
		for _, single := range []bool{false, true} {
			rr := exportRequest(t, app.Handler(), "GET", app.pipelineURL(p, single))
			if rr.Code == 200 {
				t.Fatal("old smart subscription survived", i, single)
			}
		}
	}
	afterSettings, _ := s.SmartSettings(ctx)
	if afterSettings != settings {
		t.Fatal("settings changed")
	}
	raw, err = s.vault.Open(rawHash)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := io.ReadAll(raw)
	raw.Close()
	if string(before) != string(after) {
		t.Fatal("raw mutated")
	}
	// New claims are scheduled immediately; old completion cannot earn new proof.
	for n := 0; n < 3; n++ {
		j, e := s.ClaimProbe(ctx)
		if e != nil || j == nil {
			t.Fatal("new claim", e)
		}
		result := conversionResult(*j, 0, true)
		stale := result
		stale.Version = "xray-26.11.9.template.v7"
		if e = s.CompleteProbe(ctx, stale); !errors.Is(e, ErrInvalid) {
			t.Fatal("old worker accepted", e)
		}
		if e = s.CompleteProbe(ctx, result); e != nil {
			t.Fatal(e)
		}
	}
	finishSmart(t, s)
	finishPipeline(t, s.SplashSmart())
	finishAds(t, s.AdsSmart())
	for _, p := range scopes {
		for _, single := range []bool{false, true} {
			rr := exportRequest(t, app.Handler(), "GET", app.pipelineURL(p, single))
			if rr.Code != 200 {
				t.Fatal("new proof unavailable", rr.Code, rr.Body.String())
			}
			want := "["
			if single {
				want = "{"
			}
			if !strings.HasPrefix(strings.TrimSpace(rr.Body.String()), want) {
				t.Fatal("object output changed")
			}
		}
	}
}
