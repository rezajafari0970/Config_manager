package panel

import (
	"context"
	"database/sql"
	"errors"
	"github.com/rezajafari0970/Config_manager/internal/blob"
	"github.com/rezajafari0970/Config_manager/internal/collect"
	"github.com/rezajafari0970/Config_manager/internal/schedule"
	"strings"
	"time"
)

func (s *Store) migrateV2() error {
	var version int
	if e := s.db.QueryRow("PRAGMA user_version").Scan(&version); e != nil {
		return e
	}
	if version >= 2 {
		return nil
	}
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	_, e = tx.Exec(`
ALTER TABLE sources ADD COLUMN state TEXT NOT NULL DEFAULT 'queued';
ALTER TABLE sources ADD COLUMN error_code TEXT NOT NULL DEFAULT '';
ALTER TABLE sources ADD COLUMN last_fetch INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sources ADD COLUMN last_success INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sources ADD COLUMN http_status INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sources ADD COLUMN duration_ms INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sources ADD COLUMN fetch_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sources ADD COLUMN duplicates INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sources ADD COLUMN generation INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sources ADD COLUMN lease_token TEXT NOT NULL DEFAULT '';
ALTER TABLE sources ADD COLUMN lease_until INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sources ADD COLUMN etag TEXT NOT NULL DEFAULT '';
ALTER TABLE sources ADD COLUMN modified TEXT NOT NULL DEFAULT '';
CREATE INDEX sources_due ON sources(enabled,next_run,lease_until);
CREATE TABLE settings(id INTEGER PRIMARY KEY CHECK(id=1),retention_seconds INTEGER NOT NULL,concurrency INTEGER NOT NULL,timeout_seconds INTEGER NOT NULL);
INSERT INTO settings VALUES(1,7200,16,30);
CREATE TABLE artifacts(id TEXT PRIMARY KEY,hash TEXT NOT NULL,kind TEXT NOT NULL,size INTEGER NOT NULL,created INTEGER NOT NULL,expires INTEGER NOT NULL);
CREATE INDEX artifacts_expiry ON artifacts(expires);
CREATE TABLE memberships(source_id TEXT NOT NULL REFERENCES sources(id) ON DELETE CASCADE,artifact_id TEXT NOT NULL REFERENCES artifacts(id) ON DELETE CASCADE,generation INTEGER NOT NULL,raw_hash TEXT NOT NULL,kind TEXT NOT NULL,size INTEGER NOT NULL,PRIMARY KEY(source_id,artifact_id));
CREATE INDEX memberships_artifact ON memberships(artifact_id);
CREATE TABLE responses(source_id TEXT PRIMARY KEY REFERENCES sources(id) ON DELETE CASCADE,hash TEXT NOT NULL,size INTEGER NOT NULL,created INTEGER NOT NULL,expires INTEGER NOT NULL);
CREATE INDEX responses_expiry ON responses(expires);
CREATE TABLE staged(source_id TEXT NOT NULL REFERENCES sources(id) ON DELETE CASCADE,token TEXT NOT NULL,identity TEXT NOT NULL,hash TEXT NOT NULL,kind TEXT NOT NULL,size INTEGER NOT NULL,PRIMARY KEY(token,identity));
CREATE INDEX staged_hash ON staged(hash);
DROP TABLE admin; DROP TABLE sessions; DROP TABLE attempts;
UPDATE sources SET next_run=0,state=CASE WHEN enabled=1 THEN 'queued' ELSE 'paused' END;
PRAGMA user_version=2;`)
	if e != nil {
		return e
	}
	return tx.Commit()
}

type scanner interface{ Scan(...any) error }

const baseColumns = "id,name,url,display_url,min,max,unit,enabled,revision,created,updated,next_run"
const sourceColumns = baseColumns + ",state,error_code,last_fetch,last_success,http_status,duration_ms,fetch_count,duplicates,timeout_seconds,content_state,valid_count,invalid_count,unknown_count,received_count,response_bytes,error_streak"

