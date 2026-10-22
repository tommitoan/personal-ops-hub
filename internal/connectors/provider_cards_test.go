package connectors

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tommitoan/personal-ops-hub/internal/config"
)

func TestRailwayBillingAndCPU(t *testing.T) {
	now := time.Date(2026, 10, 22, 1, 0, 0, 0, time.UTC)
	cfg := config.Config{RailwayToken: "test", RailwayWorkspaceID: "workspace", RailwayEnvironmentID: "production"}
	emptyMetrics := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Query string `json:"query"`
		}
		json.NewDecoder(r.Body).Decode(&request)
		if strings.Contains(request.Query, "currentUsage") {
			w.Write([]byte(`{"data":{"workspace":{"customer":{"currentUsage":4.5,"billingPeriod":{"start":"2026-10-01","end":"2026-11-01"}}}}}`))
			return
		}
		if emptyMetrics {
			w.Write([]byte(`{"data":{"metrics":[]}}`))
			return
		}
		w.Write([]byte(`{"data":{"metrics":[{"measurement":"CPU_USAGE","tags":{"serviceId":"one"},"values":[{"value":0.25}]},{"measurement":"CPU_USAGE","tags":{"serviceId":"two"},"values":[{"value":0.5}]}]}}`))
	}))
	defer server.Close()
	cards := New().railway(context.Background(), cfg, now, server.URL)
	if len(cards) != 2 || cards[0].Status != "healthy" || cards[0].Summary != "$4.50 current usage" || cards[1].Status != "healthy" || !strings.Contains(cards[1].Summary, "0.750 vCPU") {
		t.Fatalf("unexpected Railway cards: %+v", cards)
	}
	emptyMetrics = true
	cards = New().railway(context.Background(), cfg, now, server.URL)
	if cards[1].Status != "stale" {
		t.Fatalf("empty metrics shown as zero: %+v", cards[1])
	}
}

func TestR2BucketTotalsAndEmptyResponse(t *testing.T) {
	cfg := config.Config{CloudflareToken: "test", CloudflareAccountID: "account", R2Buckets: []string{"museum"}}
	empty := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Variables map[string]string `json:"variables"`
		}
		json.NewDecoder(r.Body).Decode(&request)
		if request.Variables["bucketName"] != "museum" {
			t.Errorf("wrong bucket filter: %+v", request.Variables)
		}
		if empty {
			w.Write([]byte(`{"data":{"viewer":{"accounts":[{"r2OperationsAdaptiveGroups":[],"r2StorageAdaptiveGroups":[],"r2BandwidthUsageAdaptiveGroups":[]}]}}}`))
			return
		}
		w.Write([]byte(`{"data":{"viewer":{"accounts":[{"r2OperationsAdaptiveGroups":[{"sum":{"requests":12}},{"sum":{"requests":3}}],"r2StorageAdaptiveGroups":[{"max":{"payloadSize":1073741824,"metadataSize":0,"objectCount":10}}],"r2BandwidthUsageAdaptiveGroups":[{"sum":{"bytesUpload":1073741824,"bytesDownload":2147483648}}]}]}}}`))
	}))
	defer server.Close()
	cards := New().r2(context.Background(), cfg, time.Now(), server.URL)
	if len(cards) != 1 || cards[0].Status != "healthy" || !strings.Contains(cards[0].Summary, "15 operations · 1.00 GiB stored · 2.00 GiB download") {
		t.Fatalf("unexpected R2 card: %+v", cards)
	}
	empty = true
	cards = New().r2(context.Background(), cfg, time.Now(), server.URL)
	if cards[0].Status != "stale" {
		t.Fatalf("empty R2 data shown as zero: %+v", cards[0])
	}
}

func TestGitHubContributions(t *testing.T) {
	cfg := config.Config{GitHubToken: "test", GitHubLogin: "tommitoan"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Variables map[string]string `json:"variables"`
		}
		json.NewDecoder(r.Body).Decode(&request)
		if request.Variables["login"] != "tommitoan" || request.Variables["from"] != "2026-01-01T00:00:00Z" {
			t.Errorf("wrong contribution window: %+v", request.Variables)
		}
		w.Write([]byte(`{"data":{"user":{"contributionsCollection":{"contributionCalendar":{"totalContributions":42}}}}}`))
	}))
	defer server.Close()
	card := New().github(context.Background(), cfg, time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC), server.URL)
	if card.Status != "healthy" || card.Summary != "42 contributions" || card.Window != "2026 year to date" {
		t.Fatalf("unexpected GitHub card: %+v", card)
	}
}

func TestRequestMetrics(t *testing.T) {
	status := http.StatusOK
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer local-test" {
			t.Error("missing metrics authorization")
		}
		w.WriteHeader(status)
		w.Write([]byte("http_requests_total{service=\"api\"} 12\n"))
	}))
	defer server.Close()
	collector := New()
	collector.SafeClient = server.Client()
	target := config.MetricTarget{Name: "api", URL: server.URL, BearerToken: "local-test"}
	card := collector.RequestMetrics(context.Background(), target, time.Now())
	if card.Status != "healthy" || card.Summary != "12 total since process start" {
		t.Fatalf("unexpected request card: %+v", card)
	}
	status = http.StatusServiceUnavailable
	card = collector.RequestMetrics(context.Background(), target, time.Now())
	if card.Status != "stale" {
		t.Fatalf("failed metrics endpoint accepted: %+v", card)
	}
}
