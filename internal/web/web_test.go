package web

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/tommitoan/personal-ops-hub/internal/config"
	"github.com/tommitoan/personal-ops-hub/internal/core"
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
	r = httptest.NewRequest("POST", "/logout", nil)
	r.AddCookie(w.Result().Cookies()[0])
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusSeeOther || len(w.Result().Cookies()) == 0 || w.Result().Cookies()[0].MaxAge >= 0 {
		t.Fatalf("logout did not clear cookie: status=%d cookies=%+v", w.Code, w.Result().Cookies())
	}
}

func TestLoginPageAndSecureCookie(t *testing.T) {
	t.Setenv("COOKIE_SECURE", "true")
	s := New(config.Config{Username: "owner", Password: "correct-long-password", SessionSecret: "12345678901234567890123456789012"}, nil)
	h := s.Handler()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/login", nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `<html lang="vi">`) || !strings.Contains(w.Body.String(), "Tên đăng nhập") {
		t.Fatalf("unexpected login page: status=%d", w.Code)
	}
	form := url.Values{"username": {"owner"}, "password": {"correct-long-password"}}
	r := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusSeeOther || len(w.Result().Cookies()) == 0 || !w.Result().Cookies()[0].Secure || !w.Result().Cookies()[0].HttpOnly || w.Result().Cookies()[0].SameSite != http.SameSiteStrictMode {
		t.Fatalf("session cookie flags missing: status=%d cookies=%+v", w.Code, w.Result().Cookies())
	}
}

func TestDashboardGroupingAndEscaping(t *testing.T) {
	now := time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)
	cards := []core.Card{
		{Key: "railway:service:one", Title: "Worker", Status: "healthy", Summary: "Running", Source: "Railway", UpdatedAt: now},
		{Key: "site:example", Title: "<script>alert(1)</script>", Status: "down", Summary: "HTTPS failed", Source: "HTTPS probe", UpdatedAt: now},
		{Key: "railway:cost", Title: "Cost", Status: "not configured", Summary: "Add token", Source: "Railway", UpdatedAt: now},
	}
	view := buildDashboard(cards, now)
	if view.Total != 3 || view.Attention != 2 || len(view.Sections) != 2 || view.Sections[0].Title != "Site health" || view.Sections[1].Title != "Railway" {
		t.Fatalf("unexpected dashboard groups: %+v", view)
	}
	if view.Sections[1].Cards[0].Title != "Cost" {
		t.Fatalf("attention card not first: %+v", view.Sections[1].Cards)
	}
	var html bytes.Buffer
	if err := page.Execute(&html, view); err != nil {
		t.Fatal(err)
	}
	result := html.String()
	for _, expected := range []string{"Đến nội dung chính", "Lịch sử", "Nguồn: Railway", "not-configured", "&lt;script&gt;"} {
		if !strings.Contains(result, expected) {
			t.Errorf("dashboard missing %q", expected)
		}
	}
	if strings.Contains(result, "<script>alert(1)</script>") {
		t.Fatal("dashboard rendered untrusted HTML")
	}
}

func TestHistoryTemplateAndParams(t *testing.T) {
	now := time.Date(2026, 10, 22, 11, 0, 0, 0, time.FixedZone("ICT", 7*60*60))
	var html bytes.Buffer
	if err := historyPage.Execute(&html, struct {
		Key   string
		Items []core.Card
	}{Key: "site:<script>", Items: []core.Card{{Status: "stale", Summary: "Provider failed", UpdatedAt: now}}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html.String(), "Provider failed") || !strings.Contains(html.String(), "&lt;script&gt;") || !strings.Contains(html.String(), "22 Oct 2026 04:00:00") {
		t.Fatal("history page did not render safely")
	}
	for _, query := range []string{"", "?key=site%3Aone&limit=0", "?key=site%3Aone&limit=201", "?key=site%3Aone&limit=bad"} {
		w := httptest.NewRecorder()
		_, _, ok := historyParams(w, httptest.NewRequest("GET", "/history"+query, nil))
		if ok || w.Code != http.StatusBadRequest {
			t.Fatalf("accepted invalid history query %q: %d", query, w.Code)
		}
	}
	w := httptest.NewRecorder()
	key, limit, ok := historyParams(w, httptest.NewRequest("GET", "/history?key=site%3Aone&limit=20", nil))
	if !ok || key != "site:one" || limit != 20 {
		t.Fatalf("valid history query: %q %d %v", key, limit, ok)
	}
}

func TestPrivateRoutesAndSessionExpiry(t *testing.T) {
	s := New(config.Config{Username: "owner", Password: "correct-long-password", SessionSecret: "12345678901234567890123456789012"}, nil)
	h := s.Handler()
	for _, path := range []string{"/", "/history?key=site%3Aone", "/api/cards", "/api/history?key=site%3Aone"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/login" {
			t.Errorf("route %q not protected: %d", path, w.Code)
		}
		if w.Header().Get("Content-Security-Policy") == "" || w.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("route %q missing security headers", path)
		}
	}
	payload := strconv.FormatInt(time.Now().Add(-time.Minute).Unix(), 10) + ".nonce"
	cookie := &http.Cookie{Name: "hub_session", Value: payload + "." + s.sign(payload)}
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(cookie)
	if s.authenticated(r) {
		t.Fatal("expired session accepted")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/unknown", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("unknown route status %d", w.Code)
	}
}
