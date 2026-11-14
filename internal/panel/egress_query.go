package panel

import (
	"errors"
	"strings"
)

const egressInventorySet = "j.hash IN(SELECT hash FROM egress_inventory)"
const egressInventoryLookup = "EXISTS(SELECT 1 FROM egress_inventory i WHERE i.hash=j.hash)"

const egressGuardScan = "FROM egress_owners o JOIN egress_iran_guard g ON g.identity=o.identity"
const egressGuardLookup = "FROM egress_iran_guard g CROSS JOIN egress_owners o ON o.identity=g.identity"

// The guard set is small; the ownership history is not. This view participates
// repeatedly in ordinary, Smart and Residential eligibility. Start from guards
// and use the existing identity index for every historical alias. No guard state,
// scope, expiry or evidence predicate changes, and no stored data is rewritten.
func (s *Store) ensureEgressGuardLookup() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var view string
	if err = tx.QueryRow("SELECT sql FROM sqlite_master WHERE name='input_blocked'").Scan(&view); err != nil {
		return err
	}
	if strings.Contains(view, egressGuardLookup) {
		return nil
	}
	if strings.Count(view, egressGuardScan) != 1 {
		return errors.New("unrecognized blocked-input guard predicate")
	}
	view = strings.Replace(view, egressGuardScan, egressGuardLookup, 1)
	if _, err = tx.Exec("DROP VIEW input_blocked;" + view); err != nil {
		return err
	}
	return tx.Commit()
}

// Eligibility asks whether one proven hash is in scope. Building the entire
// historical inventory for each such check also stalls ordinary conversions,
// Smart and single exports through their shared input-blocking views. Keep the
// exact live predicate, but let SQLite use its unique hash indexes.
// This is an idempotent query-plan repair, not a new classification epoch.
func (s *Store) ensureEgressEligibilityLookup() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var view string
	if err = tx.QueryRow("SELECT sql FROM sqlite_master WHERE name='egress_eligible'").Scan(&view); err != nil {
		return err
	}
	if strings.Contains(view, egressInventoryLookup) {
		return nil
	}
	if strings.Count(view, egressInventorySet) != 1 {
		return errors.New("unrecognized egress inventory eligibility predicate")
	}
	view = strings.Replace(view, egressInventorySet, egressInventoryLookup, 1)
	if _, err = tx.Exec("DROP VIEW egress_eligible;" + view); err != nil {
		return err
	}
	return tx.Commit()
}
