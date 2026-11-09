package panel

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"
)

type readQuerier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type snapshotKey struct{ store *Store }
type readTimeKey struct{}
type snapshotReader struct {
	tx    *sql.Tx
	stamp int64
}

// Freeze the known wall-clock expression in presentation queries as well as
// their explicit expiry parameters. A response cannot cross a TTL boundary
// halfway through and call the same input both present and absent.
func (r snapshotReader) query(q string) string {
	return strings.ReplaceAll(q, "CAST(unixepoch('subsec')*1000 AS INTEGER)", strconv.FormatInt(r.stamp, 10))
}
func (r snapshotReader) QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error) {
	return r.tx.QueryContext(ctx, r.query(q), args...)
}
func (r snapshotReader) QueryRowContext(ctx context.Context, q string, args ...any) *sql.Row {
	return r.tx.QueryRowContext(ctx, r.query(q), args...)
}
func readStamp(ctx context.Context) int64 {
	if stamp, ok := ctx.Value(readTimeKey{}).(int64); ok {
		return stamp
	}
	return time.Now().UnixMilli()
}
func (s *Store) read(ctx context.Context) readQuerier {
	if r, ok := ctx.Value(snapshotKey{s}).(snapshotReader); ok {
		return r
	}
	return s.reader()
}

// HTTP summaries share one committed snapshot while writers remain independent.
// No claims, mutations, blob I/O or long exports may run inside this context.
func (s *Store) readSnapshot(ctx context.Context) (context.Context, func(), error) {
	tx, e := s.reader().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return ctx, func() {}, e
	}
	stamp := time.Now().UnixMilli()
	ctx = context.WithValue(ctx, readTimeKey{}, stamp)
	return context.WithValue(ctx, snapshotKey{s}, snapshotReader{tx, stamp}), func() { _ = tx.Rollback() }, nil
}
