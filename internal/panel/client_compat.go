package panel

import (
	"github.com/rezajafari0970/Config_manager/internal/convert"
	"github.com/rezajafari0970/Config_manager/internal/smart"
	"strings"
)

// Persisted SQL views carry the admission version. Changing Go constants alone
// leaves installed databases accepting old evidence and rejecting fresh results.
func (s *Store) migrateV19() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var version int
	if err = tx.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	if version >= 19 {
		return nil
	}
	replacements := strings.NewReplacer(
		"xray-26.7.28.template.v4", convert.Version,
		"xray-26.7.28.smart.v8", smart.Version,
		"xray-26.7.28.smart-ads.v1", smart.AdsVersion,
	)
	for _, name := range []string{"smart_inputs", "splash_smart_inputs", "smart_ready", "splash_smart_ready", "ads_smart_ready"} {
		var sql string
		if err = tx.QueryRow("SELECT sql FROM sqlite_master WHERE type='view' AND name=?", name).Scan(&sql); err != nil {
			return err
		}
		updated := replacements.Replace(sql)
		if updated != sql {
			if _, err = tx.Exec("DROP VIEW " + name + ";" + updated); err != nil {
				return err
			}
		}
	}
	// Fence old leases and wake validation without changing source settings,
	// immutable blobs, counters, rotation queues or the external deletion journal.
	if _, err = tx.Exec("UPDATE conversion_jobs SET token='',lease_until=0,state='queued',next_run=0 WHERE version!=?", convert.Version); err != nil {
		return err
	}
	for _, scope := range []struct{ table, version string }{
		{"smart_heads", smart.Version}, {"splash_smart_heads", smart.Version}, {"ads_smart_heads", smart.AdsVersion},
	} {
		if _, err = tx.Exec("UPDATE "+scope.table+" SET token='',lease_until=0,state='queued',next_run=0 WHERE version!=?", scope.version); err != nil {
			return err
		}
	}
	if _, err = tx.Exec("PRAGMA user_version=19"); err != nil {
		return err
	}
	return tx.Commit()
}
