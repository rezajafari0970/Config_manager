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
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, "UPDATE egress_settings SET generation=generation WHERE id=1"); e != nil {
		return nil, e
	}
	now := egress.Now()
	var active, enabled int
	if e = tx.QueryRowContext(ctx, "SELECT enabled,(SELECT count(*) FROM egress_jobs WHERE token!='' AND lease_until>?) FROM egress_settings WHERE id=1", now).Scan(&enabled, &active); e != nil {
		return nil, e
	}
	if enabled == 0 || active >= egress.MaxWorkers {
		return nil, tx.Commit()
	}
	j := &egress.Job{}
	var size int64
	e = tx.QueryRowContext(ctx, `SELECT hash,kind,size,revision,ever_accepted,attempts FROM egress_jobs WHERE state!='DELETED' AND paused=0 AND next_run<=? AND (token='' OR lease_until<=?) ORDER BY CASE WHEN ever_accepted=1 AND valid_until<=? THEN 0 WHEN reason='manual_recheck' THEN 1 WHEN tested=0 AND EXISTS(SELECT 1 FROM memberships m WHERE m.raw_hash=egress_jobs.hash) THEN 2 WHEN attempts=1 AND EXISTS(SELECT 1 FROM memberships m WHERE m.raw_hash=egress_jobs.hash) THEN 3 WHEN ever_accepted=1 THEN 4 WHEN tested=0 THEN 5 ELSE 6 END,next_run,first_seen,hash LIMIT 1`, now, now, now+60000).Scan(&j.Hash, &j.Kind, &size, &j.Revision, &j.Deep, &j.Attempt)
	if errors.Is(e, sql.ErrNoRows) {
		return nil, tx.Commit()
	}
	if e != nil {
		return nil, e
	}
	j.Deep = j.Deep || j.Attempt > 0
	j.Attempt++
	j.Token = randomToken()
	if _, e = tx.ExecContext(ctx, "UPDATE egress_jobs SET token=?,lease_until=?,claimed_at=?,attempts=attempts+1 WHERE hash=?", j.Token, now+120000, now, j.Hash); e != nil {
		return nil, e
	}
	if e = tx.Commit(); e != nil {
		return nil, e
	}
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
	r, e := s.db.ExecContext(ctx, "UPDATE egress_jobs SET lease_until=? WHERE hash=? AND token=? AND revision=? AND lease_until>? AND paused=0 AND state!='DELETED'", egress.Now()+120000, j.Hash, j.Token, j.Revision, egress.Now())
	if e != nil {
		return e
	}
	n, _ := r.RowsAffected()
	if n != 1 {
		return ErrConflict
	}
	return nil
}
func (s *Store) CompleteEgress(ctx context.Context, c egress.Completion) error {
	j, r := c.Job, c.Report
	if !validDigest(j.Hash) || r.Hash != j.Hash || r.Version != egress.Version || len(r.Samples) > 256 || len(r.Facts) > 64 || len(r.Attempts) > 8 || len(r.Checks) > 2 {
		return ErrInvalid
	}
	b, e := s.vault.Open(j.Hash)
	var raw []byte
	if e == nil {
		raw, e = io.ReadAll(io.LimitReader(b, repair.MaxBytes+1))
		b.Close()
	}
	if e != nil || egress.Hash(raw) != j.Hash {
		raw = nil
	}
	plan := egress.Prepare(raw)
	now := egress.Now()
	d := egress.Evaluate(r, plan, j.Hash, now)
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
	var accepted bool
	var failures int
	var last, claimed int64
	e = tx.QueryRowContext(ctx, "SELECT ever_accepted,failures,last_failure,claimed_at FROM egress_jobs WHERE hash=? AND revision=? AND token=? AND lease_until>? AND paused=0 AND state!='DELETED'", j.Hash, j.Revision, j.Token, now).Scan(&accepted, &failures, &last, &claimed)
	if errors.Is(e, sql.ErrNoRows) {
		return ErrConflict
	}
	if e != nil {
		return e
	}
	if r.Started < claimed-2000 || r.Finished > now+2000 {
		return ErrConflict
	}
	state := d.State
	if accepted && !d.Eligible {
		state = "QUARANTINED"
	}
	if d.Eligible {
		accepted = true
		failures = 0
		last = 0
	} else if accepted && d.Negative && now-last >= 30000 {
		failures++
		last = now
	} else if !d.Negative {
		failures = 0
		last = 0
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO egress_events(hash,at,state,reason) VALUES(?,?,?,?)", j.Hash, now, state, d.Reason); e != nil {
		return e
	}
	var correlated int
	if e = tx.QueryRowContext(ctx, "SELECT count(DISTINCT hash) FROM egress_events WHERE at>? AND reason='controls_incomplete' AND state='QUARANTINED'", now-60000).Scan(&correlated); e != nil {
		return e
	}
	if correlated >= 3 {
		if _, e = tx.ExecContext(ctx, "UPDATE egress_settings SET circuit_until=max(circuit_until,?)", now+300000); e != nil {
			return e
		}
	}
	var circuit int64
	if e = tx.QueryRowContext(ctx, "SELECT circuit_until FROM egress_settings").Scan(&circuit); e != nil {
		return e
	}
	if accepted && failures >= 3 && now >= circuit {
		if e = s.revokeEgress(ctx, tx, j.Hash, "confirmed_loss", now); e != nil {
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
	delay := int64(300000)
	if r.Code == "network_retry_pending" || j.Attempt == 1 && (d.Reason == "controls_incomplete" || d.Reason == "ads_peer_echo_incomplete") {
		delay = 30000
	}
	if state == "UNSUPPORTED" {
		delay = 3600000
	}
	if accepted {
		delay = 120000
	}
	if accepted && !d.Eligible {
		delay = 30000
	}
	_, e = tx.ExecContext(ctx, `UPDATE egress_jobs SET state=?,reason=?,ever_accepted=?,tested=?,valid_until=?,candidate_hash=?,candidate_size=?,format=?,decision=?,report=?,token='',lease_until=0,next_run=?,failures=?,last_failure=? WHERE hash=?`, state, d.Reason, accepted, now, d.Until, candidateHash, candidateSize, plan.Format, string(decision), string(report), now+delay, failures, last, j.Hash)
	if e != nil {
		return e
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
	for _, q := range []string{
		"DELETE FROM responses WHERE source_id IN (SELECT source FROM egress_owners WHERE hash=?)",
		"DELETE FROM artifacts WHERE id IN (SELECT identity FROM egress_owners WHERE hash=?)",
		"DELETE FROM conversion_results WHERE hash=?", "DELETE FROM conversion_jobs WHERE hash=?", "DELETE FROM repairs WHERE raw_hash=?", "DELETE FROM egress_turns WHERE hash=?",
	} {
		if _, e := tx.ExecContext(ctx, q, hash); e != nil {
			return e
		}
	}
	_, e := tx.ExecContext(ctx, "UPDATE egress_jobs SET state='DELETED',reason=?,revision=revision+1,token='',lease_until=0,valid_until=0,candidate_hash='',candidate_size=0,report='{}',decision='{}' WHERE hash=?", reason, hash)
	return e
}
func (s *Store) manageEgress(ctx context.Context, tx *sql.Tx, action string, all bool, now int64) error {
	condition := "hash IN (SELECT hash FROM egress_owners WHERE identity IN (SELECT id FROM selected_configs))"
	if all {
		condition = "1=1"
	}
	if action != "delete" {
		_, e := tx.ExecContext(ctx, "UPDATE egress_jobs SET paused=?,revision=revision+1,token='',lease_until=0,valid_until=0,next_run=0,state=CASE WHEN ever_accepted=1 THEN 'QUARANTINED' ELSE 'PENDING' END WHERE "+condition, action == "disable")
		return e
	}
	if _, e := tx.ExecContext(ctx, "UPDATE egress_jobs SET state='DELETED',reason='ordinary_delete',revision=revision+1,token='',lease_until=0,valid_until=0,candidate_hash='',candidate_size=0,report='{}',decision='{}' WHERE ("+condition+") AND ever_accepted=0"); e != nil {
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
	if _, e := tx.ExecContext(ctx, "UPDATE egress_jobs SET state='DELETED',reason='retention_expired',candidate_hash='',candidate_size=0,report='{}',decision='{}',revision=revision+1,token='',lease_until=0 WHERE ever_accepted=0 AND tested>0 AND state!='DELETED' AND first_seen+(SELECT retention_seconds*1000 FROM settings)<=?", now); e != nil {
		return e
	}
	_, e := tx.ExecContext(ctx, "DELETE FROM egress_events WHERE at<?", now-int64(24*time.Hour/time.Millisecond))
	return e
}
