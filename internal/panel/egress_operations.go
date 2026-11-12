package panel

import (
	"context"
)

// Operational counters are committed with the work they describe. Rolling
// events may expire; lifetime counters never depend on that cleanup window.
func (s *Store) migrateV30() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var version int
	if err = tx.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return err
	}
	if version >= 30 {
		return nil
	}
	_, err = tx.Exec(`
 ALTER TABLE egress_jobs ADD COLUMN retry_streak INTEGER NOT NULL DEFAULT 0;
 ALTER TABLE egress_jobs ADD COLUMN first_completed INTEGER NOT NULL DEFAULT 0;
 UPDATE egress_jobs SET first_completed=1 WHERE tested>0;
 ALTER TABLE egress_events ADD COLUMN definitive INTEGER NOT NULL DEFAULT 0;
 ALTER TABLE egress_events ADD COLUMN duration_ms INTEGER NOT NULL DEFAULT 0;
 UPDATE egress_events SET definitive=1 WHERE state IN ('RESIDENTIAL','NO_SPLIT');
 CREATE TABLE egress_metrics(id INTEGER PRIMARY KEY CHECK(id=1),since INTEGER NOT NULL,baseline_inputs INTEGER NOT NULL,registered INTEGER NOT NULL DEFAULT 0,completed INTEGER NOT NULL DEFAULT 0,first_assessments INTEGER NOT NULL DEFAULT 0,definitive INTEGER NOT NULL DEFAULT 0,lease_recoveries INTEGER NOT NULL DEFAULT 0,unhealthy_deleted INTEGER NOT NULL DEFAULT 0);
 INSERT INTO egress_metrics(id,since,baseline_inputs) SELECT 1,CAST(unixepoch('subsec')*1000 AS INTEGER),count(*) FROM egress_jobs;
 CREATE TRIGGER egress_metric_ingest AFTER INSERT ON egress_jobs BEGIN UPDATE egress_metrics SET registered=registered+1 WHERE id=1; END;
 CREATE TRIGGER egress_metric_delete AFTER UPDATE OF state ON egress_jobs WHEN OLD.state!='DELETED' AND NEW.state='DELETED' AND NEW.reason='confirmed_unhealthy' BEGIN UPDATE egress_metrics SET unhealthy_deleted=unhealthy_deleted+1 WHERE id=1; END;
 CREATE TRIGGER egress_retry_revived AFTER UPDATE OF reason ON egress_jobs WHEN OLD.reason IN ('retention_expired','ordinary_delete') AND NEW.reason='new_input' BEGIN UPDATE egress_jobs SET retry_streak=0 WHERE hash=NEW.hash; END;
 CREATE INDEX egress_partial ON egress_jobs(state) WHERE json_extract(decision,'$.route')='PARTIAL_DIFFERENCE';
 PRAGMA user_version=30;`)
	if err != nil {
		return err
	}
	if err = backfillStagedEgress(tx); err != nil {
		return err
	}
	return tx.Commit()
}

type egressMetrics struct {
	Since            int64 `json:"since"`
	BaselineInputs   int64 `json:"baseline_inputs"`
	Registered       int64 `json:"registered"`
	Completed        int64 `json:"completed"`
	FirstAssessments int64 `json:"first_assessments"`
	Definitive       int64 `json:"definitive"`
	LeaseRecoveries  int64 `json:"lease_recoveries"`
	UnhealthyDeleted int64 `json:"unhealthy_deleted"`
}

func egressMetricState(ctx context.Context, q databaseQuery) (egressMetrics, error) {
	var m egressMetrics
	err := q.QueryRowContext(ctx, "SELECT since,baseline_inputs,registered,completed,first_assessments,definitive,lease_recoveries,unhealthy_deleted FROM egress_metrics WHERE id=1").Scan(&m.Since, &m.BaselineInputs, &m.Registered, &m.Completed, &m.FirstAssessments, &m.Definitive, &m.LeaseRecoveries, &m.UnhealthyDeleted)
	return m, err
}

// Keep first deep retries and credible health failures prompt. Repeated
// uncertainty backs off without deleting raw bytes or changing its verdict.
func egressRetryDelay(hash, state, health string, accepted bool, streak int, interval int64, fastRetry bool) int64 {
	if accepted && state == "RESIDENTIAL" {
		return interval
	}
	if accepted && health == "unhealthy" {
		return 30000
	}
	if fastRetry && streak <= 1 {
		return 30000
	}
	base, ceiling := int64(300000), int64(3600000)
	if state == "UNSUPPORTED" {
		base, ceiling = 3600000, 21600000
	}
	if accepted {
		base, ceiling = 30000, interval
	}
	delay := min(ceiling, base<<min(max(streak-1, 0), 6))
	// Stable bounded jitter avoids synchronized retry storms, without delaying
	// accepted health schedules, first retries or confirmed-negative spacing.
	if delay >= 300000 && !accepted && len(hash) >= 2 {
		delay += int64(hash[0]^hash[1]) * delay / 2550
	}
	return min(delay, ceiling)
}
