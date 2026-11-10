package panel

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"time"

	"github.com/rezajafari0970/Config_manager/internal/convert"
	"github.com/rezajafari0970/Config_manager/internal/probe"
	"github.com/rezajafari0970/Config_manager/internal/repair"
	"github.com/rezajafari0970/Config_manager/internal/residential"
)

func residentialSettings(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}) (residential.Policy, residential.Trust, error) {
	var p residential.Policy
	var t residential.Trust
	var pb, tb string
	e := q.QueryRowContext(ctx, "SELECT policy,trust FROM residential_settings WHERE id=1").Scan(&pb, &tb)
	if e == nil {
		e = json.Unmarshal([]byte(pb), &p)
	}
	if e == nil {
		e = json.Unmarshal([]byte(tb), &t)
	}
	return p, t, e
}
func (s *Store) ClaimResidential(ctx context.Context) (*residential.Job, error) {
	return s.claimResidentialFor(ctx, "")
}
func (s *Store) claimResidentialFor(ctx context.Context, worker string) (*residential.Job, error) {
	if e := s.residentialBarrier(ctx); e != nil {
		if e = s.initResidentialJournal(ctx, s.residentialDBPath); e != nil {
			return nil, e
		}
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, "UPDATE residential_settings SET generation=generation WHERE id=1"); e != nil {
		return nil, e
	}
	p, t, e := residentialSettings(ctx, tx)
	if e != nil {
		return nil, e
	}
	now := time.Now().UnixMilli()
	var enabled bool
	var active int
	if e = tx.QueryRowContext(ctx, "SELECT enabled,(SELECT count(*) FROM residential_records WHERE token!='' AND lease_until>?) FROM residential_settings WHERE id=1", now).Scan(&enabled, &active); e != nil {
		return nil, e
	}
	if !enabled || active >= p.Concurrency {
		return nil, tx.Commit()
	}
	j := &residential.Job{Policy: p, TrustHash: t.Digest()}
	var size int64
	priority := s.residentialTurn.Add(1)%4 != 0
	e = tx.QueryRowContext(ctx, `SELECT hash,kind,size,revision,epoch,ever_accepted FROM residential_records r WHERE state!='DELETED' AND retain_until>CAST(unixepoch('subsec')*1000 AS INTEGER) AND paused=0 AND (token='' OR lease_until<=?) AND next_run<=? AND (?='' OR negative_worker!=? OR negative_last<?) ORDER BY CASE WHEN ever_accepted=1 AND ? THEN 0 ELSE 1 END,coalesce((SELECT min(coalesce(f.last_claim,0)) FROM residential_owners o LEFT JOIN residential_source_fairness f ON f.source=o.source WHERE o.hash=r.hash),0),next_run,first_seen,hash LIMIT 1`, now, now, worker, worker, now-120000, priority).Scan(&j.Hash, &j.Kind, &size, &j.Revision, &j.Epoch, &j.Deep)
	if errors.Is(e, sql.ErrNoRows) {
		return nil, tx.Commit()
	}
	if e != nil {
		return nil, e
	}
	j.Token = randomToken()
	if _, e = tx.ExecContext(ctx, "INSERT INTO residential_source_fairness SELECT source,? FROM residential_owners WHERE hash=? ON CONFLICT(source) DO UPDATE SET last_claim=excluded.last_claim", now, j.Hash); e != nil {
		return nil, e
	}
	if _, e = tx.ExecContext(ctx, "UPDATE residential_records SET token=?,lease_until=?,claimed_at=?,attempts=attempts+1 WHERE hash=?", j.Token, now+120000, now, j.Hash); e != nil {
		return nil, e
	}
	if e = tx.Commit(); e != nil {
		return nil, e
	}
	if size > repair.MaxBytes {
		j.Issue = "inspection_size_limit"
		return j, nil
	}
	body, e := s.vault.Open(j.Hash)
	if e != nil {
		j.Issue = "original_blob_unavailable"
		return j, nil
	}
	defer body.Close()
	j.Raw, e = io.ReadAll(io.LimitReader(body, repair.MaxBytes+1))
	if e != nil || repair.Digest(j.Raw) != j.Hash {
		j.Raw = nil
		j.Issue = "original_blob_unavailable"
	}
	return j, nil
}
func (s *Store) RenewResidential(ctx context.Context, j residential.Job) error {
	now := time.Now().UnixMilli()
	r, e := s.db.ExecContext(ctx, `UPDATE residential_records SET lease_until=? WHERE hash=? AND revision=? AND epoch=? AND token=? AND lease_until>? AND state!='DELETED' AND retain_until>CAST(unixepoch('subsec')*1000 AS INTEGER) AND EXISTS(SELECT 1 FROM residential_settings WHERE enabled=1 AND policy_hash=? AND trust_hash=?)`, now+120000, j.Hash, j.Revision, j.Epoch, j.Token, now, j.Policy.Digest(), j.TrustHash)
	if e != nil {
		return e
	}
	n, e := r.RowsAffected()
	if e == nil && n != 1 {
		return ErrConflict
	}
	return e
}

