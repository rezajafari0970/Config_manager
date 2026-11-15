package panel

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// SQLite's wall clock is statement-scoped, even inside a read transaction.
// Inline the authoritative view definitions with one presentation timestamp so
// counts, country membership and entries cannot disagree at a TTL boundary.
type egressSnapshotReader struct {
	q      databaseQuery
	prefix string
}

func (r egressSnapshotReader) QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error) {
	return r.q.QueryContext(ctx, r.prefix+q, args...)
}
func (r egressSnapshotReader) QueryRowContext(ctx context.Context, q string, args ...any) *sql.Row {
	return r.q.QueryRowContext(ctx, r.prefix+q, args...)
}
func freezeEgress(ctx context.Context, q databaseQuery) (databaseQuery, error) {
	parts := []string{}
	for _, name := range []string{"egress_inventory", "egress_work", "egress_eligible", "egress_geo_eligible", "egress_export_eligible"} {
		var definition string
		if err := q.QueryRowContext(ctx, "SELECT sql FROM sqlite_master WHERE type='view' AND name=?", name).Scan(&definition); err != nil {
			return nil, err
		}
		match := regexp.MustCompile(`(?i)\bAS\s+(?:SELECT|WITH)\b`).FindStringIndex(definition)
		index := -1
		if match != nil {
			index = match[0]
		}
		if index < 0 {
			return nil, fmt.Errorf("invalid eligibility view: %s", name)
		}
		body := strings.ReplaceAll(strings.TrimSpace(definition[index+2:]), "CAST(unixepoch('subsec')*1000 AS INTEGER)", strconv.FormatInt(readStamp(ctx), 10))
		if name == "egress_inventory" {
			// Presentation must start with current/retained identities, just like
			// claims. Preserve the authoritative predicate verbatim; the sparse
			// candidate set is only a query-plan bound, never eligibility proof.
			const join = "FROM egress_unique u JOIN egress_jobs j ON j.hash=u.hash"
			if strings.Count(body, join) != 1 {
				return nil, fmt.Errorf("unrecognized inventory scope")
			}
			body = strings.Replace(body, join, "FROM ("+egressScopeCandidates+") scope CROSS JOIN egress_unique u ON u.hash=scope.hash CROSS JOIN egress_jobs j ON j.hash=u.hash", 1)
		}
		// Avoid copying every large diagnostic report into a temporary table.
		parts = append(parts, name+" AS NOT MATERIALIZED ("+strings.TrimSuffix(strings.TrimSpace(body), ";")+")")
	}
	return egressSnapshotReader{q, "WITH " + strings.Join(parts, ",") + " "}, nil
}
