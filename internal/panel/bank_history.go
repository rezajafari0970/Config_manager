package panel

import (
	"encoding/json"
	"net/http"
)

type bankVariant struct {
	Hash       string `json:"hash"`
	Size       int64  `json:"size"`
	Rule       string `json:"rule"`
	Created    int64  `json:"created"`
	Tested     int64  `json:"tested"`
	Connection string `json:"connection"`
}

func (s *Server) bankReport(w http.ResponseWriter, r *http.Request) {
	v, e := s.store.BankOne(r.Context(), r.PathValue("id"))
	if e != nil {
		failure(w, e)
		return
	}
	report := json.RawMessage(`[]`)
	var encoded string
	if e = s.store.db.QueryRowContext(r.Context(), "SELECT report FROM probe_results WHERE hash=?", v.Hash).Scan(&encoded); e == nil {
		report = json.RawMessage(encoded)
	}
	rows, e := s.store.db.QueryContext(r.Context(), "SELECT candidate_hash,size,rule,created,last_tested,last_status FROM bank_variants WHERE hash=? ORDER BY last_tested DESC", v.Hash)
	if e != nil {
		failure(w, e)
		return
	}
	variants := []bankVariant{}
	for rows.Next() {
		var item bankVariant
		if e = rows.Scan(&item.Hash, &item.Size, &item.Rule, &item.Created, &item.Tested, &item.Connection); e != nil {
			break
		}
		variants = append(variants, item)
	}
	if e == nil {
		e = rows.Err()
	}
	rows.Close()
	if e != nil {
		failure(w, e)
		return
	}
	respond(w, 200, map[string]any{"entry": v, "attempts": report, "variants": variants})
}
func (s *Server) bankVariant(w http.ResponseWriter, r *http.Request) {
	v, e := s.store.BankOne(r.Context(), r.PathValue("id"))
	if e != nil {
		failure(w, e)
		return
	}
	if !v.Enabled {
		failure(w, ErrNotFound)
		return
	}
	hash := r.PathValue("hash")
	var size int64
	if e = s.store.db.QueryRowContext(r.Context(), "SELECT size FROM bank_variants WHERE hash=? AND candidate_hash=?", v.Hash, hash).Scan(&size); e != nil {
		failure(w, ErrNotFound)
		return
	}
	w.Header().Set("X-Android-Import", "not-run")
	w.Header().Set("X-Verification-Scope", "historical-candidate-not-a-fresh-health-certificate")
	s.stream(w, r, hash, size, hash[:12]+".json")
}
