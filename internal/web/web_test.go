package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/tommitoan/personal-ops-hub/internal/config"
)

func TestLoginAndProtectedRoute(t *testing.T) {
	s := New(config.Config{Username: "owner", Password: "correct-long-password", SessionSecret: "12345678901234567890123456789012"}, nil)
	s.secureCookie = false
	h := s.Handler()
	r := httptest.NewRequest("GET", "/api/cards", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("unauthenticated status %d", w.Code)
	}
	form := url.Values{"username": {"owner"}, "password": {"correct-long-password"}}
	r = httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusSeeOther || len(w.Result().Cookies()) == 0 {
		t.Fatalf("login status %d", w.Code)
	}
	r = httptest.NewRequest("GET", "/api/cards", nil)
	r.AddCookie(w.Result().Cookies()[0])
	if !s.authenticated(r) {
		t.Fatal("valid cookie rejected")
	}
	cookie := w.Result().Cookies()[0]
	cookie.Value += "x"
	r = httptest.NewRequest("GET", "/api/cards", nil)
	r.AddCookie(cookie)
	if s.authenticated(r) {
		t.Fatal("tampered cookie accepted")
	}
}
