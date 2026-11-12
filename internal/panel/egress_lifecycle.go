package panel

import (
	"context"
	"net/http"
)

type egressLifecycle struct {
	Revision      int64 `json:"revision"`
	HealthMinutes int64 `json:"health_minutes"`
}

func (s *Store) migrateV28() error {
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var v int
	if e = tx.QueryRow("PRAGMA user_version").Scan(&v); e != nil {
		return e
	}
	if v >= 28 {
		return nil
	}
	_, e = tx.Exec(`CREATE TABLE egress_lifecycle(id INTEGER PRIMARY KEY CHECK(id=1),revision INTEGER NOT NULL,health_minutes INTEGER NOT NULL CHECK(health_minutes BETWEEN 1 AND 60));
 INSERT INTO egress_lifecycle VALUES(1,1,15);
 CREATE TABLE egress_health(hash TEXT PRIMARY KEY REFERENCES egress_jobs(hash) ON DELETE CASCADE,status TEXT NOT NULL,checked_at INTEGER NOT NULL,valid_until INTEGER NOT NULL,report TEXT NOT NULL);
 DROP TRIGGER IF EXISTS egress_ingest;
 UPDATE egress_jobs SET tested=0,attempts=0,deep_completed=0,report='{}',decision='{}',valid_until=0,revision=revision+1,token='',lease_until=0,next_run=0 WHERE state='PENDING' AND reason='new_input' AND tested>0;
 UPDATE egress_jobs SET next_run=0 WHERE ever_accepted=1 AND state!='DELETED';
 PRAGMA user_version=28;`)
	if e != nil {
		return e
	}
	// Both staging and membership commits durably register immutable input. A
	// failed later source transaction cannot silently discard a validated item.
	for _, t := range []struct{ name, table, hash, identity string }{{"egress_ingest", "memberships", "raw_hash", "artifact_id"}, {"egress_stage", "staged", "hash", "identity"}} {
		revive := "egress_jobs.reason IN ('retention_expired','ordinary_delete')"
		q := "CREATE TRIGGER " + t.name + " AFTER INSERT ON " + t.table + " BEGIN INSERT INTO egress_jobs(hash,kind,size,first_seen,last_seen) SELECT new." + t.hash + ",new.kind,new.size,CAST(unixepoch('subsec')*1000 AS INTEGER),CAST(unixepoch('subsec')*1000 AS INTEGER) WHERE NOT EXISTS(SELECT 1 FROM input_exclusions WHERE identity=new." + t.identity + ") ON CONFLICT(hash) DO UPDATE SET last_seen=excluded.last_seen"
		for _, x := range []struct{ key, value string }{{"state", "'PENDING'"}, {"reason", "'new_input'"}, {"tested", "0"}, {"attempts", "0"}, {"deep_completed", "0"}, {"next_run", "0"}, {"report", "'{}'"}, {"decision", "'{}'"}, {"valid_until", "0"}, {"token", "''"}, {"lease_until", "0"}, {"revision", "egress_jobs.revision+1"}} {
			q += ", " + x.key + "=CASE WHEN " + revive + " THEN " + x.value + " ELSE egress_jobs." + x.key + " END"
		}
		q += "; INSERT OR IGNORE INTO egress_owners SELECT new." + t.hash + ",new." + t.identity + ",new.source_id WHERE EXISTS(SELECT 1 FROM egress_jobs WHERE hash=new." + t.hash + "); END;"
		if _, e = tx.Exec(q); e != nil {
			return e
		}
	}
	// Publication independently requires a fresh full-payload health receipt.
	var view string
	if e = tx.QueryRow("SELECT sql FROM sqlite_master WHERE name='egress_eligible'").Scan(&view); e != nil {
		return e
	}
	view += " AND EXISTS(SELECT 1 FROM egress_health h WHERE h.hash=j.hash AND h.status='healthy' AND h.valid_until>CAST(unixepoch('subsec')*1000 AS INTEGER))"
	if _, e = tx.Exec("DROP VIEW egress_eligible;" + view); e != nil {
		return e
	}
	return tx.Commit()
}
func lifecycleState(ctx context.Context, q databaseQuery) (egressLifecycle, error) {
	var p egressLifecycle
	e := q.QueryRowContext(ctx, "SELECT revision,health_minutes FROM egress_lifecycle WHERE id=1").Scan(&p.Revision, &p.HealthMinutes)
	return p, e
}
func (m egressModule) lifecycle(w http.ResponseWriter, r *http.Request) {
	var p egressLifecycle
	if decode(r, &p) != nil || p.HealthMinutes < 1 || p.HealthMinutes > 60 {
		failure(w, ErrInvalid)
		return
	}
	tx, e := m.s.store.db.BeginTx(r.Context(), nil)
	if e != nil {
		failure(w, e)
		return
	}
	defer tx.Rollback()
	result, e := tx.ExecContext(r.Context(), "UPDATE egress_lifecycle SET revision=revision+1,health_minutes=? WHERE id=1 AND revision=?", p.HealthMinutes, p.Revision)
	if e != nil {
		failure(w, e)
		return
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		failure(w, ErrConflict)
		return
	}
	// Shortening takes effect immediately; lengthening cannot manufacture a
	// longer health/route receipt. Existing proof must first be renewed.
	if _, e = tx.ExecContext(r.Context(), "UPDATE egress_jobs SET next_run=min(next_run,tested+?) WHERE ever_accepted=1 AND state!='DELETED'", p.HealthMinutes*60000); e != nil {
		failure(w, e)
		return
	}
	if e = tx.Commit(); e != nil {
		failure(w, e)
		return
	}
	p.Revision++
	respond(w, 200, p)
}
