package panel

import "strings"

// Infrastructure recovery is separate from classification. Only these named
// faults remain retryable after the finite UNKNOWN route budget. Existing
// backoff, scope, leases, exclusions and resource admission still apply.
var egressRecoveryReasons = []string{
	"ads_iran_quarantined", "implementation_changed", "local_resource_pressure",
	"driver_session_failed", "driver_subprocess_timeout", "driver_subprocess_failed",
	"driver_infrastructure_error", "input_unavailable", "missing_runtime_asset",
	"endpoint_dns_failed", "ads_delivery_unproven",
}

func (s *Store) ensureEgressRecovery() error {
	definition := `CREATE VIEW egress_work AS SELECT * FROM egress_inventory WHERE tested=0 OR ever_accepted=1 OR reason IN('` + strings.Join(egressRecoveryReasons, "','") + `') OR (state='UNKNOWN' AND deep_completed=0 AND retry_streak<2) OR hash IN(SELECT hash FROM egress_jobs WHERE json_extract(decision,'$.country_conflict')=1) OR EXISTS(SELECT 1 FROM egress_retirements x WHERE x.hash=egress_inventory.hash)`
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var current string
	if err = tx.QueryRow("SELECT sql FROM sqlite_master WHERE name='egress_work'").Scan(&current); err != nil {
		return err
	}
	if current == definition {
		return nil
	}
	if _, err = tx.Exec("DROP VIEW egress_work;" + definition); err != nil {
		return err
	}
	return tx.Commit()
}
