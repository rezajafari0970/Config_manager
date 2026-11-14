package panel

import (
	"github.com/rezajafari0970/Config_manager/internal/egress"
)

// One durable representative per collector fingerprint. Fingerprint includes
// credentials, routing and transport; only cosmetic names/encoding are removed.
// Receipts stay bound to the exact tested representative, never copied to aliases.
func (s *Store) migrateV34() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var v int
	if err = tx.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		return err
	}
	if v >= 34 {
		return nil
	}
	now := egress.Now()
	_, err = tx.Exec(`
 CREATE TABLE egress_unique(identity TEXT PRIMARY KEY,hash TEXT NOT NULL UNIQUE REFERENCES egress_jobs(hash),created INTEGER NOT NULL);
 CREATE INDEX egress_membership_identity ON memberships(artifact_id,raw_hash);
 DROP TRIGGER egress_metric_ingest;
 DROP TRIGGER egress_ingest;
 CREATE TRIGGER egress_ingest AFTER INSERT ON memberships BEGIN
 INSERT INTO egress_jobs(hash,kind,size,first_seen,last_seen)
 SELECT new.raw_hash,new.kind,new.size,CAST(unixepoch('subsec')*1000 AS INTEGER),CAST(unixepoch('subsec')*1000 AS INTEGER)
 WHERE NOT EXISTS(SELECT 1 FROM input_exclusions WHERE identity=new.artifact_id)
 ON CONFLICT(hash) DO UPDATE SET last_seen=excluded.last_seen,
 state=CASE WHEN egress_jobs.reason IN('ordinary_delete','retention_expired') AND NOT EXISTS(SELECT 1 FROM egress_unique u WHERE u.hash=egress_jobs.hash) THEN 'PENDING' ELSE egress_jobs.state END,
 reason=CASE WHEN egress_jobs.reason IN('ordinary_delete','retention_expired') AND NOT EXISTS(SELECT 1 FROM egress_unique u WHERE u.hash=egress_jobs.hash) THEN 'unique_epoch' ELSE egress_jobs.reason END;
 INSERT OR IGNORE INTO egress_owners SELECT new.raw_hash,new.artifact_id,new.source_id WHERE EXISTS(SELECT 1 FROM egress_jobs WHERE hash=new.raw_hash);
 INSERT OR IGNORE INTO egress_unique SELECT new.artifact_id,new.raw_hash,CAST(unixepoch('subsec')*1000 AS INTEGER)
 WHERE EXISTS(SELECT 1 FROM egress_jobs WHERE hash=new.raw_hash AND state!='DELETED') AND NOT EXISTS(SELECT 1 FROM input_exclusions WHERE identity=new.artifact_id);
 END;
 CREATE TRIGGER egress_metric_unique AFTER INSERT ON egress_unique BEGIN UPDATE egress_metrics SET registered=registered+1 WHERE id=1; END;
 INSERT OR IGNORE INTO egress_unique
 SELECT m.artifact_id,min(m.raw_hash),CAST(unixepoch('subsec')*1000 AS INTEGER) FROM memberships m JOIN egress_jobs j ON j.hash=m.raw_hash
 WHERE j.state!='DELETED' AND NOT EXISTS(SELECT 1 FROM input_exclusions x WHERE x.identity=m.artifact_id)
 GROUP BY m.artifact_id;
 CREATE VIEW egress_inventory AS SELECT j.* FROM egress_unique u JOIN egress_jobs j ON j.hash=u.hash
 WHERE j.state!='DELETED' AND NOT EXISTS(SELECT 1 FROM input_exclusions x WHERE x.identity=u.identity) AND NOT EXISTS(SELECT 1 FROM egress_iran_guard g WHERE g.identity=u.identity AND g.state='BLOCKED') AND (EXISTS(SELECT 1 FROM memberships m WHERE m.artifact_id=u.identity) OR j.ever_accepted=1 OR EXISTS(SELECT 1 FROM egress_iran_guard g WHERE g.identity=u.identity AND g.state='QUARANTINED'));
 CREATE VIEW egress_work AS SELECT * FROM egress_inventory WHERE tested=0 OR ever_accepted=1 OR reason IN('ads_iran_quarantined','implementation_changed','manual_recheck') OR (state='UNKNOWN' AND deep_completed=0 AND retry_streak<2);
 DELETE FROM egress_run_members;
 DELETE FROM egress_runs;
 DELETE FROM egress_events;
 DELETE FROM egress_health;
 DELETE FROM egress_geo_results;
 DELETE FROM egress_countries;
 DELETE FROM egress_turns;
 UPDATE egress_jobs SET revision=revision+1,token='',lease_until=0,claimed_at=0,tested=0,valid_until=0,ever_accepted=0,candidate_hash='',candidate_size=0,report='{}',decision='{}',attempts=0,failures=0,last_failure=0,retry_streak=0,first_completed=0,deep_completed=0,next_run=0,
 state=CASE WHEN state='DELETED' THEN state ELSE 'PENDING' END,
 reason=CASE WHEN state='DELETED' THEN reason ELSE 'unique_epoch' END;
 UPDATE egress_jobs SET reason='ads_iran_quarantined',state='QUARANTINED' WHERE state!='DELETED' AND hash IN(SELECT u.hash FROM egress_unique u JOIN egress_iran_guard g ON g.identity=u.identity WHERE g.state='QUARANTINED');
 UPDATE egress_settings SET generation=generation+1,circuit_until=0;
 UPDATE egress_worker SET status='offline',heartbeat=0,active=0,code='';
 UPDATE egress_metrics SET since=?,baseline_inputs=(SELECT count(*) FROM egress_unique),registered=0,completed=0,first_assessments=0,definitive=0,lease_recoveries=0,unhealthy_deleted=0 WHERE id=1;
 INSERT INTO egress_runs(started) SELECT ? WHERE EXISTS(SELECT 1 FROM egress_inventory);
 INSERT INTO egress_run_members(run,hash,accepted) SELECT (SELECT max(id) FROM egress_runs),hash,0 FROM egress_inventory WHERE paused=0
 AND NOT EXISTS(SELECT 1 FROM egress_owners o JOIN egress_iran_guard g ON g.identity=o.identity WHERE o.hash=egress_inventory.hash AND g.state='BLOCKED');
 PRAGMA user_version=34;`, now, now)
	if err != nil {
		return err
	}
	var view string
	if err = tx.QueryRow("SELECT sql FROM sqlite_master WHERE name='egress_eligible'").Scan(&view); err != nil {
		return err
	}
	if _, err = tx.Exec("DROP VIEW egress_eligible;" + view + " AND j.hash IN(SELECT hash FROM egress_inventory)"); err != nil {
		return err
	}
	return tx.Commit()
}
