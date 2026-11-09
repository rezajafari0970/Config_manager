package panel

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"github.com/rezajafari0970/Config_manager/internal/transient"
	"io"
	"time"

	"github.com/rezajafari0970/Config_manager/internal/collect"
)

func (s *Store) migrateV10() error {
	var version int
	if e := s.db.QueryRow("PRAGMA user_version").Scan(&version); e != nil {
		return e
	}
	if version >= 10 {
		return nil
	}
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	// Also tolerate additive columns retained by an earlier partial upgrade.
	rows, e := tx.Query("PRAGMA table_info(sources)")
	if e != nil {
		return e
	}
	columns := map[string]bool{}
	for rows.Next() {
		var cid, notnull, pk int
		var name, typ string
		var def any
		if e = rows.Scan(&cid, &name, &typ, &notnull, &def, &pk); e != nil {
			rows.Close()
			return e
		}
		columns[name] = true
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, name := range []string{"stored_count", "suppressed_count"} {
		if !columns[name] {
			if _, e = tx.Exec("ALTER TABLE sources ADD COLUMN " + name + " INTEGER NOT NULL DEFAULT 0"); e != nil {
				return e
			}
		}
	}
	if _, e = tx.Exec("PRAGMA user_version=10"); e != nil {
		return e
	}
	return tx.Commit()
}

type acquisitionReader struct {
	ctx context.Context
	r   io.Reader
}

func (r acquisitionReader) Read(b []byte) (int, error) {
	if e := r.ctx.Err(); e != nil {
		return 0, e
	}
	return r.r.Read(b)
}

// RestoreSource is an explicit, source-scoped undo of deletion controls. It does
// not restore old output/proofs: ordinary collection and validation must run again.
// The response is read outside SQLite transactions to preserve vault lock order.
func (s *Store) RestoreSource(ctx context.Context, id string, revision int64) (int64, error) {
	source, e := s.Get(ctx, id)
	if e != nil {
		return 0, e
	}
	if source.Revision != revision || !source.Enabled {
		return 0, ErrConflict
	}
	ref, e := s.Response(ctx, id)
	if e != nil {
		return 0, e
	}
	body, e := s.vault.Open(ref.Hash)
	if e != nil {
		return 0, e
	}
	defer body.Close()
	spool, spoolPath, e := transient.NewFile()
	if e != nil {
		return 0, e
	}
	defer spool.Close()
	if _, e = io.Copy(spool, acquisitionReader{ctx, body}); e != nil {
		return 0, e
	}
	ids := map[string]bool{}
	e = collect.NewRegistry().Extract(ctx, spoolPath, func(ctx context.Context, r io.Reader, kind string) error {
		// Match collector identity exactly, including the one-MiB inspection boundary.
		h := sha256.New()
		r = io.TeeReader(acquisitionReader{ctx, r}, h)
		sample, err := io.ReadAll(io.LimitReader(r, 1024*1024))
		if err != nil {
			return err
		}
		rest, err := io.Copy(io.Discard, r)
		if err != nil {
			return err
		}
		ids[collect.Fingerprint(sample, rest == 0, kind, hex.EncodeToString(h.Sum(nil)))] = true
		return nil
	})
	if e != nil && !collect.IsContentError(e) {
		return 0, e
	}
	if len(ids) == 0 {
		return 0, ErrInvalid
	}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return 0, e
	}
	defer tx.Rollback()
	stamp := time.Now().UnixMilli()
	// The latest complete response may have changed while it was being parsed.
	var current bool
	e = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM sources s JOIN responses r ON r.source_id=s.id WHERE s.id=? AND s.revision=? AND s.enabled=1 AND r.hash=? AND r.expires>?)", id, revision, ref.Hash, stamp).Scan(&current)
	if e != nil {
		return 0, e
	}
	if !current {
		return 0, ErrConflict
	}
	var restored int64
	for ident := range ids {
		res, err := tx.ExecContext(ctx, "DELETE FROM config_controls WHERE id=? AND action='deleted' AND expires>?", ident, stamp)
		if err != nil {
			return 0, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return 0, err
		}
		restored += n
	}
	// Fence pre-restore workers. A later manual deletion remains authoritative.
	if _, e = tx.ExecContext(ctx, "UPDATE sources SET revision=revision+1,lease_token='',lease_until=0,next_run=?,state='queued',error_code='',etag='',modified='' WHERE id=?", time.Now().UnixNano(), id); e != nil {
		return 0, e
	}
	if _, e = tx.ExecContext(ctx, "DELETE FROM staged WHERE source_id=?", id); e != nil {
		return 0, e
	}
	if e = syncConversions(ctx, tx); e != nil {
		return 0, e
	}
	return restored, tx.Commit()
}
