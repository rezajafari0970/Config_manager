package panel

import "github.com/rezajafari0970/Config_manager/internal/convert"

// Build the two current Splash reservation sets once per SQL statement.
// A correlated identity-OR-candidate join rescanned Splash memberships for every
// member of every cached profile and could monopolize the shared connection.
// Both sets are non-null. They remain live views, never a stale ownership cache.
func (s *Store) migrateV13() error {
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var version int
	if e = tx.QueryRow("PRAGMA user_version").Scan(&version); e != nil {
		return e
	}
	if version >= 13 {
		return nil
	}
	_, e = tx.Exec(`
 DROP VIEW smart_inputs;
 CREATE VIEW smart_inputs AS SELECT m.source_id AS source,m.artifact_id AS id,m.raw_hash,r.candidate_hash AS hash,min(a.expires,r.valid_until) AS expires
 FROM memberships m JOIN artifacts a ON a.id=m.artifact_id JOIN sources s ON s.id=m.source_id JOIN conversion_results r ON r.hash=m.raw_hash
 WHERE s.splash=0
 AND m.artifact_id NOT IN(
  SELECT sm.artifact_id FROM memberships sm JOIN sources ss ON ss.id=sm.source_id JOIN artifacts sa ON sa.id=sm.artifact_id
  WHERE ss.splash=1 AND sa.expires>CAST(unixepoch('subsec')*1000 AS INTEGER))
 AND r.candidate_hash NOT IN(
  SELECT sr.candidate_hash FROM memberships sm JOIN sources ss ON ss.id=sm.source_id JOIN artifacts sa ON sa.id=sm.artifact_id JOIN conversion_results sr ON sr.hash=sm.raw_hash
  WHERE ss.splash=1 AND sa.expires>CAST(unixepoch('subsec')*1000 AS INTEGER) AND sr.candidate_hash IS NOT NULL)
 AND s.enabled=1 AND a.expires>CAST(unixepoch('subsec')*1000 AS INTEGER)
 AND NOT EXISTS(SELECT 1 FROM config_controls c WHERE c.id=a.id AND c.expires>CAST(unixepoch('subsec')*1000 AS INTEGER))
 AND r.core='passed' AND r.candidate_hash!='' AND r.version='` + convert.Version + `'
 AND r.connection='passed' AND r.code='https_payload_verified' AND r.valid_until>CAST(unixepoch('subsec')*1000 AS INTEGER);
 PRAGMA user_version=13;`)
	if e != nil {
		return e
	}
	return tx.Commit()
}
