package panel

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
)

type outputModule struct{ s *Server }

func (m outputModule) Register(mux *http.ServeMux) {
	s := m.s
	mux.HandleFunc("GET /api/repairs/{id}", s.repairReport)
	mux.HandleFunc("GET /output/repaired/{id}", s.repaired)
	mux.HandleFunc("POST /api/configs/batch", s.configBatch)
	mux.HandleFunc("GET /output/raw/{id}", s.raw)
	mux.HandleFunc("GET /output/raw.txt", s.rawAll)
	mux.HandleFunc("GET /output/archive.zip", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/output/raw.txt", http.StatusPermanentRedirect)
	})
}
func (s *Server) stream(w http.ResponseWriter, r *http.Request, hash string, size int64, filename string) {
	body, e := s.store.vault.Open(hash)
	if e != nil {
		failure(w, ErrNotFound)
		return
	}
	defer body.Close()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.Header().Set("X-Content-SHA256", hash)
	if r.URL.Query().Get("preview") != "1" {
		w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	}
	_, _ = io.Copy(w, body)
}
func (s *Server) raw(w http.ResponseWriter, r *http.Request) {
	a, e := s.store.Output(r.Context(), r.PathValue("id"))
	if e != nil {
		failure(w, e)
		return
	}
	if a.Disabled && r.URL.Query().Get("preview") != "1" {
		respond(w, 409, map[string]string{"error": "این کانفیگ غیرفعال است."})
		return
	}
	ext := "txt"
	if a.Kind == "json" {
		ext = "json"
	}
	s.stream(w, r, a.Hash, a.Size, a.ID[:12]+"."+ext)
}
func (s *Server) response(w http.ResponseWriter, r *http.Request) {
	ref, e := s.store.Response(r.Context(), r.PathValue("id"))
	if e != nil {
		failure(w, e)
		return
	}
	s.stream(w, r, ref.Hash, ref.Size, "original-response.txt")
}

// rawAll prints immutable original bodies; no ZIP, wrapper or JSON rewriting.
func (s *Server) rawAll(w http.ResponseWriter, r *http.Request) {
	index, e := newExportSpool()
	if e != nil {
		failure(w, e)
		return
	}
	defer index.Close()
	before := sha256.New()
	members := make(map[string]struct{})
	encoder := json.NewEncoder(io.MultiWriter(index.file, before))
	if e = s.store.WalkOutputs(r.Context(), func(a Artifact) error {
		members[a.ID] = struct{}{}
		return encoder.Encode(a)
	}); e != nil {
		failure(w, e)
		return
	}
	if e = index.Rewind(); e != nil {
		failure(w, e)
		return
	}
	output, e := newExportSpool()
	if e != nil {
		failure(w, e)
		return
	}
	defer output.Close()
	decoder := json.NewDecoder(index.file)
	count := 0
	var size int64
	writer := io.MultiWriter(output.file, output.digest)
	for {
		var a Artifact
		e = decoder.Decode(&a)
		if e == io.EOF {
			break
		}
		if e != nil {
			failure(w, e)
			return
		}
		if r.Context().Err() != nil {
			return
		}
		if size+a.Size+1 > 256<<20 {
			http.Error(w, "raw output size limit", 503)
			return
		}
		body, err := s.store.vault.Open(a.Hash)
		if err != nil {
			http.Error(w, "raw body unavailable; retry", 503)
			return
		}
		if count > 0 {
			if _, e = writer.Write([]byte{'\n'}); e != nil {
				body.Close()
				failure(w, e)
				return
			}
			size++
		}
		n, err := io.Copy(writer, body)
		body.Close()
		if err != nil || n != a.Size {
			http.Error(w, "raw body incomplete; retry", 503)
			return
		}
		size += n
		count++
	}
	if count == 0 {
		http.Error(w, "no current raw configurations", 503)
		return
	}
	// Vault I/O is complete before taking the serialized admission transaction.
	tx, e := s.store.db.BeginTx(r.Context(), nil)
	if e != nil {
		failure(w, e)
		return
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(r.Context(), "UPDATE metadata SET value=value WHERE key='key-check'"); e != nil {
		failure(w, e)
		return
	}
	valid, e := rawSnapshotCurrent(r.Context(), tx, members, before.Sum(nil))
	if e != nil {
		failure(w, e)
		return
	}
	if !valid {
		http.Error(w, "raw membership changed; retry", 503)
		return
	}
	if e = tx.Commit(); e != nil {
		failure(w, e)
		return
	}
	if e = output.Rewind(); e != nil {
		failure(w, e)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	w.Header().Set("X-Content-SHA256", output.Hash())
	w.Header().Set("X-Config-Count", strconv.Itoa(count))
	w.Header().Set("X-Raw-Framing", "original-bytes; one-LF-between-members")
	if r.Method != http.MethodHead {
		_, _ = output.WriteTo(w)
	}
}

// New arrivals belong to the next request. Every captured member must still
// have the same current representation and lifetime at serialized admission.
func rawSnapshotCurrent(ctx context.Context, q databaseQuery, members map[string]struct{}, before []byte) (bool, error) {
	after := sha256.New()
	verify := json.NewEncoder(after)
	if e := walkOutputs(ctx, q, func(a Artifact) error {
		if _, ok := members[a.ID]; !ok {
			return nil
		}
		return verify.Encode(a)
	}); e != nil {
		return false, e
	}
	return equal(string(before), string(after.Sum(nil))), nil
}
