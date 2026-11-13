package panel

import "context"

// Counts describe one committed snapshot. A raw version, collector identity,
// emitted template and completed attempt are deliberately different units.
// No rolling event log is used to reconstruct lifetime inventory.
func egressAccounting(ctx context.Context, q databaseQuery) (map[string]any, error) {
	now := readStamp(ctx)
	var total, paused, running, due, scheduled, first, assessed, retained int
	e := q.QueryRowContext(ctx, `SELECT count(*),
 coalesce(sum(paused!=0),0),
 coalesce(sum(paused=0 AND token!='' AND lease_until>?),0),
 coalesce(sum(paused=0 AND (token='' OR lease_until<=?) AND next_run<=?),0),
 coalesce(sum(paused=0 AND (token='' OR lease_until<=?) AND next_run>?),0),
 coalesce(sum(tested=0),0),coalesce(sum(tested>0),0),coalesce(sum(ever_accepted=1),0)
 FROM egress_jobs WHERE state!='DELETED'`, now, now, now, now, now).Scan(&total, &paused, &running, &due, &scheduled, &first, &assessed, &retained)
	if e != nil {
		return nil, e
	}
	var current, currentIDs, currentActive, staged, missing, stagedMissing, deleted int
	e = q.QueryRowContext(ctx, `SELECT
 (SELECT count(DISTINCT raw_hash) FROM memberships),
 (SELECT count(DISTINCT artifact_id) FROM memberships),
 (SELECT count(DISTINCT m.raw_hash) FROM memberships m JOIN egress_jobs j ON j.hash=m.raw_hash WHERE j.state!='DELETED'),
 (SELECT count(DISTINCT hash) FROM staged),
 (SELECT count(DISTINCT raw_hash) FROM memberships m WHERE NOT EXISTS(SELECT 1 FROM egress_jobs j WHERE j.hash=m.raw_hash) AND NOT EXISTS(SELECT 1 FROM input_exclusions x WHERE x.identity=m.artifact_id)),
 (SELECT count(DISTINCT hash) FROM staged s WHERE NOT EXISTS(SELECT 1 FROM egress_jobs j WHERE j.hash=s.hash) AND NOT EXISTS(SELECT 1 FROM input_exclusions x WHERE x.identity=s.identity)),
 (SELECT count(*) FROM egress_jobs WHERE state='DELETED')`).Scan(&current, &currentIDs, &currentActive, &staged, &missing, &stagedMissing, &deleted)
	if e != nil {
		return nil, e
	}
	var eligible, known, published, templates, identities int
	e = q.QueryRowContext(ctx, `SELECT
 (SELECT count(*) FROM egress_eligible),
 (SELECT count(*) FROM egress_geo_eligible),
 (SELECT count(*) FROM egress_export_eligible),
 (SELECT count(DISTINCT candidate_hash) FROM egress_export_eligible),
 (SELECT count(DISTINCT o.identity) FROM egress_owners o JOIN egress_export_eligible e ON e.hash=o.hash)`).Scan(&eligible, &known, &published, &templates, &identities)
	if e != nil {
		return nil, e
	}
	return map[string]any{
		"at": now, "unit": "raw_sha256", "live_versions": total, "deleted_versions": deleted,
		"current_versions": current, "current_identities": currentIDs, "current_live_versions": currentActive,
		"archive_versions": total - currentActive, "staged_versions": staged,
		"coverage_missing": missing, "staged_coverage_missing": stagedMissing,
		"awaiting_first": first, "assessed_versions": assessed,
		"queue": map[string]int{"paused": paused, "running": running, "due": due, "scheduled": scheduled},
		"publication": map[string]int{"retained": retained, "awaiting_proof": retained - eligible,
			"eligible": eligible, "country_unknown": eligible - known, "country_filtered": known - published,
			"published": published, "distinct_templates": templates, "identities": identities},
	}, nil
}
