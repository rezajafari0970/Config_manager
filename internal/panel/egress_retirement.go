package panel

import (
	"context"
	"database/sql"
	"github.com/rezajafari0970/Config_manager/internal/egress"
)

// Only measured loss can retire an input. UNKNOWN, deadlines, no-fill and
// provider outages never become a fabricated negative. Each hit must be a
// separate, revision-fenced report started after the previous measurement.
func retirementReason(d egress.Decision, health string, accepted bool) string {
	if health == "unhealthy" {
		return "confirmed_ads_delivery_unhealthy"
	}
	if accepted && d.State == "NO_SPLIT" && d.Negative {
		return "confirmed_ads_route_lost"
	}
	return ""
}
func (s *Store) retirement(ctx context.Context, tx *sql.Tx, hash, reason string, eligible bool, started, finished, now int64) (bool, int, error) {
	var old string
	var hits int
	var first, last int64
	err := tx.QueryRowContext(ctx, "SELECT reason,hits,first_at,at FROM egress_retirements WHERE hash=?", hash).Scan(&old, &hits, &first, &last)
	if err != nil && err != sql.ErrNoRows {
		return false, 0, err
	}
	if eligible {
		if _, err = tx.ExecContext(ctx, "DELETE FROM egress_retirements WHERE hash=?", hash); err != nil {
			return false, 0, err
		}
		return false, 0, nil
	}
	if first > 0 && now-first > 600000 {
		// Confirmation evidence expires; identity-wide withdrawal does not.
		if _, err = tx.ExecContext(ctx, "UPDATE egress_retirements SET hits=0,first_at=0,at=0 WHERE hash=?", hash); err != nil {
			return false, 0, err
		}
		hits, first, last = 0, 0, 0
	}
	if reason == "" {
		return false, hits, nil
	}
	if old != reason {
		hits = 0
		last = 0
		first = 0
	}
	if last == 0 || started >= last && finished-last >= 30000 {
		hits++
		if first == 0 {
			first = finished
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO egress_retirements(hash,reason,hits,at,first_at) VALUES(?,?,?,?,?) ON CONFLICT(hash) DO UPDATE SET reason=excluded.reason,hits=excluded.hits,at=excluded.at,first_at=excluded.first_at`, hash, reason, hits, finished, first)
	}
	return hits >= 3, hits, err
}
