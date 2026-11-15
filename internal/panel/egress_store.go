package panel

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/rezajafari0970/Config_manager/internal/egress"
	"github.com/rezajafari0970/Config_manager/internal/repair"
	"io"
	"time"
)

func (s *Store) ClaimEgress(ctx context.Context) (*egress.Job, error) {
	return s.ClaimEgressLane(ctx, "")
}

// Enumerate the small current/retained identity set before looking up jobs.
// Starting with the due-job index instead scans the entire historical archive on
// every idle worker poll. The authoritative views still apply all exclusions,
// Iran guards and completed-result rules to each candidate; this is not a cache.
const egressClaimCandidates = `WITH claim_scope(hash) AS MATERIALIZED (
 SELECT u.hash FROM (SELECT DISTINCT artifact_id FROM memberships) m JOIN egress_unique u ON u.identity=m.artifact_id
 UNION SELECT hash FROM egress_jobs WHERE ever_accepted=1
 UNION SELECT u.hash FROM egress_iran_guard g JOIN egress_unique u ON u.identity=g.identity WHERE g.state='QUARANTINED'
) `

// Independent first-assessment and follow-up lanes share revision-fenced leases.
// Due accepted proofs and manual requests can use either lane. Idle slots
// steal first assessments before spending capacity on routine NO_SPLIT rechecks.
// next_run already reserves measured probe duration before expiry. Waiting for
// the last minute here would discard that lead and starve renewal under load.
func (s *Store) ClaimEgressLane(ctx context.Context, lane string) (*egress.Job, error) {
	if lane != "" && lane != "initial" && lane != "followup" {
		return nil, ErrInvalid
	}
	release, err := s.admitClaim(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	// Idle workers inspect committed WAL state without queuing for the sole
	// writer. This is only a negative fast path: claims and scope maintenance
	// below still recheck everything after acquiring the writer. Work committed
	// after a negative read is picked up by the next ordinary poll, without a cache.
	now := egress.Now()
	var needed bool
	e := s.workerReader().QueryRowContext(ctx, egressClaimCandidates+`SELECT
 EXISTS(SELECT 1 FROM egress_jobs WHERE token!='' AND NOT EXISTS(SELECT 1 FROM egress_inventory i WHERE i.hash=egress_jobs.hash))
 OR EXISTS(SELECT 1 FROM egress_run_members WHERE run=(SELECT max(id) FROM egress_runs) AND status='pending' AND NOT EXISTS(SELECT 1 FROM egress_inventory i WHERE i.hash=egress_run_members.hash))
 OR (EXISTS(SELECT 1 FROM egress_settings WHERE id=1 AND enabled!=0)
 AND (SELECT count(*) FROM egress_jobs WHERE token!='' AND lease_until>?)<?
 AND EXISTS(SELECT 1 FROM claim_scope c CROSS JOIN egress_work j ON j.hash=c.hash WHERE paused=0 AND next_run<=? AND (token='' OR lease_until<=?)))`, now, egress.MaxWorkers, now, now).Scan(&needed)
	if e != nil || !needed {
		return nil, e
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, "UPDATE egress_settings SET generation=generation WHERE id=1"); e != nil {
		return nil, e
	}
	now = egress.Now()
	var active, enabled int
	if _, e = tx.ExecContext(ctx, `UPDATE egress_jobs SET revision=revision+1,token='',lease_until=0 WHERE token!='' AND NOT EXISTS(SELECT 1 FROM egress_inventory i WHERE i.hash=egress_jobs.hash)`); e != nil {
		return nil, e
	}
	if _, e = tx.ExecContext(ctx, `UPDATE egress_run_members SET status='excluded',state='OUT_OF_SCOPE',completed_at=? WHERE run=(SELECT max(id) FROM egress_runs) AND status='pending' AND NOT EXISTS(SELECT 1 FROM egress_inventory i WHERE i.hash=egress_run_members.hash)`, now); e != nil {
		return nil, e
	}
	if e = tx.QueryRowContext(ctx, "SELECT enabled,(SELECT count(*) FROM egress_jobs WHERE token!='' AND lease_until>?) FROM egress_settings WHERE id=1", now).Scan(&enabled, &active); e != nil {
		return nil, e
	}
	if enabled == 0 || active >= egress.MaxWorkers {
		return nil, tx.Commit()
	}
	j := &egress.Job{}
	var size int64
	var state, previousToken string
	base := egressClaimCandidates + `SELECT egress_jobs.hash,kind,size,revision,ever_accepted,attempts,state,token FROM claim_scope c CROSS JOIN egress_work AS egress_jobs ON egress_jobs.hash=c.hash WHERE paused=0 AND next_run<=? AND (token='' OR lease_until<=?)`
	order := ` ORDER BY CASE WHEN reason='ads_iran_quarantined' THEN 0 WHEN ever_accepted=1 THEN 0 WHEN tested=0 AND EXISTS(SELECT 1 FROM memberships m WHERE m.raw_hash=egress_jobs.hash) THEN 2 WHEN tested=0 THEN 3 WHEN ever_accepted=1 THEN 4 WHEN state='UNKNOWN' AND attempts=1 AND EXISTS(SELECT 1 FROM memberships m WHERE m.raw_hash=egress_jobs.hash) THEN 5 ELSE 6 END,next_run,first_seen,egress_jobs.hash LIMIT 1`
	filters := []string{""}
	if lane == "initial" {
		filters = []string{` AND (tested=0 OR reason='ads_iran_quarantined' OR ever_accepted=1)`, ""}
	}
	if lane == "followup" {
		filters = []string{` AND (ever_accepted=1 OR tested>0)`, ` AND tested=0`, ""}
		order = ` ORDER BY CASE WHEN reason='ads_iran_quarantined' THEN 0 WHEN ever_accepted=1 THEN 0 ELSE 2 END,next_run,first_seen,egress_jobs.hash LIMIT 1`
	}
	for _, filter := range filters {
		args := []any{now, now}
		e = tx.QueryRowContext(ctx, base+filter+order, args...).Scan(&j.Hash, &j.Kind, &size, &j.Revision, &j.Deep, &j.Attempt, &state, &previousToken)
		if !errors.Is(e, sql.ErrNoRows) {
			break
		}
	}

	if errors.Is(e, sql.ErrNoRows) {
		return nil, tx.Commit()
	}
	if e != nil {
		return nil, e
	}
	if previousToken != "" {
		if _, e = tx.ExecContext(ctx, "UPDATE egress_metrics SET lease_recoveries=lease_recoveries+1 WHERE id=1"); e != nil {
			return nil, e
		}
	}
	var freshRun bool
	if e = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM egress_run_members WHERE run=(SELECT max(id) FROM egress_runs) AND hash=? AND status='pending')", j.Hash).Scan(&freshRun); e != nil {
		return nil, e
	}
	if freshRun {
		j.Attempt = 0
	}
	j.Accepted = j.Deep
	j.Deep = j.Deep || j.Attempt > 0 && state != "NO_SPLIT" && state != "UNSUPPORTED"
	j.Attempt++
	j.Token = randomToken()
	if _, e = tx.ExecContext(ctx, "UPDATE egress_jobs SET token=?,lease_until=?,claimed_at=?,attempts=attempts+1 WHERE hash=?", j.Token, now+120000, now, j.Hash); e != nil {
		return nil, e
	}
	if e = tx.Commit(); e != nil {
		return nil, e
	}
	release()
	if size > repair.MaxBytes {
		j.Issue = "input_size_limit"
		return j, nil
	}
	b, e := s.vault.Open(j.Hash)
	if e != nil {
		j.Issue = "input_unavailable"
		return j, nil
	}
	defer b.Close()
	j.Raw, e = io.ReadAll(io.LimitReader(b, repair.MaxBytes+1))
	if e != nil || egress.Hash(j.Raw) != j.Hash {
		j.Raw = nil
		j.Issue = "input_unavailable"
	}
	return j, nil
}
func (s *Store) RenewEgress(ctx context.Context, j egress.Job) error {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, "UPDATE egress_settings SET generation=generation WHERE id=1"); e != nil {
		return e
	}
	now := egress.Now()
	r, e := tx.ExecContext(ctx, "UPDATE egress_jobs SET lease_until=? WHERE hash=? AND token=? AND revision=? AND lease_until>? AND paused=0 AND state!='DELETED' AND hash IN(SELECT hash FROM egress_inventory)", now+120000, j.Hash, j.Token, j.Revision, now)
	if e != nil {
		return e
	}
	n, _ := r.RowsAffected()
	if n != 1 {
		return ErrConflict
	}
	return tx.Commit()
}
func (s *Store) CompleteEgress(ctx context.Context, c egress.Completion) error {
	j, r := c.Job, c.Report
	if !validDigest(j.Hash) || r.Hash != j.Hash || r.Version != egress.Version || len(r.Samples) > 256 || len(r.Attempts) > 8 || len(r.Checks) > 2 || len(r.Health) > 3 || len(r.Evidence) > 4 || len(r.Delivery) > 3 {
		return ErrInvalid
	}
	for _, part := range r.Evidence {
		if len(part.Evidence) != 0 || len(part.Samples) > 256 || len(part.Checks) > 2 || len(part.Health) > 3 || len(part.Delivery) > 3 || part.Hash != j.Hash || part.Version != egress.Version {
			return ErrInvalid
		}
	}
	b, e := s.vault.Open(j.Hash)
	var raw []byte
	if e == nil {
		raw, e = io.ReadAll(io.LimitReader(b, repair.MaxBytes+1))
		b.Close()
	}
	inputUnavailable := e != nil || egress.Hash(raw) != j.Hash
	if inputUnavailable {
		raw = nil
	}
	plan := egress.Prepare(raw)
	now := egress.Now()
	evaluate := func(now int64) (egress.Decision, string) {
		d := egress.Evaluate(r, plan, j.Hash, now)
		if inputUnavailable {
			d.State = "UNKNOWN"
			d.Reason = "input_unavailable"
			d.Eligible = false
			d.Until = 0
		}
		health := egress.AdsHealthForDecision(r, plan, d, now)
		if d.Eligible && health != "healthy" {
			d.Eligible = false
			d.State = "UNKNOWN"
			d.Reason = "payload_health_" + health
			d.Until = 0
		}
		return d, health
	}
	r.GeneralHealth = egress.Health(r, plan, now)
	d, health := evaluate(now)
	var candidateHash string
	var candidateSize int64
	if d.Eligible {
		ref, e := s.vault.Put(ctx, bytes.NewReader(plan.Variants[1].Config))
		if e != nil {
			return e
		}
		defer s.vault.Release(ref.Hash)
		candidateHash, candidateSize = ref.Hash, ref.Size
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, "UPDATE egress_settings SET generation=generation+1 WHERE id=1"); e != nil {
		return e
	}
	// The writer reservation is held before reading the authoritative clock.
	// Waiting for SQLite or candidate storage must not extend an expired lease.
	now = egress.Now()
	d, health = evaluate(now)
	if d.Eligible && candidateHash == "" {
		return ErrConflict
	}
	var accepted bool
	var failures, streak, firstCompleted int
	var last, claimed int64
	e = tx.QueryRowContext(ctx, "SELECT ever_accepted,failures,last_failure,claimed_at,retry_streak,first_completed FROM egress_jobs WHERE hash=? AND revision=? AND token=? AND lease_until>? AND paused=0 AND state!='DELETED' AND hash IN(SELECT hash FROM egress_inventory)", j.Hash, j.Revision, j.Token, now).Scan(&accepted, &failures, &last, &claimed, &streak, &firstCompleted)
	if errors.Is(e, sql.ErrNoRows) {
		return ErrConflict
	}
	if e != nil {
		return e
	}
	if r.Started < claimed-2000 || r.Finished > now+2000 {
		return ErrConflict
	}
	lifecycle, e := lifecycleState(ctx, tx)
	if e != nil {
		return e
	}
	// Reserve completion time before the ten-minute evidence deadline. Slow
	// probes are scheduled earlier, while admission remains resource bounded.
	lead := min(int64(240000), max(int64(60000), 2*(r.Finished-r.Started)+30000))
	interval := min(lifecycle.HealthMinutes*60000, egress.IranInterval-lead)
	healthUntil := int64(0)
	if health == "healthy" {
		healthUntil = now + lifecycle.HealthMinutes*60000 + 120000
	}
	if d.Eligible {
		d.Until = min(d.Until, healthUntil, r.Finished+egress.IranInterval)
	}
	hb, _ := json.Marshal(map[string]any{"scope": "ads_application_exchange", "status": health, "general_status": r.GeneralHealth, "general_payload": r.Health, "delivery": d.Delivery})
	if _, e = tx.ExecContext(ctx, `INSERT INTO egress_health VALUES(?,?,?,?,?) ON CONFLICT(hash) DO UPDATE SET status=excluded.status,checked_at=excluded.checked_at,valid_until=excluded.valid_until,report=excluded.report`, j.Hash, health, now, healthUntil, string(hb)); e != nil {
		return e
	}
	if health == "unhealthy" {
		d.Reason = "ads_delivery_unhealthy"
	}
	guard, e := s.applyIran(ctx, tx, r, plan, now)
	if e != nil {
		return e
	}
	if guard.Quarantined {
		d.Eligible = false
		d.Until = 0
		d.State = "QUARANTINED"
		d.Reason = "ads_iran_quarantined"
	}
	if guard.Blocked {
		d.State = "DELETED"
		d.Reason = "confirmed_ads_iran"
	}
	state := d.State
	if accepted && !d.Eligible && !guard.Blocked {
		state = "QUARANTINED"
	}
	reason := retirementReason(d, health, accepted)
	retired, hits, err := s.retirement(ctx, tx, j.Hash, reason, d.Eligible, r.Started, r.Finished, now)
	if err != nil {
		return err
	}
	failures = hits
	if d.Eligible {
		accepted = true
		last = 0
	} else if reason != "" {
		last = now
	}
	definitive := d.Eligible || d.Negative || retired || guard.Blocked
	eventReason := d.Reason
	if retired {
		state = "DELETED"
		eventReason = reason
	}
	if health == "unhealthy" && !guard.Blocked && !retired {
		eventReason = "ads_delivery_unhealthy"
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO egress_events(hash,at,state,reason,definitive,duration_ms) VALUES(?,?,?,?,?,?)", j.Hash, now, state, eventReason, definitive, max(0, r.Finished-r.Started)); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "UPDATE egress_metrics SET completed=completed+1,first_assessments=first_assessments+?,definitive=definitive+? WHERE id=1", firstCompleted == 0, definitive); e != nil {
		return e
	}
	if e = recordEgressRun(ctx, tx, j.Hash, state, definitive, r.Started, now, max(int64(0), r.Finished-r.Started)); e != nil {
		return e
	}
	if guard.Blocked {
		return tx.Commit()
	}
	if retired {
		if e = s.revokeEgress(ctx, tx, j.Hash, reason, now); e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, "UPDATE egress_metrics SET unhealthy_deleted=unhealthy_deleted+1 WHERE id=1"); e != nil {
			return e
		}
		return tx.Commit()
	}
	decision, _ := json.Marshal(d)
	report, _ := json.Marshal(r)
	// Keep proof-bearing material for an accepted member during uncertainty; every
	// output uses current eligibility, so retained data is never a stale fallback.
	if candidateHash == "" && accepted {
		if e = tx.QueryRowContext(ctx, "SELECT candidate_hash,candidate_size FROM egress_jobs WHERE hash=?", j.Hash).Scan(&candidateHash, &candidateSize); e != nil {
			return e
		}
	}
	streak++
	if d.Eligible {
		streak = 0
	}
	fastRetry := r.Code == "network_retry_pending" || j.Attempt == 1 && (d.Reason == "controls_incomplete" || d.Reason == "ads_peer_echo_incomplete" || d.Reason == "ads_profile_evidence_incomplete")
	delay := egressRetryDelay(j.Hash, state, health, accepted, streak, interval, fastRetry)
	if hits > 0 {
		delay = 30000
	}
	if guard.Quarantined {
		delay = guard.Delay
	}
	_, e = tx.ExecContext(ctx, `UPDATE egress_jobs SET state=?,reason=?,ever_accepted=?,tested=?,valid_until=?,candidate_hash=?,candidate_size=?,format=?,decision=?,report=?,token='',lease_until=0,next_run=?,failures=?,last_failure=?,retry_streak=?,first_completed=1 WHERE hash=?`, state, d.Reason, accepted, now, d.Until, candidateHash, candidateSize, plan.Format, string(decision), string(report), now+delay, failures, last, streak, j.Hash)
	if e != nil {
		return e
	}
	if e = saveEgressCountries(ctx, tx, j.Hash, r, d, now); e != nil {
		return e
	}
	if j.Deep {
		if _, e = tx.ExecContext(ctx, "UPDATE egress_jobs SET deep_completed=1 WHERE hash=?", j.Hash); e != nil {
			return e
		}
	}
	return tx.Commit()
}
func (s *Store) revokeEgress(ctx context.Context, tx *sql.Tx, hash, reason string, now int64) error {
	for _, q := range []string{
		"INSERT INTO input_exclusions SELECT identity,?,hash,1 FROM egress_owners WHERE hash=? ON CONFLICT(identity) DO UPDATE SET at=excluded.at,hash=excluded.hash,manual=1",
	} {
		if _, e := tx.ExecContext(ctx, q, now, hash); e != nil {
			return e
		}
	}
	aliases := "SELECT o.hash FROM egress_owners o WHERE o.identity IN(SELECT identity FROM egress_owners WHERE hash=?)"
	for _, q := range []string{
		"DELETE FROM responses WHERE source_id IN(SELECT source FROM egress_owners WHERE identity IN(SELECT identity FROM egress_owners WHERE hash=?))",
		"DELETE FROM staged WHERE identity IN(SELECT identity FROM egress_owners WHERE hash=?)",
		"DELETE FROM artifacts WHERE id IN(SELECT identity FROM egress_owners WHERE hash=?)",
	} {
		if _, err := tx.ExecContext(ctx, q, hash); err != nil {
			return err
		}
	}
	for _, table := range []string{"egress_health", "egress_geo_results", "egress_countries", "conversion_results", "conversion_jobs", "egress_turns", "egress_retirements"} {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+table+" WHERE hash=? OR hash IN("+aliases+")", hash, hash); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM repairs WHERE raw_hash=? OR raw_hash IN("+aliases+")", hash, hash); err != nil {
		return err
	}
	_, e := tx.ExecContext(ctx, "UPDATE egress_jobs SET state='DELETED',reason=CASE WHEN hash=? THEN ? ELSE 'identity_excluded' END,ever_accepted=0,revision=revision+1,token='',lease_until=0,valid_until=0,candidate_hash='',candidate_size=0,report='{}',decision='{}' WHERE hash=? OR hash IN("+aliases+")", hash, reason, hash, hash)
	return e
}
func (s *Store) manageEgress(ctx context.Context, tx *sql.Tx, action string, all bool, now int64) error {
	condition := "hash IN (SELECT hash FROM egress_owners WHERE identity IN (SELECT id FROM selected_configs))"
	if all {
		condition = "1=1"
	}
	if action != "delete" {
		_, e := tx.ExecContext(ctx, "UPDATE egress_jobs SET paused=?,revision=revision+1,token='',lease_until=0,valid_until=0,next_run=0,state=CASE WHEN ever_accepted=1 THEN 'QUARANTINED' ELSE state END WHERE ("+condition+") AND state!='DELETED'", action == "disable")
		return e
	}
	if _, e := tx.ExecContext(ctx, "UPDATE egress_jobs SET revision=revision+1,token='',lease_until=0,valid_until=0 WHERE ("+condition+") AND ever_accepted=0"); e != nil {
		return e
	}
	rows, e := tx.QueryContext(ctx, "SELECT hash FROM egress_jobs WHERE ("+condition+") AND ever_accepted=1")
	if e != nil {
		return e
	}
	hashes := []string{}
	for rows.Next() {
		var h string
		if e = rows.Scan(&h); e != nil {
			rows.Close()
			return e
		}
		hashes = append(hashes, h)
	}
	rows.Close()
	for _, h := range hashes {
		if e = s.revokeEgress(ctx, tx, h, "manual_delete", now); e != nil {
			return e
		}
	}
	return nil
}
func (s *Store) expireEgress(ctx context.Context, tx *sql.Tx, now int64) error {
	if _, e := tx.ExecContext(ctx, "UPDATE egress_jobs SET state='QUARANTINED',reason='evidence_expired' WHERE ever_accepted=1 AND state='RESIDENTIAL' AND valid_until<=?", now); e != nil {
		return e
	}
	// Durable investigation inputs are not ordinary output artifacts. Unknown,
	// unsupported and previously accepted inputs survive ordinary retention.

	if _, e := tx.ExecContext(ctx, "DELETE FROM egress_countries WHERE last_seen<=? OR hash IN(SELECT hash FROM egress_jobs WHERE state='DELETED')", now-egress.CountryWindow); e != nil {
		return e
	}
	if _, e := tx.ExecContext(ctx, "DELETE FROM egress_geo_results WHERE hash IN(SELECT hash FROM egress_jobs WHERE state='DELETED')"); e != nil {
		return e
	}
	_, e := tx.ExecContext(ctx, "DELETE FROM egress_events WHERE at<?", now-int64(24*time.Hour/time.Millisecond))
	return e
}
