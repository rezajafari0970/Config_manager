package panel

import (
	"context"
	"crypto/subtle"
	"embed"
	"errors"
	"html/template"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

//go:embed web/*
var assets embed.FS

type Server struct {
	store     *Store
	origin    string
	host      string
	secure    bool
	template  *template.Template
	authSlots chan struct{}
}
type view struct {
	Page, Title, Error, CSRF, LoginCSRF string
	Sources                             []Source
	Source                              Source
	Total, Active, Paused               int
	Editing                             bool
	Message                             string
}

func NewServer(store *Store, origin string) (*Server, error) {
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("public origin must be scheme://host[:port]")
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost" || u.Hostname() == "::1")) {
		return nil, errors.New("HTTPS public origin required")
	}
	t, err := template.New("page.html").Funcs(template.FuncMap{"unit": func(v string) string {
		return map[string]string{"seconds": "ثانیه", "minutes": "دقیقه", "hours": "ساعت"}[v]
	}}).ParseFS(assets, "web/page.html")
	if err != nil {
		return nil, err
	}
	return &Server{store: store, origin: origin, host: u.Host, secure: u.Scheme == "https", template: t, authSlots: make(chan struct{}, 2)}, nil
}
func (s *Server) cookieName(kind string) string {
	if s.secure {
		return "__Host-cr_" + kind
	}
	return "cr_" + kind
}
func (s *Server) cookie(w http.ResponseWriter, kind, value string, expires time.Time) {
	max := 0
	if value == "" {
		max = -1
	}
	http.SetCookie(w, &http.Cookie{Name: s.cookieName(kind), Value: value, Path: "/", HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteStrictMode, Expires: expires, MaxAge: max})
}
func (s *Server) token(r *http.Request, kind string) string {
	c, e := r.Cookie(s.cookieName(kind))
	if e != nil {
		return ""
	}
	return c.Value
}
func (s *Server) csrf(r *http.Request) string { return s.store.digest("csrf", s.token(r, "session")) }
func equal(a, b string) bool {
	return len(a) == len(b) && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
func (s *Server) render(w http.ResponseWriter, r *http.Request, status int, v view) {
	if v.Page != "login" {
		v.CSRF = s.csrf(r)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := s.template.Execute(w, v); err != nil {
		log.Print("panel template rendering failed")
	}
}
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	static, _ := fs.Sub(assets, "web")
	mux.Handle("GET /static/style.css", http.StripPrefix("/static/", http.FileServer(http.FS(static))))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if s.store.Ping(r.Context()) != nil {
			http.Error(w, "unavailable", 503)
			return
		}
		w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("GET /login", s.loginPage)
	mux.HandleFunc("POST /login", s.login)
	mux.HandleFunc("GET /{$}", s.protected(s.dashboard))
	mux.HandleFunc("POST /logout", s.protected(s.logout))
	mux.HandleFunc("GET /sources/new", s.protected(s.sourceForm))
	mux.HandleFunc("POST /sources", s.protected(s.saveSource))
	mux.HandleFunc("GET /sources/{id}/edit", s.protected(s.sourceForm))
	mux.HandleFunc("POST /sources/{id}/edit", s.protected(s.saveSource))
	mux.HandleFunc("POST /sources/{id}/toggle", s.protected(s.toggleSource))
	mux.HandleFunc("GET /sources/{id}/delete", s.protected(s.deleteForm))
	mux.HandleFunc("POST /sources/{id}/delete", s.protected(s.deleteSource))
	mux.HandleFunc("GET /settings", s.protected(func(w http.ResponseWriter, r *http.Request) {
		s.render(w, r, 200, view{Page: "settings", Title: "امنیت حساب"})
	}))
	mux.HandleFunc("POST /settings/password", s.protected(s.changePassword))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'self'; img-src 'self'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'")
		w.Header().Set("X-Frame-Options", "DENY")
		if r.Host != s.host && r.URL.Path != "/healthz" {
			http.Error(w, "invalid host", 400)
			return
		}
		if r.Method == http.MethodPost {
			if r.Header.Get("Origin") != s.origin {
				http.Error(w, "درخواست از مبدأ نامعتبر است.", 403)
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
			if err := r.ParseForm(); err != nil {
				http.Error(w, "اطلاعات فرم معتبر نیست یا بیش از حد طولانی است.", 400)
				return
			}
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		mux.ServeHTTP(w, r.WithContext(ctx))
	})
}
func (s *Server) protected(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.store.Session(r.Context(), s.token(r, "session")) {
			http.Redirect(w, r, "/login", 303)
			return
		}
		if r.Method == "POST" && !equal(r.PostForm.Get("csrf"), s.csrf(r)) {
			http.Error(w, "فرم منقضی شده است؛ صفحه را دوباره باز کنید.", 403)
			return
		}
		next(w, r)
	}
}
func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) {
	if s.store.Session(r.Context(), s.token(r, "session")) {
		http.Redirect(w, r, "/", 303)
		return
	}
	nonce := randomToken()
	s.cookie(w, "login", nonce, time.Now().Add(10*time.Minute))
	message := ""
	if r.URL.Query().Get("changed") == "1" {
		message = "رمز عبور تغییر کرد. دوباره وارد شوید."
	}
	s.render(w, r, 200, view{Page: "login", Title: "ورود به پنل", LoginCSRF: nonce, Message: message})
}
func (s *Server) authPermit(w http.ResponseWriter, r *http.Request) bool {
	select {
	case s.authSlots <- struct{}{}:
	default:
		w.Header().Set("Retry-After", "5")
		http.Error(w, "چند لحظه دیگر دوباره تلاش کنید.", 429)
		return false
	}
	client, _, _ := net.SplitHostPort(r.RemoteAddr)
	if ip := net.ParseIP(client); ip != nil && ip.IsLoopback() {
		if real := net.ParseIP(r.Header.Get("X-Real-IP")); real != nil {
			client = real.String()
		}
	}
	ok, err := s.store.AllowAuth(r.Context(), client)
	if err != nil || !ok {
		<-s.authSlots
		w.Header().Set("Retry-After", "900")
		http.Error(w, "تعداد تلاش‌ها زیاد است. ۱۵ دقیقه دیگر دوباره تلاش کنید.", 429)
		return false
	}
	return true
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	nonce := s.token(r, "login")
	if len(nonce) != 43 || !equal(nonce, r.PostForm.Get("csrf")) {
		http.Error(w, "فرم ورود منقضی شده است؛ صفحه را دوباره باز کنید.", 403)
		return
	}
	if !s.authPermit(w, r) {
		return
	}
	defer func() { <-s.authSlots }()
	token, expiry, err := s.store.Authenticate(r.Context(), r.PostForm.Get("password"), r.PostForm.Get("remember") == "on")
	if err != nil {
		s.render(w, r, 401, view{Page: "login", Title: "ورود به پنل", Error: "ورود انجام نشد. رمز عبور را بررسی کنید.", LoginCSRF: nonce})
		return
	}
	if previous := s.token(r, "session"); previous != "" {
		_ = s.store.Logout(r.Context(), previous)
	}
	s.cookie(w, "session", token, expiry)
	s.cookie(w, "login", "", time.Time{})
	http.Redirect(w, r, "/", 303)
}
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Logout(r.Context(), s.token(r, "session")); err != nil {
		s.failure(w, r, err)
		return
	}
	s.cookie(w, "session", "", time.Time{})
	http.Redirect(w, r, "/login", 303)
}
func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	sources, err := s.store.List(r.Context())
	if err != nil {
		s.failure(w, r, err)
		return
	}
	v := view{Page: "dashboard", Title: "منابع کانفیگ", Sources: sources, Total: len(sources)}
	for _, source := range sources {
		if source.Enabled {
			v.Active++
		} else {
			v.Paused++
		}
	}
	s.render(w, r, 200, v)
}
func (s *Server) failure(w http.ResponseWriter, r *http.Request, err error) {
	status := 500
	message := "عملیات انجام نشد؛ دوباره تلاش کنید."
	if errors.Is(err, ErrNotFound) {
		status = 404
		message = "این منبع وجود ندارد."
	}
	if errors.Is(err, ErrConflict) {
		status = 409
		message = "این منبع تغییر کرده یا لینک تکراری است. فهرست را تازه کنید."
	}
	if errors.Is(err, ErrInvalid) {
		status = 422
		message = "اطلاعات منبع معتبر نیست."
	}
	s.render(w, r, status, view{Page: "error", Title: "عملیات انجام نشد", Error: message})
}
func (s *Server) sourceForm(w http.ResponseWriter, r *http.Request) {
	v := view{Page: "form", Title: "افزودن منبع", Source: Source{SourceInput: SourceInput{Min: 15, Max: 20, Unit: "minutes", Enabled: true}}}
	if id := r.PathValue("id"); id != "" {
		source, err := s.store.Get(r.Context(), id)
		if err != nil {
			s.failure(w, r, err)
			return
		}
		v.Source = source
		v.Editing = true
		v.Title = "ویرایش منبع"
	}
	s.render(w, r, 200, v)
}
func (s *Server) saveSource(w http.ResponseWriter, r *http.Request) {
	min, _ := strconv.ParseInt(r.PostForm.Get("min"), 10, 64)
	max, _ := strconv.ParseInt(r.PostForm.Get("max"), 10, 64)
	rev, _ := strconv.ParseInt(r.PostForm.Get("revision"), 10, 64)
	input := SourceInput{Name: strings.TrimSpace(r.PostForm.Get("name")), URL: strings.TrimSpace(r.PostForm.Get("url")), Min: min, Max: max, Unit: r.PostForm.Get("unit"), Enabled: r.PostForm.Get("enabled") == "on"}
	id := r.PathValue("id")
	_, err := s.store.Save(r.Context(), id, rev, input)
	if err != nil {
		if errors.Is(err, ErrInvalid) || errors.Is(err, ErrConflict) {
			message := "نام، لینک عمومی HTTP/HTTPS و بازهٔ زمانی را بررسی کنید. حداکثر بازه ۳۰ روز است."
			status := 422
			if errors.Is(err, ErrConflict) {
				message = "لینک تکراری است یا منبع در صفحهٔ دیگری تغییر کرده؛ صفحه را دوباره باز کنید."
				status = 409
			}
			s.render(w, r, status, view{Page: "form", Title: "ذخیرهٔ منبع", Source: Source{SourceInput: input, ID: id, Revision: rev}, Editing: id != "", Error: message})
			return
		}
		s.failure(w, r, err)
		return
	}
	http.Redirect(w, r, "/", 303)
}
func (s *Server) toggleSource(w http.ResponseWriter, r *http.Request) {
	source, err := s.store.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		s.failure(w, r, err)
		return
	}
	rev, _ := strconv.ParseInt(r.PostForm.Get("revision"), 10, 64)
	source.Enabled = !source.Enabled
	_, err = s.store.Save(r.Context(), source.ID, rev, source.SourceInput)
	if err != nil {
		s.failure(w, r, err)
		return
	}
	http.Redirect(w, r, "/", 303)
}
func (s *Server) deleteForm(w http.ResponseWriter, r *http.Request) {
	source, err := s.store.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		s.failure(w, r, err)
		return
	}
	s.render(w, r, 200, view{Page: "delete", Title: "حذف منبع", Source: source})
}
func (s *Server) deleteSource(w http.ResponseWriter, r *http.Request) {
	rev, _ := strconv.ParseInt(r.PostForm.Get("revision"), 10, 64)
	if err := s.store.Delete(r.Context(), r.PathValue("id"), rev); err != nil {
		s.failure(w, r, err)
		return
	}
	http.Redirect(w, r, "/", 303)
}
func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	if !s.authPermit(w, r) {
		return
	}
	defer func() { <-s.authSlots }()
	if r.PostForm.Get("next") != r.PostForm.Get("confirm") {
		s.render(w, r, 422, view{Page: "settings", Title: "امنیت حساب", Error: "تکرار رمز عبور یکسان نیست."})
		return
	}
	if err := s.store.ChangePassword(r.Context(), r.PostForm.Get("current"), r.PostForm.Get("next")); err != nil {
		s.render(w, r, 422, view{Page: "settings", Title: "امنیت حساب", Error: "رمز فعلی را بررسی کنید. رمز جدید باید حداقل ۱۴ نویسهٔ انگلیسی باشد."})
		return
	}
	s.cookie(w, "session", "", time.Time{})
	http.Redirect(w, r, "/login?changed=1", 303)
}
