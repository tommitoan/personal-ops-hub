package connectors

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestParsePrometheusRequests(t *testing.T) {
	v, e := ParsePrometheusRequests(`# HELP http_requests_total Requests
http_requests_total{method="GET",status="200"} 12
http_requests_total{method="POST",status="500"} 3
other_total 900
`)
	if e != nil || v != 15 {
		t.Fatalf("got %v, %v", v, e)
	}
	if _, e = ParsePrometheusRequests("other_total 2"); e == nil {
		t.Fatal("missing metric accepted")
	}
	if _, e = ParsePrometheusRequests("http_requests_total -1"); e == nil {
		t.Fatal("negative metric accepted")
	}
}

func TestGraphQLFixtureAndErrors(t *testing.T) {
	sample, err := os.ReadFile("../../testdata/railway-billing.json")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test" {
			t.Error("missing auth header")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(sample)
	}))
	defer server.Close()
	collector := New()
	var data map[string]json.RawMessage
	if err := collector.graphql(context.Background(), server.URL, "test", "query { x }", nil, &data); err != nil {
		t.Fatal(err)
	}
	if len(data["workspace"]) == 0 {
		t.Fatal("workspace absent")
	}
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"data":null,"errors":[{"message":"denied"}]}`))
	}))
	defer bad.Close()
	if err := collector.graphql(context.Background(), bad.URL, "test", "query { x }", nil, &data); err == nil {
		t.Fatal("GraphQL errors accepted")
	}
}
