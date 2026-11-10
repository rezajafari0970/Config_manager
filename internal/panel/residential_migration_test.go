package panel

import (
	"context"
	"strings"
	"testing"
)

func TestResidentialV16MigrationPreservesAgeAndSources(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	resPolicy(t, s)
	source, j, _ := resSeed(t, s, 1)
	var first int64
	s.db.QueryRow("SELECT first_seen FROM residential_records WHERE hash=?", j.Hash).Scan(&first)
	restoreV17ForTest(t, s)
	// Recreate the actual preceding schema without the P29 field/index.
	var view string
	s.db.QueryRow("SELECT sql FROM sqlite_master WHERE name='residential_eligible'").Scan(&view)
	view = strings.Replace(view, " AND r.retain_until>CAST(unixepoch('subsec')*1000 AS INTEGER)", "", 1)
	if _, e := s.db.Exec("DROP TRIGGER residential_ingest;DROP VIEW residential_eligible;" + view + ";DROP INDEX residential_lifetime;ALTER TABLE residential_records DROP COLUMN retain_until;CREATE TRIGGER residential_ingest AFTER INSERT ON memberships BEGIN SELECT 1;END;UPDATE settings SET retention_seconds=36000 WHERE id=1;PRAGMA user_version=16;"); e != nil {
		t.Fatal(e)
	}
	if e := s.migrateV17(); e != nil {
		t.Fatal(e)
	}
	if e := s.migrateV17(); e != nil {
		t.Fatal("non-idempotent", e)
	}
	var until, after int64
	if e := s.db.QueryRow("SELECT first_seen,retain_until FROM residential_records WHERE hash=?", j.Hash).Scan(&after, &until); e != nil {
		t.Fatal(e)
	}
	if after != first || until != first+36000000 {
		t.Fatal("migration renewed age", after, until)
	}
	var sources int
	if e := s.db.QueryRowContext(ctx, "SELECT count(*) FROM sources WHERE id=?", source).Scan(&sources); e != nil || sources != 1 {
		t.Fatal("source lost", e)
	}
}

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
	if _, e := s.db.Exec(`DROP VIEW IF EXISTS ads_different_eligible;DROP VIEW IF EXISTS residential_active;DROP TRIGGER residential_ingest;DROP VIEW residential_blocked_identities;DROP VIEW residential_blocked;DROP VIEW residential_eligible;DROP TABLE residential_owners;DROP TABLE residential_turns;DROP TABLE residential_rotation;DROP TABLE residential_records;DROP TABLE residential_source_fairness;DROP TABLE residential_tombstones;DROP TABLE residential_identity_tombstones;DROP TABLE residential_audit;DROP TABLE residential_worker;DROP TABLE residential_metrics;DROP TABLE residential_settings;PRAGMA user_version=15;`); e != nil {
		t.Fatal(e)
	}
}

func restoreV17ForTest(t *testing.T, s *Store) {
	t.Helper()
	var view string
	if e := s.db.QueryRow("SELECT sql FROM sqlite_master WHERE name='residential_active'").Scan(&view); e != nil {
		t.Fatal(e)
	}
	view = strings.Replace(view, "CREATE VIEW residential_active AS", "CREATE VIEW residential_eligible AS", 1)
	view = strings.Replace(view, "r.state IN ('ACCEPTED','ADS_DIFFERENT')", "r.state='ACCEPTED'", 1)
	view += " AND r.retain_until>CAST(unixepoch('subsec')*1000 AS INTEGER)"
	for _, name := range []string{"residential_blocked", "residential_blocked_identities"} {
		var q string
		if e := s.db.QueryRow("SELECT sql FROM sqlite_master WHERE name=?", name).Scan(&q); e != nil {
			t.Fatal(e)
		}
		q = strings.ReplaceAll(q, "residential_active", "residential_eligible")
		if _, e := s.db.Exec("DROP VIEW " + name + ";" + q); e != nil {
			t.Fatal(e)
		}
	}
	var ingest string
	if e := s.db.QueryRow("SELECT sql FROM sqlite_master WHERE name='residential_ingest'").Scan(&ingest); e != nil {
		t.Fatal(e)
	}
	ingest = strings.Replace(ingest, ",tested=0,ever_accepted=0,accepted_decision='{}'", "", 1)
	if _, e := s.db.Exec("DROP TRIGGER residential_ingest;" + ingest); e != nil {
		t.Fatal(e)
	}
	if _, e := s.db.Exec("DROP VIEW ads_different_eligible;DROP VIEW residential_eligible;DROP VIEW residential_active;" + view + ";ALTER TABLE residential_records DROP COLUMN accepted_decision;PRAGMA user_version=17;"); e != nil {
		t.Fatal(e)
	}
}
func TestResidentialV18MigrationIsIdempotentAndFencesOldEvidence(t *testing.T) {
	s := testStore(t)
	resPolicy(t, s)
	source, j, _ := resSeed(t, s, 1)
	restoreV17ForTest(t, s)
	for i := 0; i < 2; i++ {
		if e := s.migrateV18(); e != nil {
			t.Fatal(e)
		}
	}
	var state, accepted string
	if e := s.db.QueryRow("SELECT state,accepted_decision FROM residential_records WHERE hash=?", j.Hash).Scan(&state, &accepted); e != nil || state != "QUARANTINED" || accepted == "{}" {
		t.Fatal(state, accepted, e)
	}
	var n int
	s.db.QueryRow("SELECT count(*) FROM sources WHERE id=?", source).Scan(&n)
	if n != 1 {
		t.Fatal("source changed")
	}
	if w := resRequest(s, "all", "raw.txt"); w.Code != 503 {
		t.Fatal("old policy proof published")
	}
}
