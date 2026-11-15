package panel

// Non-reset maintenance migration. Preserve the reset clock, source inputs,
// positive proof, country policy and sticky withdrawal. Old confirmation counts
// cannot be reused after tightening destructive-negative evidence validation.
func (s *Store) migrateV37() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var v int
	if err = tx.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		return err
	}
	if v >= 37 {
		return nil
	}
	var installed int
	if err = tx.QueryRow("SELECT count(*) FROM pragma_table_info('egress_retirements') WHERE name='first_at'").Scan(&installed); err != nil {
		return err
	}
	if installed == 0 {
		if _, err = tx.Exec(`ALTER TABLE egress_retirements ADD COLUMN first_at INTEGER NOT NULL DEFAULT 0;
 UPDATE egress_retirements SET hits=0,at=0;`); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(`CREATE TABLE IF NOT EXISTS egress_settlements(hash TEXT PRIMARY KEY,at INTEGER NOT NULL,state TEXT NOT NULL);
 INSERT OR IGNORE INTO egress_settlements SELECT e.hash,e.at,e.state FROM egress_events e JOIN
 (SELECT hash,min(id) AS id FROM egress_events WHERE definitive=1 AND at>=(SELECT since FROM egress_metrics) GROUP BY hash) f ON e.id=f.id;
 UPDATE egress_metrics SET definitive=(SELECT count(*) FROM egress_settlements) WHERE id=1;`); err != nil {
		return err
	}
	if _, err = tx.Exec("PRAGMA user_version=37"); err != nil {
		return err
	}
	return tx.Commit()
}
