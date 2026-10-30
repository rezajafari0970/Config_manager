package panel

import (
	"context"
	"database/sql"
	"net/http"
	"strconv"
	"strings"
)

// The queue contains only hashes for the current round, never config history.
// New arrivals wait for the next snapshot, so a continuously changing feed
// cannot prevent a round from finishing.
func (s *Store) migrateV7() error {
	var v int
	if e := s.db.QueryRow("PRAGMA user_version").Scan(&v); e != nil {
		return e
	}
	if v >= 7 {
		return nil
	}
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	_, e = tx.Exec(`
 CREATE TABLE export_rotation_state(id INTEGER PRIMARY KEY CHECK(id=1),cycle INTEGER NOT NULL CHECK(cycle>=0));
 INSERT INTO export_rotation_state VALUES(1,0);
 CREATE TABLE export_rotation_queue(position INTEGER PRIMARY KEY,candidate_hash TEXT NOT NULL UNIQUE);
 PRAGMA user_version=7;`)
	if e != nil {
		return e
	}
	return tx.Commit()
}

type rotatingConfig struct {
	Conversion Conversion
	Cycle      int64
	Bytes      []byte
}

// Acquire the SQLite write lock before reading state. This also serializes
// claims made through different Store instances/processes, not just goroutines.
func (s *Store) rotationTransaction(ctx context.Context) (*sql.Tx, error) {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return nil, e
	}
	if _, e = tx.ExecContext(ctx, "UPDATE export_rotation_state SET cycle=cycle WHERE id=1"); e != nil {
		tx.Rollback()
		return nil, e
	}
	return tx, nil
}

func (s *Store) prepareRotation(ctx context.Context) (*rotatingConfig, error) {
	tx, e := s.rotationTransaction(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	entries, e := queryConversions(ctx, tx, 0, true)
	if e != nil {
		return nil, e
	}
	available := map[string]Conversion{}
	ordered := []Conversion{}
	for _, c := range entries {
		if !c.Available {
			continue
		}
		if _, exists := available[c.CandidateHash]; exists {
			continue
		}
		available[c.CandidateHash] = c
		ordered = append(ordered, c)
	}
	var cycle int64
	if e = tx.QueryRowContext(ctx, "SELECT cycle FROM export_rotation_state WHERE id=1").Scan(&cycle); e != nil {
		return nil, e
	}
	rows, e := tx.QueryContext(ctx, "SELECT candidate_hash FROM export_rotation_queue ORDER BY position")
	if e != nil {
		return nil, e
	}
	queue := []string{}
	for rows.Next() {
		var hash string
		if e = rows.Scan(&hash); e != nil {
			rows.Close()
			return nil, e
		}
		queue = append(queue, hash)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	for _, hash := range queue {
		if c, exists := available[hash]; exists {
			return &rotatingConfig{Conversion: c, Cycle: cycle}, tx.Commit()
		}
		// Removed, disabled, expired and no-longer-accepted entries cannot block a round.
		if _, e = tx.ExecContext(ctx, "DELETE FROM export_rotation_queue WHERE candidate_hash=?", hash); e != nil {
			return nil, e
		}
	}
	if len(ordered) == 0 {
		return nil, tx.Commit()
	}
	if _, e = tx.ExecContext(ctx, "UPDATE export_rotation_state SET cycle=cycle+1 WHERE id=1"); e != nil {
		return nil, e
	}
	insert, e := tx.PrepareContext(ctx, "INSERT INTO export_rotation_queue(position,candidate_hash) VALUES(?,?)")
	if e != nil {
		return nil, e
	}
	defer insert.Close()
	for i, c := range ordered {
		if _, e = insert.ExecContext(ctx, i, c.CandidateHash); e != nil {
			return nil, e
		}
	}
	return &rotatingConfig{Conversion: ordered[0], Cycle: cycle + 1}, tx.Commit()
}

func (s *Store) claimRotation(ctx context.Context, c *rotatingConfig) (bool, error) {
	tx, e := s.rotationTransaction(ctx)
	if e != nil {
		return false, e
	}
	defer tx.Rollback()
	var cycle int64
	var next string
	e = tx.QueryRowContext(ctx, `SELECT cycle,coalesce((SELECT candidate_hash FROM export_rotation_queue ORDER BY position LIMIT 1),'') FROM export_rotation_state WHERE id=1`).Scan(&cycle, &next)
	if e != nil {
		return false, e
	}
	if cycle != c.Cycle || next != c.Conversion.CandidateHash {
		return false, nil
	}
	// Recheck current source ownership, TTL, controls, version and proof after
	// reading the blob: a concurrent deletion or failed retest cannot be exported.
	entries, e := queryConversions(ctx, tx, 0, true)
	if e != nil {
		return false, e
	}
	eligible := false
	for _, current := range entries {
		if current.Available && current.CandidateHash == next {
			eligible = true
			break
		}
	}
	if !eligible {
		return false, nil
	}
	if _, e = tx.ExecContext(ctx, "DELETE FROM export_rotation_queue WHERE candidate_hash=?", next); e != nil {
		return false, e
	}
	return true, tx.Commit()
}

func (s *Store) NextConversion(ctx context.Context) (*rotatingConfig, error) {
	// Limit optimistic retries within this process. SQLite provides the durable
	// cross-process guard. No network write is performed while holding this lock.
	s.rotationMu.Lock()
	defer s.rotationMu.Unlock()
	for {
		if e := ctx.Err(); e != nil {
			return nil, e
		}
		item, e := s.prepareRotation(ctx)
		if e != nil || item == nil {
			return nil, e
		}
		// Never hold a DB transaction while opening the vault: GC takes its locks
		// in vault -> DB order. Validate the exact bytes before consuming the turn.
		item.Bytes, e = s.conversionBytes(item.Conversion)
		if e != nil {
			if e = s.retryConversionBlob(ctx, item.Conversion); e != nil {
				return nil, e
			}
			continue
		}
		claimed, e := s.claimRotation(ctx, item)
		if e != nil {
			return nil, e
		}
		if claimed {
			return item, nil
		}
	}
}

type rotationModule struct{ s *Server }

func (s *Server) singleExportURL() string {
	return s.origin + "/subscription/" + s.store.digest("subscription", "template-single-v1") + "/next.json"
}
func (m rotationModule) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /subscription/{token}/next.json", m.s.nextJSON)
}
func (s *Server) nextJSON(w http.ResponseWriter, r *http.Request) {
	if !equal(r.PathValue("token"), s.store.digest("subscription", "template-single-v1")) {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "inline")
	// Go's GET routes also match HEAD. Link checks and browser prefetches must
	// not consume configurations intended for actual GET requests.
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	if strings.Contains(strings.ToLower(r.Header.Get("Sec-Purpose")+" "+r.Header.Get("Purpose")), "prefetch") {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	item, e := s.store.NextConversion(r.Context())
	if e != nil {
		http.Error(w, "output temporarily unavailable", http.StatusServiceUnavailable)
		return
	}
	if item == nil {
		w.Header().Set("X-Config-Count", "0")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("X-Config-Count", "1")
	w.Header().Set("X-Export-Cycle", strconv.FormatInt(item.Cycle, 10))
	w.Header().Set("X-Content-SHA256", item.Conversion.CandidateHash)
	w.Header().Set("X-Android-Import", "not-run")
	w.Header().Set("X-Verification-Scope", "pinned-core-template; connection-is-separate")
	w.Write(item.Bytes)
	w.Write([]byte{'\n'})
}
