package panel

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/egress"
)

// Existing collector identities normalize URI fragments/JSON remarks and key
// order while preserving credentials and routing. Never blacklist an IP/source.
func (s *Store) migrateV31() error {
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var v int
	if e = tx.QueryRow("PRAGMA user_version").Scan(&v); e != nil {
		return e
	}
	if v >= 31 {
		return nil
	}
	_, e = tx.Exec(`CREATE TABLE IF NOT EXISTS egress_iran_policy(id INTEGER PRIMARY KEY CHECK(id=1),started INTEGER NOT NULL);
 INSERT OR IGNORE INTO egress_iran_policy VALUES(1,CAST(unixepoch('subsec')*1000 AS INTEGER));
 CREATE TABLE IF NOT EXISTS egress_iran_guard(
 identity TEXT PRIMARY KEY,state TEXT NOT NULL CHECK(state IN ('QUARANTINED','BLOCKED')),
 first_seen INTEGER NOT NULL,last_round INTEGER NOT NULL DEFAULT 0,positives INTEGER NOT NULL DEFAULT 0,
 followups INTEGER NOT NULL DEFAULT 0,last_checked INTEGER NOT NULL DEFAULT 0,
 first_evidence TEXT NOT NULL DEFAULT '{}',last_evidence TEXT NOT NULL DEFAULT '{}');
 CREATE INDEX IF NOT EXISTS egress_iran_state ON egress_iran_guard(state);
 INSERT OR IGNORE INTO egress_iran_guard(identity,state,first_seen)
 SELECT o.identity,'QUARANTINED',max(c.last_seen) FROM egress_countries c JOIN egress_owners o ON o.hash=c.hash JOIN egress_jobs j ON j.hash=c.hash
 WHERE c.country='IR' AND c.last_seen>CAST(unixepoch('subsec')*1000 AS INTEGER)-86400000 AND j.state!='DELETED' GROUP BY o.identity;
 DROP VIEW input_blocked;
 CREATE VIEW input_blocked AS SELECT hash FROM input_exclusions UNION SELECT hash FROM egress_jobs WHERE ever_accepted=1 AND NOT EXISTS(SELECT 1 FROM egress_eligible e WHERE e.hash=egress_jobs.hash)
 UNION SELECT o.hash FROM egress_owners o JOIN egress_iran_guard g ON g.identity=o.identity;
 UPDATE egress_jobs SET valid_until=min(valid_until,tested+600000),next_run=0 WHERE ever_accepted=1 AND state!='DELETED';
 UPDATE egress_jobs SET state='QUARANTINED',reason='ads_iran_quarantined',valid_until=0,next_run=0 WHERE state!='DELETED' AND hash IN(SELECT o.hash FROM egress_owners o JOIN egress_iran_guard g ON g.identity=o.identity);
 DELETE FROM responses WHERE source_id IN(SELECT o.source FROM egress_owners o JOIN egress_iran_guard g ON g.identity=o.identity);
 PRAGMA user_version=31;`)
	if e != nil {
		return e
	}
	var view string
	if e = tx.QueryRow("SELECT sql FROM sqlite_master WHERE name='egress_eligible'").Scan(&view); e != nil {
		return e
	}
	view += " AND j.tested>=(SELECT started FROM egress_iran_policy WHERE id=1) AND NOT EXISTS(SELECT 1 FROM egress_owners o JOIN egress_iran_guard g ON g.identity=o.identity WHERE o.hash=j.hash)"
	if _, e = tx.Exec("DROP VIEW egress_eligible;" + view); e != nil {
		return e
	}
	return tx.Commit()
}

type iranOutcome struct {
	Quarantined, Blocked bool
	Delay                int64
}

