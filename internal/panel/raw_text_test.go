package panel

import (
	"bytes"
	"context"
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
