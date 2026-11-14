package panel

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	_ "modernc.org/sqlite"
	"os"
	"testing"
)

func TestEgressPrivateMigrationRehearsal(t *testing.T) {
	path := os.Getenv("EGRESS_REHEARSAL_CONFIG")
	if path == "" {
		t.Skip("private snapshot not requested")
	}
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	var c struct {
		Database string `json:"database"`
		Key      string `json:"master_key"`
	}
	if json.Unmarshal(b, &c) != nil {
		t.Fatal("bad private fixture")
	}
	db, e := sql.Open("sqlite", c.Database)
	if e != nil {
		t.Fatal(e)
	}
	var beforeSources, beforeMembers int
	db.QueryRow("SELECT count(*) FROM sources").Scan(&beforeSources)
	db.QueryRow("SELECT count(*) FROM memberships").Scan(&beforeMembers)
	db.Close()
	key, e := base64.StdEncoding.DecodeString(c.Key)
	if e != nil {
		t.Fatal(e)
	}
	s, e := Open(c.Database, key)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	var afterSources, afterMembers, jobs, raws, legacy, version int
	s.db.QueryRow("SELECT count(*) FROM sources").Scan(&afterSources)
	s.db.QueryRow("SELECT count(*) FROM memberships").Scan(&afterMembers)
	s.db.QueryRow("SELECT count(*) FROM egress_jobs").Scan(&jobs)
	s.db.QueryRow("SELECT count(DISTINCT raw_hash) FROM memberships").Scan(&raws)
	s.db.QueryRow("SELECT count(*) FROM sqlite_master WHERE name GLOB 'residential_*'").Scan(&legacy)
	s.db.QueryRow("PRAGMA user_version").Scan(&version)
	if beforeSources != afterSources || beforeMembers != afterMembers || jobs < raws || legacy != 0 || version != 32 {
		t.Fatal("migration counts", beforeSources, afterSources, beforeMembers, afterMembers, jobs, raws, legacy, version)
	}
	t.Logf("preserved %d sources and %d memberships; seeded %d unique inputs; zero legacy objects; schema %d", afterSources, afterMembers, jobs, version)
}
