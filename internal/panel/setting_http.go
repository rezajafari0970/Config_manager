package panel

import (
	"github.com/rezajafari0970/Config_manager/internal/collect"
	"net/http"
	"time"
)

type settingModule struct{ s *Server }

func (m settingModule) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/settings", func(w http.ResponseWriter, r *http.Request) {
		settings, e := m.s.store.Settings(r.Context())
		if e != nil {
			failure(w, e)
			return
		}
		capacity := 0
		if m.s.engine != nil {
			capacity = m.s.engine.Capacity()
		}
		respond(w, 200, map[string]any{"settings": settings, "capacity": capacity})
	})
	mux.HandleFunc("POST /api/settings", m.s.settings)
}
func (s *Server) settings(w http.ResponseWriter, r *http.Request) {
	var v collect.Settings
	if e := decode(r, &v); e != nil {
		failure(w, e)
		return
	}
	if e := s.store.SaveSettings(r.Context(), v); e != nil {
		failure(w, e)
		return
	}
	if e := s.store.Expire(r.Context(), time.Now()); e != nil {
		failure(w, e)
		return
	}
	if s.engine != nil {
		s.engine.Wake("", false)
	}
	respond(w, 200, map[string]bool{"saved": true})
}
