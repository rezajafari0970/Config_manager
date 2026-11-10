package panel

import (
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/convert"
	"github.com/rezajafari0970/Config_manager/internal/residential"
)

func (s *Store) migrateV16() error {
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var version int
	if e = tx.QueryRow("PRAGMA user_version").Scan(&version); e != nil {
		return e
	}
	if version >= 16 {
		return nil
	}
	_, e = tx.Exec(`
 CREATE TABLE residential_settings(id INTEGER PRIMARY KEY CHECK(id=1),policy TEXT NOT NULL,policy_hash TEXT NOT NULL,trust TEXT NOT NULL,trust_hash TEXT NOT NULL,template_version TEXT NOT NULL,revision INTEGER NOT NULL DEFAULT 1,enabled INTEGER NOT NULL DEFAULT 1,generation INTEGER NOT NULL DEFAULT 1,updated INTEGER NOT NULL DEFAULT 0,journal_seq INTEGER NOT NULL DEFAULT 0,circuit_until INTEGER NOT NULL DEFAULT 0);
 CREATE TABLE residential_records(hash TEXT PRIMARY KEY,kind TEXT NOT NULL,size INTEGER NOT NULL,first_seen INTEGER NOT NULL,last_seen INTEGER NOT NULL,revision INTEGER NOT NULL DEFAULT 1,epoch INTEGER NOT NULL DEFAULT 0,paused INTEGER NOT NULL DEFAULT 0,claimed_at INTEGER NOT NULL DEFAULT 0,state TEXT NOT NULL DEFAULT 'UNKNOWN',reason TEXT NOT NULL DEFAULT 'awaiting_assessment',ever_accepted INTEGER NOT NULL DEFAULT 0,candidate_hash TEXT NOT NULL DEFAULT '',candidate_size INTEGER NOT NULL DEFAULT 0,policy_hash TEXT NOT NULL DEFAULT '',trust_hash TEXT NOT NULL DEFAULT '',template_version TEXT NOT NULL DEFAULT '',tested INTEGER NOT NULL DEFAULT 0,valid_until INTEGER NOT NULL DEFAULT 0,decision TEXT NOT NULL DEFAULT '{}',report TEXT NOT NULL DEFAULT '{}',token TEXT NOT NULL DEFAULT '',lease_until INTEGER NOT NULL DEFAULT 0,next_run INTEGER NOT NULL DEFAULT 0,attempts INTEGER NOT NULL DEFAULT 0,negative_count INTEGER NOT NULL DEFAULT 0,negative_first INTEGER NOT NULL DEFAULT 0,negative_last INTEGER NOT NULL DEFAULT 0,negative_worker TEXT NOT NULL DEFAULT '',negative_epoch TEXT NOT NULL DEFAULT '');
 CREATE INDEX residential_due ON residential_records(next_run,lease_until);
 CREATE INDEX residential_candidate_refs ON residential_records(candidate_hash) WHERE state!='DELETED';
 CREATE INDEX residential_retained ON residential_records(ever_accepted,state,valid_until);
 CREATE TABLE residential_source_fairness(source TEXT PRIMARY KEY,last_claim INTEGER NOT NULL DEFAULT 0);
 CREATE TABLE residential_owners(hash TEXT NOT NULL REFERENCES residential_records(hash),identity TEXT NOT NULL,source TEXT NOT NULL,PRIMARY KEY(hash,identity,source));
 CREATE INDEX residential_identity ON residential_owners(identity);
 CREATE TABLE residential_tombstones(hash TEXT PRIMARY KEY,at INTEGER NOT NULL,epoch INTEGER NOT NULL,reason TEXT NOT NULL,manual INTEGER NOT NULL,seq INTEGER NOT NULL);
 CREATE TABLE residential_identity_tombstones(identity TEXT PRIMARY KEY,at INTEGER NOT NULL,hash TEXT NOT NULL,manual INTEGER NOT NULL);
 CREATE TABLE residential_audit(id INTEGER PRIMARY KEY,hash TEXT NOT NULL,revision INTEGER NOT NULL,at INTEGER NOT NULL,state TEXT NOT NULL,reason TEXT NOT NULL,report TEXT NOT NULL,decision TEXT NOT NULL);
 CREATE INDEX residential_audit_hash ON residential_audit(hash,id);
 CREATE TABLE residential_rotation(scope TEXT PRIMARY KEY,cycle INTEGER NOT NULL DEFAULT 0);
 CREATE TABLE residential_turns(scope TEXT NOT NULL REFERENCES residential_rotation(scope) ON DELETE CASCADE,position INTEGER NOT NULL,hash TEXT NOT NULL,PRIMARY KEY(scope,position),UNIQUE(scope,hash));
 CREATE TABLE residential_worker(id INTEGER PRIMARY KEY CHECK(id=1),status TEXT NOT NULL,code TEXT NOT NULL,version TEXT NOT NULL,active INTEGER NOT NULL,heartbeat INTEGER NOT NULL);
 INSERT INTO residential_worker VALUES(1,'offline','','',0,0);
 CREATE TABLE residential_metrics(id INTEGER PRIMARY KEY,at INTEGER NOT NULL,wall_ms REAL NOT NULL,cpu_ms REAL NOT NULL,rss_bytes INTEGER NOT NULL,state TEXT NOT NULL);
 CREATE TRIGGER residential_ingest AFTER INSERT ON memberships BEGIN
 INSERT INTO residential_records(hash,kind,size,first_seen,last_seen) VALUES(new.raw_hash,new.kind,new.size,CAST(unixepoch('subsec')*1000 AS INTEGER),CAST(unixepoch('subsec')*1000 AS INTEGER)) ON CONFLICT(hash) DO UPDATE SET last_seen=excluded.last_seen;
 INSERT OR IGNORE INTO residential_owners VALUES(new.raw_hash,new.artifact_id,new.source_id);
 UPDATE residential_records SET state='UNKNOWN',reason='fresh_readmission_required',revision=revision+1,epoch=epoch+1,next_run=0,valid_until=0,token='',lease_until=0 WHERE hash=new.raw_hash AND state='DELETED' AND NOT EXISTS(SELECT 1 FROM residential_tombstones WHERE hash=new.raw_hash AND manual=1) AND NOT EXISTS(SELECT 1 FROM residential_identity_tombstones WHERE identity=new.artifact_id AND manual=1);
 END;
 INSERT INTO residential_records(hash,kind,size,first_seen,last_seen) SELECT m.raw_hash,min(m.kind),max(m.size),min(a.created),max(a.created) FROM memberships m JOIN artifacts a ON a.id=m.artifact_id GROUP BY m.raw_hash;
 INSERT INTO residential_owners SELECT DISTINCT raw_hash,artifact_id,source_id FROM memberships;
 CREATE VIEW residential_eligible AS SELECT r.* FROM residential_records r JOIN residential_settings s ON s.id=1 WHERE s.enabled=1 AND r.paused=0 AND r.state='ACCEPTED' AND r.ever_accepted=1 AND r.tested<=CAST(unixepoch('subsec')*1000 AS INTEGER)+2000 AND r.valid_until>CAST(unixepoch('subsec')*1000 AS INTEGER) AND r.policy_hash=s.policy_hash AND r.trust_hash=s.trust_hash AND r.template_version=s.template_version AND r.candidate_hash!='' AND NOT EXISTS(SELECT 1 FROM residential_owners o JOIN residential_identity_tombstones t ON t.identity=o.identity WHERE o.hash=r.hash AND t.manual=1) AND NOT EXISTS(SELECT 1 FROM residential_tombstones t WHERE t.hash=r.hash AND (t.manual=1 OR r.tested<=t.at OR r.epoch<=t.epoch));
 CREATE VIEW residential_blocked AS SELECT r.hash FROM residential_records r WHERE r.ever_accepted=1 AND NOT EXISTS(SELECT 1 FROM residential_eligible e WHERE e.hash=r.hash) UNION SELECT t.hash FROM residential_tombstones t WHERE NOT EXISTS(SELECT 1 FROM residential_eligible e WHERE e.hash=t.hash);
 CREATE VIEW residential_blocked_identities AS
 SELECT o.identity FROM residential_owners o JOIN residential_records r ON r.hash=o.hash
 WHERE r.ever_accepted=1 AND NOT EXISTS(SELECT 1 FROM residential_eligible e WHERE e.hash=r.hash)
 AND NOT EXISTS(SELECT 1 FROM residential_owners fresh JOIN residential_eligible accepted ON accepted.hash=fresh.hash WHERE fresh.identity=o.identity AND accepted.tested>r.tested)
 UNION SELECT t.identity FROM residential_identity_tombstones t WHERE NOT EXISTS(SELECT 1 FROM residential_owners o JOIN residential_eligible e ON e.hash=o.hash WHERE o.identity=t.identity AND e.tested>t.at AND t.manual=0);
 PRAGMA user_version=16;`)
	if e != nil {
		return e
	}
	p := residential.DefaultPolicy()
	t := residential.EmptyTrust()
	pb, _ := json.Marshal(p)
	tb, _ := json.Marshal(t)
	if _, e = tx.Exec("INSERT INTO residential_settings(id,policy,policy_hash,trust,trust_hash,template_version) VALUES(1,?,?,?,?,?)", string(pb), p.Digest(), string(tb), t.Digest(), convert.Version); e != nil {
		return e
	}
	// These fixed existing views are reused by claim, completion, renew and all
	// Smart ready-cache provenance. Filtering here closes every composed scope.
	for _, name := range []string{"conversion_inputs", "smart_inputs", "splash_smart_inputs"} {
		var q string
		if e = tx.QueryRow("SELECT sql FROM sqlite_master WHERE type='view' AND name=?", name).Scan(&q); e != nil {
			return e
		}
		q += " AND m.raw_hash NOT IN (SELECT hash FROM residential_blocked) AND m.artifact_id NOT IN (SELECT identity FROM residential_blocked_identities)"
		if _, e = tx.Exec("DROP VIEW " + name + ";" + q); e != nil {
			return e
		}
	}
	return tx.Commit()
}
