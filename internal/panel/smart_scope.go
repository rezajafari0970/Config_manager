package panel

import (
	"context"
	"database/sql"
	"regexp"

	"github.com/rezajafari0970/Config_manager/internal/probe"
	"github.com/rezajafari0970/Config_manager/internal/smart"
)

// SmartPipeline binds the common builder to one isolated publication repository.
// Collection and immutable content storage stay shared; no queue/cache/counter is
// shared across the three pipelines. Scope comes only from these constructors.
type SmartPipeline struct {
	*Store
	db     smartDB
	splash bool
	ads    bool
}

func (s *Store) Smart() *SmartPipeline { return &SmartPipeline{Store: s, db: smartDB{DB: s.db}} }
func (s *Store) SplashSmart() *SmartPipeline {
	return &SmartPipeline{Store: s, db: smartDB{DB: s.db, splash: true}, splash: true}
}
func (s *Store) AdsSmart() *SmartPipeline {
	return &SmartPipeline{Store: s, db: smartDB{DB: s.db, ads: true}, ads: true}
}
func (s *SmartPipeline) version() string {
	if s.ads {
		return smart.AdsVersion
	}
	return smart.Version
}
func (s *SmartPipeline) tokenMode(v string) string {
	if s.ads {
		return "ads-" + v
	}
	if s.splash {
		return "splash-" + v
	}
	return v
}
func (s *SmartPipeline) reader() smartDB {
	return smartDB{DB: s.Store.reader(), splash: s.splash, ads: s.ads}
}
func (s *SmartPipeline) path() string {
	if s.ads {
		return "/smart-ads"
	}
	if s.splash {
		return "/smart-splash"
	}
	return "/smart"
}

// Rewrite only fixed SQL identifiers, never values, credentials or user input.
// The vocabulary is closed and the regexp uses whole-word matches (one pass).
var smartIdentifiers = regexp.MustCompile(`\bsmart_(?:evidence|cache_members_source|cache_members|cache_hash|cache_owned|cache|rotation_state|rotation_queue|source_usage|refresh_needed|owned_members|members_source|members|heads_hash|heads|settings|worker|offers|inputs|ready|usage|due|ids)\b`)

func smartSQL(q string, splash bool) string {
	if !splash {
		return q
	}
	return smartIdentifiers.ReplaceAllString(q, "splash_$0")
}

func scopedSmartSQL(q string, splash, ads bool) string {
	if ads {
		return smartIdentifiers.ReplaceAllString(q, "ads_$0")
	}
	return smartSQL(q, splash)
}

type smartDB struct {
	*sql.DB
	splash bool
	ads    bool
}

func (d smartDB) QueryContext(c context.Context, q string, a ...any) (*sql.Rows, error) {
	return d.DB.QueryContext(c, scopedSmartSQL(q, d.splash, d.ads), a...)
}
func (d smartDB) QueryRowContext(c context.Context, q string, a ...any) *sql.Row {
	return d.DB.QueryRowContext(c, scopedSmartSQL(q, d.splash, d.ads), a...)
}
func (d smartDB) ExecContext(c context.Context, q string, a ...any) (sql.Result, error) {
	return d.DB.ExecContext(c, scopedSmartSQL(q, d.splash, d.ads), a...)
}
func (d smartDB) BeginTx(c context.Context, o *sql.TxOptions) (*smartTx, error) {
	t, e := d.DB.BeginTx(c, o)
	if e != nil {
		return nil, e
	}
	return &smartTx{Tx: t, splash: d.splash, ads: d.ads}, nil
}

type smartTx struct {
	*sql.Tx
	splash bool
	ads    bool
}

func (t *smartTx) QueryContext(c context.Context, q string, a ...any) (*sql.Rows, error) {
	return t.Tx.QueryContext(c, scopedSmartSQL(q, t.splash, t.ads), a...)
}
func (t *smartTx) QueryRowContext(c context.Context, q string, a ...any) *sql.Row {
	return t.Tx.QueryRowContext(c, scopedSmartSQL(q, t.splash, t.ads), a...)
}
func (t *smartTx) ExecContext(c context.Context, q string, a ...any) (sql.Result, error) {
	return t.Tx.ExecContext(c, scopedSmartSQL(q, t.splash, t.ads), a...)
}

type smartTransaction interface {
	conversionQuerier
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// Compatibility facade: existing ordinary Smart callers retain their scope.
func (s *Store) SmartSettings(c context.Context) (SmartSettings, error) {
	return s.Smart().SmartSettings(c)
}
func (s *Store) SaveSmartSettings(c context.Context, v SmartSettings) error {
	return s.Smart().SaveSmartSettings(c, v)
}
func (s *Store) ClaimSmart(c context.Context) (*probe.Job, error) { return s.Smart().ClaimSmart(c) }
func (s *Store) RenewSmart(c context.Context, j probe.Job) error  { return s.Smart().RenewSmart(c, j) }
func (s *Store) CompleteSmart(c context.Context, r probe.Result) error {
	return s.Smart().CompleteSmart(c, r)
}
func (s *Store) SmartWorker(c context.Context) (probe.WorkerStatus, error) {
	return s.Smart().SmartWorker(c)
}
func (s *Store) SetSmartWorker(c context.Context, v probe.WorkerStatus) error {
	return s.Smart().SetSmartWorker(c, v)
}
func (s *Store) smartBytes(c context.Context, v SmartEntry) ([]byte, error) {
	return s.Smart().smartBytes(c, v)
}
func (s *Store) smartNext(c context.Context) (*SmartEntry, []byte, int64, error) {
	return s.Smart().smartNext(c)
}
func (s *Store) retrySmartBlob(c context.Context, v SmartEntry) { s.Smart().retrySmartBlob(c, v) }
