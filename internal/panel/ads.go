package panel

import (
	"github.com/rezajafari0970/Config_manager/internal/smart"
	"strings"
)

// The Ads repository shares immutable input evidence, never another scope's
// cache, leases, settings, usage counters or delivery turns. Position zero is
// the one Splash exit; every other position is owned by the ordinary pool.
func (s *Store) migrateV15() error {
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var version int
	if e = tx.QueryRow("PRAGMA user_version").Scan(&version); e != nil {
		return e
	}
	if version >= 15 {
		return nil
	}
	rows, e := tx.Query("SELECT type,name,sql FROM sqlite_master WHERE name GLOB 'smart_*' AND sql IS NOT NULL ORDER BY type DESC,name")
	if e != nil {
		return e
	}
	type definition struct{ kind, name, sql string }
	var defs []definition
	for rows.Next() {
		var d definition
		if e = rows.Scan(&d.kind, &d.name, &d.sql); e != nil {
			rows.Close()
			return e
		}
		defs = append(defs, d)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, d := range defs {
		if d.kind == "view" {
			continue
		}
		q := scopedSmartSQL(d.sql, false, true)
		if !strings.Contains(q, "IF NOT EXISTS") {
			q = strings.Replace(q, "CREATE TABLE ", "CREATE TABLE IF NOT EXISTS ", 1)
			q = strings.Replace(q, "CREATE INDEX ", "CREATE INDEX IF NOT EXISTS ", 1)
		}
		if _, e = tx.Exec(q); e != nil {
			return e
		}
	}
	if _, e = tx.Exec(`INSERT OR IGNORE INTO ads_smart_settings SELECT 1,1,member_count,1,0 FROM smart_settings WHERE id=1;
 INSERT OR IGNORE INTO ads_smart_rotation_state VALUES(1,0);
 INSERT OR IGNORE INTO ads_smart_worker VALUES(1,'offline','','',0,0);`); e != nil {
		return e
	}
	for _, d := range defs {
		if d.kind != "view" {
			continue
		}
		q := scopedSmartSQL(d.sql, false, true)
		switch d.name {
		case "smart_inputs":
			q = `CREATE VIEW ads_smart_inputs AS SELECT * FROM splash_smart_inputs UNION ALL SELECT * FROM smart_inputs`
		case "smart_owned_members", "smart_cache_owned":
			table := "ads_smart_members"
			if d.name == "smart_cache_owned" {
				table = "ads_smart_cache_members"
			}
			q = `CREATE VIEW ads_` + d.name + ` AS SELECT m.head,m.position,i.expires FROM ` + table + ` m JOIN ads_smart_inputs i ON i.source=m.source AND i.id=m.id AND i.raw_hash=m.raw_hash AND i.hash=m.hash JOIN sources src ON src.id=i.source WHERE (m.position=0 AND m.id=m.head AND src.splash=1) OR (m.position>0 AND src.splash=0)`
		case "smart_ready":
			q = strings.ReplaceAll(q, smart.Version, smart.AdsVersion)
			q = strings.Replace(q, "c.member_count>0", "c.member_count>1 AND EXISTS(SELECT 1 FROM ads_smart_cache_owned own WHERE own.head=c.id AND own.position=0)", 1)
		case "smart_refresh_needed":
			// Extra Splash candidates are future heads, not missing followers.
			q = strings.Replace(q, "SELECT * FROM ads_smart_offers", "SELECT * FROM smart_offers", 1)
			q = strings.Replace(q, "c.member_count<s.member_count", "c.member_count<s.member_count+1", 1)
		}
		if _, e = tx.Exec("DROP VIEW IF EXISTS ads_" + d.name + ";" + q); e != nil {
			return e
		}
	}
	if _, e = tx.Exec("PRAGMA user_version=15"); e != nil {
		return e
	}
	return tx.Commit()
}
