package panel

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/rezajafari0970/Config_manager/internal/convert"
	"github.com/rezajafari0970/Config_manager/internal/egress"
	"strings"
)

type databaseQuery interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// The one-way reset is explicitly requested. Only old investigation state is
// discarded; acquisitions, templates, ordinary outputs and manual exclusions survive.
func (s *Store) migrateEgress() error {
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var v int
	if e = tx.QueryRow("PRAGMA user_version").Scan(&v); e != nil {
		return e
	}
	if v >= 23 {
		return nil
	}
	if _, e = tx.Exec(`CREATE TABLE IF NOT EXISTS input_exclusions(identity TEXT PRIMARY KEY,at INTEGER NOT NULL,hash TEXT NOT NULL,manual INTEGER NOT NULL DEFAULT 1);`); e != nil {
		return e
	}
	var legacy int
	if e = tx.QueryRow("SELECT count(*) FROM sqlite_master WHERE name='residential_identity_tombstones'").Scan(&legacy); e != nil {
		return e
	}
	if legacy > 0 {
		if _, e = tx.Exec("INSERT OR IGNORE INTO input_exclusions SELECT identity,at,hash,manual FROM residential_identity_tombstones WHERE manual=1"); e != nil {
			return e
		}
	}
	rows, e := tx.Query("SELECT name,sql FROM sqlite_master WHERE type='view'")
	if e != nil {
		return e
	}
	type view struct{ n, q string }
	views := []view{}
	for rows.Next() {
		var x view
		if e = rows.Scan(&x.n, &x.q); e != nil {
			rows.Close()
			return e
		}
		if !strings.HasPrefix(x.n, "residential_") && x.n != "ads_different_eligible" {
			views = append(views, x)
		}
	}
	rows.Close()
	// Rebuild all retained views without references to the retired subsystem.
	for _, x := range views {
		q := strings.ReplaceAll(x.q, " AND m.raw_hash NOT IN (SELECT hash FROM residential_blocked) AND m.artifact_id NOT IN (SELECT identity FROM residential_blocked_identities)", "")
		q = strings.ReplaceAll(q, " UNION SELECT hash,kind,size FROM residential_eligible UNION SELECT hash,kind,size FROM ads_different_eligible", "")
		if q != x.q {
			if _, e = tx.Exec("DROP VIEW " + x.n + ";" + q); e != nil {
				return e
			}
		}
	}
	for _, kind := range []string{"trigger", "view", "table"} {
		rows, e = tx.Query("SELECT name FROM sqlite_master WHERE type=? AND (name GLOB 'residential_*' OR name='ads_different_eligible')", kind)
		if e != nil {
			return e
		}
		names := []string{}
		for rows.Next() {
			var n string
			rows.Scan(&n)
			names = append(names, n)
		}
		rows.Close()
		// Child tables precede parents while FK enforcement stays enabled.
		if kind == "table" {
			first := []string{"residential_owners", "residential_turns"}
			for _, n := range first {
				if _, e = tx.Exec("DROP TABLE IF EXISTS " + n); e != nil {
					return e
				}
			}
		}
		for _, n := range names {
			if _, e = tx.Exec("DROP " + kind + " IF EXISTS " + n); e != nil {
				return e
			}
		}
	}
	_, e = tx.Exec(`
 CREATE TABLE IF NOT EXISTS egress_settings(id INTEGER PRIMARY KEY CHECK(id=1),epoch INTEGER NOT NULL,enabled INTEGER NOT NULL,version TEXT NOT NULL,generation INTEGER NOT NULL DEFAULT 1,circuit_until INTEGER NOT NULL DEFAULT 0);
 INSERT OR IGNORE INTO egress_settings VALUES(1,1,1,'ads-egress.v2',1,0);
 CREATE TABLE IF NOT EXISTS egress_jobs(hash TEXT PRIMARY KEY,kind TEXT NOT NULL,size INTEGER NOT NULL,first_seen INTEGER NOT NULL,last_seen INTEGER NOT NULL,revision INTEGER NOT NULL DEFAULT 1,claimed_at INTEGER NOT NULL DEFAULT 0,state TEXT NOT NULL DEFAULT 'PENDING',reason TEXT NOT NULL DEFAULT 'new_input',ever_accepted INTEGER NOT NULL DEFAULT 0,paused INTEGER NOT NULL DEFAULT 0,tested INTEGER NOT NULL DEFAULT 0,valid_until INTEGER NOT NULL DEFAULT 0,candidate_hash TEXT NOT NULL DEFAULT '',candidate_size INTEGER NOT NULL DEFAULT 0,format TEXT NOT NULL DEFAULT 'xray',decision TEXT NOT NULL DEFAULT '{}',report TEXT NOT NULL DEFAULT '{}',token TEXT NOT NULL DEFAULT '',lease_until INTEGER NOT NULL DEFAULT 0,next_run INTEGER NOT NULL DEFAULT 0,attempts INTEGER NOT NULL DEFAULT 0,failures INTEGER NOT NULL DEFAULT 0,last_failure INTEGER NOT NULL DEFAULT 0);
 CREATE INDEX IF NOT EXISTS egress_due ON egress_jobs(paused,next_run,lease_until);
 CREATE INDEX IF NOT EXISTS egress_blob ON egress_jobs(candidate_hash);
 CREATE TABLE IF NOT EXISTS egress_owners(hash TEXT NOT NULL REFERENCES egress_jobs(hash) ON DELETE CASCADE,identity TEXT NOT NULL,source TEXT NOT NULL,PRIMARY KEY(hash,identity,source));
 CREATE INDEX IF NOT EXISTS egress_identity ON egress_owners(identity);
 CREATE TABLE IF NOT EXISTS egress_events(id INTEGER PRIMARY KEY,hash TEXT NOT NULL,at INTEGER NOT NULL,state TEXT NOT NULL,reason TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS egress_turns(hash TEXT PRIMARY KEY,turn INTEGER NOT NULL);
 CREATE TABLE IF NOT EXISTS egress_worker(id INTEGER PRIMARY KEY CHECK(id=1),status TEXT NOT NULL,heartbeat INTEGER NOT NULL,active INTEGER NOT NULL,code TEXT NOT NULL);
 INSERT OR IGNORE INTO egress_worker VALUES(1,'offline',0,0,'');
 CREATE TRIGGER IF NOT EXISTS egress_ingest AFTER INSERT ON memberships BEGIN
 INSERT INTO egress_jobs(hash,kind,size,first_seen,last_seen) VALUES(new.raw_hash,new.kind,new.size,CAST(unixepoch('subsec')*1000 AS INTEGER),CAST(unixepoch('subsec')*1000 AS INTEGER)) ON CONFLICT(hash) DO UPDATE SET last_seen=excluded.last_seen,state=CASE WHEN egress_jobs.reason IN ('retention_expired','ordinary_delete') THEN 'PENDING' ELSE egress_jobs.state END,first_seen=CASE WHEN egress_jobs.reason IN ('retention_expired','ordinary_delete') THEN excluded.first_seen ELSE egress_jobs.first_seen END,next_run=CASE WHEN egress_jobs.reason IN ('retention_expired','ordinary_delete') THEN 0 ELSE egress_jobs.next_run END,reason=CASE WHEN egress_jobs.reason IN ('retention_expired','ordinary_delete') THEN 'new_input' ELSE egress_jobs.reason END;
 INSERT OR IGNORE INTO egress_owners VALUES(new.raw_hash,new.artifact_id,new.source_id);
 END;
 INSERT OR IGNORE INTO egress_jobs(hash,kind,size,first_seen,last_seen) SELECT raw_hash,min(kind),max(size),CAST(unixepoch('subsec')*1000 AS INTEGER),CAST(unixepoch('subsec')*1000 AS INTEGER) FROM memberships GROUP BY raw_hash;
 INSERT OR IGNORE INTO egress_owners SELECT DISTINCT raw_hash,artifact_id,source_id FROM memberships;
 CREATE VIEW IF NOT EXISTS egress_eligible AS SELECT j.* FROM egress_jobs j JOIN egress_settings s ON s.id=1 WHERE s.enabled=1 AND j.state='RESIDENTIAL' AND j.paused=0 AND j.valid_until>CAST(unixepoch('subsec')*1000 AS INTEGER) AND j.candidate_hash!='' AND NOT EXISTS(SELECT 1 FROM egress_owners o JOIN input_exclusions x ON x.identity=o.identity WHERE o.hash=j.hash) AND NOT EXISTS(SELECT 1 FROM egress_owners o JOIN config_controls c ON c.id=o.identity WHERE o.hash=j.hash AND c.expires>CAST(unixepoch('subsec')*1000 AS INTEGER));
 CREATE VIEW IF NOT EXISTS input_blocked AS SELECT hash FROM input_exclusions UNION SELECT hash FROM egress_jobs WHERE ever_accepted=1 AND NOT EXISTS(SELECT 1 FROM egress_eligible e WHERE e.hash=egress_jobs.hash);
 CREATE VIEW IF NOT EXISTS input_blocked_identities AS SELECT identity FROM input_exclusions UNION SELECT identity FROM egress_owners WHERE hash IN (SELECT hash FROM input_blocked);
 `)
	if e != nil {
		return fmt.Errorf("new egress schema: %w", e)
	}
	for _, name := range []string{"conversion_inputs", "smart_inputs", "splash_smart_inputs"} {
		var q string
		if e = tx.QueryRow("SELECT sql FROM sqlite_master WHERE type='view' AND name=?", name).Scan(&q); e != nil {
			return e
		}
		if !strings.Contains(q, "input_blocked") {
			q += " AND m.raw_hash NOT IN (SELECT hash FROM input_blocked) AND m.artifact_id NOT IN (SELECT identity FROM input_blocked_identities)"
		}
		if name == "conversion_inputs" && !strings.Contains(q, "egress_eligible") {
			q += " UNION SELECT hash,kind,size FROM egress_eligible"
		}
		if _, e = tx.Exec("DROP VIEW " + name + ";" + q); e != nil {
			return e
		}
	}
	if _, e = tx.Exec("PRAGMA user_version=23"); e != nil {
		return e
	}
	return tx.Commit()
}

