package panel

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

func (s *Store) migrateV3() error {
	var version int
	if e := s.db.QueryRow("PRAGMA user_version").Scan(&version); e != nil {
		return e
	}
	if version >= 3 {
		return nil
	}
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	_, e = tx.Exec(`
ALTER TABLE sources ADD COLUMN timeout_seconds INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sources ADD COLUMN content_state TEXT NOT NULL DEFAULT 'unknown';
ALTER TABLE sources ADD COLUMN valid_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sources ADD COLUMN invalid_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sources ADD COLUMN unknown_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sources ADD COLUMN received_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sources ADD COLUMN response_bytes INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sources ADD COLUMN error_streak INTEGER NOT NULL DEFAULT 0;
ALTER TABLE artifacts ADD COLUMN quality TEXT NOT NULL DEFAULT 'unknown';
ALTER TABLE staged ADD COLUMN quality TEXT NOT NULL DEFAULT 'unknown';
CREATE TABLE config_controls(id TEXT PRIMARY KEY,action TEXT NOT NULL CHECK(action IN ('disabled','deleted')),created INTEGER NOT NULL,expires INTEGER NOT NULL);
CREATE INDEX config_controls_expiry ON config_controls(expires);
UPDATE sources SET etag='',modified='';
PRAGMA user_version=3;`)
	if e != nil {
		return e
	}
	return tx.Commit()
}

type SourceSelection struct {
	ID       string `json:"id"`
	Revision int64  `json:"revision"`
}
type SourceBatch struct {
	Action string            `json:"action"`
	All    bool              `json:"all"`
	Items  []SourceSelection `json:"items"`
}
type ConfigBatch struct {
	Action string   `json:"action"`
	All    bool     `json:"all"`
	IDs    []string `json:"ids"`
}
type BatchResult struct {
	Affected int64    `json:"affected"`
	IDs      []string `json:"-"`
}

func (s *Store) ManageSources(ctx context.Context, b SourceBatch) (BatchResult, error) {
	result := BatchResult{}
	if b.Action != "delete" && b.Action != "disable" && b.Action != "enable" {
		return result, ErrInvalid
	}
	if b.All && len(b.Items) > 0 || !b.All && len(b.Items) == 0 {
		return result, ErrInvalid
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return result, e
	}
	defer tx.Rollback()
	items := b.Items
	if b.All {
		rows, e := tx.QueryContext(ctx, "SELECT id,revision FROM sources")
		if e != nil {
			return result, e
		}
		for rows.Next() {
			var item SourceSelection
			if e = rows.Scan(&item.ID, &item.Revision); e != nil {
				rows.Close()
				return result, e
			}
			items = append(items, item)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return result, e
		}
	}
	seen := map[string]bool{}
	for _, item := range items {
		if seen[item.ID] {
			continue
		}
		seen[item.ID] = true
		var revision int64
		if e = tx.QueryRowContext(ctx, "SELECT revision FROM sources WHERE id=?", item.ID).Scan(&revision); errors.Is(e, sql.ErrNoRows) {
			return result, ErrConflict
		} else if e != nil {
			return result, e
		}
		if revision != item.Revision {
			return result, ErrConflict
		}
		if b.Action == "delete" {
			_, e = tx.ExecContext(ctx, "DELETE FROM sources WHERE id=?", item.ID)
		} else {
			enabled := b.Action == "enable"
			state := "paused"
			if enabled {
				state = "queued"
			}
			_, e = tx.ExecContext(ctx, "UPDATE sources SET enabled=?,revision=revision+1,lease_token='',lease_until=0,state=?,next_run=?,updated=? WHERE id=?", enabled, state, time.Now().UnixNano(), time.Now().Unix(), item.ID)
			if e == nil {
				_, e = tx.ExecContext(ctx, "DELETE FROM staged WHERE source_id=?", item.ID)
			}
		}
		if e != nil {
			return result, e
		}
		result.IDs = append(result.IDs, item.ID)
		result.Affected++
	}
	if _, e = tx.ExecContext(ctx, "DELETE FROM artifacts WHERE NOT EXISTS(SELECT 1 FROM memberships m WHERE m.artifact_id=artifacts.id)"); e != nil {
		return result, e
	}
	return result, tx.Commit()
}

func (s *Store) ManageConfigs(ctx context.Context, b ConfigBatch) (BatchResult, error) {
	result := BatchResult{}
	if b.Action != "delete" && b.Action != "disable" && b.Action != "enable" {
		return result, ErrInvalid
	}
	if b.All && len(b.IDs) > 0 || !b.All && len(b.IDs) == 0 {
		return result, ErrInvalid
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return result, e
	}
	defer tx.Rollback()
	stamp := time.Now().UnixMilli()
	// A temporary selection table avoids SQL placeholder limits and duplicate IDs.
	if _, e = tx.ExecContext(ctx, "CREATE TEMP TABLE IF NOT EXISTS selected_configs(id TEXT PRIMARY KEY); DELETE FROM selected_configs;"); e != nil {
		return result, e
	}
	if b.All {
		_, e = tx.ExecContext(ctx, "INSERT INTO selected_configs SELECT id FROM artifacts WHERE expires>?", stamp)
	} else {
		for _, id := range b.IDs {
			if _, e = tx.ExecContext(ctx, "INSERT OR IGNORE INTO selected_configs VALUES(?)", id); e != nil {
				return result, e
			}
		}
	}
	if e != nil {
		return result, e
	}
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM artifacts a JOIN selected_configs s ON s.id=a.id WHERE a.expires>?", stamp).Scan(&result.Affected); e != nil {
		return result, e
	}
	if b.Action == "enable" {
		_, e = tx.ExecContext(ctx, "DELETE FROM config_controls WHERE action='disabled' AND id IN (SELECT id FROM selected_configs)")
	} else {
		action := "disabled"
		if b.Action == "delete" {
			action = "deleted"
		}
		_, e = tx.ExecContext(ctx, "INSERT INTO config_controls SELECT a.id,?,a.created,a.expires FROM artifacts a JOIN selected_configs s ON s.id=a.id WHERE a.expires>? ON CONFLICT(id) DO UPDATE SET action=excluded.action,expires=excluded.expires", action, stamp)
	}
	if e != nil {
		return result, e
	}
	if b.Action == "delete" {
		if _, e = tx.ExecContext(ctx, "DELETE FROM artifacts WHERE id IN (SELECT id FROM selected_configs)"); e != nil {
			return result, e
		}
		if b.All {
			rows, e := tx.QueryContext(ctx, "SELECT id FROM sources")
			if e != nil {
				return result, e
			}
			for rows.Next() {
				var id string
				if e = rows.Scan(&id); e != nil {
					rows.Close()
					return result, e
				}
				result.IDs = append(result.IDs, id)
			}
			e = rows.Err()
			rows.Close()
			if e != nil {
				return result, e
			}
			if _, e = tx.ExecContext(ctx, "DELETE FROM staged; DELETE FROM responses; UPDATE sources SET revision=revision+1,lease_token='',lease_until=0,etag='',modified='',state=CASE WHEN enabled=1 THEN 'queued' ELSE 'paused' END,content_state='unknown',valid_count=0,invalid_count=0,unknown_count=0,received_count=0,response_bytes=0,duplicates=0;"); e != nil {
				return result, e
			}
		}
	}
	if _, e = tx.ExecContext(ctx, "DELETE FROM selected_configs"); e != nil {
		return result, e
	}
	return result, tx.Commit()
}
