package panel

import (
	"context"
	"database/sql"
	"errors"
	"github.com/rezajafari0970/Config_manager/internal/probe"
)

// Successful model-specific rules are tried first on later samples, but never
// grant another sample a pass. Every candidate still needs its own fresh test.
func (s *Store) PreferredRule(ctx context.Context, model string) (string, error) {
	if !validDigest(model) {
		return "", ErrInvalid
	}
	var rule string
	e := s.db.QueryRowContext(ctx, `SELECT rule FROM probe_rule_evidence WHERE model=? AND version=? AND passed>0 ORDER BY CAST(passed AS REAL)/(passed+failed) DESC,passed DESC,last_tested DESC LIMIT 1`, model, probe.Version).Scan(&rule)
	if errors.Is(e, sql.ErrNoRows) {
		return "", nil
	}
	return rule, e
}
