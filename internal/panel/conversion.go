package panel

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/rezajafari0970/Config_manager/internal/blob"
	"github.com/rezajafari0970/Config_manager/internal/convert"
	"github.com/rezajafari0970/Config_manager/internal/probe"
	"github.com/rezajafari0970/Config_manager/internal/repair"
	"io"
	"strings"
	"time"
)

// Schema 6 deliberately removes the persistent laboratory. Only current source
// membership may own conversion jobs or results. Original payloads remain exact.
func (s *Store) migrateV6() error {
	var v int
	if e := s.db.QueryRow("PRAGMA user_version").Scan(&v); e != nil {
		return e
	}
	if v >= 6 {
		return nil
	}
	previous := probe.Settings{Enabled: true, Concurrency: 6, TimeoutSeconds: 10, RetestSeconds: 900, HealthTTLSeconds: 1800}
	if v == 5 {
		if e := s.db.QueryRow("SELECT enabled,concurrency,timeout_seconds,retest_seconds,health_ttl_seconds FROM probe_settings WHERE id=1").Scan(&previous.Enabled, &previous.Concurrency, &previous.TimeoutSeconds, &previous.RetestSeconds, &previous.HealthTTLSeconds); e != nil {
			return e
		}
	}
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	_, e = tx.Exec(`
 DROP TABLE IF EXISTS bank_variants;DROP TABLE IF EXISTS probe_rule_evidence;
 DROP TABLE IF EXISTS probe_results;DROP TABLE IF EXISTS probe_jobs;
 DROP TABLE IF EXISTS probe_models;DROP TABLE IF EXISTS probe_settings;
 DROP TABLE IF EXISTS probe_worker;DROP TABLE IF EXISTS bank_samples;
 CREATE VIEW conversion_inputs AS SELECT DISTINCT m.raw_hash AS hash,m.kind,m.size
 FROM memberships m JOIN artifacts a ON a.id=m.artifact_id JOIN sources s ON s.id=m.source_id
 WHERE a.expires>CAST(unixepoch('subsec')*1000 AS INTEGER) AND s.enabled=1
 AND NOT EXISTS(SELECT 1 FROM config_controls c WHERE c.id=a.id AND c.expires>CAST(unixepoch('subsec')*1000 AS INTEGER));
 CREATE TABLE conversion_jobs(hash TEXT PRIMARY KEY,revision INTEGER NOT NULL DEFAULT 1,state TEXT NOT NULL DEFAULT 'queued',token TEXT NOT NULL DEFAULT '',lease_until INTEGER NOT NULL DEFAULT 0,next_run INTEGER NOT NULL DEFAULT 0,failures INTEGER NOT NULL DEFAULT 0,version TEXT NOT NULL DEFAULT '');
 CREATE INDEX conversion_due ON conversion_jobs(next_run,lease_until);
 CREATE TABLE conversion_results(hash TEXT PRIMARY KEY REFERENCES conversion_jobs(hash) ON DELETE CASCADE,candidate_hash TEXT NOT NULL,candidate_size INTEGER NOT NULL,rule TEXT NOT NULL,core TEXT NOT NULL,connection TEXT NOT NULL,code TEXT NOT NULL,tested INTEGER NOT NULL,valid_until INTEGER NOT NULL,runtime_hash TEXT NOT NULL,version TEXT NOT NULL,requires_connection INTEGER NOT NULL);
 CREATE TABLE conversion_settings(id INTEGER PRIMARY KEY CHECK(id=1),enabled INTEGER NOT NULL,concurrency INTEGER NOT NULL,timeout_seconds INTEGER NOT NULL,retest_seconds INTEGER NOT NULL,health_ttl_seconds INTEGER NOT NULL);
 INSERT INTO conversion_settings VALUES(1,1,6,10,900,1800);
 CREATE TABLE conversion_worker(id INTEGER PRIMARY KEY CHECK(id=1),status TEXT NOT NULL,code TEXT NOT NULL,version TEXT NOT NULL,active INTEGER NOT NULL,heartbeat INTEGER NOT NULL);
 INSERT INTO conversion_worker VALUES(1,'offline','','',0,0);
 INSERT INTO conversion_jobs(hash) SELECT hash FROM conversion_inputs;
 PRAGMA user_version=6;`)
	if e != nil {
		return e
	}
	if _, e = tx.Exec("UPDATE conversion_settings SET enabled=?,concurrency=?,timeout_seconds=?,retest_seconds=?,health_ttl_seconds=? WHERE id=1", previous.Enabled, previous.Concurrency, previous.TimeoutSeconds, previous.RetestSeconds, previous.HealthTTLSeconds); e != nil {
		return e
	}
	return tx.Commit()
}
func syncConversions(ctx context.Context, tx *sql.Tx) error {
	if _, e := tx.ExecContext(ctx, `DELETE FROM conversion_jobs WHERE NOT EXISTS(SELECT 1 FROM conversion_inputs i WHERE i.hash=conversion_jobs.hash)`); e != nil {
		return e
	}
	_, e := tx.ExecContext(ctx, `INSERT INTO conversion_jobs(hash) SELECT hash FROM conversion_inputs WHERE true ON CONFLICT(hash) DO NOTHING`)
	return e
}
func (s *Store) ProbeSettings(ctx context.Context) (probe.Settings, error) {
	var v probe.Settings
	e := s.db.QueryRowContext(ctx, "SELECT enabled,concurrency,timeout_seconds,retest_seconds,health_ttl_seconds FROM conversion_settings WHERE id=1").Scan(&v.Enabled, &v.Concurrency, &v.TimeoutSeconds, &v.RetestSeconds, &v.HealthTTLSeconds)
	return v, e
}
func (s *Store) SaveProbeSettings(ctx context.Context, v probe.Settings) error {
	if v.Concurrency < 1 || v.Concurrency > 8 || v.TimeoutSeconds < 1 || v.TimeoutSeconds > 120 || v.RetestSeconds < 30 || v.HealthTTLSeconds < v.RetestSeconds || v.HealthTTLSeconds > 86400 {
		return ErrInvalid
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	_, e = tx.ExecContext(ctx, "UPDATE conversion_settings SET enabled=?,concurrency=?,timeout_seconds=?,retest_seconds=?,health_ttl_seconds=? WHERE id=1", v.Enabled, v.Concurrency, v.TimeoutSeconds, v.RetestSeconds, v.HealthTTLSeconds)
	if e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "UPDATE conversion_results SET valid_until=min(valid_until,tested+?)", v.HealthTTLSeconds*1000); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "UPDATE conversion_jobs SET next_run=min(next_run,?)", time.Now().UnixMilli()+int64(v.RetestSeconds)*1000); e != nil {
		return e
	}
	if !v.Enabled {
		if _, e = tx.ExecContext(ctx, "UPDATE conversion_jobs SET token='',lease_until=0,state='queued' WHERE token!=''"); e != nil {
			return e
		}
	}
	return tx.Commit()
}
func validDigest(s string) bool { b, e := hex.DecodeString(s); return e == nil && len(b) == 32 }

