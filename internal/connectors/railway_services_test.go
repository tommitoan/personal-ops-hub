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

func TestRailwayServices(t *testing.T) {
	now := time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)
	cfg := config.Config{RailwayToken: "test", RailwayProjectID: "project", RailwayEnvironmentID: "production"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Query     string            `json:"query"`
			Variables map[string]string `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		if request.Variables["projectId"] != "project" || !strings.Contains(request.Query, "activeDeployments") {
			t.Errorf("unexpected GraphQL request: %+v", request)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":{"project":{"services":{"pageInfo":{"hasNextPage":false},"edges":[{"node":{"id":"two","name":"Web","serviceInstances":{"pageInfo":{"hasNextPage":false},"edges":[{"node":{"environmentId":"production","activeDeployments":[{"status":"SUCCESS"}]}}]}}},{"node":{"id":"one","name":"Worker","serviceInstances":{"pageInfo":{"hasNextPage":false},"edges":[{"node":{"environmentId":"production","activeDeployments":[]}}]}}}]}}}}`))
	}))
	defer server.Close()
	cards := New().railwayServices(context.Background(), cfg, now, server.URL)
	if len(cards) != 3 || cards[0].Status != "degraded" || cards[0].Summary != "2 services · 1 active · 1 without active deployment" {
		t.Fatalf("unexpected summary: %+v", cards)
	}
	if cards[1].Title != "Web" || cards[1].Status != "healthy" || cards[2].Title != "Worker" || cards[2].Status != "degraded" {
		t.Fatalf("unexpected service cards: %+v", cards)
	}
	if cards[2].Status == "down" {
		t.Fatal("no active deployment must not imply an HTTP outage")
	}
}

func TestRailwayServicesUnavailable(t *testing.T) {
	cfg := config.Config{RailwayToken: "test", RailwayProjectID: "project", RailwayEnvironmentID: "production"}
	for name, response := range map[string]string{
		"missing project": `{"data":{"project":null}}`,
		"partial page":    `{"data":{"project":{"services":{"pageInfo":{"hasNextPage":true},"edges":[]}}}}`,
		"provider error":  `{"data":null,"errors":[{"message":"denied"}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(response))
			}))
			defer server.Close()
			cards := New().railwayServices(context.Background(), cfg, time.Now(), server.URL)
			if len(cards) != 1 || cards[0].Status != "stale" {
				t.Fatalf("unexpected cards: %+v", cards)
			}
		})
	}
	cfg.RailwayProjectID = ""
	cards := New().railwayServices(context.Background(), cfg, time.Now(), "https://example.com")
	if len(cards) != 1 || cards[0].Status != "not configured" {
		t.Fatalf("unexpected unconfigured cards: %+v", cards)
	}
}
