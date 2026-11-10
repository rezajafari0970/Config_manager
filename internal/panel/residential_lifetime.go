package panel

import (
	"context"
	"database/sql"
	"strings"
)

// P29 replaces the former age exemption with the user's configured fixed
// lifetime. Re-fetch/retest never extends a current generation's deadline.
func (s *Store) migrateV17() error {
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var version int
	if e = tx.QueryRow("PRAGMA user_version").Scan(&version); e != nil {
		return e
	}
	if version >= 17 {
		return nil
	}
	if _, e = tx.Exec(`ALTER TABLE residential_records ADD COLUMN retain_until INTEGER NOT NULL DEFAULT 0;
 UPDATE residential_records SET retain_until=first_seen+(SELECT retention_seconds*1000 FROM settings WHERE id=1);
 CREATE INDEX residential_lifetime ON residential_records(retain_until) WHERE state!='DELETED';
 DROP TRIGGER residential_ingest;
 CREATE TRIGGER residential_ingest AFTER INSERT ON memberships BEGIN
 INSERT INTO residential_records(hash,kind,size,first_seen,last_seen,retain_until)
 VALUES(new.raw_hash,new.kind,new.size,CAST(unixepoch('subsec')*1000 AS INTEGER),CAST(unixepoch('subsec')*1000 AS INTEGER),CAST(unixepoch('subsec')*1000 AS INTEGER)+(SELECT retention_seconds*1000 FROM settings WHERE id=1))
 ON CONFLICT(hash) DO UPDATE SET last_seen=excluded.last_seen;
 INSERT OR IGNORE INTO residential_owners VALUES(new.raw_hash,new.artifact_id,new.source_id);
 UPDATE residential_records SET state='UNKNOWN',reason='fresh_readmission_required',revision=revision+1,epoch=epoch+1,next_run=0,valid_until=0,token='',lease_until=0,first_seen=CAST(unixepoch('subsec')*1000 AS INTEGER),retain_until=CAST(unixepoch('subsec')*1000 AS INTEGER)+(SELECT retention_seconds*1000 FROM settings WHERE id=1)
 WHERE hash=new.raw_hash AND state='DELETED' AND NOT EXISTS(SELECT 1 FROM residential_tombstones WHERE hash=new.raw_hash AND manual=1) AND NOT EXISTS(SELECT 1 FROM residential_identity_tombstones WHERE identity=new.artifact_id AND manual=1);
 END;`); e != nil {
		return e
	}
	var view string
	if e = tx.QueryRow("SELECT sql FROM sqlite_master WHERE type='view' AND name='residential_eligible'").Scan(&view); e != nil {
		return e
	}
	view = strings.TrimSuffix(view, ";") + " AND r.retain_until>CAST(unixepoch('subsec')*1000 AS INTEGER)"
	if _, e = tx.Exec("DROP VIEW residential_eligible;" + view + ";PRAGMA user_version=17;"); e != nil {
		return e
	}
	return tx.Commit()
}

func (s *Store) expireResidentialLifetime(ctx context.Context, tx *sql.Tx, now int64) error {
	// Bound each cleanup transaction so one source cannot monopolize the writer.
	rows, e := tx.QueryContext(ctx, "SELECT hash,ever_accepted FROM residential_records WHERE state!='DELETED' AND retain_until<=? ORDER BY retain_until,hash LIMIT 64", now)
	if e != nil {
		return e
	}
	type item struct {
		hash     string
		accepted bool
	}
	var items []item
	for rows.Next() {
		var v item
		if e = rows.Scan(&v.hash, &v.accepted); e != nil {
			rows.Close()
			return e
		}
		items = append(items, v)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, v := range items {
		if v.accepted {
			if e = s.revokeResidential(ctx, tx, v.hash, "retention_expired", false, now); e != nil {
				return e
			}
			if _, e = tx.ExecContext(ctx, "UPDATE residential_records SET reason='retention_expired' WHERE hash=?", v.hash); e != nil {
				return e
			}
		} else {
			if _, e = tx.ExecContext(ctx, `UPDATE residential_records SET state='DELETED',reason='retention_expired',candidate_hash='',candidate_size=0,report='{}',decision='{}',revision=revision+1,epoch=epoch+1,token='',lease_until=0,valid_until=0 WHERE hash=?`, v.hash); e != nil {
				return e
			}
		}
	}
	return nil
}