func safeCode(s string) bool {
	if len(s) > 200 || s == "" {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("._-", r)) {
			return false
		}
	}
	return true
}
func (s *Store) ClaimProbe(ctx context.Context) (*probe.Job, error) {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	if e = syncConversions(ctx, tx); e != nil {
		return nil, e
	}
	var settings probe.Settings
	if e = tx.QueryRowContext(ctx, "SELECT enabled,concurrency,timeout_seconds,retest_seconds,health_ttl_seconds FROM conversion_settings WHERE id=1").Scan(&settings.Enabled, &settings.Concurrency, &settings.TimeoutSeconds, &settings.RetestSeconds, &settings.HealthTTLSeconds); e != nil {
		return nil, e
	}
	now := time.Now().UnixMilli()
	var active int
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM conversion_jobs WHERE token!='' AND lease_until>?", now).Scan(&active); e != nil {
		return nil, e
	}
	if !settings.Enabled || active >= min(8, settings.Concurrency) {
		return nil, tx.Commit()
	}
	j := &probe.Job{Settings: settings}
	var size int64
	priority := s.probeTurn.Add(1)%4 == 0
	e = tx.QueryRowContext(ctx, `SELECT i.hash,i.kind,i.size,j.revision FROM conversion_inputs i JOIN conversion_jobs j ON j.hash=i.hash WHERE (j.token='' OR j.lease_until<=?) AND (j.next_run<=? OR j.version!=?) ORDER BY CASE WHEN ((j.version=? AND j.next_run>0)=?) THEN 0 ELSE 1 END,j.next_run,j.hash LIMIT 1`, now, now, convert.Version, convert.Version, priority).Scan(&j.Hash, &j.Kind, &size, &j.Revision)
	if errors.Is(e, sql.ErrNoRows) {
		return nil, tx.Commit()
	}
	if e != nil {
		return nil, e
	}
	j.Token = randomToken()
	if _, e = tx.ExecContext(ctx, "UPDATE conversion_jobs SET token=?,lease_until=?,state='running',version=? WHERE hash=?", j.Token, now+120000, convert.Version, j.Hash); e != nil {
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
	if e != nil {
		j.Raw = nil
		j.Issue = "original_blob_unavailable"
	}
	return j, nil
}
func (s *Store) RenewProbe(ctx context.Context, j probe.Job) error {
	res, e := s.db.ExecContext(ctx, `UPDATE conversion_jobs SET lease_until=? WHERE hash=? AND token=? AND revision=? AND lease_until>? AND EXISTS(SELECT 1 FROM conversion_inputs i WHERE i.hash=conversion_jobs.hash)`, time.Now().Add(2*time.Minute).UnixMilli(), j.Hash, j.Token, j.Revision, time.Now().UnixMilli())
	if e != nil {
		return e
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return ErrConflict
	}
	return nil
}
func (s *Store) CompleteProbe(ctx context.Context, r probe.Result) error {
	if !safeCode(r.Rule) || !safeCode(r.Code) || !validDigest(r.Hash) || r.Version != convert.Version || r.CoreSHA256 != probe.CoreSHA256 {
		return ErrInvalid
	}
	if r.Core != "passed" && r.Core != "failed" && r.Core != "blocked" && r.Core != "not_run" {
		return ErrInvalid
	}
	if r.Connection != "passed" && r.Connection != "partial" && r.Connection != "failed" && r.Connection != "not_run" {
		return ErrInvalid
	}
	if r.Connection == "passed" && (r.Core != "passed" || r.EndpointsPassed != 2 || !validDigest(r.RuntimeSHA256)) {
		return ErrInvalid
	}
	var ref blob.Ref
	requires := false
	if len(r.Candidate) > 0 {
		if len(r.Candidate) > repair.MaxBytes || repair.Digest(r.Candidate) != r.SHA256 || !json.Valid(r.Candidate) {
			return ErrInvalid
		}
		// Only this version's deterministic template candidates may be completed.
		original, e := s.vault.Open(r.Hash)
		if e != nil {
			return ErrInvalid
		}
		raw, e := io.ReadAll(io.LimitReader(original, repair.MaxBytes+1))
		original.Close()
		if e != nil {
			return e
		}
		p := convert.Prepare(raw)
		matched := false
		for _, c := range p.Candidates {
			if c.SHA256 == r.SHA256 && c.Rule == r.Rule {
				matched = true
				requires = c.RequiresConnection
				break
			}
		}
		if !matched {
			return ErrInvalid
		}
		ref, e = s.vault.Put(ctx, bytes.NewReader(r.Candidate))
		if e != nil {
			return e
		}
		defer s.vault.Release(ref.Hash)
	} else if r.Core == "passed" || r.Connection == "passed" {
		return ErrInvalid
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	now := time.Now().UnixMilli()
	var failures int
	e = tx.QueryRowContext(ctx, `SELECT failures FROM conversion_jobs j WHERE hash=? AND token=? AND revision=? AND lease_until>? AND EXISTS(SELECT 1 FROM conversion_inputs i WHERE i.hash=j.hash)`, r.Hash, r.Token, r.Revision, now).Scan(&failures)
	if errors.Is(e, sql.ErrNoRows) {
		return ErrConflict
	}
	if e != nil {
		return e
	}
	var retest, ttl int64
	if e = tx.QueryRowContext(ctx, "SELECT retest_seconds,health_ttl_seconds FROM conversion_settings WHERE id=1").Scan(&retest, &ttl); e != nil {
		return e
	}
	state := "blocked"
	valid := int64(0)
	delay := retest
	if r.Core == "passed" {
		state = "converted"
	}
	if r.Connection == "passed" {
		valid = now + ttl*1000
		failures = 0
	} else {
		failures++
		delay = min(retest, int64(30<<min(failures, 6)))
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO conversion_results VALUES(?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(hash) DO UPDATE SET candidate_hash=excluded.candidate_hash,candidate_size=excluded.candidate_size,rule=excluded.rule,core=excluded.core,connection=excluded.connection,code=excluded.code,tested=excluded.tested,valid_until=excluded.valid_until,runtime_hash=excluded.runtime_hash,version=excluded.version,requires_connection=excluded.requires_connection`, r.Hash, ref.Hash, ref.Size, r.Rule, r.Core, r.Connection, r.Code, now, valid, r.RuntimeSHA256, r.Version, requires)
	if e != nil {
		return e
	}
	_, e = tx.ExecContext(ctx, "UPDATE conversion_jobs SET state=?,token='',lease_until=0,next_run=?,failures=?,version=? WHERE hash=?", state, now+delay*1000, failures, r.Version, r.Hash)
	if e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Store) ProbeWorker(ctx context.Context) (probe.WorkerStatus, error) {
	var v probe.WorkerStatus
	e := s.db.QueryRowContext(ctx, "SELECT status,code,version,active,heartbeat FROM conversion_worker WHERE id=1").Scan(&v.Status, &v.Code, &v.Version, &v.Active, &v.Heartbeat)
	return v, e
}
func (s *Store) SetProbeWorker(ctx context.Context, v probe.WorkerStatus) error {
	if !safeCode(v.Status) || v.Code != "" && !safeCode(v.Code) {
		return ErrInvalid
	}
	_, e := s.db.ExecContext(ctx, "UPDATE conversion_worker SET status=?,code=?,version=?,active=?,heartbeat=? WHERE id=1", v.Status, v.Code, v.Version, v.Active, time.Now().UnixMilli())
	return e
}
