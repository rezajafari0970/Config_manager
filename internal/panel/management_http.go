package panel

import "net/http"

func (s *Server) sourceBatch(w http.ResponseWriter, r *http.Request) {
	var b SourceBatch
	if e := decode(r, &b); e != nil {
		failure(w, e)
		return
	}
	result, e := s.store.ManageSources(r.Context(), b)
	if e != nil {
		failure(w, e)
		return
	}
	if s.engine != nil {
		for _, id := range result.IDs {
			s.engine.Wake(id, true)
		}
	}
	respond(w, 200, result)
}
func (s *Server) configBatch(w http.ResponseWriter, r *http.Request) {
	var b ConfigBatch
	if e := decode(r, &b); e != nil {
		failure(w, e)
		return
	}
	result, e := s.store.ManageConfigs(r.Context(), b)
	if e != nil {
		failure(w, e)
		return
	}
	if s.engine != nil {
		for _, id := range result.IDs {
			s.engine.Wake(id, true)
		}
	}
	respond(w, 200, result)
}
