package panel

import "context"

// Counts use the same read snapshot and authoritative eligibility views as the
// panel and exports. Profile counts may overlap; identity totals never do.
func egressProfileSummary(ctx context.Context, q databaseQuery) (map[string]any, error) {
	var complete, partial, published int
	if err := q.QueryRowContext(ctx, `SELECT coalesce(sum(json_extract(decision,'$.ads_coverage')='ALL_TESTED'),0),coalesce(sum(json_extract(decision,'$.ads_coverage')='PARTIAL'),0),(SELECT count(*) FROM egress_export_eligible) FROM egress_eligible`).Scan(&complete, &partial, &published); err != nil {
		return nil, err
	}
	rows, err := q.QueryContext(ctx, `SELECT p.value,count(*) FROM egress_eligible e,json_each(e.decision,'$.qualified_profiles') p GROUP BY p.value`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := map[string]int{}
	for rows.Next() {
		var id string
		var n int
		if err = rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		counts[id] = n
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return map[string]any{"complete_tested_ads": complete, "partial_tested_ads": partial, "published": published, "profiles": counts, "profile_counts_overlap": true}, nil
}
