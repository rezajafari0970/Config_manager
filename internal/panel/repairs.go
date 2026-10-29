package panel

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/rezajafari0970/Config_manager/internal/collect"
	"github.com/rezajafari0970/Config_manager/internal/repair"
)

func (s *Store) migrateV4() error {
	var v int
	if e := s.db.QueryRow("PRAGMA user_version").Scan(&v); e != nil {
		return e
	}
	if v >= 4 {
		return nil
	}
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	_, e = tx.Exec(`CREATE TABLE repairs(raw_hash TEXT PRIMARY KEY,candidate_hash TEXT NOT NULL,size INTEGER NOT NULL,status TEXT NOT NULL,report TEXT NOT NULL,version TEXT NOT NULL);
 CREATE INDEX repairs_candidate ON repairs(candidate_hash);
 CREATE INDEX memberships_raw_hash ON memberships(raw_hash);
 UPDATE sources SET etag='',modified='',next_run=0;
 PRAGMA user_version=4;`)
	if e != nil {
		return e
	}
	return tx.Commit()
}
func saveRepair(ctx context.Context, tx *sql.Tx, item collect.Item) error {
	if item.Repair.Version == "" {
		return nil
	}
	b, e := json.Marshal(item.Repair)
	if e != nil {
		return e
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO repairs VALUES(?,?,?,?,?,?) ON CONFLICT(raw_hash) DO UPDATE SET candidate_hash=excluded.candidate_hash,size=excluded.size,status=excluded.status,report=excluded.report,version=excluded.version`, item.Hash, item.Candidate.Hash, item.Candidate.Size, item.Repair.Status, string(b), item.Repair.Version)
	return e
}

// A derived file lives only while its exact original span is still referenced.
const repairReferences = `EXISTS(SELECT 1 FROM memberships WHERE raw_hash=repairs.raw_hash UNION ALL SELECT 1 FROM staged WHERE hash=repairs.raw_hash)`

type RepairSummary struct {
	Outputs map[string]string         `json:"outputs"`
	Sources map[string]map[string]int `json:"sources"`
	Counts  map[string]int            `json:"counts"`
}

func (s *Store) RepairSummary(ctx context.Context, outputs []Artifact) (RepairSummary, error) {
	result := RepairSummary{Outputs: map[string]string{}, Sources: map[string]map[string]int{}, Counts: map[string]int{}}
	// Bound output metadata by the current UI page; do not expose raw data in state.
	for _, a := range outputs {
		var status string
		e := s.db.QueryRowContext(ctx, "SELECT status FROM repairs WHERE raw_hash=?", a.Hash).Scan(&status)
		if e == sql.ErrNoRows {
			continue
		}
		if e != nil {
			return result, e
		}
		result.Outputs[a.Hash] = status
	}
	rows, e := s.db.QueryContext(ctx, `SELECT m.source_id,r.status,count(*) FROM memberships m JOIN artifacts a ON a.id=m.artifact_id JOIN repairs r ON r.raw_hash=m.raw_hash WHERE a.expires>? GROUP BY m.source_id,r.status`, time.Now().UnixMilli())
	if e != nil {
		return result, e
	}
	for rows.Next() {
		var id, status string
		var n int
		if e = rows.Scan(&id, &status, &n); e != nil {
			rows.Close()
			return result, e
		}
		if result.Sources[id] == nil {
			result.Sources[id] = map[string]int{}
		}
		result.Sources[id][status] = n
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return result, e
	}
	rows, e = s.db.QueryContext(ctx, `SELECT r.status,count(*)`+representativeJoin+`JOIN repairs r ON r.raw_hash=m.raw_hash WHERE a.expires>? GROUP BY r.status`, time.Now().UnixMilli())
	if e != nil {
		return result, e
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var n int
		if e = rows.Scan(&status, &n); e != nil {
			return result, e
		}
		result.Counts[status] = n
	}
	return result, rows.Err()
}
func (s *Server) repairReport(w http.ResponseWriter, r *http.Request) {
	a, e := s.store.Output(r.Context(), r.PathValue("id"))
	if e != nil {
		failure(w, e)
		return
	}
	var report string
	e = s.store.db.QueryRowContext(r.Context(), "SELECT report FROM repairs WHERE raw_hash=?", a.Hash).Scan(&report)
	if e != nil {
		failure(w, ErrNotFound)
		return
	}
	var d repair.Diagnostic
	if json.Unmarshal([]byte(report), &d) != nil {
		failure(w, ErrInvalid)
		return
	}
	respond(w, 200, d)
}
func (s *Server) repaired(w http.ResponseWriter, r *http.Request) {
	a, e := s.store.Output(r.Context(), r.PathValue("id"))
	if e != nil {
		failure(w, e)
		return
	}
	if a.Disabled {
		respond(w, 409, map[string]string{"error": "این کانفیگ غیرفعال است."})
		return
	}
	var hash string
	var size int64
	e = s.store.db.QueryRowContext(r.Context(), "SELECT candidate_hash,size FROM repairs WHERE raw_hash=? AND status='repaired' AND candidate_hash!=''", a.Hash).Scan(&hash, &size)
	if e != nil {
		failure(w, ErrNotFound)
		return
	}
	w.Header().Set("X-Original-SHA256", a.Hash)
	w.Header().Set("X-Validation-Stage", "structural-only")
	s.stream(w, r, hash, size, a.ID[:12]+"-repaired.json")
}