func validateResidentialReport(r residential.Report) bool {
	if r.Version != residential.Version || r.CoreSHA256 != probe.CoreSHA256 || r.TemplateVersion != convert.Version || !safeCode(r.Worker) || !safeCode(r.InfrastructureEpoch) || len(r.Variants) > 2 || len(r.Observations) > 4096 || len(r.Controls) > 128 || len(r.InfrastructureControls) > 8 || len(r.IPFacts) > 8192 {
		return false
	}
	for _, o := range append(append([]residential.Observation{}, r.Observations...), append(r.Controls, r.InfrastructureControls...)...) {
		for _, v := range []string{o.Variant, o.Domain, o.Path, o.Method, o.Outcome, o.Code, o.Transport} {
			if v != "" && !safeCode(v) {
				return false
			}
		}
		if len(o.IP) > 45 || len(o.Nonce) > 128 || len(o.ConnectionID) > 128 || len(o.Capability) > 128 || len(o.Attestation) > 128 || len(o.ConfigHash) > 64 || len(o.RuntimeHash) > 64 || len(o.PolicyHash) > 64 {
			return false
		}
	}
	for _, f := range r.IPFacts {
		if !safeCode(f.Source) || len(f.IP) > 45 || len(f.Country) > 3 || len(f.Class) > 16 || len(f.MAC) > 64 {
			return false
		}
	}
	return true
}

