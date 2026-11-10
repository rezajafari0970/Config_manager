package panel

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
)

func TestRawAllPrintsOriginalBodiesAndRetiresZIP(t *testing.T) {
	s := testStore(t)
	addConversion(t, s, " \n"+conversionSample+"\n ")
	addConversion(t, s, "vless://00000000-0000-4000-8000-000000000002@192.0.2.15:443?security=tls&sni=example.com")
	var refs []Artifact
	if e := s.WalkOutputs(context.Background(), func(a Artifact) error { refs = append(refs, a); return nil }); e != nil {
		t.Fatal(e)
	}
	var raw [][]byte
	for _, a := range refs {
		r, e := s.vault.Open(a.Hash)
		if e != nil {
			t.Fatal(e)
		}
		b, e := io.ReadAll(r)
		r.Close()
		if e != nil {
			t.Fatal(e)
		}
		raw = append(raw, b)
	}
	server, _ := NewServer(s, "http://127.0.0.1:18191")
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, httptest.NewRequest("GET", "http://127.0.0.1:18191/output/raw.txt", nil))
	if w.Code != 200 || !bytes.Equal(w.Body.Bytes(), bytes.Join(raw, []byte{'\n'})) || w.Header().Get("Content-Disposition") != "inline" {
		t.Fatal(w.Code, w.Header())
	}
	z := httptest.NewRecorder()
	server.Handler().ServeHTTP(z, httptest.NewRequest("GET", "http://127.0.0.1:18191/output/archive.zip", nil))
	if z.Code != 308 || z.Header().Get("Location") != "/output/raw.txt" {
		t.Fatal(z.Code, z.Header())
	}
}

func TestRawSnapshotAllowsArrivalsButFencesInvalidatedMembers(t *testing.T) {
	for _, mutation := range []string{"arrival", "disable", "delete", "expire", "representation"} {
		t.Run(mutation, func(t *testing.T) {
			s := testStore(t)
			ctx := context.Background()
			addConversion(t, s, conversionSample)
			members := make(map[string]struct{})
			before := sha256.New()
			encoder := json.NewEncoder(before)
			var captured Artifact
			if e := s.WalkOutputs(ctx, func(a Artifact) error {
				captured = a
				members[a.ID] = struct{}{}
				return encoder.Encode(a)
			}); e != nil {
				t.Fatal(e)
			}
			addConversion(t, s, "vless://00000000-0000-4000-8000-000000000002@192.0.2.15:443?security=tls&sni=example.com")
			var e error
			switch mutation {
			case "disable", "delete":
				_, e = s.ManageConfigs(ctx, ConfigBatch{Action: mutation, IDs: []string{captured.ID}})
			case "expire":
				_, e = s.db.ExecContext(ctx, "UPDATE artifacts SET expires=1 WHERE id=?", captured.ID)
			case "representation":
				_, e = s.db.ExecContext(ctx, "UPDATE memberships SET size=size+1 WHERE artifact_id=?", captured.ID)
			}
			if e != nil {
				t.Fatal(e)
			}
			valid, e := rawSnapshotCurrent(ctx, s.read(ctx), members, before.Sum(nil))
			if e != nil || valid != (mutation == "arrival") {
				t.Fatal("snapshot admission", valid, e)
			}
		})
	}
}
