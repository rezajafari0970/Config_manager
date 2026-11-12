package panel

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/egress"
	"net/http"
	"sort"
)

func (s *Store) migrateV27() error {
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var v int
	if e = tx.QueryRow("PRAGMA user_version").Scan(&v); e != nil {
		return e
	}
	if v >= 27 {
		return nil
	}
	_, e = tx.Exec(`
 CREATE TABLE IF NOT EXISTS egress_country_policy(id INTEGER PRIMARY KEY CHECK(id=1),revision INTEGER NOT NULL,allowed TEXT NOT NULL,updated INTEGER NOT NULL);
 CREATE TABLE IF NOT EXISTS egress_geo_results(hash TEXT PRIMARY KEY REFERENCES egress_jobs(hash) ON DELETE CASCADE,complete INTEGER NOT NULL,status TEXT NOT NULL,checked_at INTEGER NOT NULL,valid_until INTEGER NOT NULL,summary TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS egress_countries(hash TEXT NOT NULL REFERENCES egress_jobs(hash) ON DELETE CASCADE,country TEXT NOT NULL,first_seen INTEGER NOT NULL,last_seen INTEGER NOT NULL,PRIMARY KEY(hash,country));
 CREATE INDEX IF NOT EXISTS egress_country_recent ON egress_countries(hash,last_seen,country);
 CREATE VIEW IF NOT EXISTS egress_geo_eligible AS SELECT e.* FROM egress_eligible e JOIN egress_geo_results g ON g.hash=e.hash WHERE g.complete=1 AND g.valid_until>CAST(unixepoch('subsec')*1000 AS INTEGER) AND EXISTS(SELECT 1 FROM egress_countries c WHERE c.hash=e.hash AND c.last_seen>CAST(unixepoch('subsec')*1000 AS INTEGER)-86400000);
 CREATE VIEW IF NOT EXISTS egress_export_eligible AS SELECT e.* FROM egress_geo_eligible e WHERE NOT EXISTS(SELECT 1 FROM egress_countries c WHERE c.hash=e.hash AND c.last_seen>CAST(unixepoch('subsec')*1000 AS INTEGER)-86400000 AND c.country NOT IN(SELECT value FROM json_each((SELECT allowed FROM egress_country_policy WHERE id=1))));

 UPDATE egress_jobs SET next_run=0 WHERE ever_accepted=1 AND state!='DELETED';
 PRAGMA user_version=27;`)
	if e != nil {
		return e
	}
	var column int
	if e = tx.QueryRow("SELECT count(*) FROM pragma_table_info('egress_jobs') WHERE name='deep_completed'").Scan(&column); e != nil {
		return e
	}
	if column == 0 {
		if _, e = tx.Exec("ALTER TABLE egress_jobs ADD COLUMN deep_completed INTEGER NOT NULL DEFAULT 0"); e != nil {
			return e
		}
	}
	allowed, _ := json.Marshal(egress.CountryCodes())
	if _, e = tx.Exec("INSERT OR IGNORE INTO egress_country_policy VALUES(1,1,?,?)", string(allowed), egress.Now()); e != nil {
		return e
	}
	return tx.Commit()
}

func saveEgressCountries(ctx context.Context, tx *sql.Tx, hash string, r egress.Report, d egress.Decision, now int64) error {
	g := egress.Countries(r, d, now)
	b, _ := json.Marshal(g)
	if _, e := tx.ExecContext(ctx, `INSERT INTO egress_geo_results VALUES(?,?,?,?,?,?) ON CONFLICT(hash) DO UPDATE SET complete=excluded.complete,status=excluded.status,checked_at=excluded.checked_at,valid_until=excluded.valid_until,summary=excluded.summary`, hash, g.Complete, g.Status, now, g.ValidUntil, string(b)); e != nil {
		return e
	}
	for _, country := range g.Countries {
		if _, e := tx.ExecContext(ctx, `INSERT INTO egress_countries VALUES(?,?,?,?) ON CONFLICT(hash,country) DO UPDATE SET last_seen=excluded.last_seen`, hash, country, now, now); e != nil {
			return e
		}
	}
	_, e := tx.ExecContext(ctx, "DELETE FROM egress_countries WHERE hash=? AND last_seen<=?", hash, now-egress.CountryWindow)
	return e
}

