package panel

import (
	"strings"
)

// Fence only old Smart envelopes. A failure on one phone/carrier never deletes
// raw inputs, invalidates a globally healthy ordinary receipt, or changes users.
func (s *Store) migrateV27() error {
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var version int
	if e = tx.QueryRow("PRAGMA user_version").Scan(&version); e != nil {
		return e
	}
	if version >= 27 {
		return nil
	}
	rows, e := tx.Query("SELECT name,sql FROM sqlite_master WHERE type='view' AND name IN ('smart_ready','splash_smart_ready','ads_smart_ready') AND sql IS NOT NULL")
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
	replace := strings.NewReplacer("xray-26.11.9.smart.v14", "xray-26.11.9.smart.v15", "xray-26.11.9.smart-ads.v7", "xray-26.11.9.smart-ads.v8")
	for _, v := range views {
		q := replace.Replace(v.q)
		if q != v.q {
			if _, e = tx.Exec("DROP VIEW \"" + strings.ReplaceAll(v.name, "\"", "\"\"") + "\";" + q); e != nil {
				return e
			}
		}
	}
	for _, item := range []struct{ table, version string }{{"smart_heads", "xray-26.11.9.smart.v15"}, {"splash_smart_heads", "xray-26.11.9.smart.v15"}, {"ads_smart_heads", "xray-26.11.9.smart-ads.v8"}} {
		if _, e = tx.Exec("UPDATE "+item.table+" SET token='',lease_until=0,state='queued',next_run=0 WHERE version!=?", item.version); e != nil {
			return e
		}
	}
	if _, e = tx.Exec("PRAGMA user_version=27"); e != nil {
		return e
	}
	return tx.Commit()
}
