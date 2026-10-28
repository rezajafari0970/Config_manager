package schedule

import (
	"crypto/rand"
	"errors"
	"math/big"
	"time"
)

const MaxInterval = 30 * 24 * time.Hour

// Next samples uniformly in the inclusive interval. Callers persist this deadline;
// restarting a process must not draw a new time for an already scheduled job.
func Next(now time.Time, min, max time.Duration) (time.Time, error) {
	if min <= 0 || max < min || max > MaxInterval {
		return time.Time{}, errors.New("invalid interval: require 0 < min <= max <= 30 days")
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max-min)+1))
	if err != nil {
		return time.Time{}, err
	}
	return now.Add(min + time.Duration(n.Int64())), nil
}
