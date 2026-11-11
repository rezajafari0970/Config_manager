package panel

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Optional private clone only. Never opens the production database, launches a
// collector, or writes source credentials into test output.
func TestResidentialMigrationRehearsal(t *testing.T) {
	path := os.Getenv("RESIDENTIAL_MIGRATION_REHEARSAL")
	if path == "" {
		t.Skip("explicit private clone rehearsal")
	}
	if !strings.Contains(path, "/.local/p31-rehearsal-") {
		t.Fatal("private clone required")
	}
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	var c struct {
		Database  string `json:"database"`
		MasterKey string `json:"master_key"`
	}
	if json.Unmarshal(b, &c) != nil || filepath.Dir(c.Database) != filepath.Dir(path) {
		t.Fatal("invalid clone config")
	}
	key, e := base64.StdEncoding.DecodeString(c.MasterKey)
	if e != nil {
		t.Fatal("invalid clone key")
	}
	t.Setenv("SHEN_RESIDENTIAL_JOURNAL", filepath.Join(filepath.Dir(path), "revocations.jsonl"))
	for i := 0; i < 2; i++ {
		s, e := Open(c.Database, key)
		if e != nil {
			t.Fatal(e)
		}
		var version, active int
		var integrity string
		s.db.QueryRow("PRAGMA user_version").Scan(&version)
		s.db.QueryRow("PRAGMA quick_check").Scan(&integrity)
		s.db.QueryRow("SELECT count(*) FROM residential_active").Scan(&active)
		if version != 21 || integrity != "ok" || active != 0 {
			s.Close()
			t.Fatal("migration did not fence old evidence", version, integrity, active)
		}
		if e = s.Close(); e != nil {
			t.Fatal(e)
		}
	}
}