type countryPolicy struct {
	Revision  int64    `json:"revision"`
	Countries []string `json:"countries"`
}

func (m egressModule) countries(w http.ResponseWriter, r *http.Request) {
	var b countryPolicy
	if decode(r, &b) != nil || len(b.Countries) > 250 {
		failure(w, ErrInvalid)
		return
	}
	seen := map[string]bool{}
	for _, c := range b.Countries {
		if !egress.ValidCountry(c) || seen[c] {
			failure(w, ErrInvalid)
			return
		}
		seen[c] = true
	}
	sort.Strings(b.Countries)
	if b.Countries == nil {
		b.Countries = []string{}
	}
	raw, _ := json.Marshal(b.Countries)
	tx, e := m.s.store.db.BeginTx(r.Context(), nil)
	if e != nil {
		failure(w, e)
		return
	}
	defer tx.Rollback()
	result, e := tx.ExecContext(r.Context(), "UPDATE egress_country_policy SET allowed=?,revision=revision+1,updated=? WHERE id=1 AND revision=?", string(raw), egress.Now(), b.Revision)
	if e != nil {
		failure(w, e)
		return
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		failure(w, ErrConflict)
		return
	}
	info, e := countryState(r.Context(), tx)
	if e != nil {
		failure(w, e)
		return
	}
	if e = tx.Commit(); e != nil {
		failure(w, e)
		return
	}
	b.Revision++
	respond(w, 200, map[string]any{"revision": b.Revision, "countries": b.Countries, "published": info["published"], "country": info, "saved_at": egress.Now()})

}

func countryState(ctx context.Context, q databaseQuery) (map[string]any, error) {
	var p countryPolicy
	var raw string
	if e := q.QueryRowContext(ctx, "SELECT revision,allowed FROM egress_country_policy WHERE id=1").Scan(&p.Revision, &raw); e != nil {
		return nil, e
	}
	if e := json.Unmarshal([]byte(raw), &p.Countries); e != nil {
		return nil, e
	}
	counts := map[string]int{}
	rows, e := q.QueryContext(ctx, "SELECT c.country,count(*) FROM egress_countries c JOIN egress_geo_eligible g ON g.hash=c.hash WHERE c.last_seen>? GROUP BY c.country", egress.Now()-egress.CountryWindow)
	if e != nil {
		return nil, e
	}
	for rows.Next() {
		var c string
		var n int
		if e = rows.Scan(&c, &n); e != nil {
			rows.Close()
			return nil, e
		}
		counts[c] = n
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	var known, shared, published, route int
	if e = q.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM egress_geo_eligible),(SELECT count(*) FROM egress_export_eligible),(SELECT count(*) FROM egress_eligible),(SELECT count(*) FROM (SELECT c.hash FROM egress_countries c JOIN egress_geo_eligible g ON g.hash=c.hash WHERE c.last_seen>? GROUP BY c.hash HAVING count(*)>1))`, egress.Now()-egress.CountryWindow).Scan(&known, &published, &route, &shared); e != nil {
		return nil, e
	}
	return map[string]any{"policy": p, "codes": egress.CountryCodes(), "counts": counts, "known": known, "shared": shared, "published": published, "unknown": route - known, "window_ms": egress.CountryWindow}, nil
}

func entryCountries(ctx context.Context, q databaseQuery, hash string) ([]string, error) {
	rows, e := q.QueryContext(ctx, "SELECT country FROM egress_countries WHERE hash=? AND last_seen>? ORDER BY country", hash, egress.Now()-egress.CountryWindow)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var c string
		if e = rows.Scan(&c); e != nil {
			return nil, e
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
