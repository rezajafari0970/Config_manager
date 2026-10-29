package panel

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
)

type outputModule struct{ s *Server }

func (m outputModule) Register(mux *http.ServeMux) {
	s := m.s
	mux.HandleFunc("POST /api/configs/batch", s.configBatch)
	mux.HandleFunc("GET /output/raw/{id}", s.raw)
	mux.HandleFunc("GET /output/archive.zip", s.archive)
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
func (s *Server) archive(w http.ResponseWriter, r *http.Request) {
	snapshot, e := os.CreateTemp("", "config-output-index-")
	if e != nil {
		failure(w, e)
		return
	}
	defer os.Remove(snapshot.Name())
	defer snapshot.Close()
	encoder := json.NewEncoder(snapshot)
	if e = s.store.WalkOutputs(r.Context(), func(a Artifact) error { return encoder.Encode(a) }); e != nil {
		failure(w, e)
		return
	}
	if _, e = snapshot.Seek(0, io.SeekStart); e != nil {
		failure(w, e)
		return
	}
	decoder := json.NewDecoder(snapshot)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="configs-raw.zip"`)
	archive := zip.NewWriter(w)
	defer archive.Close()
	for {
		var a Artifact
		if e = decoder.Decode(&a); e == io.EOF {
			break
		} else if e != nil {
			return
		}
		if r.Context().Err() != nil {
			return
		}
		body, e := s.store.vault.Open(a.Hash)
		if e != nil {
			continue
		}
		ext := "txt"
		if a.Kind == "json" {
			ext = "json"
		}
		entry, e := archive.Create(fmt.Sprintf("%s/%s.%s", a.Kind, a.ID, ext))
		if e == nil {
			_, e = io.Copy(entry, body)
		}
		body.Close()
		if e != nil {
			return
		}
	}
}
