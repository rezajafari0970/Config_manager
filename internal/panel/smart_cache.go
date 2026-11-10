package panel

import (
	"context"

	"github.com/rezajafari0970/Config_manager/internal/convert"
	"github.com/rezajafari0970/Config_manager/internal/smart"
)

// Jobs and published cache have independent lifetimes. Improving capacity must
// not remove a safe cache; revocation of any exact member still takes effect at
// read time. Only CompleteSmart publishes a validated replacement atomically.
func (s *Store) migrateV11() error {
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var version int
	if e = tx.QueryRow("PRAGMA user_version").Scan(&version); e != nil {
		return e
	}
	if version >= 12 {
		return nil
	}
	_, e = tx.Exec(`
 CREATE TABLE IF NOT EXISTS smart_cache(id TEXT PRIMARY KEY REFERENCES smart_heads(id) ON DELETE CASCADE,hash TEXT NOT NULL,member_count INTEGER NOT NULL,created INTEGER NOT NULL,expires INTEGER NOT NULL,revision INTEGER NOT NULL,version TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS smart_cache_members(head TEXT NOT NULL REFERENCES smart_cache(id) ON DELETE CASCADE,position INTEGER NOT NULL,id TEXT NOT NULL REFERENCES artifacts(id) ON DELETE CASCADE,source TEXT NOT NULL REFERENCES sources(id) ON DELETE CASCADE,raw_hash TEXT NOT NULL,hash TEXT NOT NULL,PRIMARY KEY(head,position),UNIQUE(head,id));
 CREATE INDEX IF NOT EXISTS smart_cache_hash ON smart_cache(hash,expires);
 CREATE INDEX IF NOT EXISTS smart_cache_members_source ON smart_cache_members(source,id,raw_hash,hash);
 CREATE INDEX IF NOT EXISTS memberships_raw_hash ON memberships(raw_hash);
 CREATE INDEX IF NOT EXISTS responses_hash ON responses(hash);
 CREATE INDEX IF NOT EXISTS conversion_candidate_hash ON conversion_results(candidate_hash);
 CREATE INDEX IF NOT EXISTS smart_heads_hash ON smart_heads(hash,expires);
 DROP VIEW IF EXISTS smart_refresh_needed;
 DROP VIEW IF EXISTS smart_ready;
 DROP VIEW IF EXISTS smart_cache_owned;
 DROP VIEW IF EXISTS smart_owned_members;
 DROP VIEW IF EXISTS smart_offers;
 DROP VIEW IF EXISTS smart_inputs;
 CREATE VIEW smart_inputs AS SELECT m.source_id AS source,m.artifact_id AS id,m.raw_hash,r.candidate_hash AS hash,min(a.expires,r.valid_until) AS expires
 FROM memberships m JOIN artifacts a ON a.id=m.artifact_id JOIN sources s ON s.id=m.source_id JOIN conversion_results r ON r.hash=m.raw_hash
 WHERE s.enabled=1 AND a.expires>CAST(unixepoch('subsec')*1000 AS INTEGER)
 AND NOT EXISTS(SELECT 1 FROM config_controls c WHERE c.id=a.id AND c.expires>CAST(unixepoch('subsec')*1000 AS INTEGER))
 AND r.core='passed' AND r.candidate_hash!='' AND r.version='` + convert.Version + `'
 AND r.connection='passed' AND r.code='https_multi_site_verified' AND r.valid_until>CAST(unixepoch('subsec')*1000 AS INTEGER);
 CREATE VIEW smart_offers AS SELECT * FROM smart_inputs;
 CREATE VIEW smart_owned_members AS SELECT m.head,m.position,i.expires FROM smart_members m JOIN smart_inputs i ON i.source=m.source AND i.id=m.id AND i.raw_hash=m.raw_hash AND i.hash=m.hash;
 CREATE VIEW smart_cache_owned AS SELECT m.head,m.position,i.expires FROM smart_cache_members m JOIN smart_inputs i ON i.source=m.source AND i.id=m.id AND i.raw_hash=m.raw_hash AND i.hash=m.hash;
 CREATE VIEW smart_ready AS SELECT c.* FROM smart_cache c JOIN smart_settings s ON s.id=1 JOIN settings ttl ON ttl.id=1
 WHERE s.enabled=1 AND c.version='` + smart.Version + `' AND c.revision=s.revision AND c.hash!=''
 AND c.expires>CAST(unixepoch('subsec')*1000 AS INTEGER) AND c.created+ttl.retention_seconds*1000>CAST(unixepoch('subsec')*1000 AS INTEGER)
 AND c.member_count>0 AND (SELECT count(*) FROM smart_cache_owned m WHERE m.head=c.id)=c.member_count;
 CREATE VIEW smart_refresh_needed AS WITH offers AS MATERIALIZED(SELECT * FROM smart_offers), source_ids AS MATERIALIZED(SELECT DISTINCT source FROM offers)
 SELECT h.id FROM smart_heads h JOIN smart_settings s ON s.id=1 LEFT JOIN smart_ready c ON c.id=h.id WHERE c.id IS NULL
 OR (c.member_count<s.member_count AND EXISTS(SELECT 1 FROM offers i WHERE NOT EXISTS(SELECT 1 FROM smart_cache_members m WHERE m.head=h.id AND (m.id=i.id OR m.hash=i.hash))))
 OR (s.member_count>=(SELECT count(*) FROM source_ids) AND
 (SELECT count(*) FROM (SELECT i.source FROM smart_cache_members m JOIN offers i ON i.id=m.id WHERE m.head=h.id UNION SELECT i.source FROM smart_cache_members m JOIN offers i ON i.hash=m.hash WHERE m.head=h.id))<(SELECT count(*) FROM source_ids));
 `)
	if e != nil {
		return e
	}
	if version < 11 {
		// Never promote unfinished/blocked work or infer proof from a prior code.
		if _, e = tx.Exec(`INSERT OR IGNORE INTO smart_cache SELECT id,hash,member_count,created,expires,revision,version FROM smart_heads WHERE state='ready' AND hash!='';
   INSERT OR IGNORE INTO smart_cache_members SELECT m.* FROM smart_members m JOIN smart_cache c ON c.id=m.head;`); e != nil {
			return e
		}
	}
	if _, e = tx.Exec("PRAGMA user_version=11"); e != nil {
		return e
	}
	return tx.Commit()
}

