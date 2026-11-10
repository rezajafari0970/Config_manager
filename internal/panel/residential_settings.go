package panel

import (
	"context"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/convert"
	"github.com/rezajafari0970/Config_manager/internal/residential"
	"io"
	"os"
	"time"
)

// ConfigureResidential is a trusted operator boundary, not a public proof API.
// Only an administrator-controlled local file can add evidence capabilities or
// provider independence assertions. Any change fences in-flight assessments.
func (s *Store) ConfigureResidential(ctx context.Context, p residential.Policy, t residential.Trust) error {
	if p.Validate() != nil || t.Validate() != nil {
		return ErrInvalid
	}
	pb, _ := json.Marshal(p)
	tb, _ := json.Marshal(t)
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var oldPolicy, oldTrust, oldTemplate string
	if e = tx.QueryRowContext(ctx, "SELECT policy_hash,trust_hash,template_version FROM residential_settings WHERE id=1").Scan(&oldPolicy, &oldTrust, &oldTemplate); e != nil {
		return e
	}
	if oldPolicy == p.Digest() && oldTrust == t.Digest() && oldTemplate == convert.Version {
		return nil
	}
	if _, e = tx.ExecContext(ctx, "UPDATE residential_settings SET policy=?,policy_hash=?,trust=?,trust_hash=?,template_version=?,revision=revision+1,generation=generation+1,updated=? WHERE id=1", string(pb), p.Digest(), string(tb), t.Digest(), convert.Version, time.Now().UnixMilli()); e != nil {
		return e
	}
	if _, e = tx.ExecContext(ctx, "UPDATE residential_records SET state=CASE WHEN ever_accepted=1 THEN 'QUARANTINED' ELSE 'UNKNOWN' END,reason='policy_template_or_trust_changed',valid_until=0,revision=revision+1,token='',lease_until=0,next_run=0 WHERE state!='DELETED'"); e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Store) LoadResidentialConfiguration(ctx context.Context, path string) error {
	p, t, e := residentialSettings(ctx, s.db)
	if e != nil {
		return e
	}
	if path != "" {
		f, e := os.Open(path)
		if e != nil {
			return e
		}
		defer f.Close()
		info, e := f.Stat()
		if e != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0007 != 0 {
			return ErrInvalid
		}
		var c struct {
			Policy residential.Policy `json:"policy"`
			Trust  residential.Trust  `json:"trust"`
		}
		d := json.NewDecoder(io.LimitReader(f, 65537))
		d.DisallowUnknownFields()
		if d.Decode(&c) != nil {
			return ErrInvalid
		}
		if d.Decode(new(any)) != io.EOF {
			return ErrInvalid
		}
		p, t = c.Policy, c.Trust
	}
	return s.ConfigureResidential(ctx, p, t)
}
