package panel

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/rezajafari0970/Config_manager/internal/blob"
	"github.com/rezajafari0970/Config_manager/internal/probe"
	"github.com/rezajafari0970/Config_manager/internal/repair"
)

// P6: source membership is transient; encrypted laboratory samples are retained
// independently by explicit user instruction. Only fresh connection evidence is
// eligible for the server-verified subscription. Android evidence stays separate.
func (s *Store) migrateV5() error {
	var v int
	if e := s.db.QueryRow("PRAGMA user_version").Scan(&v); e != nil {
		return e
	}
	if v >= 5 {
		return nil
	}
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	_, e = tx.Exec(`CREATE TABLE bank_samples(hash TEXT PRIMARY KEY,identity TEXT NOT NULL,kind TEXT NOT NULL,size INTEGER NOT NULL,created INTEGER NOT NULL,last_seen INTEGER NOT NULL,enabled INTEGER NOT NULL DEFAULT 1,deleted INTEGER NOT NULL DEFAULT 0,revision INTEGER NOT NULL DEFAULT 1,name TEXT NOT NULL DEFAULT '');
 CREATE INDEX bank_identity ON bank_samples(identity);
 CREATE TABLE probe_jobs(hash TEXT PRIMARY KEY REFERENCES bank_samples(hash),state TEXT NOT NULL DEFAULT 'queued',token TEXT NOT NULL DEFAULT '',lease_until INTEGER NOT NULL DEFAULT 0,revision INTEGER NOT NULL DEFAULT 1,next_run INTEGER NOT NULL DEFAULT 0,failures INTEGER NOT NULL DEFAULT 0,version TEXT NOT NULL DEFAULT '');
 CREATE INDEX probe_due ON probe_jobs(next_run,lease_until);
 CREATE TABLE probe_results(hash TEXT PRIMARY KEY REFERENCES bank_samples(hash),candidate_hash TEXT NOT NULL,candidate_size INTEGER NOT NULL,rule TEXT NOT NULL,core TEXT NOT NULL,connection TEXT NOT NULL,android TEXT NOT NULL,code TEXT NOT NULL,tested INTEGER NOT NULL,valid_until INTEGER NOT NULL,runtime_hash TEXT NOT NULL,version TEXT NOT NULL,model TEXT NOT NULL,report TEXT NOT NULL);
 CREATE INDEX probe_candidate ON probe_results(candidate_hash);
 CREATE TABLE bank_variants(hash TEXT NOT NULL REFERENCES bank_samples(hash),candidate_hash TEXT NOT NULL,size INTEGER NOT NULL,rule TEXT NOT NULL,created INTEGER NOT NULL,last_tested INTEGER NOT NULL,last_status TEXT NOT NULL,PRIMARY KEY(hash,candidate_hash));
 CREATE TABLE probe_models(model TEXT PRIMARY KEY,kind TEXT NOT NULL,observed INTEGER NOT NULL,passed INTEGER NOT NULL,failed INTEGER NOT NULL,last_seen INTEGER NOT NULL,version TEXT NOT NULL,diagnosis TEXT NOT NULL);
 CREATE TABLE probe_rule_evidence(model TEXT NOT NULL,rule TEXT NOT NULL,version TEXT NOT NULL,passed INTEGER NOT NULL,failed INTEGER NOT NULL,last_tested INTEGER NOT NULL,PRIMARY KEY(model,rule,version));
 CREATE TABLE probe_settings(id INTEGER PRIMARY KEY CHECK(id=1),enabled INTEGER NOT NULL,concurrency INTEGER NOT NULL,timeout_seconds INTEGER NOT NULL,retest_seconds INTEGER NOT NULL,health_ttl_seconds INTEGER NOT NULL);
 INSERT INTO probe_settings VALUES(1,1,6,10,900,1800);
 CREATE TABLE probe_worker(id INTEGER PRIMARY KEY CHECK(id=1),status TEXT NOT NULL,code TEXT NOT NULL,version TEXT NOT NULL,active INTEGER NOT NULL,heartbeat INTEGER NOT NULL);
 INSERT INTO probe_worker VALUES(1,'offline','','',0,0);
 INSERT OR IGNORE INTO bank_samples(hash,identity,kind,size,created,last_seen,name,enabled)
 SELECT m.raw_hash,m.artifact_id,m.kind,m.size,a.created,CAST(unixepoch('subsec')*1000 AS INTEGER),s.name,NOT EXISTS(SELECT 1 FROM config_controls c WHERE c.id=a.id AND c.action='disabled') FROM memberships m JOIN artifacts a ON a.id=m.artifact_id JOIN sources s ON s.id=m.source_id WHERE a.expires>CAST(unixepoch('subsec')*1000 AS INTEGER);
 INSERT INTO probe_jobs(hash,revision) SELECT hash,revision FROM bank_samples WHERE enabled=1;
 PRAGMA user_version=5;`)
	if e != nil {
		return e
	}
	return tx.Commit()
}
func saveSample(ctx context.Context, tx *sql.Tx, hash, identity, kind string, size, stamp int64, name string) error {
	_, e := tx.ExecContext(ctx, `INSERT INTO bank_samples(hash,identity,kind,size,created,last_seen,name) VALUES(?,?,?,?,?,?,?) ON CONFLICT(hash) DO UPDATE SET last_seen=excluded.last_seen WHERE bank_samples.deleted=0`, hash, identity, kind, size, stamp, stamp, name)
	if e != nil {
		return e
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO probe_jobs(hash,revision) SELECT hash,revision FROM bank_samples WHERE hash=? AND enabled=1 AND deleted=0 ON CONFLICT(hash) DO NOTHING`, hash)
	return e
}
func (s *Store) ProbeSettings(ctx context.Context) (probe.Settings, error) {
	var v probe.Settings
	e := s.db.QueryRowContext(ctx, "SELECT enabled,concurrency,timeout_seconds,retest_seconds,health_ttl_seconds FROM probe_settings WHERE id=1").Scan(&v.Enabled, &v.Concurrency, &v.TimeoutSeconds, &v.RetestSeconds, &v.HealthTTLSeconds)
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
	_, e = tx.ExecContext(ctx, "UPDATE probe_settings SET enabled=?,concurrency=?,timeout_seconds=?,retest_seconds=?,health_ttl_seconds=? WHERE id=1", v.Enabled, v.Concurrency, v.TimeoutSeconds, v.RetestSeconds, v.HealthTTLSeconds)
	if e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "UPDATE probe_results SET valid_until=min(valid_until,tested+?)", v.HealthTTLSeconds*1000); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "UPDATE probe_jobs SET next_run=min(next_run,?)", time.Now().UnixMilli()+int64(v.RetestSeconds)*1000); e != nil {
		return e
	}
	if !v.Enabled {
		if _, e = tx.ExecContext(ctx, "UPDATE probe_jobs SET token='',lease_until=0,state='queued' WHERE token!=''"); e != nil {
			return e
		}
	}
	return tx.Commit()
}
func validDigest(s string) bool { b, e := hex.DecodeString(s); return e == nil && len(b) == 32 }

func (s *Store) ClaimProbe(ctx context.Context) (*probe.Job, error) {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	now := time.Now().UnixMilli()
	var settings probe.Settings
	if e = tx.QueryRowContext(ctx, "SELECT enabled,concurrency,timeout_seconds,retest_seconds,health_ttl_seconds FROM probe_settings WHERE id=1").Scan(&settings.Enabled, &settings.Concurrency, &settings.TimeoutSeconds, &settings.RetestSeconds, &settings.HealthTTLSeconds); e != nil {
		return nil, e
	}
	if !settings.Enabled {
		return nil, nil
	}
	var active int
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM probe_jobs WHERE token!='' AND lease_until>?", now).Scan(&active); e != nil {
		return nil, e
	}
	capacity := settings.Concurrency
	if capacity > 8 {
		capacity = 8
	}
	if active >= capacity {
		return nil, nil
	}
	j := &probe.Job{Settings: settings}
	var size int64
	prioritizeRetest := s.probeTurn.Add(1)%4 == 0
	e = tx.QueryRowContext(ctx, `SELECT b.hash,b.kind,b.revision,b.size FROM bank_samples b JOIN probe_jobs j ON j.hash=b.hash WHERE b.enabled=1 AND b.deleted=0 AND (j.token='' OR j.lease_until<=?) AND (j.next_run<=? OR j.version!=?) ORDER BY CASE WHEN ((j.version=? AND j.next_run>0)=?) THEN 0 ELSE 1 END,j.next_run,b.created LIMIT 1`, now, now, probe.Version, probe.Version, prioritizeRetest).Scan(&j.Hash, &j.Kind, &j.Revision, &size)
	if errors.Is(e, sql.ErrNoRows) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	j.Token = randomToken()
	_, e = tx.ExecContext(ctx, "UPDATE probe_jobs SET token=?,lease_until=?,revision=?,state='running',version=? WHERE hash=?", j.Token, now+120000, j.Revision, probe.Version, j.Hash)
	if e != nil {
		return nil, e
	}
	if e = tx.Commit(); e != nil {
		return nil, e
	}
	if size > repair.MaxBytes {
		j.Raw = nil
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
	res, e := s.db.ExecContext(ctx, `UPDATE probe_jobs SET lease_until=? WHERE hash=? AND token=? AND revision=? AND EXISTS(SELECT 1 FROM bank_samples b WHERE b.hash=probe_jobs.hash AND b.enabled=1 AND b.deleted=0 AND b.revision=?)`, time.Now().Add(2*time.Minute).UnixMilli(), j.Hash, j.Token, j.Revision, j.Revision)
	if e != nil {
		return e
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return ErrConflict
	}
	return nil
}
func safeCode(s string) bool {
	if len(s) > 128 || s == "" {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("._-", r)) {
			return false
		}
	}
	return true
}
func (s *Store) CompleteProbe(ctx context.Context, r probe.Result) error {
	if !safeCode(r.Rule) || !validDigest(r.Hash) || r.Version != probe.Version || r.CoreSHA256 != probe.CoreSHA256 || !safeCode(r.Code) || len(r.Attempts) > 12 || !validDigest(r.Model) {
		return ErrInvalid
	}
	if r.Core != "passed" && r.Core != "failed" && r.Core != "blocked" && r.Core != "not_run" {
		return ErrInvalid
	}
	if r.Connection != "passed" && r.Connection != "partial" && r.Connection != "failed" && r.Connection != "not_run" {
		return ErrInvalid
	}
	if r.Connection == "passed" && (r.Core != "passed" || r.EndpointsPassed != 2 || len(r.Candidate) == 0 || !validDigest(r.RuntimeSHA256)) {
		return ErrInvalid
	}
	var ref blob.Ref
	var e error
	if len(r.Candidate) > 0 {
		if len(r.Candidate) > repair.MaxBytes || repair.Digest(r.Candidate) != r.SHA256 || !json.Valid(r.Candidate) {
			return ErrInvalid
		}
		ref, e = s.vault.Put(ctx, bytes.NewReader(r.Candidate))
		if e != nil {
			return e
		}
		defer s.vault.Release(ref.Hash)
	}
	// Reports cannot contain credential-bearing arbitrary diagnostics.
	for _, a := range r.Attempts {
		if !safeCode(a.Code) || !safeCode(a.Rule) || !validDigest(a.SHA256) {
			return ErrInvalid
		}
	}
	report, _ := json.Marshal(r.Attempts)
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var kind string
	var failures int
	e = tx.QueryRowContext(ctx, `SELECT b.kind,j.failures FROM bank_samples b JOIN probe_jobs j ON j.hash=b.hash WHERE b.hash=? AND b.revision=? AND b.enabled=1 AND b.deleted=0 AND j.token=? AND j.revision=? AND j.lease_until>?`, r.Hash, r.Revision, r.Token, r.Revision, time.Now().UnixMilli()).Scan(&kind, &failures)
	if errors.Is(e, sql.ErrNoRows) {
		return ErrConflict
	}
	if e != nil {
		return e
	}
	var retest, ttl int64
	if e = tx.QueryRowContext(ctx, "SELECT retest_seconds,health_ttl_seconds FROM probe_settings WHERE id=1").Scan(&retest, &ttl); e != nil {
		return e
	}
	now := time.Now().UnixMilli()
	valid := int64(0)
	state := "failed"
	if r.Connection == "passed" {
		valid = now + ttl*1000
		state = "verified"
		failures = 0
	} else {
		failures++
		if r.Connection == "partial" {
			state = "partial"
		}
		if r.Connection == "not_run" {
			state = "blocked"
		}
	}
	delay := retest
	if state != "verified" {
		delay = 30 << min(failures, 6)
		if delay > retest {
			delay = retest
		}
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO probe_results VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(hash) DO UPDATE SET candidate_hash=excluded.candidate_hash,candidate_size=excluded.candidate_size,rule=excluded.rule,core=excluded.core,connection=excluded.connection,android=excluded.android,code=excluded.code,tested=excluded.tested,valid_until=excluded.valid_until,runtime_hash=excluded.runtime_hash,version=excluded.version,model=excluded.model,report=excluded.report`, r.Hash, ref.Hash, ref.Size, r.Rule, r.Core, r.Connection, "not_run", r.Code, now, valid, r.RuntimeSHA256, r.Version, r.Model, string(report))
	if e != nil {
		return e
	}
	_, e = tx.ExecContext(ctx, "UPDATE probe_jobs SET state=?,token='',lease_until=0,next_run=?,failures=?,version=? WHERE hash=?", state, now+delay*1000, failures, r.Version, r.Hash)
	if e != nil {
		return e
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO probe_models VALUES(?,?,1,?,?,?, ?,?) ON CONFLICT(model) DO UPDATE SET observed=observed+1,passed=passed+excluded.passed,failed=failed+excluded.failed,last_seen=excluded.last_seen,version=excluded.version,diagnosis=excluded.diagnosis`, r.Model, kind, boolInt(r.Connection == "passed"), boolInt(r.Connection != "passed"), now, r.Version, r.Code)
	if e != nil {
		return e
	}
	if ref.Hash != "" {
		_, e = tx.ExecContext(ctx, `INSERT INTO bank_variants VALUES(?,?,?,?,?,?,?) ON CONFLICT(hash,candidate_hash) DO UPDATE SET last_tested=excluded.last_tested,last_status=excluded.last_status`, r.Hash, ref.Hash, ref.Size, r.Rule, now, now, r.Connection)
		if e != nil {
			return e
		}
	}
	for _, a := range r.Attempts {
		if a.Connection == "not_run" {
			continue
		}
		_, e = tx.ExecContext(ctx, `INSERT INTO probe_rule_evidence VALUES(?,?,?,?,?,?) ON CONFLICT(model,rule,version) DO UPDATE SET passed=passed+excluded.passed,failed=failed+excluded.failed,last_tested=excluded.last_tested`, r.Model, a.Rule, r.Version, boolInt(a.Connection == "passed"), boolInt(a.Connection != "passed"), now)
		if e != nil {
			return e
		}
	}
	return tx.Commit()
}
func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

type BankEntry struct {
	Hash          string `json:"hash"`
	Kind          string `json:"kind"`
	Name          string `json:"name"`
	Size          int64  `json:"size"`
	Created       int64  `json:"created"`
	LastSeen      int64  `json:"last_seen"`
	Enabled       bool   `json:"enabled"`
	Revision      int64  `json:"revision"`
	State         string `json:"state"`
	Code          string `json:"code"`
	Core          string `json:"core"`
	Connection    string `json:"connection"`
	Android       string `json:"android"`
	Tested        int64  `json:"tested"`
	ValidUntil    int64  `json:"valid_until"`
	Rule          string `json:"rule"`
	CandidateHash string `json:"candidate_hash"`
	Model         string `json:"model"`
}

const bankColumns = `b.hash,b.kind,b.name,b.size,b.created,b.last_seen,b.enabled,b.revision,coalesce(j.state,'queued'),coalesce(r.code,''),coalesce(r.core,'not_run'),coalesce(r.connection,'not_run'),coalesce(r.android,'not_run'),coalesce(r.tested,0),coalesce(r.valid_until,0),coalesce(r.rule,''),coalesce(r.candidate_hash,''),coalesce(r.model,'')`
const bankJoin = ` FROM bank_samples b LEFT JOIN probe_jobs j ON j.hash=b.hash LEFT JOIN probe_results r ON r.hash=b.hash `

func scanBank(row scanner) (b BankEntry, e error) {
	e = row.Scan(&b.Hash, &b.Kind, &b.Name, &b.Size, &b.Created, &b.LastSeen, &b.Enabled, &b.Revision, &b.State, &b.Code, &b.Core, &b.Connection, &b.Android, &b.Tested, &b.ValidUntil, &b.Rule, &b.CandidateHash, &b.Model)
	return
}
func (s *Store) Bank(ctx context.Context, kind, state string, page int) ([]BankEntry, error) {
	args := []any{}
	q := "SELECT " + bankColumns + bankJoin + "WHERE b.deleted=0 "
	if kind != "" {
		q += "AND b.kind=? "
		args = append(args, kind)
	}
	if state == "verified" {
		q += "AND b.enabled=1 AND r.connection='passed' AND r.valid_until>? "
		args = append(args, time.Now().UnixMilli())
	} else if state != "" {
		q += "AND j.state=? "
		args = append(args, state)
	}
	q += "ORDER BY b.created DESC,b.hash LIMIT 100 OFFSET ?"
	args = append(args, max(0, page)*100)
	rows, e := s.db.QueryContext(ctx, q, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []BankEntry{}
	for rows.Next() {
		v, e := scanBank(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) BankOne(ctx context.Context, hash string) (BankEntry, error) {
	v, e := scanBank(s.db.QueryRowContext(ctx, "SELECT "+bankColumns+bankJoin+"WHERE b.hash=? AND b.deleted=0", hash))
	if errors.Is(e, sql.ErrNoRows) {
		e = ErrNotFound
	}
	return v, e
}
func (s *Store) BankStats(ctx context.Context) (map[string]int64, error) {
	out := map[string]int64{}
	var total, verified, running, partial, blocked, models int64
	e := s.db.QueryRowContext(ctx, `SELECT count(*),coalesce(sum(b.enabled=1 AND r.connection='passed' AND r.valid_until>?),0),coalesce(sum(j.state='running' AND j.lease_until>?),0),coalesce(sum(j.state='partial'),0),coalesce(sum(j.state IN ('blocked','failed')),0)`+bankJoin+`WHERE b.deleted=0`, time.Now().UnixMilli(), time.Now().UnixMilli()).Scan(&total, &verified, &running, &partial, &blocked)
	if e != nil {
		return nil, e
	}
	e = s.db.QueryRowContext(ctx, "SELECT count(*) FROM probe_models").Scan(&models)
	out["total"] = total
	out["verified"] = verified
	out["running"] = running
	out["partial"] = partial
	out["blocked"] = blocked
	out["models"] = models
	return out, e
}
func (s *Store) ManageBank(ctx context.Context, b ConfigBatch) (int64, error) {
	if b.Action != "delete" && b.Action != "disable" && b.Action != "enable" && b.Action != "retest" {
		return 0, ErrInvalid
	}
	if b.All && len(b.IDs) > 0 || !b.All && len(b.IDs) == 0 {
		return 0, ErrInvalid
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return 0, e
	}
	defer tx.Rollback()
	ids := b.IDs
	if b.All {
		rows, e := tx.QueryContext(ctx, "SELECT hash FROM bank_samples WHERE deleted=0")
		if e != nil {
			return 0, e
		}
		for rows.Next() {
			var id string
			rows.Scan(&id)
			ids = append(ids, id)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return 0, e
		}
	}
	n := int64(0)
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		var res sql.Result
		if b.Action == "delete" {
			res, e = tx.ExecContext(ctx, "UPDATE bank_samples SET deleted=1,enabled=0,revision=revision+1 WHERE hash=? AND deleted=0", id)
		} else if b.Action != "retest" {
			res, e = tx.ExecContext(ctx, "UPDATE bank_samples SET enabled=?,revision=revision+1 WHERE hash=? AND deleted=0", b.Action == "enable", id)
		} else {
			res, e = tx.ExecContext(ctx, "UPDATE bank_samples SET revision=revision+1 WHERE hash=? AND deleted=0 AND enabled=1", id)
		}
		if e != nil {
			return 0, e
		}
		c, _ := res.RowsAffected()
		n += c
		if c == 0 {
			continue
		}
		if _, e = tx.ExecContext(ctx, "DELETE FROM probe_jobs WHERE hash=?", id); e != nil {
			return 0, e
		}
		if b.Action == "delete" {
			if _, e = tx.ExecContext(ctx, "DELETE FROM bank_variants WHERE hash=?", id); e != nil {
				return 0, e
			}
			_, e = tx.ExecContext(ctx, "DELETE FROM probe_results WHERE hash=?", id)
		} else {
			_, e = tx.ExecContext(ctx, "UPDATE probe_results SET valid_until=0 WHERE hash=?", id)
		}
		if e != nil {
			return 0, e
		}
		if _, e = tx.ExecContext(ctx, "INSERT INTO probe_jobs(hash,revision) SELECT hash,revision FROM bank_samples WHERE hash=? AND deleted=0 AND enabled=1", id); e != nil {
			return 0, e
		}
	}
	return n, tx.Commit()
}
func (s *Store) ProbeWorker(ctx context.Context) (probe.WorkerStatus, error) {
	var v probe.WorkerStatus
	e := s.db.QueryRowContext(ctx, "SELECT status,code,version,active,heartbeat FROM probe_worker WHERE id=1").Scan(&v.Status, &v.Code, &v.Version, &v.Active, &v.Heartbeat)
	return v, e
}
func (s *Store) SetProbeWorker(ctx context.Context, v probe.WorkerStatus) error {
	if !safeCode(v.Status) || v.Code != "" && !safeCode(v.Code) {
		return ErrInvalid
	}
	_, e := s.db.ExecContext(ctx, "UPDATE probe_worker SET status=?,code=?,version=?,active=?,heartbeat=? WHERE id=1", v.Status, v.Code, v.Version, v.Active, time.Now().UnixMilli())
	return e
}
