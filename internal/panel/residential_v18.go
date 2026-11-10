package panel

import (
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/residential"
	"strings"
)

// P31 restores age-independent retention for accepted records and separates
// observed Ads routing from verified Residential network classification.
func (s *Store) migrateV18() error {
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var version int
	if e = tx.QueryRow("PRAGMA user_version").Scan(&version); e != nil {
		return e
	}
	if version >= 18 {
		return nil
	}
	var view string
	if e = tx.QueryRow("SELECT sql FROM sqlite_master WHERE name='residential_eligible'").Scan(&view); e != nil {
		return e
	}
	view = strings.Replace(view, "CREATE VIEW residential_eligible AS", "CREATE VIEW residential_active AS", 1)
	view = strings.Replace(view, "r.state='ACCEPTED'", "r.state IN ('ACCEPTED','ADS_DIFFERENT')", 1)
	view = strings.Replace(view, " AND r.retain_until>CAST(unixepoch('subsec')*1000 AS INTEGER)", "", 1)
	if _, e = tx.Exec(`ALTER TABLE residential_records ADD COLUMN accepted_decision TEXT NOT NULL DEFAULT '{}';
UPDATE residential_records SET accepted_decision=coalesce((SELECT decision FROM residential_audit a WHERE a.hash=residential_records.hash AND a.state='ACCEPTED' ORDER BY a.id DESC LIMIT 1),'{}') WHERE ever_accepted=1;
DROP VIEW residential_eligible;` + view + `;
CREATE VIEW residential_eligible AS SELECT * FROM residential_active WHERE state='ACCEPTED';
CREATE VIEW ads_different_eligible AS SELECT * FROM residential_active WHERE json_extract(decision,'$.ads_route.state')='DIFFERENT';`); e != nil {
		return e
	}
	for _, name := range []string{"residential_blocked", "residential_blocked_identities"} {
		var q string
		if e = tx.QueryRow("SELECT sql FROM sqlite_master WHERE name=?", name).Scan(&q); e != nil {
			return e
		}
		q = strings.ReplaceAll(q, "residential_eligible", "residential_active")
		if _, e = tx.Exec("DROP VIEW " + name + ";" + q); e != nil {
			return e
		}
	}
	var ingest string
	if e = tx.QueryRow("SELECT sql FROM sqlite_master WHERE name='residential_ingest'").Scan(&ingest); e != nil {
		return e
	}
	ingest = strings.Replace(ingest, "epoch=epoch+1,next_run=0", "epoch=epoch+1,next_run=0,tested=0,ever_accepted=0,accepted_decision='{}'", 1)
	if _, e = tx.Exec("DROP TRIGGER residential_ingest;" + ingest); e != nil {
		return e
	}
	var p residential.Policy
	var body string
	if e = tx.QueryRow("SELECT policy FROM residential_settings WHERE id=1").Scan(&body); e != nil {
		return e
	}
	if e = json.Unmarshal([]byte(body), &p); e != nil {
		return e
	}
	p.RouteComparisonVersion = residential.RouteComparisonVersion
	b, _ := json.Marshal(p)
	if _, e = tx.Exec("UPDATE residential_settings SET policy=?,policy_hash=?,revision=revision+1,generation=generation+1 WHERE id=1", string(b), p.Digest()); e != nil {
		return e
	}
	if _, e = tx.Exec(`UPDATE residential_records SET state=CASE WHEN ever_accepted=1 THEN 'QUARANTINED' ELSE 'UNKNOWN' END,reason='route_policy_upgraded',valid_until=0,revision=revision+1,token='',lease_until=0,next_run=0 WHERE state!='DELETED'; PRAGMA user_version=18;`); e != nil {
		return e
	}
	return tx.Commit()
}
