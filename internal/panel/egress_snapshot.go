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
	return r.q.QueryContext(ctx, r.query(q), args...)
}
func (r egressSnapshotReader) QueryRowContext(ctx context.Context, q string, args ...any) *sql.Row {
	return r.q.QueryRowContext(ctx, r.query(q), args...)
}

// Scope only the caller's inventory/work reads. Never shadow the authoritative
// views inside eligibility: correlated eligibility lookups must keep their hash
// index. Globally replacing those views previously multiplied the sparse scan
// for every candidate and country lookup.
var egressPresentationScope = regexp.MustCompile(`\begress_(inventory|work)\b`)

func (r egressSnapshotReader) query(q string) string {
	if !egressPresentationScope.MatchString(q) {
		return r.prefix + q
	}
	const candidates = `SELECT u.hash FROM (SELECT DISTINCT artifact_id FROM memberships) m CROSS JOIN egress_unique u ON u.identity=m.artifact_id
 UNION SELECT hash FROM egress_jobs WHERE ever_accepted=1
 UNION SELECT u.hash FROM egress_iran_guard g CROSS JOIN egress_unique u ON u.identity=g.identity WHERE g.state='QUARANTINED'`
	scoped := `egress_presentation_scope(hash) AS MATERIALIZED (` + candidates + `),
 egress_presentation_inventory AS NOT MATERIALIZED (SELECT j.* FROM egress_presentation_scope c CROSS JOIN egress_inventory j ON j.hash=c.hash),
 egress_presentation_work AS NOT MATERIALIZED (SELECT j.* FROM egress_presentation_scope c CROSS JOIN egress_work j ON j.hash=c.hash),`
	q = egressPresentationScope.ReplaceAllString(q, "egress_presentation_$1")
	return "WITH " + scoped + strings.TrimPrefix(r.prefix, "WITH ") + q
}

func freezeEgress(ctx context.Context, q databaseQuery) (databaseQuery, error) {
	parts := []string{}
	for _, name := range []string{"egress_eligible", "egress_geo_eligible", "egress_export_eligible"} {
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
		parts = append(parts, name+" AS ("+strings.TrimSuffix(strings.TrimSpace(body), ";")+")")
	}
	return egressSnapshotReader{q, "WITH " + strings.Join(parts, ",") + " "}, nil
}
