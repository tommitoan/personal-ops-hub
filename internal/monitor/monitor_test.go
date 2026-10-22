package monitor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"

	"github.com/tommitoan/personal-ops-hub/internal/config"
	"github.com/tommitoan/personal-ops-hub/internal/core"
)

func TestTransition(t *testing.T) {
	now := time.Now()
	s := core.IncidentState{}
	var event string
	s, event = core.Transition(s, false, 2, now)
	if event != "" {
		t.Fatal(event)
	}
	s, event = core.Transition(s, false, 2, now)
	if event != "down" || !s.Open {
		t.Fatal(s, event)
	}
	s, event = core.Transition(s, false, 2, now)
	if event != "" {
		t.Fatal(event)
	}
	s, event = core.Transition(s, true, 2, now)
	if event != "resolved" || s.Open {
		t.Fatal(s, event)
	}
}
func TestPrivateAddressRejected(t *testing.T) {
	for _, s := range []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "100.64.0.1", "198.18.0.1", "::1", "fc00::1"} {
		if publicIP(netip.MustParseAddr(s)) {
			t.Fatalf("accepted %s", s)
		}
	}
	if !publicIP(netip.MustParseAddr("8.8.8.8")) {
		t.Fatal("public address rejected")
	}
}

func TestProbeStatusAndTLS(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) }))
	defer server.Close()
	checker := &Checker{Client: server.Client()}
	got := checker.Probe(context.Background(), config.Target{Name: "test", URL: server.URL}, time.Minute)
	if got.Status != "down" || got.UpdatedAt.IsZero() || got.ExpiresAt.Before(got.UpdatedAt) {
		t.Fatalf("unexpected card: %+v", got)
	}
}
