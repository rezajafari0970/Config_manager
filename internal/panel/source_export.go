package panel

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/url"
	"strconv"
)

// Preserve the existing global round exactly, then add independent source
// scopes. Deleting a source cascades only its own rotation state and queue.
func (s *Store) migrateV8() error {
	var v int
	if e := s.db.QueryRow("PRAGMA user_version").Scan(&v); e != nil {
		return e
	}
	if v >= 8 {
		return nil
	}
	tx, e := s.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	_, e = tx.Exec(`
 ALTER TABLE export_rotation_state RENAME TO export_rotation_state_v7;
 ALTER TABLE export_rotation_queue RENAME TO export_rotation_queue_v7;
 CREATE TABLE export_rotation_state(
 scope TEXT PRIMARY KEY,source_id TEXT UNIQUE REFERENCES sources(id) ON DELETE CASCADE,
 cycle INTEGER NOT NULL CHECK(cycle>=0),
 CHECK((scope='' AND source_id IS NULL) OR (scope<>'' AND source_id IS NOT NULL AND scope=source_id)));
 CREATE TABLE export_rotation_queue(
 scope TEXT NOT NULL REFERENCES export_rotation_state(scope) ON DELETE CASCADE,
 position INTEGER NOT NULL,candidate_hash TEXT NOT NULL,
 PRIMARY KEY(scope,position),UNIQUE(scope,candidate_hash));
 INSERT INTO export_rotation_state(scope,source_id,cycle) SELECT '',NULL,cycle FROM export_rotation_state_v7 WHERE id=1;
 INSERT INTO export_rotation_queue(scope,position,candidate_hash) SELECT '',position,candidate_hash FROM export_rotation_queue_v7;
 DROP TABLE export_rotation_queue_v7;DROP TABLE export_rotation_state_v7;
 PRAGMA user_version=8;`)
	if e != nil {
		return e
	}
	return tx.Commit()
}

type rotationLock struct {
	gate  chan struct{}
	users int
}

// Wait only for this scope; slow blob I/O for A cannot hold B's rotation lock.
// Reference counting removes idle locks, including deleted or unknown sources.
func (s *Store) lockRotation(ctx context.Context, scope string) (func(), error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	s.rotationMu.Lock()
	if s.rotationLocks == nil {
		s.rotationLocks = map[string]*rotationLock{}
	}
	lock := s.rotationLocks[scope]
	if lock == nil {
		lock = &rotationLock{gate: make(chan struct{}, 1)}
		s.rotationLocks[scope] = lock
	}
	lock.users++
	s.rotationMu.Unlock()
	releaseRef := func() {
		s.rotationMu.Lock()
		lock.users--
		if lock.users == 0 {
			delete(s.rotationLocks, scope)
		}
		s.rotationMu.Unlock()
	}
	select {
	case lock.gate <- struct{}{}:
		return func() { <-lock.gate; releaseRef() }, nil
	case <-ctx.Done():
		releaseRef()
		return nil, ctx.Err()
	}
}

func (s *Store) exportSourceName(ctx context.Context, source string) (string, error) {
	if source == "" {
		return "همهٔ منابع", nil
	}
	var name string
	e := s.db.QueryRowContext(ctx, "SELECT name FROM sources WHERE id=?", source).Scan(&name)
	if errors.Is(e, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return name, e
}

func (s *Server) exportURL(source string, single bool) string {
	if source == "" {
		if single {
			return s.singleExportURL()
		}
		return s.origin + "/subscription/" + s.store.digest("subscription", "template-jsonl-v1") + "/configs.jsonl"
	}
	file := "configs.jsonl"
	if single {
		file = "next.json"
	}
	return s.origin + "/subscription/source/" + url.PathEscape(source) + "/" + s.store.digest("source-export", source+"\x00"+file) + "/" + file
}

type SourceExport struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Enabled         bool   `json:"enabled"`
	ConfigCount     int    `json:"config_count"`
	ExportURL       string `json:"export_url"`
	SingleExportURL string `json:"single_export_url"`
	PanelURL        string `json:"panel_url"`
}

type sourceExportModule struct{ s *Server }

func (m sourceExportModule) Register(mux *http.ServeMux) {
	s := m.s
	mux.HandleFunc("GET /api/export/sources", func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		sources, e := s.store.ListPage(r.Context(), r.URL.Query().Get("q"), page)
		if e != nil {
			failure(w, e)
			return
		}
		out := make([]SourceExport, 0, len(sources))
		for _, source := range sources {
			out = append(out, SourceExport{ID: source.ID, Name: source.Name, Enabled: source.Enabled, ConfigCount: source.ConfigCount, ExportURL: s.exportURL(source.ID, false), SingleExportURL: s.exportURL(source.ID, true), PanelURL: "/output?" + url.Values{"source": {source.ID}}.Encode()})
		}
		respond(w, 200, map[string]any{"sources": out, "page": max(0, page), "has_more": len(sources) == 100})
	})
	mux.HandleFunc("GET /subscription/source/{source}/{token}/configs.jsonl", func(w http.ResponseWriter, r *http.Request) {
		if s.validSourceExport(w, r, false) {
			s.writeJSONLines(w, r, r.PathValue("source"))
		}
	})
	mux.HandleFunc("GET /subscription/source/{source}/{token}/next.json", func(w http.ResponseWriter, r *http.Request) {
		if s.validSourceExport(w, r, true) {
			s.writeNextJSON(w, r, r.PathValue("source"))
		}
	})
}
func (s *Server) validSourceExport(w http.ResponseWriter, r *http.Request, single bool) bool {
	source := r.PathValue("source")
	file := "configs.jsonl"
	if single {
		file = "next.json"
	}
	if source == "" || !equal(r.PathValue("token"), s.store.digest("source-export", source+"\x00"+file)) {
		http.NotFound(w, r)
		return false
	}
	if _, e := s.store.exportSourceName(r.Context(), source); e != nil {
		failure(w, e)
		return false
	}
	return true
}
