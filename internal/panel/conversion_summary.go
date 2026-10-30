package panel

import (
	"context"
	"github.com/rezajafari0970/Config_manager/internal/convert"
	"github.com/rezajafari0970/Config_manager/internal/repair"
	"strings"
	"time"
)

type SourceConversionSummary struct {
	Total     int `json:"total"`
	Ready     int `json:"ready"`
	Connected int `json:"connected"`
	Repaired  int `json:"repaired"`
	Pending   int `json:"pending"`
	Blocked   int `json:"blocked"`
}

// Use the same representative ranking and source membership as Output, then
// aggregate canonical identities, not every historical raw representation.
func (s *Store) SourceConversionSummaries(ctx context.Context, sources []Source) (map[string]SourceConversionSummary, error) {
	out := map[string]SourceConversionSummary{}
	if len(sources) == 0 {
		return out, nil
	}
	now := time.Now().UnixMilli()
	end := strings.Index(conversionQuery, "\n SELECT id,raw_hash")
	q := conversionQuery[:end]
	q = strings.Replace(q, "SELECT a.id,m.raw_hash", "SELECT m.source_id,r.version AS result_version,r.rule AS result_rule,a.id,m.raw_hash", 1)
	q = strings.Replace(q, "PARTITION BY a.id", "PARTITION BY m.source_id,a.id", 1)
	marks := make([]string, len(sources))
	args := []any{convert.Version, now, convert.Version, now, now, now}
	for i, source := range sources {
		marks[i] = "?"
		args = append(args, source.ID)
		out[source.ID] = SourceConversionSummary{}
	}
	q = strings.Replace(q, "/* source filter */", "AND m.source_id IN ("+strings.Join(marks, ",")+")", 1)
	q += ` SELECT source_id,count(*),
 sum(CASE WHEN available THEN 1 ELSE 0 END),
 sum(CASE WHEN available AND connection='passed' AND valid_until>? THEN 1 ELSE 0 END),
 sum(CASE WHEN available AND result_rule LIKE ? THEN 1 ELSE 0 END),
 sum(CASE WHEN coalesce(result_version,'')!=? OR state IN ('queued','running') THEN 1 ELSE 0 END),
 sum(CASE WHEN coalesce(result_version,'')=? AND state NOT IN ('queued','running') AND NOT coalesce(available,0) THEN 1 ELSE 0 END)
 FROM choices WHERE choice=1 GROUP BY source_id`
	args = append(args, now, convert.Version+"."+repair.RulesVersion+"%", convert.Version, convert.Version)
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var v SourceConversionSummary
		if err = rows.Scan(&id, &v.Total, &v.Ready, &v.Connected, &v.Repaired, &v.Pending, &v.Blocked); err != nil {
			return nil, err
		}
		out[id] = v
	}
	return out, rows.Err()
}
