package panel

import (
	"context"
	"database/sql"
	"time"
)

// Receipt counters describe one upstream body, independently of current TTL,
// global identity reuse, conversion and connection proof. -1 means not measured.
func (s *Store) migrateV14() error {
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var version int
	if e = tx.QueryRow("PRAGMA user_version").Scan(&version); e != nil {
		return e
	}
	if version >= 14 {
		return nil
	}
	rows, e := tx.Query("PRAGMA table_info(sources)")
	if e != nil {
		return e
	}
	columns := map[string]bool{}
	for rows.Next() {
		var cid, notnull, pk int
		var name, typ string
		var def any
		if e = rows.Scan(&cid, &name, &typ, &notnull, &def, &pk); e != nil {
			rows.Close()
			return e
		}
		columns[name] = true
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, col := range []struct{ name, definition string }{{"receipt_unique", "INTEGER NOT NULL DEFAULT -1"}, {"receipt_new", "INTEGER NOT NULL DEFAULT -1"}, {"receipt_at", "INTEGER NOT NULL DEFAULT 0"}, {"refresh_reason", "TEXT NOT NULL DEFAULT ''"}} {
		if !columns[col.name] {
			if _, e = tx.Exec("ALTER TABLE sources ADD COLUMN " + col.name + " " + col.definition); e != nil {
				return e
			}
		}
	}
	if _, e = tx.Exec("UPDATE sources SET receipt_unique=-1,receipt_new=-1,receipt_at=0,refresh_reason=''"); e != nil {
		return e
	}
	if _, e = tx.Exec("PRAGMA user_version=14"); e != nil {
		return e
	}
	return tx.Commit()
}

// Queue before any expiry cascade deletes the source-to-artifact association.
// Never extend an artifact lifetime, clear deletion controls, or enable a source.
func queueExpiredSources(ctx context.Context, tx *sql.Tx, now time.Time) error {
	_, e := tx.ExecContext(ctx, `UPDATE sources SET next_run=min(next_run,?),refresh_reason='retention',
 state=CASE WHEN lease_token!='' AND lease_until>? THEN state ELSE 'queued' END
 WHERE enabled=1 AND (EXISTS(SELECT 1 FROM memberships m JOIN artifacts a ON a.id=m.artifact_id WHERE m.source_id=sources.id AND a.expires<=?)
 OR EXISTS(SELECT 1 FROM responses r WHERE r.source_id=sources.id AND r.expires<=?))`, now.UnixNano(), now.UnixMilli(), now.UnixMilli(), now.UnixMilli())
	return e
}

func (s *Store) recoverInventory(ctx context.Context) error {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	now := time.Now()
	if e = queueExpiredSources(ctx, tx, now); e != nil {
		return e
	}
	// Upgrade/restart recovery for gaps whose expired ownership was already swept.
	// Explicit deletions remain suppressed on the subsequent ordinary collection.
	_, e = tx.ExecContext(ctx, `UPDATE sources SET next_run=min(next_run,?),state='queued',refresh_reason='missing_inventory'
 WHERE enabled=1 AND lease_token='' AND refresh_reason='' AND state IN ('updated','unchanged') AND stored_count>0
 AND (SELECT count(*) FROM memberships m JOIN artifacts a ON a.id=m.artifact_id WHERE m.source_id=sources.id AND a.expires>?)<stored_count`, now.UnixNano(), now.UnixMilli())
	if e != nil {
		return e
	}
	return tx.Commit()
}

func sourceRefreshDeadline(ctx context.Context, tx *sql.Tx, id string, now, next time.Time) (time.Time, error) {
	var deadline sql.NullInt64
	e := tx.QueryRowContext(ctx, `SELECT min(expires) FROM (
 SELECT a.expires FROM memberships m JOIN artifacts a ON a.id=m.artifact_id WHERE m.source_id=? AND a.expires>?
 UNION ALL SELECT expires FROM responses WHERE source_id=? AND expires>?)`, id, now.UnixMilli(), id, now.UnixMilli()).Scan(&deadline)
	if e != nil {
		return next, e
	}
	if deadline.Valid && time.UnixMilli(deadline.Int64).Before(next) {
		next = time.UnixMilli(deadline.Int64)
	}
	var reason string
	var streak int
	if e = tx.QueryRowContext(ctx, "SELECT refresh_reason,error_streak FROM sources WHERE id=?", id).Scan(&reason, &streak); e != nil {
		return next, e
	}
	if reason != "" {
		delay := min(5*time.Minute, 5*time.Second*time.Duration(1<<min(max(streak, 0), 6)))
		retry := now.Add(delay)
		if retry.Before(next) {
			next = retry
		}
	}
	return next, nil
}
