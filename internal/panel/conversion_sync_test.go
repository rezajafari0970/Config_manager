package panel

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/rezajafari0970/Config_manager/internal/egress"
	"github.com/rezajafari0970/Config_manager/internal/repair"
)

func TestConversionSyncBulkOwnershipAndRetainedProof(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	source, _ := newExportSource(t, s, "bulk-sync", sourceURI(31), sourceURI(32))
	j, err := s.ClaimEgress(ctx)
	if err != nil || j == nil {
		t.Fatal(j, err)
	}
	if err = s.CompleteEgress(ctx, egress.Completion{Job: *j, Report: syntheticEgress(*j)}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UnixMilli()
	if _, err = s.db.Exec(`WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i+1 FROM n WHERE i<2000)
 INSERT INTO artifacts(id,hash,kind,size,created,expires) SELECT printf('%064x',200000+i),printf('%064x',200000+i),'fixture',1,?,? FROM n`, now, now+3600000); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`INSERT INTO memberships(source_id,artifact_id,generation,raw_hash,kind,size) SELECT ?,id,1,hash,kind,size FROM artifacts WHERE kind='fixture'`, source); err != nil {
		t.Fatal(err)
	}
	blocked := fmt.Sprintf("%064x", 200001)
	if _, err = s.ManageConfigs(ctx, ConfigBatch{Action: "disable", IDs: []string{blocked}}); err != nil {
		t.Fatal(err)
	}
	// Retained accepted proof stays eligible without ordinary membership. The
	// other ordinary receipt must disappear through the job's foreign-key cascade.
	if _, err = s.db.Exec("DELETE FROM memberships WHERE kind!='fixture'; INSERT INTO conversion_jobs(hash) VALUES(?)", fmt.Sprintf("%064x", 300001)); err != nil {
		t.Fatal(err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err = syncConversions(ctx, tx); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	var jobs, retained, revoked, blockedJobs int
	ordinary := repair.Digest([]byte(sourceURI(31)))
	if ordinary == j.Hash {
		ordinary = repair.Digest([]byte(sourceURI(32)))
	}
	if err = s.db.QueryRow(`SELECT (SELECT count(*) FROM conversion_jobs),
 (SELECT count(*) FROM conversion_results WHERE hash=?),
 (SELECT count(*) FROM conversion_results WHERE hash=?),
 (SELECT count(*) FROM conversion_jobs WHERE hash IN(?,?))`, j.Hash, ordinary, blocked, fmt.Sprintf("%064x", 300001)).Scan(&jobs, &retained, &revoked, &blockedJobs); err != nil {
		t.Fatal(err)
	}
	if jobs != 2000 || retained != 1 || revoked != 0 || blockedJobs != 0 {
		t.Fatal("current scope, retained proof or revocation changed", jobs, retained, revoked, blockedJobs)
	}
}

func TestSourceIdentityRecheckPreservesAliasRanking(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	rawA, rawB := sourceURI(33)+"-A", sourceURI(33)+"-B"
	a, _ := newExportSource(t, s, "rank-a", rawA, sourceURI(34))
	b, _ := newExportSource(t, s, "rank-b", rawB)
	var identity string
	if err := s.db.QueryRow("SELECT artifact_id FROM memberships WHERE raw_hash=?", repair.Digest([]byte(rawA))).Scan(&identity); err != nil {
		t.Fatal(err)
	}
	candidates := map[string]bool{"missing-candidate": true}
	for step := 0; step < 2; step++ {
		for _, scope := range []string{"", a, b} {
			all, err := querySourceConversions(ctx, s.db, scope, 0, true)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range all {
				candidates[entry.CandidateHash] = true
			}
			for candidate := range candidates {
				if candidate == "" {
					continue
				}
				matching, err := querySourceCandidateConversions(ctx, s.db, scope, candidate)
				if err != nil {
					t.Fatal(err)
				}
				wanted := []Conversion{}
				for _, entry := range all {
					if entry.CandidateHash == candidate {
						wanted = append(wanted, entry)
					}
				}
				if len(matching) != len(wanted) {
					t.Fatal("candidate lookup changed winning aliases", scope, step, len(matching), len(wanted))
				}
				for i := range wanted {
					if matching[i] != wanted[i] {
						t.Fatal("candidate lookup changed source ranking", scope, step)
					}
				}
			}
			filtered, err := querySourceIdentityConversions(ctx, s.db, scope, identity, 0, true)
			if err != nil || len(filtered) != 1 {
				t.Fatal("identity recheck lost source ownership", len(filtered), err)
			}
			found := false
			for _, entry := range all {
				if entry.ID == identity {
					found = true
					if entry != filtered[0] {
						t.Fatal("identity filter changed winning raw alias", scope, step)
					}
				}
			}
			if !found {
				t.Fatal("fixture identity absent")
			}
		}
		if _, err := s.db.Exec("UPDATE conversion_results SET connection='failed' WHERE hash=?", repair.Digest([]byte(rawA))); err != nil {
			t.Fatal(err)
		}
	}
}