func scanSource(row scanner) (Source, []byte, error) {
	var s Source
	var sealed []byte
	var created, updated, next int64
	e := row.Scan(&s.ID, &s.Name, &sealed, &s.DisplayURL, &s.Min, &s.Max, &s.Unit, &s.Enabled, &s.Revision, &created, &updated, &next, &s.State, &s.ErrorCode, &s.LastFetch, &s.LastSuccess, &s.HTTPStatus, &s.DurationMS, &s.FetchCount, &s.DuplicateCount, &s.TimeoutSeconds, &s.ContentState, &s.ValidCount, &s.InvalidCount, &s.UnknownCount, &s.ReceivedCount, &s.ResponseBytes, &s.ErrorStreak, &s.ConfigCount, &s.HasResponse)
	s.CreatedAt = time.Unix(created, 0)
	s.UpdatedAt = time.Unix(updated, 0)
	s.NextRunAt = time.Unix(0, next)
	return s, sealed, e
}

const sourceQuery = "SELECT " + sourceColumns + ",(SELECT count(*) FROM memberships m JOIN artifacts a ON a.id=m.artifact_id WHERE m.source_id=sources.id AND a.expires>?),EXISTS(SELECT 1 FROM responses r WHERE r.source_id=sources.id AND r.expires>?) FROM sources "

