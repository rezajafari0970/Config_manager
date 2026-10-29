package panel

import (
	"bufio"
	"errors"
	"io"
	"net/http"
	"strings"
)

type inputModule struct{ s *Server }

func (m inputModule) Register(mux *http.ServeMux) {
	s := m.s
	mux.HandleFunc("GET /api/sources/{id}", func(w http.ResponseWriter, r *http.Request) {
		v, e := s.store.Get(r.Context(), r.PathValue("id"))
		if e != nil {
			failure(w, e)
			return
		}
		respond(w, 200, v)
	})
	mux.HandleFunc("POST /api/sources/import", s.bulk)
	mux.HandleFunc("POST /api/sources/{id}/edit", s.edit)
	mux.HandleFunc("POST /api/sources/{id}/{action}", s.action)
	mux.HandleFunc("GET /input/raw/{id}", s.response)
}
func (s *Server) bulk(w http.ResponseWriter, r *http.Request) {
	min, max, e := parseInterval(r.URL.Query().Get("interval"))
	unit := r.URL.Query().Get("unit")
	probe := SourceInput{Min: min, Max: max, Unit: unit}
	if e != nil {
		failure(w, e)
		return
	}
	if _, _, e = probe.durations(); e != nil {
		failure(w, e)
		return
	}
	reader := bufio.NewReader(r.Body)
	added, duplicates, invalid, lineNo := 0, 0, []int{}, 0
	for {
		line, err := reader.ReadString('\n')
		lineNo++
		raw := strings.TrimSpace(strings.TrimPrefix(line, "\ufeff"))
		if raw != "" {
			input := SourceInput{URL: raw, Min: min, Max: max, Unit: unit, Enabled: true}
			_, e = s.store.Save(r.Context(), "", 0, input)
			switch {
			case e == nil:
				added++
			case errors.Is(e, ErrConflict):
				duplicates++
			case errors.Is(e, ErrInvalid):
				invalid = append(invalid, lineNo)
			default:
				failure(w, e)
				return
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			failure(w, ErrInvalid)
			return
		}
	}
	if s.engine != nil {
		s.engine.Wake("", false)
	}
	respond(w, 200, map[string]any{"added": added, "duplicates": duplicates, "invalid_lines": invalid})
}
func (s *Server) edit(w http.ResponseWriter, r *http.Request) {
	var v struct {
		SourceInput
		Revision int64 `json:"revision"`
	}
	if e := decode(r, &v); e != nil {
		failure(w, e)
		return
	}
	id := r.PathValue("id")
	if _, e := s.store.Save(r.Context(), id, v.Revision, v.SourceInput); e != nil {
		failure(w, e)
		return
	}
	if s.engine != nil {
		s.engine.Wake(id, true)
	}
	respond(w, 200, map[string]bool{"saved": true})
}
func (s *Server) action(w http.ResponseWriter, r *http.Request) {
	var v struct {
		Revision int64 `json:"revision"`
	}
	if e := decode(r, &v); e != nil {
		failure(w, e)
		return
	}
	id := r.PathValue("id")
	source, e := s.store.Get(r.Context(), id)
	if e != nil {
		failure(w, e)
		return
	}
	if source.Revision != v.Revision {
		failure(w, ErrConflict)
		return
	}
	cancel := false
	switch r.PathValue("action") {
	case "delete":
		e = s.store.Delete(r.Context(), id, v.Revision)
		cancel = true
	case "toggle":
		source.Enabled = !source.Enabled
		_, e = s.store.Save(r.Context(), id, v.Revision, source.SourceInput)
		cancel = true
	case "run":
		e = s.store.RunNow(r.Context(), id)
	default:
		e = ErrNotFound
	}
	if e != nil {
		failure(w, e)
		return
	}
	if s.engine != nil {
		s.engine.Wake(id, cancel)
	}
	respond(w, 200, map[string]bool{"saved": true})
}