// A changed template or evaluator cannot reuse an older eligibility receipt.
func (s *Store) reconcileEgressVersion() error {
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var previous string
	if e = tx.QueryRow("SELECT version FROM egress_settings").Scan(&previous); e != nil {
		return e
	}
	current := egress.Version + "|" + convert.Version
	if previous != current {
		if _, e = tx.Exec("UPDATE egress_jobs SET state=CASE WHEN ever_accepted=1 THEN 'QUARANTINED' ELSE 'PENDING' END,reason='implementation_changed',valid_until=0,revision=revision+1,token='',lease_until=0,next_run=0 WHERE state!='DELETED'"); e != nil {
			return e
		}
		if _, e = tx.Exec("UPDATE egress_health SET valid_until=0"); e != nil {
			return e
		}
		if _, e = tx.Exec("UPDATE egress_settings SET version=?,generation=generation+1", current); e != nil {
			return e
		}
	}
	return tx.Commit()
}

// Additive indexes leave schema25 and the evidence contract unchanged. Metadata
// lookups must not scan large stored JSON reports on every queue/GC/UI request.
func (s *Store) ensureEgressIndexes() error {
	for _, q := range []string{
		"CREATE INDEX IF NOT EXISTS egress_eligibility ON egress_jobs(state,paused,valid_until,hash)",
		"CREATE INDEX IF NOT EXISTS egress_accepted ON egress_jobs(hash) WHERE ever_accepted=1",
		"CREATE INDEX IF NOT EXISTS egress_active ON egress_jobs(lease_until) WHERE token!=''",
		"CREATE INDEX IF NOT EXISTS egress_tested ON egress_jobs(tested)",
		"CREATE INDEX IF NOT EXISTS egress_retention ON egress_jobs(first_seen) WHERE ever_accepted=0 AND tested>0 AND state!='DELETED'",
		"CREATE INDEX IF NOT EXISTS egress_events_time ON egress_events(at)",
		// Claims and counters must not fetch large report pages for every archived input.
		"CREATE INDEX IF NOT EXISTS egress_schedule_cover ON egress_jobs(paused,next_run,lease_until,state,token,ever_accepted,tested,reason,first_seen,hash,kind,size,revision,attempts,valid_until) WHERE state!='DELETED'",
	} {
		if _, err := s.db.Exec(q); err != nil {
			return err
		}
	}
	return nil
}

// P58 changes the meaning of health and required route coverage. Older binaries
// must not consume this database. Evidence remains historical, never promoted.
func (s *Store) migrateV32() error {
	var v int
	if err := s.db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		return err
	}
	if v >= 32 {
		return nil
	}
	_, err := s.db.Exec("PRAGMA user_version=32")
	return err
}