func (s *Store) List(ctx context.Context) ([]Source, error) { return s.list(ctx, "", 0, false) }
func (s *Store) ListPage(ctx context.Context, search string, page int) ([]Source, error) {
	return s.list(ctx, search, page, true)
}
func (s *Store) list(ctx context.Context, search string, page int, paged bool) ([]Source, error) {
	now := time.Now().UnixMilli()
	args := []any{now, now}
	query := sourceQuery
	if search != "" {
		query += "WHERE name LIKE ? OR display_url LIKE ? "
		args = append(args, "%"+search+"%", "%"+search+"%")
	}
	query += "ORDER BY created DESC,id "
	if paged {
		if page < 0 {
			page = 0
		}
		query += "LIMIT 100 OFFSET ?"
		args = append(args, page*100)
	}
	rows, e := s.db.QueryContext(ctx, query, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Source{}
	for rows.Next() {
		v, _, e := scanSource(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) Get(ctx context.Context, id string) (Source, error) {
	now := time.Now().UnixMilli()
	v, sealed, e := scanSource(s.db.QueryRowContext(ctx, sourceQuery+"WHERE id=?", now, now, id))
	if errors.Is(e, sql.ErrNoRows) {
		return v, ErrNotFound
	}
	if e != nil {
		return v, e
	}
	b, e := s.open(id, sealed)
	v.URL = string(b)
	return v, e
}
func (s *Store) Save(ctx context.Context, id string, revision int64, input SourceInput) (string, error) {
	if input.Name == "" {
		input.Name = AutoName(input.URL)
	}
	display, e := validateSource(input)
	if e != nil {
		return "", e
	}
	now := time.Now()
	state := "queued"
	if !input.Enabled {
		state = "paused"
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return "", e
	}
	defer tx.Rollback()
	if id == "" {
		id = randomToken()
		_, e = tx.ExecContext(ctx, "INSERT INTO sources ("+baseColumns+",url_hash,state) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)", id, input.Name, s.seal(id, []byte(input.URL)), display, input.Min, input.Max, input.Unit, input.Enabled, 1, now.Unix(), now.Unix(), now.UnixNano(), s.digest("source-url", input.URL), state)
	} else {
		var oldHash string
		e = tx.QueryRowContext(ctx, "SELECT url_hash FROM sources WHERE id=? AND revision=?", id, revision).Scan(&oldHash)
		if errors.Is(e, sql.ErrNoRows) {
			return "", ErrConflict
		}
		if e != nil {
			return "", e
		}
		result, err := tx.ExecContext(ctx, "UPDATE sources SET name=?,url=?,url_hash=?,display_url=?,min=?,max=?,unit=?,enabled=?,revision=revision+1,updated=?,next_run=?,state=?,error_code='',lease_token='',lease_until=0 WHERE id=? AND revision=?", input.Name, s.seal(id, []byte(input.URL)), s.digest("source-url", input.URL), display, input.Min, input.Max, input.Unit, input.Enabled, now.Unix(), now.UnixNano(), state, id, revision)
		e = err
		if e == nil {
			n, _ := result.RowsAffected()
			if n != 1 {
				return "", ErrConflict
			}
		}
		if e == nil && oldHash != s.digest("source-url", input.URL) {
			if _, e = tx.ExecContext(ctx, "DELETE FROM memberships WHERE source_id=?", id); e != nil {
				return "", e
			}
			if _, e = tx.ExecContext(ctx, "DELETE FROM responses WHERE source_id=?", id); e != nil {
				return "", e
			}
			_, e = tx.ExecContext(ctx, "UPDATE sources SET etag='',modified='',last_fetch=0,last_success=0,http_status=0,duration_ms=0,duplicates=0,content_state='unknown',valid_count=0,invalid_count=0,unknown_count=0,received_count=0,response_bytes=0,error_streak=0 WHERE id=?", id)
		}
	}
	if e == nil {
		_, e = tx.ExecContext(ctx, "UPDATE sources SET timeout_seconds=? WHERE id=?", input.TimeoutSeconds, id)
	}
	if e != nil {
		if strings.Contains(e.Error(), "UNIQUE constraint") {
			return "", ErrConflict
		}
		return "", e
	}
	return id, tx.Commit()
}
func (s *Store) Delete(ctx context.Context, id string, revision int64) error {
	result, e := s.db.ExecContext(ctx, "DELETE FROM sources WHERE id=? AND revision=?", id, revision)
	if e != nil {
		return e
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return ErrConflict
	}
	return nil
}
func (s *Store) RunNow(ctx context.Context, id string) error {
	result, e := s.db.ExecContext(ctx, "UPDATE sources SET next_run=?,state=CASE WHEN lease_token='' THEN 'queued' ELSE state END WHERE id=? AND enabled=1", time.Now().UnixNano(), id)
	if e != nil {
		return e
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return ErrConflict
	}
	return nil
}
func (s *Store) Settings(ctx context.Context) (collect.Settings, error) {
	var v collect.Settings
	e := s.db.QueryRowContext(ctx, "SELECT retention_seconds,concurrency,timeout_seconds FROM settings WHERE id=1").Scan(&v.RetentionSeconds, &v.Concurrency, &v.TimeoutSeconds)
	return v, e
}
func (s *Store) SaveSettings(ctx context.Context, v collect.Settings) error {
	if v.RetentionSeconds <= 0 || v.Concurrency < 1 || v.TimeoutSeconds < 1 || v.RetentionSeconds > int64((1<<63-1)/int64(time.Second)) || v.TimeoutSeconds > int64((1<<63-1)/int64(time.Second))-30 {
		return ErrInvalid
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, "DELETE FROM artifacts WHERE expires<=?", time.Now().UnixMilli()); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "DELETE FROM responses WHERE expires<=?", time.Now().UnixMilli()); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "DELETE FROM config_controls WHERE expires<=?", time.Now().UnixMilli()); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "UPDATE config_controls SET expires=created+?", v.RetentionSeconds*1000); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "UPDATE settings SET retention_seconds=?,concurrency=?,timeout_seconds=? WHERE id=1", v.RetentionSeconds, v.Concurrency, v.TimeoutSeconds); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "UPDATE artifacts SET expires=created+?", v.RetentionSeconds*1000); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "UPDATE responses SET expires=created+?", v.RetentionSeconds*1000); e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Store) Recover(ctx context.Context) error {
	if _, e := s.db.ExecContext(ctx, "DELETE FROM staged"); e != nil {
		return e
	}
	_, e := s.db.ExecContext(ctx, "UPDATE sources SET lease_token='',lease_until=0,state=CASE WHEN enabled=1 THEN 'queued' ELSE 'paused' END,next_run=CASE WHEN enabled=1 AND lease_token!='' THEN 0 ELSE next_run END WHERE lease_token!=''")
	return e
}
func (s *Store) Claim(ctx context.Context, now time.Time, limit int, lease time.Duration) ([]collect.Job, error) {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	rows, e := tx.QueryContext(ctx, "SELECT id,url,revision,min,max,unit,etag,modified,timeout_seconds,EXISTS(SELECT 1 FROM responses r WHERE r.source_id=sources.id AND r.expires>?),EXISTS(SELECT 1 FROM memberships m JOIN artifacts a ON a.id=m.artifact_id WHERE m.source_id=sources.id AND a.expires>?) FROM sources WHERE enabled=1 AND next_run<=? AND (lease_token='' OR lease_until<=?) ORDER BY next_run,id LIMIT ?", now.UnixMilli(), now.UnixMilli(), now.UnixNano(), now.UnixMilli(), limit)
	if e != nil {
		return nil, e
	}
	jobs := []collect.Job{}
	corrupt := []string{}
	for rows.Next() {
		var j collect.Job
		var sealed []byte
		var in SourceInput
		var response, items bool
		if e = rows.Scan(&j.ID, &sealed, &j.Revision, &in.Min, &in.Max, &in.Unit, &j.ETag, &j.Modified, &j.TimeoutSeconds, &response, &items); e != nil {
			rows.Close()
			return nil, e
		}
		raw, err := s.open(j.ID, sealed)
		if err != nil {
			corrupt = append(corrupt, j.ID)
			continue
		}
		j.URL = string(raw)
		j.Min, j.Max, e = in.durations()
		if e != nil {
			corrupt = append(corrupt, j.ID)
			continue
		}
		j.Token = randomToken()
		if !response || !items {
			j.ETag = ""
			j.Modified = ""
		}
		jobs = append(jobs, j)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	for _, id := range corrupt {
		if _, e = tx.ExecContext(ctx, "UPDATE sources SET state='error',error_code='source_data_error',lease_token='',lease_until=0,last_fetch=?,error_streak=error_streak+1,next_run=? WHERE id=?", now.UnixMilli(), now.Add(time.Minute).UnixNano(), id); e != nil {
			return nil, e
		}
	}
	for _, j := range jobs {
		jobLease := lease
		if j.TimeoutSeconds > 0 {
			jobLease = time.Duration(j.TimeoutSeconds)*time.Second + 30*time.Second
		}
		if _, e = tx.ExecContext(ctx, "UPDATE sources SET lease_token=?,lease_until=?,state='fetching',error_code='' WHERE id=?", j.Token, now.Add(jobLease).UnixMilli(), j.ID); e != nil {
			return nil, e
		}
	}
	return jobs, tx.Commit()
}
func (s *Store) Complete(ctx context.Context, job collect.Job, result collect.Result, now time.Time) error {
	next, e := schedule.Next(now, job.Min, job.Max)
	if e != nil {
		return e
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var generation int64
	e = tx.QueryRowContext(ctx, "SELECT generation FROM sources WHERE id=? AND revision=? AND lease_token=? AND enabled=1", job.ID, job.Revision, job.Token).Scan(&generation)
	if errors.Is(e, sql.ErrNoRows) {
		return ErrConflict
	}
	if e != nil {
		return e
	}
	var retention int64
	if e = tx.QueryRowContext(ctx, "SELECT retention_seconds FROM settings WHERE id=1").Scan(&retention); e != nil {
		return e
	}
	stamp := now.UnixMilli()
	expiry := stamp + retention*1000
	if _, e = tx.ExecContext(ctx, "DELETE FROM artifacts WHERE expires<=?", stamp); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "DELETE FROM responses WHERE expires<=?", stamp); e != nil {
		return e
	}
	if result.Raw.Hash != "" {
		_, e = tx.ExecContext(ctx, "INSERT INTO responses VALUES (?,?,?,?,?) ON CONFLICT(source_id) DO UPDATE SET hash=excluded.hash,size=excluded.size,created=CASE WHEN responses.hash=excluded.hash THEN responses.created ELSE excluded.created END,expires=CASE WHEN responses.hash=excluded.hash THEN responses.expires ELSE excluded.expires END", job.ID, result.Raw.Hash, result.Raw.Size, stamp, expiry)
		if e != nil {
			return e
		}
	}
	duplicates := result.Extracted - len(result.Items)
	var stagedCount int
	if result.Staged {
		if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM staged WHERE token=?", job.Token).Scan(&stagedCount); e != nil {
			return e
		}
	}
	if result.Staged && result.State == "updated" && stagedCount == 0 {
		result.State = "error"
		result.ErrorCode = "staging_expired"
	}
	if result.State == "updated" && (len(result.Items) > 0 || stagedCount > 0) {
		generation++
		if _, e = tx.ExecContext(ctx, "DELETE FROM memberships WHERE source_id=?", job.ID); e != nil {
			return e
		}
		for _, item := range result.Items {
			if e = saveSample(ctx, tx, item.Hash, item.Identity, item.Kind, item.Size, stamp, ""); e != nil {
				return e
			}
			if e = saveRepair(ctx, tx, item); e != nil {
				return e
			}
			res, err := tx.ExecContext(ctx, "INSERT INTO artifacts(id,hash,kind,size,created,expires,quality) SELECT ?,?,?,?,?,?,? WHERE NOT EXISTS(SELECT 1 FROM config_controls WHERE id=? AND action='deleted' AND expires>?) ON CONFLICT(id) DO NOTHING", item.Identity, item.Hash, item.Kind, item.Size, stamp, expiry, item.Quality, item.Identity, stamp)
			if err != nil {
				return err
			}
			inserted, _ := res.RowsAffected()
			if _, e = tx.ExecContext(ctx, "UPDATE artifacts SET quality=? WHERE id=?", item.Quality, item.Identity); e != nil {
				return e
			}
			if inserted == 0 {
				duplicates++
			}
			if _, e = tx.ExecContext(ctx, "INSERT INTO memberships SELECT ?,?,?,?,?,? WHERE EXISTS(SELECT 1 FROM artifacts WHERE id=?)", job.ID, item.Identity, generation, item.Hash, item.Kind, item.Size, item.Identity); e != nil {
				return e
			}
		}
	}
	if result.State == "unchanged" {
		var count int
		if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM responses WHERE source_id=? AND expires>?", job.ID, stamp).Scan(&count); e != nil {
			return e
		}
		if count == 0 {
			result.State = "error"
			result.ErrorCode = "cache_missing"
			result.ETag = ""
			result.Modified = ""
		}
	}
	if result.State == "updated" && result.Staged && stagedCount > 0 {
		res, err := tx.ExecContext(ctx, "INSERT INTO artifacts(id,hash,kind,size,created,expires,quality) SELECT identity,hash,kind,size,?,?,quality FROM staged WHERE token=? AND NOT EXISTS(SELECT 1 FROM config_controls c WHERE c.id=staged.identity AND c.action='deleted' AND c.expires>?) ON CONFLICT(id) DO NOTHING", stamp, expiry, job.Token, stamp)
		if err != nil {
			return err
		}
		inserted, _ := res.RowsAffected()
		duplicates = result.Extracted - int(inserted)
		if _, e = tx.ExecContext(ctx, "UPDATE artifacts SET quality=(SELECT quality FROM staged WHERE token=? AND identity=artifacts.id) WHERE id IN (SELECT identity FROM staged WHERE token=?)", job.Token, job.Token); e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, "INSERT INTO memberships SELECT source_id,identity,?,hash,kind,size FROM staged WHERE token=? AND EXISTS(SELECT 1 FROM artifacts a WHERE a.id=staged.identity)", generation, job.Token); e != nil {
			return e
		}
	}
	if result.State == "updated" && result.Staged && stagedCount > 0 {
		if _, e = tx.ExecContext(ctx, `INSERT INTO bank_samples(hash,identity,kind,size,created,last_seen,name)
        SELECT st.hash,st.identity,st.kind,st.size,?,?,s.name FROM staged st JOIN sources s ON s.id=st.source_id WHERE st.token=?
        ON CONFLICT(hash) DO UPDATE SET last_seen=excluded.last_seen WHERE bank_samples.deleted=0`, stamp, stamp, job.Token); e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, `INSERT INTO probe_jobs(hash,revision) SELECT b.hash,b.revision FROM bank_samples b JOIN staged st ON st.hash=b.hash WHERE st.token=? AND b.enabled=1 AND b.deleted=0 ON CONFLICT(hash) DO NOTHING`, job.Token); e != nil {
			return e
		}
	}
	if _, e = tx.ExecContext(ctx, "DELETE FROM staged WHERE token=?", job.Token); e != nil {
		return e
	}
	success := result.State == "updated" || result.State == "unchanged"
	if result.ContentState != "" {
		_, e = tx.ExecContext(ctx, "UPDATE sources SET content_state=?,valid_count=?,invalid_count=?,unknown_count=?,received_count=?,response_bytes=? WHERE id=?", result.ContentState, result.ValidCount, result.InvalidCount, result.UnknownCount, result.Extracted, result.Raw.Size, job.ID)
		if e != nil {
			return e
		}
	}
	_, e = tx.ExecContext(ctx, "UPDATE sources SET state=?,error_code=?,last_fetch=?,last_success=CASE WHEN ? THEN ? ELSE last_success END,http_status=?,duration_ms=?,fetch_count=fetch_count+1,duplicates=?,generation=?,error_streak=CASE WHEN ? THEN 0 ELSE error_streak+1 END,lease_token='',lease_until=0,next_run=?,etag=CASE WHEN ? THEN ? ELSE etag END,modified=CASE WHEN ? THEN ? ELSE modified END WHERE id=? AND revision=? AND lease_token=?", result.State, result.ErrorCode, stamp, success, stamp, result.HTTPStatus, result.DurationMS, duplicates, generation, success, next.UnixNano(), result.HTTPStatus == 200 || result.ErrorCode == "cache_missing", result.ETag, result.HTTPStatus == 200 || result.ErrorCode == "cache_missing", result.Modified, job.ID, job.Revision, job.Token)
	if e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Store) Expire(ctx context.Context, now time.Time) error {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, "DELETE FROM config_controls WHERE expires<=?", now.UnixMilli()); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "DELETE FROM staged WHERE NOT EXISTS(SELECT 1 FROM sources s WHERE s.id=staged.source_id AND s.lease_token=staged.token AND s.lease_until>?)", now.UnixMilli()); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "DELETE FROM artifacts WHERE expires<=? OR NOT EXISTS(SELECT 1 FROM memberships m WHERE m.artifact_id=artifacts.id)", now.UnixMilli()); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "DELETE FROM responses WHERE expires<=?", now.UnixMilli()); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "DELETE FROM repairs WHERE NOT ("+repairReferences+")"); e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Store) Referenced(ctx context.Context, hash string) (bool, error) {
	var used bool
	e := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM memberships WHERE raw_hash=? UNION ALL SELECT 1 FROM responses WHERE hash=? UNION ALL SELECT 1 FROM staged WHERE hash=? UNION ALL SELECT 1 FROM repairs WHERE candidate_hash=? AND ("+repairReferences+") UNION ALL SELECT 1 FROM bank_samples WHERE hash=? AND deleted=0 UNION ALL SELECT 1 FROM probe_results r JOIN bank_samples b ON b.hash=r.hash WHERE r.candidate_hash=? AND b.deleted=0 UNION ALL SELECT 1 FROM bank_variants v JOIN bank_samples b ON b.hash=v.hash WHERE v.candidate_hash=? AND b.deleted=0)", hash, hash, hash, hash, hash, hash, hash).Scan(&used)
	return used, e
}

