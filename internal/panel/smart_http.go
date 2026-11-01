package panel

import (
	"bytes"
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/probe"
	"github.com/rezajafari0970/Config_manager/internal/smart"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type smartModule struct{ s *Server }

func (s *Server) smartURL(single bool) string {
	mode, file := "smart-jsonl-v1", "configs.jsonl"
	if single {
		mode, file = "smart-single-v1", "next.json"
	}
	return s.origin + "/smart-subscription/" + s.store.digest("subscription", mode) + "/" + file
}
func (m smartModule) Register(mux *http.ServeMux) {
	s := m.s
	mux.HandleFunc("GET /api/smart", func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		entries, e := querySmart(r.Context(), s.store.db, false, page)
		if e != nil {
			failure(w, e)
			return
		}
		settings, e := s.store.SmartSettings(r.Context())
		if e != nil {
			failure(w, e)
			return
		}
		worker, e := s.store.SmartWorker(r.Context())
		if e != nil {
			failure(w, e)
			return
		}
		if time.Now().UnixMilli()-worker.Heartbeat > 30000 {
			worker.Status = "offline"
		}
		stats := map[string]int{}
		var total, ready, blocked, sources, eligible int
		e = s.store.db.QueryRowContext(r.Context(), `SELECT (SELECT count(*) FROM smart_heads),(SELECT count(*) FROM smart_ready),(SELECT count(*) FROM smart_heads WHERE state='blocked'),(SELECT count(DISTINCT source) FROM smart_offers),(SELECT count(DISTINCT id) FROM smart_offers)`).Scan(&total, &ready, &blocked, &sources, &eligible)
		if e != nil {
			failure(w, e)
			return
		}
		stats["total"] = total
		stats["ready"] = ready
		stats["blocked"] = blocked
		stats["warming"] = max(0, total-ready-blocked)
		stats["sources"] = sources
		stats["eligible"] = eligible
		respond(w, 200, map[string]any{"entries": entries, "settings": settings, "worker": worker, "stats": stats, "export_url": s.smartURL(false), "single_export_url": s.smartURL(true), "version": smart.Version, "android_import": "not_run"})
	})
	mux.HandleFunc("POST /api/smart/settings", func(w http.ResponseWriter, r *http.Request) {
		var v SmartSettings
		if decode(r, &v) != nil || v.Count < 1 {
			respond(w, 422, map[string]string{"error": "تعداد اعضای هر اسمارت باید یک عدد صحیح مثبت باشد."})
			return
		}
		if e := s.store.SaveSmartSettings(r.Context(), v); e != nil {
			failure(w, e)
			return
		}
		respond(w, 200, true)
	})
	mux.HandleFunc("GET /smart/json/{id}", func(w http.ResponseWriter, r *http.Request) {
		rows, e := querySmart(r.Context(), s.store.db, true, 0)
		if e != nil {
			failure(w, e)
			return
		}
		for _, v := range rows {
			if v.ID == r.PathValue("id") {
				b, e := s.store.smartBytes(r.Context(), v)
				if e != nil {
					failure(w, ErrNotFound)
					return
				}
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.Header().Set("X-Content-SHA256", v.Hash)
				w.Write(b)
				return
			}
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("GET /smart-subscription/{token}/configs.jsonl", func(w http.ResponseWriter, r *http.Request) {
		if !equal(r.PathValue("token"), s.store.digest("subscription", "smart-jsonl-v1")) {
			http.NotFound(w, r)
			return
		}
		rows, e := querySmart(r.Context(), s.store.db, true, 0)
		if e != nil {
			failure(w, e)
			return
		}
		var out bytes.Buffer
		count := 0
		seen := map[string]bool{}
		for _, v := range rows {
			if !v.Available || seen[v.Hash] {
				continue
			}
			b, e := s.store.smartBytes(r.Context(), v)
			if e != nil {
				s.store.retrySmartBlob(r.Context(), v)
				continue
			}
			seen[v.Hash] = true
			out.Write(b)
			out.WriteByte('\n')
			count++
		}
		smartHeaders(w)
		w.Header().Set("X-Config-Count", strconv.Itoa(count))
		w.Write(out.Bytes())
	})
	mux.HandleFunc("GET /smart-subscription/{token}/next.json", func(w http.ResponseWriter, r *http.Request) {
		if !equal(r.PathValue("token"), s.store.digest("subscription", "smart-single-v1")) {
			http.NotFound(w, r)
			return
		}
		smartHeaders(w)
		if r.Method == http.MethodHead {
			return
		}
		if strings.Contains(strings.ToLower(r.Header.Get("Sec-Purpose")+" "+r.Header.Get("Purpose")), "prefetch") {
			w.WriteHeader(204)
			return
		}
		v, b, cycle, e := s.store.smartNext(r.Context())
		if e != nil {
			failure(w, e)
			return
		}
		if v == nil {
			w.Header().Set("X-Config-Count", "0")
			w.Header().Set("Retry-After", "3")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("هنوز کانفیگ اسمارت آماده نیست؛ چند ثانیه دیگر دوباره تلاش کنید.\n"))
			return
		}
		w.Header().Set("X-Config-Count", "1")
		w.Header().Set("X-Export-Cycle", strconv.FormatInt(cycle, 10))
		w.Header().Set("X-Content-SHA256", v.Hash)
		w.Header().Set("X-Smart-First-ID", v.ID)
		w.Write(b)
		w.Write([]byte{'\n'})
	})
}
func smartHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("X-Android-Import", "not-run")
	w.Header().Set("X-Verification-Scope", "pinned-core-smart-profile; member-connectivity-is-separate")
}
func (s *Store) registerPrivateSmart(mux *http.ServeMux) {
	mux.HandleFunc("POST /smart/claim", func(w http.ResponseWriter, r *http.Request) {
		j, e := s.ClaimSmart(r.Context())
		if e != nil {
			failure(w, e)
			return
		}
		respond(w, 200, j)
	})
	mux.HandleFunc("POST /smart/renew", func(w http.ResponseWriter, r *http.Request) {
		var j probe.Job
		if decode(r, &j) != nil {
			failure(w, ErrInvalid)
			return
		}
		if e := s.RenewSmart(r.Context(), j); e != nil {
			failure(w, e)
			return
		}
		respond(w, 200, true)
	})
	mux.HandleFunc("POST /smart/complete", func(w http.ResponseWriter, r *http.Request) {
		var v probe.Result
		if decode(r, &v) != nil {
			failure(w, ErrInvalid)
			return
		}
		if e := s.CompleteSmart(r.Context(), v); e != nil {
			failure(w, e)
			return
		}
		respond(w, 200, true)
	})
	mux.HandleFunc("POST /smart/status", func(w http.ResponseWriter, r *http.Request) {
		var v probe.WorkerStatus
		if json.NewDecoder(r.Body).Decode(&v) != nil {
			failure(w, ErrInvalid)
			return
		}
		if e := s.SetSmartWorker(r.Context(), v); e != nil {
			failure(w, e)
			return
		}
		respond(w, 200, true)
	})
}
