package panel

import (
	"reflect"
	"strings"
	"testing"
)

func TestGuardLookupPreservesArchivedAliasesAndAvoidsOwnerScan(t *testing.T) {
	s := testStore(t)
	if _, err := s.db.Exec(`WITH RECURSIVE n(i) AS (SELECT 1 UNION ALL SELECT i+1 FROM n WHERE i<30000)
 INSERT INTO egress_jobs(hash,kind,size,first_seen,last_seen) SELECT printf('%064x',400000+i),'guard-fixture',1,1,1 FROM n;
 INSERT INTO egress_owners(hash,identity,source) SELECT hash,hash,'archived' FROM egress_jobs WHERE kind='guard-fixture';
 INSERT INTO egress_owners(hash,identity,source) SELECT hash,'shared-guard','other-source' FROM egress_jobs WHERE kind='guard-fixture' ORDER BY hash LIMIT 2;
 INSERT INTO egress_iran_guard(identity,state,first_seen) VALUES('shared-guard','QUARANTINED',1),(printf('%064x',400003),'BLOCKED',1);
 INSERT INTO input_exclusions(identity,at,hash) VALUES('manual',1,printf('%064x',400004));`); err != nil {
		t.Fatal(err)
	}
	var view string
	if err := s.db.QueryRow("SELECT sql FROM sqlite_master WHERE name='input_blocked'").Scan(&view); err != nil {
		t.Fatal(err)
	}
	legacy := strings.Replace(view, egressGuardLookup, egressGuardScan, 1)
	if legacy == view {
		t.Fatal("missing indexed guard fixture")
	}
	if _, err := s.db.Exec("DROP VIEW input_blocked;" + legacy); err != nil {
		t.Fatal(err)
	}
	read := func(name, field string) []string {
		t.Helper()
		rows, err := s.db.Query("SELECT " + field + " FROM " + name + " ORDER BY 1")
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var out []string
		for rows.Next() {
			var v string
			if err := rows.Scan(&v); err != nil {
				t.Fatal(err)
			}
			out = append(out, v)
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		return out
	}
	before := read("input_blocked", "hash")
	identities := read("input_blocked_identities", "identity")
	if len(before) != 4 {
		t.Fatal("guard/manual fixture", len(before))
	}
	for i := 0; i < 2; i++ {
		if err := s.ensureEgressGuardLookup(); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(before, read("input_blocked", "hash")) || !reflect.DeepEqual(identities, read("input_blocked_identities", "identity")) {
			t.Fatal("archived alias, quarantine, permanent guard or manual exclusion changed")
		}
	}
	rows, err := s.db.Query("EXPLAIN QUERY PLAN SELECT hash FROM input_blocked")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	indexed := false
	for rows.Next() {
		var id, parent, unused int
		var detail string
		if err := rows.Scan(&id, &parent, &unused, &detail); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(detail, "SCAN o") {
			t.Fatal("guard still scans historical ownership", detail)
		}
		indexed = indexed || strings.Contains(detail, "SEARCH o USING INDEX egress_identity")
	}
	if err := rows.Err(); err != nil || !indexed {
		t.Fatal("guard did not use identity lookup", indexed, err)
	}
}