func publishSmartCache(ctx context.Context, tx smartTransaction, id string) error {
	if _, e := tx.ExecContext(ctx, "DELETE FROM smart_cache WHERE id=?", id); e != nil {
		return e
	}
	if _, e := tx.ExecContext(ctx, `INSERT INTO smart_cache SELECT id,hash,member_count,created,expires,revision,version FROM smart_heads WHERE id=?`, id); e != nil {
		return e
	}
	_, e := tx.ExecContext(ctx, "INSERT INTO smart_cache_members SELECT * FROM smart_members WHERE head=?", id)
	return e
}

// Called only for unreadable/invalid ciphertext or payload, never merely stale
// ownership or a canceled request. A healthy concurrent replacement is fenced
// by cache hash; a running builder retains its independent lease.
func (s *SmartPipeline) retrySmartBlob(ctx context.Context, v SmartEntry) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	res, e := tx.ExecContext(ctx, "DELETE FROM smart_cache WHERE id=? AND hash=?", v.ID, v.Hash)
	if e != nil {
		return e
	}
	n, e := res.RowsAffected()
	if e != nil || n == 0 {
		return e
	}
	if _, e = tx.ExecContext(ctx, "UPDATE smart_heads SET state='queued',next_run=0,code='smart_blob_unavailable' WHERE id=? AND token=''", v.ID); e != nil {
		return e
	}
	return tx.Commit()
}
