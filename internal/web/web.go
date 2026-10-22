package web

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"encoding/base64"
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tommitoan/personal-ops-hub/internal/config"
	"github.com/tommitoan/personal-ops-hub/internal/core"
	"github.com/tommitoan/personal-ops-hub/internal/store"
)

//go:embed dashboard.html
var files embed.FS
var page = template.Must(template.ParseFS(files, "dashboard.html"))

// Server serves a private owner-only dashboard and read-only JSON endpoints.
type Server struct {
	Config       config.Config
	Store        *store.Store
	secureCookie bool
}

// New creates an HTTP handler with fixed routes and secure session defaults.
func New(cfg config.Config, st *store.Store) *Server {
	return &Server{Config: cfg, Store: st, secureCookie: os.Getenv("COOKIE_SECURE") != "false"}
}

// Handler returns the hub router.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /login", s.loginPage)
	mux.HandleFunc("POST /login", s.login)
	mux.HandleFunc("POST /logout", s.private(s.logout))
	mux.HandleFunc("GET /", s.private(s.dashboard))
	mux.HandleFunc("GET /api/cards", s.private(s.cards))
	mux.HandleFunc("GET /api/history", s.private(s.history))
	return securityHeaders(mux)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'none'; form-action 'self'")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if s.Store == nil || s.Store.Pool.Ping(ctx) != nil {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Write([]byte("ok"))
}
func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) {
	if s.authenticated(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!doctype html><html><head><meta name="viewport" content="width=device-width, initial-scale=1"><title>Personal Ops Hub</title><style>body{background:#0d1117;color:#e6edf3;font:16px system-ui;max-width:360px;margin:12vh auto;padding:20px}input,button{display:block;width:100%;box-sizing:border-box;padding:12px;margin:12px 0;border-radius:8px}button{background:#58a6ff;border:0;font-weight:bold}</style></head><body><h1>Personal Ops Hub</h1><form method="post" action="/login"><label>Username<input name="username" autocomplete="username" required></label><label>Password<input type="password" name="password" autocomplete="current-password" required></label><button>Sign in</button></form></body></html>`))
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if e := r.ParseForm(); e != nil {
		http.Error(w, "invalid form", 400)
		return
	}
	user := r.Form.Get("username")
	password := r.Form.Get("password")
	validUser := subtle.ConstantTimeCompare([]byte(user), []byte(s.Config.Username)) == 1
	validPass := subtle.ConstantTimeCompare([]byte(password), []byte(s.Config.Password)) == 1
	if !validUser || !validPass {
		time.Sleep(300 * time.Millisecond)
		http.Error(w, "invalid credentials", 401)
		return
	}
	nonce := make([]byte, 16)
	if _, e := rand.Read(nonce); e != nil {
		http.Error(w, "session failed", 500)
		return
	}
	expiry := time.Now().Add(12 * time.Hour).Unix()
	payload := strconv.FormatInt(expiry, 10) + "." + base64.RawURLEncoding.EncodeToString(nonce)
	sig := s.sign(payload)
	http.SetCookie(w, &http.Cookie{Name: "hub_session", Value: payload + "." + sig, Path: "/", HttpOnly: true, Secure: s.secureCookie, SameSite: http.SameSiteStrictMode, MaxAge: 12 * 3600})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
func (s *Server) sign(payload string) string {
	mac := hmac.New(sha256.New, []byte(s.Config.SessionSecret))
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
func (s *Server) authenticated(r *http.Request) bool {
	cookie, e := r.Cookie("hub_session")
	if e != nil {
		return false
	}
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 3 {
		return false
	}
	expiry, e := strconv.ParseInt(parts[0], 10, 64)
	if e != nil || time.Now().Unix() > expiry {
		return false
	}
	payload := parts[0] + "." + parts[1]
	expected := s.sign(payload)
	return hmac.Equal([]byte(expected), []byte(parts[2]))
}
func (s *Server) private(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.authenticated(r) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "hub_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.secureCookie, SameSite: http.SameSiteStrictMode})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	cards, e := s.Store.Cards(r.Context())
	if e != nil {
		http.Error(w, "dashboard unavailable", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	page.Execute(w, struct {
		Cards []core.Card
		Now   time.Time
	}{cards, time.Now().UTC()})
}
func (s *Server) cards(w http.ResponseWriter, r *http.Request) {
	cards, e := s.Store.Cards(r.Context())
	if e != nil {
		http.Error(w, "cards unavailable", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cards)
}
func (s *Server) history(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if len(key) > 150 || key == "" {
		http.Error(w, "key required", 400)
		return
	}
	limit := 50
	if x := r.URL.Query().Get("limit"); x != "" {
		n, e := strconv.Atoi(x)
		if e != nil || n < 1 || n > 200 {
			http.Error(w, "invalid limit", 400)
			return
		}
		limit = n
	}
	items, e := s.Store.History(r.Context(), key, limit)
	if e != nil {
		http.Error(w, "history unavailable", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}
