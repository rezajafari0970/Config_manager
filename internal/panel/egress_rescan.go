package panel

import (
	"context"
	"database/sql"
	"github.com/rezajafari0970/Config_manager/internal/egress"
	"net/http"
)

// A reset is a durable finite cohort, not a reset of lifetime audit counters.
// Old leases are fenced; old receipts remain diagnostic and cannot publish.
func (s *Store) migrateV33() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var version int
	if err := tx.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	if version >= 33 {
		return nil
	}
	_, err = tx.Exec(`
 CREATE TABLE egress_runs(id INTEGER PRIMARY KEY,started INTEGER NOT NULL);
 CREATE TABLE egress_run_members(run INTEGER NOT NULL REFERENCES egress_runs(id),hash TEXT NOT NULL,accepted INTEGER NOT NULL,status TEXT NOT NULL DEFAULT 'pending',completed_at INTEGER NOT NULL DEFAULT 0,state TEXT NOT NULL DEFAULT '',definitive INTEGER NOT NULL DEFAULT 0,duration_ms INTEGER NOT NULL DEFAULT 0,PRIMARY KEY(run,hash));
 CREATE INDEX egress_run_pending ON egress_run_members(run,status,hash);
 CREATE TRIGGER egress_run_deleted AFTER UPDATE OF state ON egress_jobs WHEN NEW.state='DELETED' AND OLD.state!='DELETED' BEGIN
 UPDATE egress_run_members SET status='excluded',completed_at=CAST(unixepoch('subsec')*1000 AS INTEGER),state='DELETED' WHERE run=(SELECT max(id) FROM egress_runs) AND hash=NEW.hash AND status='pending';
 END;

 CREATE TRIGGER egress_excluded_alias_insert AFTER INSERT ON input_exclusions BEGIN
 UPDATE egress_jobs SET state='DELETED',reason='identity_excluded',revision=revision+1,token='',lease_until=0,valid_until=0
 WHERE state!='DELETED' AND hash!=NEW.hash AND hash IN(SELECT hash FROM egress_owners WHERE identity=NEW.identity);
 END;
 CREATE TRIGGER egress_excluded_alias_update AFTER UPDATE ON input_exclusions BEGIN
 UPDATE egress_jobs SET state='DELETED',reason='identity_excluded',revision=revision+1,token='',lease_until=0,valid_until=0
 WHERE state!='DELETED' AND hash!=NEW.hash AND hash IN(SELECT hash FROM egress_owners WHERE identity=NEW.identity);
 END;
 UPDATE egress_jobs SET state='DELETED',reason='identity_excluded',revision=revision+1,token='',lease_until=0,valid_until=0
 WHERE state!='DELETED' AND EXISTS(SELECT 1 FROM egress_owners o JOIN input_exclusions x ON x.identity=o.identity WHERE o.hash=egress_jobs.hash AND x.hash!=egress_jobs.hash);
 PRAGMA user_version=33;`)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ResetEgress(ctx context.Context, revision int64) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "UPDATE egress_settings SET generation=generation+1 WHERE id=1"); err != nil {
		return 0, err
	}
	var current int64
	if err = tx.QueryRowContext(ctx, "SELECT coalesce(max(id),0) FROM egress_runs").Scan(&current); err != nil {
		return 0, err
	}
	if current != revision {
		return 0, ErrConflict
	}
	now := egress.Now()
	result, err := tx.ExecContext(ctx, "INSERT INTO egress_runs(started) VALUES(?)", now)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO egress_run_members(run,hash,accepted)
 SELECT ?,j.hash,j.ever_accepted FROM egress_jobs j WHERE j.state!='DELETED' AND j.paused=0
 AND NOT EXISTS(SELECT 1 FROM egress_owners o JOIN input_exclusions x ON x.identity=o.identity WHERE o.hash=j.hash)
 AND NOT EXISTS(SELECT 1 FROM egress_owners o JOIN egress_iran_guard g ON g.identity=o.identity WHERE o.hash=j.hash AND g.state='BLOCKED')`, id)
	if err != nil {
		return 0, err
	}
	// Preserve every sticky guard, raw blob, source control, and prior audit event.
	// Tested is the current assessment clock; first_completed and attempts are lifetime.
	_, err = tx.ExecContext(ctx, `UPDATE egress_jobs SET tested=0,valid_until=0,revision=revision+1,token='',lease_until=0,next_run=0,retry_streak=0,deep_completed=0,
 state=CASE WHEN ever_accepted=1 THEN 'QUARANTINED' ELSE 'PENDING' END,
 reason=CASE WHEN reason='ads_iran_quarantined' THEN reason ELSE 'full_rescan' END
 WHERE hash IN(SELECT hash FROM egress_run_members WHERE run=?)`, id)
	if err != nil {
		return 0, err
	}
	for _, table := range []string{"egress_health", "egress_geo_results"} {
		if _, err = tx.ExecContext(ctx, "UPDATE "+table+" SET valid_until=0 WHERE hash IN(SELECT hash FROM egress_run_members WHERE run=?)", id); err != nil {
			return 0, err
		}
	}
	return id, tx.Commit()
}

// Called only after current lease/revision and timestamps pass validation, in
// the same writer transaction as completion and permanent Iran revocation.
func recordEgressRun(ctx context.Context, tx *sql.Tx, hash, state string, definitive bool, started, finished, duration int64) error {
	_, err := tx.ExecContext(ctx, `UPDATE egress_run_members SET status='completed',completed_at=?,state=?,definitive=?,duration_ms=?
 WHERE hash=? AND status!='completed' AND run=(SELECT max(id) FROM egress_runs)
 AND EXISTS(SELECT 1 FROM egress_runs r WHERE r.id=run AND r.started<=?)`, finished, state, definitive, duration, hash, started)
	return err
}

func egressRunState(ctx context.Context, q databaseQuery) (map[string]any, error) {
	var id, started int64
	err := q.QueryRowContext(ctx, "SELECT id,started FROM egress_runs ORDER BY id DESC LIMIT 1").Scan(&id, &started)
	if err == sql.ErrNoRows {
		return map[string]any{"id": int64(0)}, nil
	}
	if err != nil {
		return nil, err
	}
	var total, pending, completed, excluded, definitive, accepted, acceptedDone, acceptedExcluded, at600 int
	var last int64
	err = q.QueryRowContext(ctx, `SELECT count(*),coalesce(sum(status='pending'),0),coalesce(sum(status='completed'),0),coalesce(sum(status='excluded'),0),
 coalesce(sum(status='completed' AND definitive=1),0),coalesce(sum(accepted),0),coalesce(sum(accepted=1 AND status='completed'),0),
 coalesce(sum(accepted=1 AND status='excluded'),0),coalesce(max(completed_at),0),coalesce(sum(status='completed' AND completed_at<=?),0)
 FROM egress_run_members WHERE run=?`, started+600000, id).Scan(&total, &pending, &completed, &excluded, &definitive, &accepted, &acceptedDone, &acceptedExcluded, &last, &at600)
	if err != nil {
		return nil, err
	}
	var thousand int64
	err = q.QueryRowContext(ctx, "SELECT completed_at FROM egress_run_members WHERE run=? AND status='completed' ORDER BY completed_at,hash LIMIT 1 OFFSET 999", id).Scan(&thousand)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	elapsed := max(int64(0), readStamp(ctx)-started)
	if pending == 0 {
		elapsed = max(int64(0), last-started)
	}
	var paused int
	if err = q.QueryRowContext(ctx, "SELECT count(*) FROM egress_run_members m JOIN egress_jobs j ON j.hash=m.hash WHERE m.run=? AND m.status='pending' AND j.paused!=0", id).Scan(&paused); err != nil {
		return nil, err
	}
	return map[string]any{"id": id, "started": started, "total": total, "pending": pending, "completed": completed, "excluded": excluded, "definitive": definitive,
		"accepted": accepted, "accepted_completed": acceptedDone, "accepted_excluded": acceptedExcluded, "paused_pending": paused,
		"elapsed_ms": elapsed, "first_1000_ms": max(int64(0), thousand-started), "assessed_at_600s": at600,
		"target_600s_met": thousand > 0 && thousand-started <= 600000, "finished": pending == 0, "unit": "raw_sha256"}, nil
}

func (m egressModule) rescan(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Revision int64 `json:"revision"`
	}
	if decode(r, &b) != nil || b.Revision < 0 {
		failure(w, ErrInvalid)
		return
	}
	id, err := m.s.store.ResetEgress(r.Context(), b.Revision)
	if err != nil {
		failure(w, err)
		return
	}
	respond(w, 200, map[string]any{"id": id})
}
