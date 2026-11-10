package panel

import (
	"strings"
	"testing"
)

// Legacy fixtures must actually remove the additive schema, rather than merely
// lower user_version while leaving a future module's tables and trigger behind.
func dropResidentialForMigration(t *testing.T, s *Store) {
	t.Helper()
	var exists int
	if e := s.db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name='residential_settings'").Scan(&exists); e != nil {
		t.Fatal(e)
	}
	if exists == 0 {
		return
	}
	for _, name := range []string{"conversion_inputs", "smart_inputs", "splash_smart_inputs"} {
		var q string
		if e := s.db.QueryRow("SELECT sql FROM sqlite_master WHERE type='view' AND name=?", name).Scan(&q); e != nil {
			t.Fatal(e)
		}
		q = strings.ReplaceAll(q, " AND m.raw_hash NOT IN (SELECT hash FROM residential_blocked) AND m.artifact_id NOT IN (SELECT identity FROM residential_blocked_identities)", "")
		if _, e := s.db.Exec("DROP VIEW " + name + ";" + q); e != nil {
			t.Fatal(e)
		}
	}
	if _, e := s.db.Exec(`DROP TRIGGER residential_ingest;DROP VIEW residential_blocked_identities;DROP VIEW residential_blocked;DROP VIEW residential_eligible;DROP TABLE residential_owners;DROP TABLE residential_turns;DROP TABLE residential_rotation;DROP TABLE residential_records;DROP TABLE residential_source_fairness;DROP TABLE residential_tombstones;DROP TABLE residential_identity_tombstones;DROP TABLE residential_audit;DROP TABLE residential_worker;DROP TABLE residential_metrics;DROP TABLE residential_settings;PRAGMA user_version=15;`); e != nil {
		t.Fatal(e)
	}
}
