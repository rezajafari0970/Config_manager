package panel

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/compat"
	"github.com/rezajafari0970/Config_manager/internal/convert"
	"github.com/rezajafari0970/Config_manager/internal/probe"
	"github.com/rezajafari0970/Config_manager/internal/smart"
	"github.com/rezajafari0970/Config_manager/internal/target"
	"strings"
	"time"
)

func (s *Store) migrateV20() error {
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var version int
	if e = tx.QueryRow("PRAGMA user_version").Scan(&version); e != nil {
		return e
	}
	if version >= 20 {
		return nil
	}
	if _, e = tx.Exec("CREATE TABLE IF NOT EXISTS conversion_evidence(hash TEXT PRIMARY KEY REFERENCES conversion_jobs(hash) ON DELETE CASCADE,metadata TEXT NOT NULL)"); e != nil {
		return e
	}
	for _, prefix := range []string{"", "splash_", "ads_"} {
		if _, e = tx.Exec("CREATE TABLE IF NOT EXISTS " + prefix + "smart_evidence(head TEXT PRIMARY KEY REFERENCES " + prefix + "smart_heads(id) ON DELETE CASCADE,metadata TEXT NOT NULL)"); e != nil {
			return e
		}
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
	replace := strings.NewReplacer("xray-26.7.28.template.v5", convert.Version, "xray-26.7.28.smart.v9", smart.Version, "xray-26.7.28.smart-ads.v2", smart.AdsVersion)
	for _, v := range views {
		q := replace.Replace(v.q)
		if q != v.q {
			if _, e = tx.Exec("DROP VIEW " + v.name + ";" + q); e != nil {
				return e
			}
		}
	}
	if _, e = tx.Exec("UPDATE conversion_jobs SET token='',lease_until=0,state='queued',next_run=0 WHERE version!=?", convert.Version); e != nil {
		return e
	}
	for _, scope := range []struct{ table, version string }{{"smart_heads", smart.Version}, {"splash_smart_heads", smart.Version}, {"ads_smart_heads", smart.AdsVersion}} {
		if _, e = tx.Exec("UPDATE "+scope.table+" SET token='',lease_until=0,state='queued',next_run=0 WHERE version!=?", scope.version); e != nil {
			return e
		}
	}
	var inputs string
	if e = tx.QueryRow("SELECT sql FROM sqlite_master WHERE type='view' AND name='conversion_inputs'").Scan(&inputs); e != nil {
		return e
	}
	inputs += " UNION SELECT hash,kind,size FROM residential_eligible UNION SELECT hash,kind,size FROM ads_different_eligible"
	if _, e = tx.Exec("DROP VIEW conversion_inputs;" + inputs); e != nil {
		return e
	}
	if _, e = tx.Exec("PRAGMA user_version=20"); e != nil {
		return e
	}
	return tx.Commit()
}
func receipt(r probe.Result) ([]byte, error) {
	if len(r.Evidence) > 16 {
		return nil, ErrInvalid
	}
	for key, status := range r.Evidence {
		if !safeCode(key) || !safeCode(status) || len(status) > 80 {
			return nil, ErrInvalid
		}
	}
	var changes []compat.Change
	if len(r.Candidate) > 0 {
		_, changes, _ = compat.ApplyJSON(r.Candidate)
	}
	return json.Marshal(map[string]any{"family": compat.Family(r.Candidate), "target": target.Pinned(), "pipeline": r.Version, "driver_version": compat.Version, "raw_sha256": r.Hash, "candidate_sha256": r.SHA256, "runtime_sha256": r.RuntimeSHA256, "core_sha256": r.CoreSHA256, "bundled_core_sha256": "cc4b84d3069569f297a61f04b914f2bb37e9d202fd7b84cb50def1fc30fd40e1", "changes": changes, "checks": r.Evidence, "attempts": r.Attempts, "rule": r.Rule, "code": r.Code, "tested": time.Now().UnixMilli(), "android": "not_run", "http3": r.Evidence["http3"]})
}
func saveConversionEvidence(ctx context.Context, tx *sql.Tx, r probe.Result) error {
	b, e := receipt(r)
	if e != nil {
		return e
	}
	_, e = tx.ExecContext(ctx, "INSERT INTO conversion_evidence VALUES(?,?) ON CONFLICT(hash) DO UPDATE SET metadata=excluded.metadata", r.Hash, string(b))
	return e
}
func (s *SmartPipeline) saveSmartEvidence(ctx context.Context, tx smartTransaction, r probe.Result) error {
	b, e := receipt(r)
	if e != nil {
		return e
	}
	_, e = tx.ExecContext(ctx, "INSERT INTO smart_evidence VALUES(?,?) ON CONFLICT(head) DO UPDATE SET metadata=excluded.metadata", r.Hash, string(b))
	return e
}
