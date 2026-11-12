package panel

import "database/sql"

// Migration must register committed staging before Recover removes source
// leases. Explicit exclusions and permanently deleted jobs remain authoritative.
func backfillStagedEgress(tx *sql.Tx) error {
	_, err := tx.Exec(`
 INSERT OR IGNORE INTO egress_jobs(hash,kind,size,first_seen,last_seen)
 SELECT hash,min(kind),max(size),CAST(unixepoch('subsec')*1000 AS INTEGER),CAST(unixepoch('subsec')*1000 AS INTEGER)
 FROM staged s WHERE NOT EXISTS(SELECT 1 FROM input_exclusions x WHERE x.identity=s.identity) GROUP BY hash;
 UPDATE egress_jobs SET state='PENDING',reason='new_input',tested=0,attempts=0,deep_completed=0,next_run=0,report='{}',decision='{}',valid_until=0,token='',lease_until=0,revision=revision+1
 WHERE reason IN ('retention_expired','ordinary_delete') AND hash IN (SELECT s.hash FROM staged s WHERE NOT EXISTS(SELECT 1 FROM input_exclusions x WHERE x.identity=s.identity));
 INSERT OR IGNORE INTO egress_owners
 SELECT DISTINCT s.hash,s.identity,s.source_id FROM staged s WHERE NOT EXISTS(SELECT 1 FROM input_exclusions x WHERE x.identity=s.identity) AND EXISTS(SELECT 1 FROM egress_jobs j WHERE j.hash=s.hash);
 `)
	return err
}
