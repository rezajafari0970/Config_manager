package panel

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/rezajafari0970/Config_manager/internal/collect"
	"time"
)

func (s *Store) migrateV22() error {
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var version int
	if e = tx.QueryRow("PRAGMA user_version").Scan(&version); e != nil {
		return e
	}
	if version >= 22 {
		return nil
	}
	_, e = tx.Exec(`CREATE TABLE IF NOT EXISTS source_admission (
 source_id TEXT PRIMARY KEY REFERENCES sources(id) ON DELETE CASCADE,
 driver TEXT NOT NULL, baseline_unique INTEGER NOT NULL,pending_attempts INTEGER NOT NULL,last_counted INTEGER NOT NULL,
 previous_unique INTEGER NOT NULL,candidate_unique INTEGER NOT NULL,decision TEXT NOT NULL,observed_at INTEGER NOT NULL,raw_hash TEXT NOT NULL);
 PRAGMA user_version=22;`)
	if e != nil {
		return e
	}
	return tx.Commit()
}

// Runs in the same revision/token-fenced transaction as membership replacement.
// Deferral stores the exact response but never refreshes old artifact TTLs.
func sourceAdmission(ctx context.Context, tx *sql.Tx, source string, unique int, hash string, now time.Time) (collect.AdmissionDecision, error) {
	var existing int
	if e := tx.QueryRowContext(ctx, `SELECT count(*) FROM memberships m JOIN artifacts a ON a.id=m.artifact_id WHERE m.source_id=? AND a.expires>? AND NOT EXISTS(SELECT 1 FROM config_controls c WHERE c.id=a.id AND c.action='deleted' AND c.expires>?) AND NOT EXISTS(SELECT 1 FROM residential_identity_tombstones t WHERE t.identity=a.id AND t.manual=1)`, source, now.UnixMilli(), now.UnixMilli()).Scan(&existing); e != nil {
		return collect.AdmissionDecision{}, e
	}
	state := collect.AdmissionState{}
	var counted int64
	e := tx.QueryRowContext(ctx, "SELECT driver,baseline_unique,pending_attempts,last_counted FROM source_admission WHERE source_id=?", source).Scan(&state.Driver, &state.BaselineUnique, &state.PendingAttempts, &counted)
	if e != nil && !errors.Is(e, sql.ErrNoRows) {
		return collect.AdmissionDecision{}, e
	}
	if counted > 0 {
		state.LastCounted = time.UnixMilli(counted)
	}
	var driver collect.SourceAdmissionDriver = collect.CardinalityAdmission{}
	decision := driver.Evaluate(collect.AdmissionSample{ExistingUnique: existing, CandidateUnique: unique, Now: now}, state)
	counted = 0
	if !decision.State.LastCounted.IsZero() {
		counted = decision.State.LastCounted.UnixMilli()
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO source_admission VALUES(?,?,?,?,?,?,?,?,?,?) ON CONFLICT(source_id) DO UPDATE SET driver=excluded.driver,baseline_unique=excluded.baseline_unique,pending_attempts=excluded.pending_attempts,last_counted=excluded.last_counted,previous_unique=excluded.previous_unique,candidate_unique=excluded.candidate_unique,decision=excluded.decision,observed_at=excluded.observed_at,raw_hash=excluded.raw_hash`, source, driver.Name(), decision.State.BaselineUnique, decision.State.PendingAttempts, counted, decision.PreviousUnique, unique, decision.Code, now.UnixMilli(), hash)
	return decision, e
}

// Count replacement-eligible identities, not response occurrences or suppressed
// identities. Detect private manual revocations even when admission defers.
func eligibleSourceItems(ctx context.Context, tx *sql.Tx, r collect.Result, token string, stamp int64) (int, bool, error) {
	source := "SELECT identity FROM staged WHERE token=?"
	arg := any(token)
	if !r.Staged {
		ids := make([]string, len(r.Items))
		for i, item := range r.Items {
			ids[i] = item.Identity
		}
		b, e := json.Marshal(ids)
		if e != nil {
			return 0, false, e
		}
		arg = string(b)
		source = "SELECT value AS identity FROM json_each(?)"
	}
	var eligible int
	var manual bool
	query := "WITH candidates AS (" + source + "), flags AS (SELECT EXISTS(SELECT 1 FROM residential_identity_tombstones t WHERE t.identity=c.identity AND t.manual=1) AS manual, EXISTS(SELECT 1 FROM config_controls d WHERE d.id=c.identity AND d.action='deleted' AND d.expires>?) AS deleted FROM candidates c) SELECT coalesce(sum(CASE WHEN manual=0 AND deleted=0 THEN 1 ELSE 0 END),0),coalesce(max(manual),0) FROM flags"
	e := tx.QueryRowContext(ctx, query, arg, stamp).Scan(&eligible, &manual)
	return eligible, manual, e
}