type Artifact struct {
	ID       string `json:"id"`
	Hash     string `json:"hash"`
	Kind     string `json:"kind"`
	Size     int64  `json:"size"`
	Created  int64  `json:"created"`
	Expires  int64  `json:"expires"`
	Sources  int    `json:"sources"`
	Disabled bool   `json:"disabled"`
	Quality  string `json:"quality"`
}

const representativeJoin = " FROM artifacts a JOIN memberships m ON m.artifact_id=a.id AND m.source_id=(SELECT min(m2.source_id) FROM memberships m2 WHERE m2.artifact_id=a.id) "

func (s *Store) Outputs(ctx context.Context, kind string, page int, all bool) ([]Artifact, error) {
	return s.OutputsFiltered(ctx, kind, "", page, all)
}
func (s *Store) OutputsFiltered(ctx context.Context, kind, repairStatus string, page int, all bool) ([]Artifact, error) {
	args := []any{time.Now().UnixMilli()}
	query := "SELECT a.id,m.raw_hash,m.kind,m.size,a.created,a.expires,(SELECT count(*) FROM memberships m3 WHERE m3.artifact_id=a.id),EXISTS(SELECT 1 FROM config_controls c WHERE c.id=a.id AND c.action='disabled' AND c.expires>CAST(unixepoch('subsec')*1000 AS INTEGER)),a.quality" + representativeJoin + "WHERE a.expires>?"
	if kind != "" {
		query += " AND m.kind=?"
		args = append(args, kind)
	}
	if repairStatus != "" {
		query += " AND EXISTS(SELECT 1 FROM repairs r WHERE r.raw_hash=m.raw_hash AND r.status=?)"
		args = append(args, repairStatus)
	}
	query += " ORDER BY a.created DESC,a.id"
	if !all {
		if page < 0 {
			page = 0
		}
		query += " LIMIT 100 OFFSET ?"
		args = append(args, page*100)
	}
	rows, e := s.db.QueryContext(ctx, query, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Artifact{}
	for rows.Next() {
		var a Artifact
		if e = rows.Scan(&a.ID, &a.Hash, &a.Kind, &a.Size, &a.Created, &a.Expires, &a.Sources, &a.Disabled, &a.Quality); e != nil {
			return nil, e
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
func (s *Store) Output(ctx context.Context, id string) (Artifact, error) {
	var a Artifact
	e := s.db.QueryRowContext(ctx, "SELECT a.id,m.raw_hash,m.kind,m.size,a.created,a.expires,EXISTS(SELECT 1 FROM config_controls c WHERE c.id=a.id AND c.action='disabled' AND c.expires>CAST(unixepoch('subsec')*1000 AS INTEGER)),a.quality"+representativeJoin+"WHERE a.id=? AND a.expires>?", id, time.Now().UnixMilli()).Scan(&a.ID, &a.Hash, &a.Kind, &a.Size, &a.Created, &a.Expires, &a.Disabled, &a.Quality)
	if errors.Is(e, sql.ErrNoRows) {
		e = ErrNotFound
	}
	return a, e
}
func (s *Store) Response(ctx context.Context, id string) (blob.Ref, error) {
	var ref blob.Ref
	e := s.db.QueryRowContext(ctx, "SELECT hash,size FROM responses WHERE source_id=? AND expires>?", id, time.Now().UnixMilli()).Scan(&ref.Hash, &ref.Size)
	if errors.Is(e, sql.ErrNoRows) {
		e = ErrNotFound
	}
	return ref, e
}

type Stats struct {
	Sources         int   `json:"sources"`
	Enabled         int   `json:"enabled"`
	Fetching        int   `json:"fetching"`
	Errors          int   `json:"errors"`
	Configs         int   `json:"configs"`
	Bytes           int64 `json:"bytes"`
	GivingConfigs   int   `json:"giving_configs"`
	EmptySources    int   `json:"empty_sources"`
	InvalidSources  int   `json:"invalid_sources"`
	DisabledConfigs int   `json:"disabled_configs"`
}

func (s *Store) Stats(ctx context.Context) (Stats, error) {
	var v Stats
	e := s.db.QueryRowContext(ctx, "SELECT count(*),coalesce(sum(enabled),0),coalesce(sum(state='fetching'),0),coalesce(sum(state IN ('error','invalid')),0),coalesce(sum(content_state IN ('config','partial','unverified')),0),coalesce(sum(content_state='empty'),0),coalesce(sum(content_state IN ('invalid','partial','unrecognized','error_response')),0) FROM sources").Scan(&v.Sources, &v.Enabled, &v.Fetching, &v.Errors, &v.GivingConfigs, &v.EmptySources, &v.InvalidSources)
	if e != nil {
		return v, e
	}
	e = s.db.QueryRowContext(ctx, "SELECT count(*),coalesce(sum(m.size),0)"+representativeJoin+"WHERE a.expires>?", time.Now().UnixMilli()).Scan(&v.Configs, &v.Bytes)
	if e == nil {
		e = s.db.QueryRowContext(ctx, "SELECT count(*) FROM config_controls c JOIN artifacts a ON a.id=c.id WHERE c.action='disabled' AND c.expires>? AND a.expires>?", time.Now().UnixMilli(), time.Now().UnixMilli()).Scan(&v.DisabledConfigs)
	}
	return v, e
}

func (s *Store) NewEngine() *collect.Engine {
	return collect.NewEngine(s, s.vault, collect.HTTPDriver{})
}

func (s *Store) Stage(ctx context.Context, job collect.Job, item collect.Item) (bool, error) {
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return false, e
	}
	defer tx.Rollback()
	var valid bool
	e = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM sources WHERE id=? AND revision=? AND lease_token=? AND enabled=1)", job.ID, job.Revision, job.Token).Scan(&valid)
	if e != nil {
		return false, e
	}
	if !valid {
		return false, ErrConflict
	}
	result, e := tx.ExecContext(ctx, "INSERT INTO staged(source_id,token,identity,hash,kind,size,quality) VALUES (?,?,?,?,?,?,?) ON CONFLICT(token,identity) DO NOTHING", job.ID, job.Token, item.Identity, item.Hash, item.Kind, item.Size, item.Quality)
	if e != nil {
		return false, e
	}
	n, _ := result.RowsAffected()
	if e = saveRepair(ctx, tx, item); e != nil {
		return false, e
	}
	return n == 1, tx.Commit()
}

func (s *Store) WalkOutputs(ctx context.Context, visit func(Artifact) error) error {
	rows, e := s.db.QueryContext(ctx, "SELECT a.id,m.raw_hash,m.kind,m.size,a.created,a.expires"+representativeJoin+"WHERE a.expires>? AND NOT EXISTS(SELECT 1 FROM config_controls c WHERE c.id=a.id AND c.action='disabled' AND c.expires>CAST(unixepoch('subsec')*1000 AS INTEGER)) ORDER BY a.created DESC,a.id", time.Now().UnixMilli())
	if e != nil {
		return e
	}
	defer rows.Close()
	for rows.Next() {
		var a Artifact
		if e = rows.Scan(&a.ID, &a.Hash, &a.Kind, &a.Size, &a.Created, &a.Expires); e != nil {
			return e
		}
		if e = visit(a); e != nil {
			return e
		}
	}
	return rows.Err()
}
