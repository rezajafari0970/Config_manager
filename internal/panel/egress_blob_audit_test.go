package panel

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/blob"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Read-only authenticated full inventory audit. Never opens a live Store or
// performs schema reconciliation; legitimate concurrent deletion is rechecked.
func TestEgressPrivateBlobAudit(t *testing.T) {
	config := os.Getenv("EGRESS_BLOB_AUDIT_CONFIG")
	if config == "" {
		t.Skip("private inventory audit not requested")
	}
	b, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Database string `json:"database"`
		Key      string `json:"master_key"`
	}
	if json.Unmarshal(b, &cfg) != nil {
		t.Fatal("invalid private config")
	}
	key, err := base64.StdEncoding.DecodeString(cfg.Key)
	if err != nil {
		t.Fatal("invalid key")
	}
	v, err := blob.New(filepath.Join(filepath.Dir(cfg.Database), "blobs"), key)
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:"+cfg.Database+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows, err := db.Query("SELECT hash FROM egress_jobs WHERE state!='DELETED' UNION SELECT candidate_hash FROM egress_jobs WHERE state!='DELETED' AND candidate_hash!=''")
	if err != nil {
		t.Fatal(err)
	}
	hashes := []string{}
	for rows.Next() {
		var h string
		if err = rows.Scan(&h); err != nil {
			t.Fatal(err)
		}
		hashes = append(hashes, h)
	}
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
	rows.Close()
	verified, retired, failed := 0, 0, 0
	var size int64
	for i, h := range hashes {
		f, err := v.Open(h)
		if err == nil {
			var n int64
			n, err = io.Copy(io.Discard, f)
			f.Close()
			size += n
		}
		if err != nil {
			var live bool
			if qerr := db.QueryRow("SELECT EXISTS(SELECT 1 FROM egress_jobs WHERE state!='DELETED' AND (hash=? OR candidate_hash=?))", h, h).Scan(&live); qerr != nil {
				t.Fatal(qerr)
			}
			if live {
				failed++
			} else {
				retired++
			}
		} else {
			verified++
		}
		if i%100 == 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}
	t.Logf("authenticated inventory=%d verified=%d concurrent_retired=%d missing_or_corrupt=%d bytes=%d", len(hashes), verified, retired, failed, size)
	if failed != 0 {
		t.Fatal("referenced input bytes require recovery", failed)
	}
}
