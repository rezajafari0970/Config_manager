package panel

import "github.com/rezajafari0970/Config_manager/internal/egress"

// P66 explicitly requests one new reset. The schema transaction is the durable
// idempotency key: neither daemon restart nor a source import can repeat it.
func (s *Store) migrateV36() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var v int
	if err = tx.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		return err
	}
	if v >= 36 {
		return nil
	}
	// The feature footprint is also a reset marker. Repairing an unrelated old
	// schema or an accidentally lowered PRAGMA must never repeat the reset.
	var installed int
	if err = tx.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='egress_retirements'").Scan(&installed); err != nil {
		return err
	}
	if installed == 1 {
		if _, err = tx.Exec("SELECT settled_at,settled_state FROM egress_run_members LIMIT 0"); err != nil {
			return err
		}
		if _, err = tx.Exec("PRAGMA user_version=36"); err != nil {
			return err
		}
		return tx.Commit()
	}
	now := egress.Now()
	_, err = tx.Exec(`
 ALTER TABLE egress_run_members ADD COLUMN settled_at INTEGER NOT NULL DEFAULT 0;
 ALTER TABLE egress_run_members ADD COLUMN settled_state TEXT NOT NULL DEFAULT '';
 CREATE TABLE egress_retirements(hash TEXT PRIMARY KEY REFERENCES egress_jobs(hash) ON DELETE CASCADE,reason TEXT NOT NULL,hits INTEGER NOT NULL,at INTEGER NOT NULL);
 DROP VIEW egress_work;
 CREATE VIEW egress_work AS SELECT * FROM egress_inventory WHERE tested=0 OR ever_accepted=1 OR reason IN('ads_iran_quarantined','implementation_changed','local_resource_pressure','driver_session_failed','driver_subprocess_timeout','driver_infrastructure_error','input_unavailable','missing_runtime_asset','ads_delivery_unproven') OR (state='UNKNOWN' AND deep_completed=0 AND retry_streak<2) OR hash IN(SELECT hash FROM egress_jobs WHERE json_extract(decision,'$.country_conflict')=1) OR EXISTS(SELECT 1 FROM egress_retirements x WHERE x.hash=egress_inventory.hash);
 DELETE FROM egress_events;
 DELETE FROM egress_health;
 DELETE FROM egress_geo_results;
 DELETE FROM egress_countries;
 DELETE FROM egress_turns;
 DELETE FROM egress_run_members;
 DELETE FROM egress_runs;
 UPDATE egress_jobs SET revision=revision+1,token='',lease_until=0,claimed_at=0,tested=0,valid_until=0,candidate_hash='',candidate_size=0,report='{}',decision='{}',attempts=0,failures=0,last_failure=0,retry_streak=0,first_completed=0,deep_completed=0,next_run=0,
 state=CASE WHEN ever_accepted=1 THEN 'QUARANTINED' ELSE 'PENDING' END,reason='unified_epoch' WHERE state!='DELETED';
 UPDATE egress_jobs SET reason='ads_iran_quarantined',state='QUARANTINED' WHERE state!='DELETED' AND hash IN(SELECT u.hash FROM egress_unique u JOIN egress_iran_guard g ON g.identity=u.identity WHERE g.state='QUARANTINED');
 UPDATE egress_settings SET generation=generation+1,circuit_until=0;
 UPDATE egress_worker SET status='offline',heartbeat=0,active=0,code='';
 UPDATE egress_metrics SET since=?,baseline_inputs=(SELECT count(*) FROM egress_inventory),registered=0,completed=0,first_assessments=0,definitive=0,lease_recoveries=0,unhealthy_deleted=0 WHERE id=1;
 INSERT INTO egress_runs(started) SELECT ? WHERE EXISTS(SELECT 1 FROM egress_inventory);
 INSERT INTO egress_run_members(run,hash,accepted) SELECT (SELECT max(id) FROM egress_runs),hash,ever_accepted FROM egress_inventory WHERE paused=0;
 PRAGMA user_version=36;`, now, now)
	if err != nil {
		return err
	}
	var blocked string
	if err = tx.QueryRow("SELECT sql FROM sqlite_master WHERE name='input_blocked'").Scan(&blocked); err != nil {
		return err
	}
	// Withdrawal spans ordinary, Smart and Residential outputs, including aliases.
	blocked += " UNION SELECT hash FROM egress_retirements UNION SELECT hash FROM egress_jobs WHERE json_extract(decision,'$.country_conflict')=1"
	if _, err = tx.Exec("DROP VIEW input_blocked;" + blocked); err != nil {
		return err
	}
	return tx.Commit()
}
