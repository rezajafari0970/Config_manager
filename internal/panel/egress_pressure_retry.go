package panel

// Repair only exhausted assessments interrupted by the host's resource limit.
// No classification, report, attempt statistic, source membership or expiry is
// rewritten. Archive rows may regain a retry budget, but inventory still gates
// every claim; this does not schedule archive-wide work. Completed negatives and
// ordinary incomplete network observations retain their bounded cache policy.
func (s *Store) recoverEgressPressureBudget() error {
	_, err := s.db.Exec(`UPDATE egress_jobs SET retry_streak=0,deep_completed=0
 WHERE ever_accepted=0 AND state='UNKNOWN' AND reason='local_resource_pressure'
 AND token='' AND (deep_completed!=0 OR retry_streak>=2)
 AND NOT EXISTS(SELECT 1 FROM egress_owners o JOIN input_exclusions x ON x.identity=o.identity WHERE o.hash=egress_jobs.hash)
 AND NOT EXISTS(SELECT 1 FROM egress_owners o JOIN egress_iran_guard g ON g.identity=o.identity WHERE o.hash=egress_jobs.hash AND g.state='BLOCKED')`)
	return err
}
