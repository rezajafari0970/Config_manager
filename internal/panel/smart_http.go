package panel

import (
	"encoding/json"
	"github.com/rezajafari0970/Config_manager/internal/probe"
	"github.com/rezajafari0970/Config_manager/internal/repair"
	"github.com/rezajafari0970/Config_manager/internal/smart"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type smartModule struct {
	s        *Server
	pipeline *SmartPipeline
}

func (s *Server) smartURL(single bool) string { return s.pipelineURL(s.store.Smart(), single) }
func (s *Server) pipelineURL(p *SmartPipeline, single bool) string {
	mode, file := "smart-jsonl-v1", "configs.jsonl"
	if single {
		mode, file = "smart-single-v1", "next.json"
	}
	mode = p.tokenMode(mode)
	return s.origin + p.path() + "-subscription/" + s.store.digest("subscription", mode) + "/" + file
}
func (m smartModule) Register(mux *http.ServeMux) {
	s := m.s
	p := m.pipeline
	base := p.path()
	mux.HandleFunc("GET /api"+base+"/settings", func(w http.ResponseWriter, r *http.Request) {
		settings, e := p.SmartSettings(r.Context())
		if e != nil {
			failure(w, e)
			return
		}
		respond(w, 200, map[string]any{"settings": settings})
	})
	tokenMode := p.tokenMode
	mux.HandleFunc("GET /api"+base, func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		entries, e := querySmart(r.Context(), p.reader(), false, page)
		if e != nil {
			failure(w, e)
			return
		}
		settings, e := p.SmartSettings(r.Context())
		if e != nil {
			failure(w, e)
			return
		}
		worker, e := p.SmartWorker(r.Context())
		if e != nil {
			failure(w, e)
			return
		}
		if time.Now().UnixMilli()-worker.Heartbeat > 30000 {
			worker.Status = "offline"
		}
		stats := map[string]int{}
		var total, ready, blocked, sources, eligible int
		e = p.reader().QueryRowContext(r.Context(), `SELECT (SELECT count(*) FROM smart_heads),(SELECT count(*) FROM smart_ready),(SELECT count(*) FROM smart_heads h WHERE state='blocked' AND NOT EXISTS(SELECT 1 FROM smart_ready r WHERE r.id=h.id)),(SELECT count(DISTINCT source) FROM smart_offers),(SELECT count(DISTINCT id) FROM smart_offers)`).Scan(&total, &ready, &blocked, &sources, &eligible)
		if e != nil {
			failure(w, e)
			return
		}
		if p.ads {
			var mainEligible, adsEligible, mainSources, adsSources int
			e = p.Store.reader().QueryRowContext(r.Context(), `SELECT (SELECT count(DISTINCT id) FROM smart_offers),(SELECT count(DISTINCT id) FROM splash_smart_offers),(SELECT count(DISTINCT source) FROM smart_offers),(SELECT count(DISTINCT source) FROM splash_smart_offers)`).Scan(&mainEligible, &adsEligible, &mainSources, &adsSources)
			if e != nil {
				failure(w, e)
				return
			}
			stats["main_eligible"] = mainEligible
			stats["ads_eligible"] = adsEligible
			stats["main_sources"] = mainSources
			stats["ads_sources"] = adsSources
			eligible = adsEligible
		}
		stats["total"] = total
		stats["ready"] = ready
		stats["blocked"] = blocked
		stats["warming"] = max(0, total-ready-blocked)
		stats["sources"] = sources
		stats["eligible"] = eligible
		availability, e := p.availability(r.Context())
		if e != nil {
			failure(w, e)
			return
		}
		respond(w, 200, map[string]any{"availability": availability, "entries": entries, "settings": settings, "worker": worker, "stats": stats, "export_url": s.pipelineURL(p, false), "single_export_url": s.pipelineURL(p, true), "version": p.version(), "android_import": "not_run"})
	})
	mux.HandleFunc("POST /api"+base+"/settings", func(w http.ResponseWriter, r *http.Request) {
		var v SmartSettings
		if decode(r, &v) != nil || v.Count < 1 {
			respond(w, 422, map[string]string{"error": "تعداد اعضای هر اسمارت باید یک عدد صحیح مثبت باشد."})
			return
		}
		if e := p.SaveSmartSettings(r.Context(), v); e != nil {
			failure(w, e)
			return
		}
		respond(w, 200, true)
	})
	mux.HandleFunc("GET "+base+"/json/{id}", func(w http.ResponseWriter, r *http.Request) {
		var v SmartEntry
		e := p.reader().QueryRowContext(r.Context(), "SELECT id,hash,member_count,created,expires FROM smart_ready WHERE id=?", r.PathValue("id")).Scan(&v.ID, &v.Hash, &v.Count, &v.Created, &v.Expires)
		if e != nil {
			failure(w, ErrNotFound)
			return
		}
		v.Available = true
		b, e := p.smartBytes(r.Context(), v)
		if e != nil {
			failure(w, ErrNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("X-Content-SHA256", v.Hash)
		w.Write(b)
	})
	mux.HandleFunc("GET "+base+"-subscription/{token}/configs.jsonl", func(w http.ResponseWriter, r *http.Request) {
		if !equal(r.PathValue("token"), s.store.digest("subscription", tokenMode("smart-jsonl-v1"))) {
			http.NotFound(w, r)
			return
		}
		rows, e := querySmart(r.Context(), p.reader(), true, 0)
		if e != nil {
			failure(w, e)
			return
		}
		out, e := newExportSpool()
		if e != nil {
			failure(w, e)
			return
		}
		defer out.Close()
		count := 0
		var selected []SmartEntry
		seen := map[string]bool{}
		rand.Shuffle(len(rows), func(i, j int) { rows[i], rows[j] = rows[j], rows[i] })
		for _, v := range rows {
			if r.Context().Err() != nil {
				return
			}
			if !v.Available || seen[v.Hash] {
				continue
			}
			b, e := p.smartBytes(r.Context(), v)
			if e != nil {
				continue
			}
			seen[v.Hash] = true
			b, e = smart.Shuffle(b)
			if e != nil {
				failure(w, e)
				return
			}
			if e = out.Line(b); e != nil {
				failure(w, e)
				return
			}
			count++
			selected = append(selected, v)
		}
		valid, e := p.validateSmartExport(r.Context(), selected)
		if e != nil {
			failure(w, e)
			return
		}
		if count == 0 || !valid {
			smartHeaders(w)
			status, e := p.availability(r.Context())
			if e != nil {
				failure(w, e)
				return
			}
			if !valid {
				status.Available = false
				status.State = "inventory_changed"
				status.Message = "اعتبار یکی از اعضا هنگام آماده‌سازی تغییر کرد؛ خروجی دوباره بررسی می‌شود."
			}
			// Full export tokens cannot be used with the single-status polling route.
			w.Header().Set("X-Config-Count", "0")
			w.Header().Set("X-Smart-State", status.State)
			w.Header().Set("Retry-After", "15")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(status.Message + "\n"))
			return
		}
		if e = out.Rewind(); e != nil {
			failure(w, e)
			return
		}
		smartHeaders(w)
		w.Header().Set("X-Config-Count", strconv.Itoa(count))
		w.Header().Set("X-Content-SHA256", out.Hash())
		_, _ = out.WriteTo(w)
	})
	mux.HandleFunc("GET "+base+"-subscription/{token}/status.json", func(w http.ResponseWriter, r *http.Request) {
		if !equal(r.PathValue("token"), s.store.digest("subscription", tokenMode("smart-single-v1"))) {
			http.NotFound(w, r)
			return
		}
		v, e := p.availability(r.Context())
		if e != nil {
			failure(w, e)
			return
		}
		respond(w, http.StatusOK, v)
	})
	mux.HandleFunc("GET "+base+"-subscription/{token}/next.json", func(w http.ResponseWriter, r *http.Request) {
		if !equal(r.PathValue("token"), s.store.digest("subscription", tokenMode("smart-single-v1"))) {
			http.NotFound(w, r)
			return
		}
		smartHeaders(w)
		if r.Method == http.MethodHead {
			status, e := p.availability(r.Context())
			if e != nil {
				failure(w, e)
				return
			}
			if !status.Available {
				m.unavailable(w, r, status)
			} else {
				w.Header().Set("X-Config-Count", "1")
				w.Header().Set("X-Smart-State", "ready")
			}
			return
		}
		if strings.Contains(strings.ToLower(r.Header.Get("Sec-Purpose")+" "+r.Header.Get("Purpose")), "prefetch") {
			w.WriteHeader(204)
			return
		}
		v, b, cycle, e := p.smartNext(r.Context())
		if e != nil {
			failure(w, e)
			return
		}
		if v == nil {
			status, e := p.availability(r.Context())
			if e != nil {
				failure(w, e)
				return
			}
			m.unavailable(w, r, status)
			return
		}
		w.Header().Set("X-Config-Count", "1")
		w.Header().Set("X-Export-Cycle", strconv.FormatInt(cycle, 10))
		b, e = smart.Shuffle(b)
		if e != nil {
			failure(w, e)
			return
		}
		w.Header().Set("X-Content-SHA256", repair.Digest(b))
		w.Header().Set("X-Smart-Cache-SHA256", v.Hash)
		w.Header().Set("X-Smart-First-ID", v.ID) // Legacy profile identity, not the first outbound.
		w.Header().Set("X-Smart-Profile-ID", v.ID)
		w.Write(b)
		w.Write([]byte{'\n'})
	})
}
func smartHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Smart-Order", "random-per-request; independent-ip-first")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("X-Android-Import", "not-run")
	w.Header().Set("X-Verification-Scope", "pinned-core-smart-profile; member-connectivity-is-separate")
}
func (s *Store) registerPrivateSmart(mux *http.ServeMux) {
	s.Smart().registerPrivate(mux)
	s.SplashSmart().registerPrivate(mux)
	s.AdsSmart().registerPrivate(mux)
}
func (s *SmartPipeline) registerPrivate(mux *http.ServeMux) {
	mux.HandleFunc("POST "+s.path()+"/claim", func(w http.ResponseWriter, r *http.Request) {
		j, e := s.ClaimSmart(r.Context())
		if e != nil {
			failure(w, e)
			return
		}
		respond(w, 200, j)
	})
	mux.HandleFunc("POST "+s.path()+"/renew", func(w http.ResponseWriter, r *http.Request) {
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
	mux.HandleFunc("POST "+s.path()+"/complete", func(w http.ResponseWriter, r *http.Request) {
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
	mux.HandleFunc("POST "+s.path()+"/status", func(w http.ResponseWriter, r *http.Request) {
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
