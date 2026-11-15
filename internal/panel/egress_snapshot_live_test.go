package panel

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Opt-in release check against representative data. It opens SQLite read-only,
// bypasses Open/migrations, never loads the encryption key and prints no input,
// report or subscription URL. A few small SQL timings cannot replace this check:
// nested country/profile/entry queries regressed in the first P67 candidate.
func TestEgressSnapshotLiveReadOnly(t *testing.T) {
	path := os.Getenv("EGRESS_SNAPSHOT_READONLY_DB")
	if path == "" {
		t.Skip("opt-in representative database check")
	}
	if !filepath.IsAbs(path) {
		t.Fatal("absolute database path required")
	}
	u := url.URL{Scheme: "file", Path: path, RawQuery: "mode=ro&_pragma=query_only(1)&_pragma=busy_timeout(5000)"}
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	s := &Store{db: db, readDB: db}
	app, err := NewServer(s, "http://127.0.0.1:18191")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/egress?brief=1", "/api/egress?brief=1&view=diagnostics", "/api/egress?brief=1"} {
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		rr := httptest.NewRecorder()
		start := time.Now()
		egressModule{app}.state(rr, httptest.NewRequest("GET", path, nil).WithContext(ctx))
		cancel()
		var body struct {
			CoverageMissing int `json:"coverage_missing"`
			Queue           map[string]int
			Accounting      struct{ Queue map[string]int }
		}
		if rr.Code != 200 || json.Unmarshal(rr.Body.Bytes(), &body) != nil {
			t.Fatalf("full snapshot failed: HTTP %d after %s", rr.Code, time.Since(start))
		}
		if body.CoverageMissing != 0 || body.Queue["due"] != body.Accounting.Queue["due"] || body.Queue["active_leases"] != body.Accounting.Queue["running"] {
			t.Fatal("inconsistent inventory or queue summary")
		}
		t.Logf("full snapshot %s: %s", path, time.Since(start))
	}
}
