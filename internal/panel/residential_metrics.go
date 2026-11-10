package panel

import (
	"context"
	"math"
	"sort"
	"time"
)

func (s *Store) residentialMetrics(ctx context.Context) (map[string]any, error) {
	rows, e := s.reader().QueryContext(ctx, "SELECT at,wall_ms,cpu_ms,rss_bytes FROM residential_metrics ORDER BY id DESC LIMIT 2000")
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	wall := []float64{}
	cpu := float64(0)
	var peak int64
	recent := 0
	now := time.Now().UnixMilli()
	for rows.Next() {
		var at, rss int64
		var w, c float64
		if e = rows.Scan(&at, &w, &c, &rss); e != nil {
			return nil, e
		}
		if w > 0 {
			wall = append(wall, w)
			cpu += c
			peak = max(peak, rss)
		}
		if at > now-300000 {
			recent++
		}
	}
	if e = rows.Err(); e != nil {
		return nil, e
	}
	sort.Float64s(wall)
	quantile := func(q float64) float64 {
		if len(wall) == 0 {
			return 0
		}
		return wall[max(0, int(math.Ceil(q*float64(len(wall))))-1)]
	}
	return map[string]any{"sample_count": len(wall), "p50_ms": quantile(.50), "p95_ms": quantile(.95), "configs_per_minute_last_5m": float64(recent) / 5, "mean_cpu_ms": cpu / float64(max(1, len(wall))), "peak_rss_bytes": peak, "scope": "server-side assessment; not client throughput"}, nil
}
