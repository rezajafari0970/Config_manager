package panel

import (
	"context"
	"strings"
	"time"
)

func (s *Store) migrateV12() error {
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var version int
	if e = tx.QueryRow("PRAGMA user_version").Scan(&version); e != nil {
		return e
	}
	if version >= 12 {
		return nil
	}
	if _, e = tx.Exec(`ALTER TABLE sources ADD COLUMN splash INTEGER NOT NULL DEFAULT 0 CHECK(splash IN(0,1));
 ALTER TABLE smart_heads ADD COLUMN inventory TEXT NOT NULL DEFAULT '';
 CREATE INDEX sources_splash ON sources(splash,id);`); e != nil {
		return e
	}
	// Capture DDL before executing it: one SQLite connection, no nested reads.
	rows, e := tx.Query(`SELECT sql FROM sqlite_master WHERE (type='table' OR type='index') AND name GLOB 'smart_*' AND sql IS NOT NULL ORDER BY type DESC,name`)
	if e != nil {
		return e
	}
	var ddl []string
	for rows.Next() {
		var q string
		if e = rows.Scan(&q); e != nil {
			rows.Close()
			return e
		}
		ddl = append(ddl, q)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, q := range ddl {
		if _, e = tx.Exec(smartSQL(q, true)); e != nil {
			return e
		}
	}
	if _, e = tx.Exec(`INSERT INTO splash_smart_settings VALUES(1,1,10,1,0);
 INSERT INTO splash_smart_rotation_state VALUES(1,0);
 INSERT INTO splash_smart_worker VALUES(1,'offline','','',0,0);`); e != nil {
		return e
	}
	rows, e = tx.Query("SELECT name,sql FROM sqlite_master WHERE type='view' AND name GLOB 'smart_*'")
	if e != nil {
		return e
	}
	type view struct{ name, q string }
	var views []view
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
	for _, v := range views {
		normal, splash := v.q, smartSQL(v.q, true)
		if v.name == "smart_inputs" {
			// Splash ownership has priority for duplicate identities/candidate bytes.
			// A normal source cannot smuggle a Splash config into the ordinary pool.
			normal = strings.Replace(normal, "WHERE s.enabled=1 AND", `WHERE s.splash=0 AND NOT EXISTS(
    SELECT 1 FROM memberships sm JOIN sources ss ON ss.id=sm.source_id
    JOIN artifacts sa ON sa.id=sm.artifact_id LEFT JOIN conversion_results sr ON sr.hash=sm.raw_hash
    WHERE ss.splash=1 AND sa.expires>CAST(unixepoch('subsec')*1000 AS INTEGER)
    AND (sm.artifact_id=m.artifact_id OR sr.candidate_hash=r.candidate_hash)) AND s.enabled=1 AND`, 1)
			splash = strings.Replace(splash, "WHERE s.enabled=1 AND", "WHERE s.splash=1 AND s.enabled=1 AND", 1)
		}
		if _, e = tx.Exec("DROP VIEW " + v.name + ";" + normal + ";" + splash); e != nil {
			return e
		}
	}
	if _, e = tx.Exec("PRAGMA user_version=12"); e != nil {
		return e
	}
	return tx.Commit()
}

// A classification edit fences optimistic UI updates without rewriting source
// bytes, refreshing its immutable artifact lifetime.
func (s *Store) SetSplash(ctx context.Context, id string, revision int64, value bool) error {
	res, e := s.db.ExecContext(ctx, "UPDATE sources SET splash=?,revision=revision+1,updated=?,next_run=CASE WHEN lease_token!='' AND enabled=1 THEN 0 ELSE next_run END,state=CASE WHEN lease_token!='' THEN CASE WHEN enabled=1 THEN 'queued' ELSE 'paused' END ELSE state END,lease_token='',lease_until=0 WHERE id=? AND revision=?", value, time.Now().Unix(), id, revision)
	if e != nil {
		return e
	}
	n, e := res.RowsAffected()
	if e != nil {
		return e
	}
	if n != 1 {
		return ErrConflict
	}
	return nil
}

func (s *Store) migrateV12AfterV11() error {
	if e := s.migrateV11(); e != nil {
		return e
	}
	return s.migrateV12()
}
