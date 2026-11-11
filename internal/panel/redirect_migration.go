package panel

import (
	"github.com/rezajafari0970/Config_manager/internal/convert"
	"github.com/rezajafari0970/Config_manager/internal/smart"
	"strings"
)

// Changing the adapter invalidates final-byte evidence, not source records or
// retention/rotation settings. Fence leases before any listener is started.
func (s *Store) migrateV21() error {
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var version int
	if e = tx.QueryRow("PRAGMA user_version").Scan(&version); e != nil {
		return e
	}
	if version >= 21 {
		return nil
	}
	rows, e := tx.Query("SELECT name,sql FROM sqlite_master WHERE type='view' AND sql IS NOT NULL")
	if e != nil {
		return e
	}
	type view struct{ name, q string }
	views := []view{}
	for rows.Next() {
		var v view
		if e = rows.Scan(&v.name, &v.q); e != nil {
			rows.Close()
			return e
		}
		views = append(views, v)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	replace := strings.NewReplacer("xray-26.11.9.template.v6", convert.Version, "xray-26.11.9.smart.v10", smart.Version, "xray-26.11.9.smart-ads.v3", smart.AdsVersion)
	for _, v := range views {
		q := replace.Replace(v.q)
		if q != v.q {
			if _, e = tx.Exec("DROP VIEW " + v.name + ";" + q); e != nil {
				return e
			}
		}
	}
	for _, item := range []struct{ table, version string }{{"conversion_jobs", convert.Version}, {"smart_heads", smart.Version}, {"splash_smart_heads", smart.Version}, {"ads_smart_heads", smart.AdsVersion}} {
		if _, e = tx.Exec("UPDATE "+item.table+" SET token='',lease_until=0,state='queued',next_run=0 WHERE version!=?", item.version); e != nil {
			return e
		}
	}
	if _, e = tx.Exec("PRAGMA user_version=21"); e != nil {
		return e
	}
	return tx.Commit()
}
