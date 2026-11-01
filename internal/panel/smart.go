package panel

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/rezajafari0970/Config_manager/internal/convert"
	"github.com/rezajafari0970/Config_manager/internal/probe"
	"github.com/rezajafari0970/Config_manager/internal/repair"
	"github.com/rezajafari0970/Config_manager/internal/smart"
)

type SmartSettings struct {
	Enabled  bool  `json:"enabled"`
	Count    int   `json:"count"`
	Revision int64 `json:"revision"`
}
type SmartEntry struct {
	ID        string `json:"id"`
	Hash      string `json:"hash"`
	Count     int    `json:"count"`
	Sources   int    `json:"sources"`
	State     string `json:"state"`
	Code      string `json:"code"`
	Created   int64  `json:"created"`
	Expires   int64  `json:"expires"`
	Available bool   `json:"available"`
}

func (s *Store) migrateV9() error {
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	_, e = tx.Exec(`
 CREATE TABLE IF NOT EXISTS smart_settings(id INTEGER PRIMARY KEY CHECK(id=1),enabled INTEGER NOT NULL,member_count INTEGER NOT NULL,revision INTEGER NOT NULL,sequence INTEGER NOT NULL);
 INSERT OR IGNORE INTO smart_settings VALUES(1,1,10,1,0);
 CREATE TABLE IF NOT EXISTS smart_heads(id TEXT PRIMARY KEY REFERENCES artifacts(id) ON DELETE CASCADE,hash TEXT NOT NULL DEFAULT '',member_count INTEGER NOT NULL DEFAULT 0,created INTEGER NOT NULL,expires INTEGER NOT NULL DEFAULT 0,revision INTEGER NOT NULL DEFAULT 0,version TEXT NOT NULL DEFAULT '',state TEXT NOT NULL DEFAULT 'queued',code TEXT NOT NULL DEFAULT '',token TEXT NOT NULL DEFAULT '',lease_until INTEGER NOT NULL DEFAULT 0,next_run INTEGER NOT NULL DEFAULT 0,sequence INTEGER NOT NULL DEFAULT 0);
 CREATE INDEX IF NOT EXISTS smart_due ON smart_heads(state,next_run,lease_until);
 CREATE TABLE IF NOT EXISTS smart_members(head TEXT NOT NULL REFERENCES smart_heads(id) ON DELETE CASCADE,position INTEGER NOT NULL,id TEXT NOT NULL REFERENCES artifacts(id) ON DELETE CASCADE,source TEXT NOT NULL REFERENCES sources(id) ON DELETE CASCADE,raw_hash TEXT NOT NULL,hash TEXT NOT NULL,PRIMARY KEY(head,position),UNIQUE(head,id));
 CREATE INDEX IF NOT EXISTS smart_members_source ON smart_members(source,id,raw_hash,hash);
 CREATE TABLE IF NOT EXISTS smart_usage(source TEXT NOT NULL REFERENCES sources(id) ON DELETE CASCADE,id TEXT NOT NULL REFERENCES artifacts(id) ON DELETE CASCADE,uses INTEGER NOT NULL,last_used INTEGER NOT NULL,PRIMARY KEY(source,id));
 CREATE TABLE IF NOT EXISTS smart_source_usage(source TEXT PRIMARY KEY REFERENCES sources(id) ON DELETE CASCADE,uses INTEGER NOT NULL,last_used INTEGER NOT NULL);
 CREATE TABLE IF NOT EXISTS smart_rotation_state(id INTEGER PRIMARY KEY CHECK(id=1),cycle INTEGER NOT NULL);
 INSERT OR IGNORE INTO smart_rotation_state VALUES(1,0);
 CREATE TABLE IF NOT EXISTS smart_rotation_queue(position INTEGER PRIMARY KEY,head TEXT NOT NULL UNIQUE REFERENCES smart_heads(id) ON DELETE CASCADE);
 CREATE TABLE IF NOT EXISTS smart_worker(id INTEGER PRIMARY KEY CHECK(id=1),status TEXT NOT NULL,code TEXT NOT NULL,version TEXT NOT NULL,active INTEGER NOT NULL,heartbeat INTEGER NOT NULL);
 INSERT OR IGNORE INTO smart_worker VALUES(1,'offline','','',0,0);
 DROP VIEW IF EXISTS smart_ready;
 DROP VIEW IF EXISTS smart_owned_members;
 DROP VIEW IF EXISTS smart_offers;
 DROP VIEW IF EXISTS smart_inputs;
 CREATE VIEW smart_inputs AS SELECT m.source_id AS source,m.artifact_id AS id,m.raw_hash,r.candidate_hash AS hash,min(a.expires,r.valid_until) AS expires
 FROM memberships m JOIN artifacts a ON a.id=m.artifact_id JOIN sources s ON s.id=m.source_id JOIN conversion_results r ON r.hash=m.raw_hash
 WHERE s.enabled=1 AND a.expires>CAST(unixepoch('subsec')*1000 AS INTEGER)
 AND NOT EXISTS(SELECT 1 FROM config_controls c WHERE c.id=a.id AND c.expires>CAST(unixepoch('subsec')*1000 AS INTEGER))
 AND r.core='passed' AND r.candidate_hash!='' AND r.version='` + convert.Version + `'
 AND r.connection='passed' AND r.valid_until>CAST(unixepoch('subsec')*1000 AS INTEGER);
 CREATE VIEW smart_offers AS SELECT source,id,raw_hash,hash,expires FROM (
 SELECT i.*,row_number() OVER(PARTITION BY i.source,i.id ORDER BY (r.connection='passed') DESC,r.tested DESC,i.raw_hash) AS choice FROM smart_inputs i JOIN conversion_results r ON r.hash=i.raw_hash) WHERE choice=1;
 CREATE VIEW smart_owned_members AS SELECT m.head,m.position,min(i.expires) AS expires FROM smart_members m JOIN smart_inputs i ON i.source=m.source AND i.id=m.id AND i.raw_hash=m.raw_hash AND i.hash=m.hash GROUP BY m.head,m.position;
 CREATE VIEW smart_ready AS WITH offers AS MATERIALIZED(SELECT * FROM smart_offers), source_ids AS MATERIALIZED(SELECT DISTINCT source FROM offers) SELECT h.* FROM smart_heads h JOIN smart_settings s ON s.id=1 JOIN settings ttl ON ttl.id=1
 WHERE s.enabled=1 AND h.state='ready' AND h.version='` + smart.Version + `' AND h.revision=s.revision
 AND h.hash!='' AND h.expires>CAST(unixepoch('subsec')*1000 AS INTEGER) AND h.created+ttl.retention_seconds*1000>CAST(unixepoch('subsec')*1000 AS INTEGER)
 AND h.member_count>0 AND (SELECT count(*) FROM smart_owned_members m WHERE m.head=h.id)=h.member_count
 AND (h.member_count>=s.member_count OR NOT EXISTS(
 SELECT 1 FROM offers i WHERE NOT EXISTS(SELECT 1 FROM smart_members m WHERE m.head=h.id AND (m.id=i.id OR m.hash=i.hash))))
 AND (s.member_count<(SELECT count(*) FROM source_ids) OR
 (SELECT count(*) FROM (
 SELECT i.source FROM smart_members m JOIN offers i ON i.id=m.id WHERE m.head=h.id
 UNION SELECT i.source FROM smart_members m JOIN offers i ON i.hash=m.hash WHERE m.head=h.id
 ))=(SELECT count(*) FROM source_ids));
 PRAGMA user_version=9;`)
	if e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Store) SmartSettings(ctx context.Context) (SmartSettings, error) {
	var v SmartSettings
	e := s.db.QueryRowContext(ctx, "SELECT enabled,member_count,revision FROM smart_settings WHERE id=1").Scan(&v.Enabled, &v.Count, &v.Revision)
	return v, e
}
func (s *Store) SaveSmartSettings(ctx context.Context, v SmartSettings) error {
	if v.Count < 1 {
		return ErrInvalid
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	res, e := tx.ExecContext(ctx, "UPDATE smart_settings SET enabled=?,member_count=?,revision=revision+1 WHERE id=1 AND revision=?", v.Enabled, v.Count, v.Revision)
	if e != nil {
		return e
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return ErrConflict
	}
	// A changed envelope invalidates only smart claims and cached profiles.
	if _, e = tx.ExecContext(ctx, "UPDATE smart_heads SET state='queued',token='',lease_until=0,next_run=0"); e != nil {
		return e
	}
	return tx.Commit()
}
func smartInventory(ctx context.Context, db conversionQuerier) ([]smart.Member, error) {
	rows, e := db.QueryContext(ctx, "SELECT id,source,raw_hash,hash,expires FROM smart_offers ORDER BY source,id")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []smart.Member{}
	for rows.Next() {
		var m smart.Member
		if e = rows.Scan(&m.ID, &m.Source, &m.RawHash, &m.Hash, &m.Expires); e != nil {
			return nil, e
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
func smartLedger(ctx context.Context, tx *sql.Tx) (smart.Ledger, error) {
	v := smart.Ledger{Sources: map[string]smart.Usage{}, Members: map[string]smart.Usage{}}
	rows, e := tx.QueryContext(ctx, "SELECT source,uses,last_used FROM smart_source_usage")
	if e != nil {
		return v, e
	}
	for rows.Next() {
		var source string
		var u smart.Usage
		if e = rows.Scan(&source, &u.Uses, &u.Last); e != nil {
			rows.Close()
			return v, e
		}
		v.Sources[source] = u
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return v, e
	}
	rows, e = tx.QueryContext(ctx, "SELECT source,id,uses,last_used FROM smart_usage")
	if e != nil {
		return v, e
	}
	defer rows.Close()
	for rows.Next() {
		var m smart.Member
		var u smart.Usage
		if e = rows.Scan(&m.Source, &m.ID, &u.Uses, &u.Last); e != nil {
			return v, e
		}
		v.Members[smart.Key(m)] = u
	}
	return v, rows.Err()
}
func (s *Store) ClaimSmart(ctx context.Context) (*probe.Job, error) {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, "UPDATE smart_settings SET sequence=sequence WHERE id=1"); e != nil {
		return nil, e
	}
	var enabled bool
	var count int
	var revision, sequence int64
	if e = tx.QueryRowContext(ctx, "SELECT enabled,member_count,revision,sequence FROM smart_settings WHERE id=1").Scan(&enabled, &count, &revision, &sequence); e != nil {
		return nil, e
	}
	now := time.Now().UnixMilli()
	// Ownership can disappear while a blob is composed or a helper is running.
	// Revoke invalid leases before counting capacity; they must not occupy both
	// independent slots until their crash-recovery timeout.
	if _, e = tx.ExecContext(ctx, `UPDATE smart_heads SET state='queued',token='',lease_until=0,next_run=0
 WHERE token!='' AND (version!=? OR revision!=? OR expires<=? OR
 (SELECT count(*) FROM smart_owned_members m WHERE m.head=smart_heads.id)!=member_count)`, smart.Version, revision, now); e != nil {
		return nil, e
	}
	var active int
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM smart_heads WHERE token!='' AND lease_until>?", now).Scan(&active); e != nil {
		return nil, e
	}
	if !enabled || active >= 2 {
		return nil, tx.Commit()
	}
	pool, e := smartInventory(ctx, tx)
	if e != nil {
		return nil, e
	}
	// One cached smart per unique first member. This makes complete cached rounds
	// cover every first position exactly once, even with uneven source sizes.
	ids := map[string]bool{}
	hashes := map[string]string{}
	for _, m := range pool {
		if old := hashes[m.Hash]; old != "" && old != m.ID {
			continue
		}
		hashes[m.Hash] = m.ID
		ids[m.ID] = true
	}
	// A byte-identical candidate belongs to one canonical head in this snapshot.
	// Filter follower/leader offers by the same ownership, so a secondary raw
	// representation cannot reintroduce an already-deduplicated first candidate.
	uniquePool := make([]smart.Member, 0, len(pool))
	for _, m := range pool {
		if hashes[m.Hash] == m.ID {
			uniquePool = append(uniquePool, m)
		}
	}
	pool = uniquePool
	if _, e = tx.ExecContext(ctx, "CREATE TEMP TABLE IF NOT EXISTS smart_ids(id TEXT PRIMARY KEY);DELETE FROM smart_ids"); e != nil {
		return nil, e
	}
	for id := range ids {
		if _, e = tx.ExecContext(ctx, "INSERT INTO smart_ids VALUES(?)", id); e != nil {
			return nil, e
		}
	}
	if _, e = tx.ExecContext(ctx, "DELETE FROM smart_heads WHERE id NOT IN(SELECT id FROM smart_ids)"); e != nil {
		return nil, e
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO smart_heads(id,created) SELECT id,? FROM smart_ids WHERE true ON CONFLICT(id) DO NOTHING", now); e != nil {
		return nil, e
	}
	// Invalidated ownership/expiry/proof is hidden by the view immediately. Rebuild
	// independently; a pending job never retains an obsolete completion token.
	_, e = tx.ExecContext(ctx, "UPDATE smart_heads SET state='queued',next_run=0 WHERE state='ready' AND id NOT IN(SELECT id FROM smart_ready)")
	if e != nil {
		return nil, e
	}
	var head string
	e = tx.QueryRowContext(ctx, "SELECT id FROM smart_heads WHERE (token='' OR lease_until<=?) AND (state!='ready' OR revision!=? OR version!=?) AND next_run<=? ORDER BY sequence,created,id LIMIT 1", now, revision, smart.Version, now).Scan(&head)
	if errors.Is(e, sql.ErrNoRows) {
		return nil, tx.Commit()
	}
	if e != nil {
		return nil, e
	}
	ledger, e := smartLedger(ctx, tx)
	if e != nil {
		return nil, e
	}
	members, e := smart.Select(pool, head, count, ledger)
	if e != nil {
		return nil, e
	}
	sequence++
	token := randomToken()
	var ttl int64
	if e = tx.QueryRowContext(ctx, "SELECT retention_seconds FROM settings WHERE id=1").Scan(&ttl); e != nil {
		return nil, e
	}
	expires := now + ttl*1000
	for _, m := range members {
		expires = min(expires, m.Expires)
	}
	if _, e = tx.ExecContext(ctx, "UPDATE smart_settings SET sequence=? WHERE id=1", sequence); e != nil {
		return nil, e
	}
	if _, e = tx.ExecContext(ctx, "DELETE FROM smart_members WHERE head=?", head); e != nil {
		return nil, e
	}
	for i, m := range members {
		if _, e = tx.ExecContext(ctx, "INSERT INTO smart_members VALUES(?,?,?,?,?,?)", head, i, m.ID, m.Source, m.RawHash, m.Hash); e != nil {
			return nil, e
		}
		if _, e = tx.ExecContext(ctx, "INSERT INTO smart_usage VALUES(?,?,1,?) ON CONFLICT(source,id) DO UPDATE SET uses=uses+1,last_used=excluded.last_used", m.Source, m.ID, sequence); e != nil {
			return nil, e
		}
		if _, e = tx.ExecContext(ctx, "INSERT INTO smart_source_usage VALUES(?,1,?) ON CONFLICT(source) DO UPDATE SET uses=uses+1,last_used=excluded.last_used", m.Source, sequence); e != nil {
			return nil, e
		}
	}
	_, e = tx.ExecContext(ctx, "UPDATE smart_heads SET hash='',member_count=?,created=?,expires=?,revision=?,version=?,state='running',code='',token=?,lease_until=?,next_run=0,sequence=? WHERE id=?", len(members), now, expires, revision, smart.Version, token, now+120000, sequence, head)
	if e != nil {
		return nil, e
	}
	if e = tx.Commit(); e != nil {
		return nil, e
	}
	j := &probe.Job{Hash: head, Token: token, Revision: revision, Kind: "smart", Settings: probe.Settings{Enabled: true, Concurrency: 2, TimeoutSeconds: 10}}
	for i := range members {
		members[i].Body, e = s.conversionBytes(Conversion{Available: true, CandidateHash: members[i].Hash})
		if e != nil {
			j.Issue = "smart_member_blob_unavailable"
			return j, nil
		}
	}
	j.Raw, e = smart.Compose(members)
	if e != nil {
		j.Issue = e.Error()
		j.Raw = nil
		return j, nil
	}
	ref, e := s.vault.Put(ctx, bytes.NewReader(j.Raw))
	if e != nil {
		return nil, e
	}
	defer s.vault.Release(ref.Hash)
	res, e := s.db.ExecContext(ctx, `UPDATE smart_heads SET hash=? WHERE id=? AND token=? AND revision=? AND lease_until>? AND (SELECT count(*) FROM smart_owned_members m WHERE m.head=smart_heads.id)=member_count`, ref.Hash, head, token, revision, time.Now().UnixMilli())
	if e != nil {
		return nil, e
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return nil, ErrConflict
	}
	return j, nil
}
func (s *Store) RenewSmart(ctx context.Context, j probe.Job) error {
	res, e := s.db.ExecContext(ctx, `UPDATE smart_heads SET lease_until=? WHERE id=? AND token=? AND revision=? AND lease_until>? AND revision=(SELECT revision FROM smart_settings WHERE id=1 AND enabled=1) AND (SELECT count(*) FROM smart_owned_members m WHERE m.head=smart_heads.id)=member_count`, time.Now().Add(2*time.Minute).UnixMilli(), j.Hash, j.Token, j.Revision, time.Now().UnixMilli())
	if e != nil {
		return e
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return ErrConflict
	}
	return nil
}
func (s *Store) CompleteSmart(ctx context.Context, r probe.Result) error {
	if r.Version != smart.Version || r.CoreSHA256 != probe.CoreSHA256 || !validDigest(r.Hash) || !safeCode(r.Code) || r.Connection != "not_run" {
		return ErrInvalid
	}
	if r.Core != "passed" && r.Core != "failed" && r.Core != "blocked" && r.Core != "not_run" {
		return ErrInvalid
	}
	if r.Core == "passed" && (repair.Digest(r.Candidate) != r.SHA256 || len(r.Candidate) > repair.MaxBytes || !json.Valid(r.Candidate)) {
		return ErrInvalid
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var hash string
	var members, owned int
	var expires int64
	now := time.Now().UnixMilli()
	e = tx.QueryRowContext(ctx, `SELECT hash,member_count,(SELECT count(*) FROM smart_owned_members m WHERE m.head=h.id),expires FROM smart_heads h WHERE id=? AND token=? AND revision=? AND lease_until>? AND revision=(SELECT revision FROM smart_settings WHERE id=1 AND enabled=1)`, r.Hash, r.Token, r.Revision, now).Scan(&hash, &members, &owned, &expires)
	if errors.Is(e, sql.ErrNoRows) {
		return ErrConflict
	}
	if e != nil {
		return e
	}
	if members != owned || expires <= now {
		return ErrConflict
	}
	if r.Core == "passed" && hash != r.SHA256 {
		return ErrInvalid
	}
	state := "blocked"
	next := now + 60000
	if r.Core == "passed" {
		state = "ready"
		next = 0
	}
	_, e = tx.ExecContext(ctx, "UPDATE smart_heads SET state=?,code=?,token='',lease_until=0,next_run=? WHERE id=?", state, r.Code, next, r.Hash)
	if e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Store) SmartWorker(ctx context.Context) (probe.WorkerStatus, error) {
	var v probe.WorkerStatus
	e := s.db.QueryRowContext(ctx, "SELECT status,code,version,active,heartbeat FROM smart_worker WHERE id=1").Scan(&v.Status, &v.Code, &v.Version, &v.Active, &v.Heartbeat)
	return v, e
}
func (s *Store) SetSmartWorker(ctx context.Context, v probe.WorkerStatus) error {
	if !safeCode(v.Status) || v.Code != "" && !safeCode(v.Code) {
		return ErrInvalid
	}
	_, e := s.db.ExecContext(ctx, "UPDATE smart_worker SET status=?,code=?,version=?,active=?,heartbeat=? WHERE id=1", v.Status, v.Code, v.Version, v.Active, time.Now().UnixMilli())
	return e
}
func querySmart(ctx context.Context, db conversionQuerier, all bool, page int) ([]SmartEntry, error) {
	query := `SELECT h.id,h.hash,h.member_count,(SELECT count(DISTINCT source) FROM smart_members m WHERE m.head=h.id),h.state,h.code,h.created,h.expires,EXISTS(SELECT 1 FROM smart_ready r WHERE r.id=h.id) FROM smart_heads h ORDER BY CASE h.state WHEN 'ready' THEN 0 WHEN 'running' THEN 1 WHEN 'queued' THEN 2 ELSE 3 END,h.sequence,h.id`
	if !all {
		query += " LIMIT 100 OFFSET ?"
	}
	args := []any{}
	if !all {
		args = append(args, max(0, page)*100)
	}
	rows, e := db.QueryContext(ctx, query, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []SmartEntry{}
	for rows.Next() {
		var v SmartEntry
		if e = rows.Scan(&v.ID, &v.Hash, &v.Count, &v.Sources, &v.State, &v.Code, &v.Created, &v.Expires, &v.Available); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) smartBytes(ctx context.Context, v SmartEntry) ([]byte, error) {
	if !v.Available {
		return nil, ErrNotFound
	}
	body, e := s.vault.Open(v.Hash)
	if e != nil {
		return nil, e
	}
	defer body.Close()
	b, e := io.ReadAll(io.LimitReader(body, repair.MaxBytes+1))
	if e != nil {
		return nil, e
	}
	if len(b) > repair.MaxBytes || repair.Digest(b) != v.Hash || !json.Valid(b) {
		return nil, ErrInvalid
	}
	var present bool
	e = s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM smart_ready WHERE id=? AND hash=?)", v.ID, v.Hash).Scan(&present)
	if e != nil {
		return nil, e
	}
	if !present {
		return nil, ErrNotFound
	}
	return b, nil
}
func (s *Store) retrySmartBlob(ctx context.Context, v SmartEntry) {
	_, _ = s.db.ExecContext(ctx, "UPDATE smart_heads SET state='queued',next_run=0,token='',lease_until=0,code='smart_blob_unavailable' WHERE id=? AND hash=?", v.ID, v.Hash)
}
func (s *Store) smartNext(ctx context.Context) (*SmartEntry, []byte, int64, error) {
	unlock, e := s.lockRotation(ctx, "smart-output")
	if e != nil {
		return nil, nil, 0, e
	}
	defer unlock()
	for {
		if e = ctx.Err(); e != nil {
			return nil, nil, 0, e
		}
		tx, e := s.db.BeginTx(ctx, nil)
		if e != nil {
			return nil, nil, 0, e
		}
		if _, e = tx.ExecContext(ctx, "UPDATE smart_rotation_state SET cycle=cycle WHERE id=1"); e != nil {
			tx.Rollback()
			return nil, nil, 0, e
		}
		var cycle int64
		var queued int
		if e = tx.QueryRowContext(ctx, "SELECT cycle,(SELECT count(*) FROM smart_rotation_queue) FROM smart_rotation_state WHERE id=1").Scan(&cycle, &queued); e != nil {
			tx.Rollback()
			return nil, nil, 0, e
		}
		if queued == 0 {
			// Snapshot all current heads, including warming jobs: early-ready profiles
			// must not repeat while other first positions are still being built.
			if _, e = tx.ExecContext(ctx, "INSERT INTO smart_rotation_queue(position,head) SELECT row_number() OVER(ORDER BY sequence,id),id FROM smart_heads"); e != nil {
				tx.Rollback()
				return nil, nil, 0, e
			}
			if _, e = tx.ExecContext(ctx, "UPDATE smart_rotation_state SET cycle=cycle+1 WHERE id=1"); e != nil {
				tx.Rollback()
				return nil, nil, 0, e
			}
			cycle++
		}
		// Permanently failed entries do not prevent the finite round from completing.
		if _, e = tx.ExecContext(ctx, "DELETE FROM smart_rotation_queue WHERE head IN(SELECT id FROM smart_heads WHERE state='blocked')"); e != nil {
			tx.Rollback()
			return nil, nil, 0, e
		}
		var v SmartEntry
		e = tx.QueryRowContext(ctx, `SELECT h.id,h.hash,h.member_count,h.created,h.expires FROM smart_rotation_queue q JOIN smart_ready h ON h.id=q.head ORDER BY q.position LIMIT 1`).Scan(&v.ID, &v.Hash, &v.Count, &v.Created, &v.Expires)
		if errors.Is(e, sql.ErrNoRows) {
			tx.Commit()
			return nil, nil, cycle, nil
		}
		if e != nil {
			tx.Rollback()
			return nil, nil, 0, e
		}
		if e = tx.Commit(); e != nil {
			return nil, nil, 0, e
		}
		v.Available = true
		b, e := s.smartBytes(ctx, v)
		if e != nil {
			s.retrySmartBlob(ctx, v)
			continue
		}
		tx, e = s.db.BeginTx(ctx, nil)
		if e != nil {
			return nil, nil, 0, e
		}
		res, e := tx.ExecContext(ctx, `DELETE FROM smart_rotation_queue WHERE head=? AND EXISTS(SELECT 1 FROM smart_ready WHERE id=? AND hash=?) AND (SELECT cycle FROM smart_rotation_state WHERE id=1)=?`, v.ID, v.ID, v.Hash, cycle)
		if e != nil {
			tx.Rollback()
			return nil, nil, 0, e
		}
		n, _ := res.RowsAffected()
		if e = tx.Commit(); e != nil {
			return nil, nil, 0, e
		}
		if n == 1 {
			return &v, b, cycle, nil
		}
	}
}

// Store views make removals/disable/proof expiry visible immediately, even before
// the next builder sweep. GC keeps only transient generated profiles.
