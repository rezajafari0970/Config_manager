package panel

import (
	"context"
	"crypto/subtle"
	"embed"
	"encoding/json"
	"errors"
	"github.com/rezajafari0970/Config_manager/internal/collect"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

//go:embed web/*
var assets embed.FS

type Server struct {
	store        *Store
	origin, host string
	secure       bool
	template     *template.Template
	engine       *collect.Engine
	modules      []PanelModule
}

func NewServer(store *Store, origin string) (*Server, error) {
	u, e := url.Parse(origin)
	if e != nil || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("invalid public origin")
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost" || u.Hostname() == "::1")) {
		return nil, errors.New("HTTPS public origin required")
	}
	t, e := template.New("page.html").ParseFS(assets, "web/*.html")
	if e != nil {
		return nil, e
	}
	return &Server{store: store, origin: origin, host: u.Host, secure: u.Scheme == "https", template: t}, nil
}
func (s *Server) SetEngine(e *collect.Engine) { s.engine = e }
func (s *Server) cookieName() string {
	if s.secure {
		return "__Host-cr_workspace"
	}
	return "cr_workspace"
}
func (s *Server) token(r *http.Request) string {
	c, e := r.Cookie(s.cookieName())
	if e != nil {
		return ""
	}
	return c.Value
}
func (s *Server) csrf(token string) string { return s.store.digest("anonymous-csrf", token) }
func equal(a, b string) bool {
	return len(a) == len(b) && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
func respond(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func failure(w http.ResponseWriter, e error) {
	status := 500
	message := "عملیات انجام نشد؛ دوباره تلاش کنید."
	switch {
	case errors.Is(e, ErrInvalid):
		status = 422
		message = "لینک، بازهٔ درخواست و تایم‌اوت را بررسی کنید؛ نمونهٔ بازه: ۲ تا ۳."
	case errors.Is(e, ErrConflict):
		status = 409
		message = "لینک تکراری است یا وضعیت آن تغییر کرده است."
	case errors.Is(e, ErrNotFound):
		status = 404
		message = "این مورد وجود ندارد یا زمان نگهداری آن تمام شده است."
	}
	respond(w, status, map[string]any{"error": message})
}
func decode(r *http.Request, v any) error {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return ErrInvalid
	}
	if e := d.Decode(new(any)); e != io.EOF {
		return ErrInvalid
	}
	return nil
}
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	static, _ := fs.Sub(assets, "web")
	files := http.StripPrefix("/assets/", http.FileServer(http.FS(static)))
	for _, asset := range []string{"style.css", "app.js", "core.js", "input.js", "output.js", "setting.js", "actions.js", "repairs.js", "Vazirmatn.woff2"} {
		mux.Handle("GET /assets/"+asset, files)
	}
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if s.store.Ping(r.Context()) != nil {
			http.Error(w, "unavailable", 503)
			return
		}
		if s.engine != nil {
			running, _, tick := s.engine.Status()
			if !running || time.Now().UnixMilli()-tick > 30000 {
				http.Error(w, "worker unavailable", 503)
				return
			}
		}
		w.Write([]byte("ok\n"))
	})
	for _, path := range []string{"/", "/login", "/settings", "/sources/new"} {
		pattern := "GET " + path
		if path == "/" {
			pattern = "GET /{$}"
		}
		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/input", 303) })
	}
	for _, page := range []string{"input", "output", "setting"} {
		page := page
		mux.HandleFunc("GET /"+page, func(w http.ResponseWriter, r *http.Request) {
			token := s.token(r)
			if len(token) != 43 {
				token = randomToken()
				http.SetCookie(w, &http.Cookie{Name: s.cookieName(), Value: token, Path: "/", Secure: s.secure, HttpOnly: true, SameSite: http.SameSiteStrictMode, Expires: time.Now().Add(365 * 24 * time.Hour)})
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_ = s.template.Execute(w, map[string]string{"Page": page, "CSRF": s.csrf(token)})
		})
	}
	mux.HandleFunc("GET /api/state", s.state)
	modules := []PanelModule{inputModule{s}, outputModule{s}, settingModule{s}, conversionModule{s}, rotationModule{s}, sourceExportModule{s}}
	modules = append(modules, s.modules...)
	for _, module := range modules {
		module.Register(mux)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'self'; script-src 'self'; connect-src 'self'; img-src 'self'; font-src 'self'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
		w.Header().Set("X-Frame-Options", "DENY")
		if r.Host != s.host && r.URL.Path != "/healthz" {
			http.Error(w, "invalid host", 400)
			return
		}
		if r.Method == "POST" {
			token := s.token(r)
			if r.Header.Get("Origin") != s.origin || len(token) != 43 || !equal(r.Header.Get("X-CSRF-Token"), s.csrf(token)) {
				respond(w, 403, map[string]string{"error": "صفحه را تازه کنید و دوباره تلاش کنید."})
				return
			}
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancel()
		mux.ServeHTTP(w, r.WithContext(ctx))
	})
}
func (s *Server) state(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	sources, e := s.store.ListPage(ctx, r.URL.Query().Get("q"), page)
	if e != nil {
		failure(w, e)
		return
	}
	for i := range sources {
		sources[i].ExportURL = s.exportURL(sources[i].ID, false)
		sources[i].SingleExportURL = s.exportURL(sources[i].ID, true)
	}
	outputs, e := s.store.OutputsFiltered(ctx, r.URL.Query().Get("kind"), r.URL.Query().Get("repair"), page, false)
	if e != nil {
		failure(w, e)
		return
	}
	settings, e := s.store.Settings(ctx)
	if e != nil {
		failure(w, e)
		return
	}
	stats, e := s.store.Stats(ctx)
	if e != nil {
		failure(w, e)
		return
	}
	running, active, tick := false, 0, int64(0)
	blocked := ""
	capacity := 0
	if s.engine != nil {
		running, active, tick = s.engine.Status()
		blocked = s.engine.BlockedReason()
		capacity = s.engine.Capacity()
	}
	repairs, e := s.store.RepairSummary(ctx, outputs)
	if e != nil {
		failure(w, e)
		return
	}
	conversions, e := s.store.SourceConversionSummaries(ctx, sources)
	if e != nil {
		failure(w, e)
		return
	}
	respond(w, 200, map[string]any{"conversions": conversions, "repairs": repairs, "sources": sources, "outputs": outputs, "settings": settings, "stats": stats, "worker": map[string]any{"running": running, "active": active, "tick": tick, "blocked": blocked, "capacity": capacity}, "now": time.Now().UnixMilli()})
}

// PanelModule is the compiled extension boundary for independent feature routes.
type PanelModule interface{ Register(*http.ServeMux) }

func (s *Server) RegisterModule(module PanelModule) { s.modules = append(s.modules, module) }