func (s *Store) CompleteResidential(ctx context.Context, c residential.Completion) error {
	if !validDigest(c.Hash) || !validateResidentialReport(c.Report) || len(c.Candidate) > repair.MaxBytes {
		return ErrInvalid
	}
	// Validate exact raw->common-template pair again in the unprivileged owner.
	// A worker cannot replace a config with unrelated credentials or a hypothesis.
	candidateHash := ""
	candidateSize := int64(0)
	original, readErr := s.residentialBytes(c.Hash, false)
	if readErr != nil {
		return ErrConflict
	}
	expectedPlan := residential.NewCoreRegistry().Prepare("xray-26.7.28", original)
	reportBound := len(expectedPlan.Variants) == 2 && len(c.Report.Variants) == 2
	for i := range c.Report.Variants {
		v := &c.Report.Variants[i]
		matched := false
		for _, expected := range expectedPlan.Variants {
			if expected.Name == v.Name && expected.SHA256 == v.Hash {
				matched = true
				v.PathsComplete = v.PathsComplete && expected.PathsComplete && slices.Equal(v.Paths, expected.Paths)
				v.Paths = expected.Paths
				v.ImportRule = expected.ImportRule
				break
			}
		}
		if !matched {
			reportBound = false
			v.PathsComplete = false
		}
	}

	if len(c.Candidate) > 0 {
		body, e := s.vault.Open(c.Hash)
		if e != nil {
			return e
		}
		raw, e := io.ReadAll(io.LimitReader(body, repair.MaxBytes+1))
		body.Close()
		if e != nil {
			return e
		}
		plan := probe.Prepare(raw)
		matched := false
		for _, v := range plan.Candidates {
			if v.RequiresConnection {
				continue
			}
			expected, err := convert.Apply(v.Config)
			if err != nil || !bytes.Equal(expected, c.Candidate) {
				continue
			}
			rawMatch, templateMatch := false, false
			expectedRaw := ""
			for _, execution := range expectedPlan.Variants {
				if execution.Name == "raw" {
					expectedRaw = execution.SHA256
				}
			}
			for _, variant := range c.Report.Variants {
				if variant.Name == "raw" && variant.Hash == expectedRaw {
					rawMatch = true
				}
				if variant.Name == "template" && variant.Hash == repair.Digest(expected) {
					templateMatch = true
				}
			}
			if rawMatch && templateMatch {
				matched = true
				break
			}
		}
		if !matched {
			return ErrInvalid
		}
		ref, e := s.vault.Put(ctx, bytes.NewReader(c.Candidate))
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
	if _, e = tx.ExecContext(ctx, "UPDATE residential_settings SET generation=generation WHERE id=1"); e != nil {
		return e
	}
	now := time.Now().UnixMilli()
	p, t, e := residentialSettings(ctx, tx)
	if e != nil {
		return e
	}
	if c.Report.PolicyHash != p.Digest() || c.TrustHash != t.Digest() || c.Report.Finished > now+2000 || c.Report.Started < now-1200000 {
		return ErrConflict
	}
	var accepted bool
	var claimed int64
	var count int
	var first, last int64
	var worker, epoch string
	e = tx.QueryRowContext(ctx, `SELECT ever_accepted,claimed_at,negative_count,negative_first,negative_last,negative_worker,negative_epoch FROM residential_records WHERE hash=? AND revision=? AND epoch=? AND token=? AND lease_until>? AND state!='DELETED' AND retain_until>CAST(unixepoch('subsec')*1000 AS INTEGER)`, c.Hash, c.Revision, c.Epoch, c.Token, now).Scan(&accepted, &claimed, &count, &first, &last, &worker, &epoch)
	if errors.Is(e, sql.ErrNoRows) {
		return ErrConflict
	}
	if e != nil {
		return e
	}
	if c.Report.Started < claimed-2000 {
		return ErrConflict
	}
	for _, o := range c.Report.Observations {
		if o.At < c.Report.Started || o.At > c.Report.Finished+2000 {
			return ErrInvalid
		}
	}
	var revokedAt int64
	if e = tx.QueryRowContext(ctx, `SELECT max(coalesce((SELECT max(at) FROM residential_tombstones WHERE hash=?),0),coalesce((SELECT max(t.at) FROM residential_identity_tombstones t JOIN residential_owners o ON o.identity=t.identity WHERE o.hash=?),0))`, c.Hash, c.Hash).Scan(&revokedAt); e != nil {
		return e
	}
	if revokedAt > 0 && c.Report.Started <= revokedAt {
		return ErrConflict
	}
	c.Report.IPFacts = residential.AuthenticatedFacts(c.Report.IPFacts, s.key)
	d := residential.Evaluate(c.Report, p, t, now)
	if !reportBound && (d.Eligible || d.CredibleNegative) {
		d.Eligible = false
		d.CredibleNegative = false
		d.State = "UNKNOWN"
		d.Reason = "report_not_bound_to_original"
		d.ValidUntil = 0
	}
	if d.Eligible && candidateHash == "" {
		return ErrInvalid
	}
	state := d.State
	if accepted && !d.Eligible {
		state = "QUARANTINED"
	}
	if d.Eligible {
		accepted = true
		count = 0
		first = 0
		last = 0
		worker = ""
		epoch = ""
	}
	if accepted && d.CredibleNegative && c.Report.InfrastructureOK && c.Report.Started >= last+int64(p.ConfirmationSeconds)*1000 {
		if count == 0 {
			first = c.Report.Started
		}
		count++
		last = c.Report.Finished
		worker = c.Report.Worker
		epoch = c.Report.InfrastructureEpoch
	} else if !d.CredibleNegative && !d.Eligible {
		count = 0
		first = 0
		last = 0
		worker = ""
		epoch = ""
	}
	report, _ := json.Marshal(c.Report)
	decision, _ := json.Marshal(d)
	if len(report) > 2<<20 {
		return ErrInvalid
	}
	next := now + int64(p.RetestSeconds)*1000 + int64(c.Hash[0]%31)*1000 - 15000
	if state == "QUARANTINED" {
		next = now + int64(p.ConfirmationSeconds)*1000 + int64(c.Hash[1]%11)*1000
	}
	if _, e = tx.ExecContext(ctx, `UPDATE residential_records SET state=?,reason=?,ever_accepted=?,candidate_hash=CASE WHEN ?!='' THEN ? ELSE candidate_hash END,candidate_size=CASE WHEN ?!='' THEN ? ELSE candidate_size END,policy_hash=?,trust_hash=?,template_version=?,tested=?,valid_until=?,decision=?,report=?,token='',lease_until=0,next_run=?,negative_count=?,negative_first=?,negative_last=?,negative_worker=?,negative_epoch=? WHERE hash=?`, state, d.Reason, accepted, candidateHash, candidateHash, candidateHash, candidateSize, p.Digest(), t.Digest(), convert.Version, c.Report.Finished, d.ValidUntil, string(decision), string(report), next, count, first, last, worker, epoch, c.Hash); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO residential_audit(hash,revision,at,state,reason,report,decision) VALUES(?,?,?,?,?,?,?)", c.Hash, c.Revision, now, state, d.Reason, string(report), string(decision)); e != nil {
		return e
	}
	// Keep a bounded diagnostic trail per hash without expiring the config itself.
	if _, e = tx.ExecContext(ctx, "DELETE FROM residential_audit WHERE hash=? AND id NOT IN (SELECT id FROM residential_audit WHERE hash=? ORDER BY id DESC LIMIT ?)", c.Hash, c.Hash, func() int {
		if accepted {
			return 20
		}
		return 2
	}()); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO residential_metrics(at,wall_ms,cpu_ms,rss_bytes,state) VALUES(?,?,?,?,?); DELETE FROM residential_metrics WHERE id<(SELECT max(id)-2000 FROM residential_metrics)", now, c.Report.WallMS, c.Report.CPUMS, c.Report.PeakRSSBytes, state); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "UPDATE residential_settings SET generation=generation+CASE WHEN ? THEN 1 ELSE 0 END,updated=? WHERE id=1", accepted, now); e != nil {
		return e
	}
	if accepted && count >= p.ConfirmationAttempts && last-first >= int64(p.ConfirmationAttempts-1)*int64(p.ConfirmationSeconds)*1000 {
		// Two different retained configs failing together suspend destructive action.
		// They remain unpublished; infrastructure/intel uncertainty never deletes.
		var shared int
		var circuit int64
		if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM residential_records WHERE ever_accepted=1 AND negative_count>0 AND negative_last>?", now-120000).Scan(&shared); e != nil {
			return e
		}
		if e = tx.QueryRowContext(ctx, "SELECT circuit_until FROM residential_settings WHERE id=1").Scan(&circuit); e != nil {
			return e
		}
		if shared > 1 {
			if _, e = tx.ExecContext(ctx, "UPDATE residential_settings SET circuit_until=? WHERE id=1", now+300000); e != nil {
				return e
			}
		} else if circuit <= now {
			if e = s.revokeResidential(ctx, tx, c.Hash, d.Reason, false, now); e != nil {
				return e
			}
		}
	}
	return tx.Commit()
}

// Expiry is a publication gate even when workers and scheduled cleanup are down.
func (s *Store) expireResidential(ctx context.Context, tx *sql.Tx, now int64) error {
	if e := s.expireResidentialLifetime(ctx, tx, now); e != nil {
		return e
	}
	r, e := tx.ExecContext(ctx, `UPDATE residential_records SET state='QUARANTINED',reason='evidence_expired',next_run=min(next_run,?),valid_until=0 WHERE ever_accepted=1 AND state='ACCEPTED' AND valid_until<=?`, now, now)
	if e != nil {
		return e
	}
	n, e := r.RowsAffected()
	if e != nil {
		return e
	}
	if n > 0 {
		_, e = tx.ExecContext(ctx, "UPDATE residential_settings SET generation=generation+1,updated=? WHERE id=1", now)
	}
	return e
}