// Called only under the queue's writer reservation and validated lease. The
// sticky identity veto and exports are committed atomically with job completion.
func (s *Store) applyIran(ctx context.Context, tx *sql.Tx, r egress.Report, p egress.Plan, now int64) (iranOutcome, error) {
	out := iranOutcome{Delay: egress.IranInterval - 60000}
	signal := egress.Iran(r, p, r.Hash, now)
	proof := "{}"
	if signal != nil {
		b, _ := json.Marshal(signal)
		proof = string(b)
	}
	rows, e := tx.QueryContext(ctx, "SELECT DISTINCT identity FROM egress_owners WHERE hash=?", r.Hash)
	if e != nil {
		return out, e
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			rows.Close()
			return out, e
		}
		ids = append(ids, id)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return out, e
	}
	for _, id := range ids {
		if signal != nil {
			if _, e = tx.ExecContext(ctx, `INSERT OR IGNORE INTO egress_iran_guard(identity,state,first_seen) VALUES(?,'QUARANTINED',?)`, id, now); e != nil {
				return out, e
			}
		}
		var state string
		var last int64
		var positives, followups int
		e = tx.QueryRowContext(ctx, "SELECT state,last_round,positives,followups FROM egress_iran_guard WHERE identity=?", id).Scan(&state, &last, &positives, &followups)
		if e == sql.ErrNoRows {
			continue
		}
		if e != nil {
			return out, e
		}
		out.Quarantined = true
		// Concurrent aliases and waves from one job never count as a second round.
		if state != "BLOCKED" && (last == 0 || r.Started >= last+egress.IranSpacing) {
			if last != 0 {
				followups++
			}
			if signal != nil {
				positives++
			}
			if positives >= 2 {
				state = "BLOCKED"
			}
			if _, e = tx.ExecContext(ctx, `UPDATE egress_iran_guard SET state=?,last_round=?,positives=?,followups=?,last_checked=?,first_evidence=CASE WHEN first_evidence='{}' AND ?!='{}' THEN ? ELSE first_evidence END,last_evidence=CASE WHEN ?!='{}' THEN ? ELSE last_evidence END WHERE identity=?`, state, r.Finished, positives, followups, now, proof, proof, proof, proof, id); e != nil {
				return out, e
			}
		}
		if followups < 2 {
			out.Delay = egress.IranSpacing
		}
		if state == "BLOCKED" {
			out.Blocked = true
		}
	}
	if !out.Quarantined {
		return out, nil
	}
	if _, e = tx.ExecContext(ctx, `UPDATE egress_jobs SET state='QUARANTINED',reason='ads_iran_quarantined',valid_until=0,next_run=min(next_run,?) WHERE state!='DELETED' AND (reason!='ads_iran_quarantined' OR valid_until!=0) AND hash!=? AND hash IN(SELECT o.hash FROM egress_owners o JOIN egress_iran_guard g ON g.identity=o.identity)`, now+egress.IranSpacing, r.Hash); e != nil {
		return out, e
	}
	// Invalidate cached source responses on the first signal, including aliases.
	if _, e = tx.ExecContext(ctx, `DELETE FROM responses WHERE source_id IN(SELECT o.source FROM egress_owners o JOIN egress_iran_guard g ON g.identity=o.identity)`); e != nil {
		return out, e
	}
	if out.Blocked {
		// Revoke every representation of every confirmed identity; persistent
		// input_exclusions also suppress newly fetched renamed copies at admission.
		rows, e = tx.QueryContext(ctx, `SELECT DISTINCT o.hash FROM egress_owners o JOIN egress_iran_guard g ON g.identity=o.identity JOIN egress_jobs j ON j.hash=o.hash WHERE g.state='BLOCKED' AND j.state!='DELETED'`)
		if e != nil {
			return out, e
		}
		hashes := []string{}
		for rows.Next() {
			var h string
			if e = rows.Scan(&h); e != nil {
				rows.Close()
				return out, e
			}
			hashes = append(hashes, h)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return out, e
		}
		for _, h := range hashes {
			if e = s.revokeEgress(ctx, tx, h, "confirmed_ads_iran", now); e != nil {
				return out, e
			}
		}
	}
	return out, nil
}

func iranState(ctx context.Context, q databaseQuery, now int64) (map[string]any, error) {
	var quarantined, blocked, overdue int
	e := q.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM egress_iran_guard WHERE state='QUARANTINED'),(SELECT count(*) FROM egress_iran_guard WHERE state='BLOCKED'),(SELECT count(*) FROM egress_jobs WHERE state!='DELETED' AND paused=0 AND ever_accepted=1 AND tested<=?)`, now-egress.IranInterval).Scan(&quarantined, &blocked, &overdue)
	return map[string]any{"version": egress.IranVersion, "interval_minutes": 10, "quarantined_identities": quarantined, "blocked_identities": blocked, "overdue_retained": overdue, "country_veto": "IR", "sticky": true}, e
}
