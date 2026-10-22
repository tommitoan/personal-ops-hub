package connectors

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
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
	if _, e = ParsePrometheusRequests("http_requests_total NaN"); e == nil {
		t.Fatal("NaN metric accepted")
	}
	if _, e = ParsePrometheusRequests("http_requests_total +Inf"); e == nil {
		t.Fatal("infinite metric accepted")
	}
	v, e = ParsePrometheusRequests("http_requests_total{route=\"/path with space\"} 7\n")
	if e != nil || v != 7 {
		t.Fatalf("metric with spaced label: %v, %v", v, e)
	}
	v, e = ParsePrometheusRequests("http_requests_total{route=\"/path}x\"} 9 # {trace_id=\"abc\"}\n")
	if e != nil || v != 9 {
		t.Fatalf("metric with brace in label and exemplar: %v, %v", v, e)
	}
}

func TestGraphQLRetryIsBounded(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte(`{"data":{"ok":true}}`))
	}))
	defer server.Close()
	var data struct {
		OK bool `json:"ok"`
	}
	if err := New().graphql(context.Background(), server.URL, "test", "query { ok }", nil, &data); err != nil || !data.OK || calls.Load() != 2 {
		t.Fatalf("retry failed: data=%+v calls=%d err=%v", data, calls.Load(), err)
	}
	calls.Store(0)
	forbidden := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusForbidden)
	}))
	defer forbidden.Close()
	if err := New().graphql(context.Background(), forbidden.URL, "test", "query { ok }", nil, &data); err == nil || calls.Load() != 1 {
		t.Fatalf("authorization error retried: calls=%d err=%v", calls.Load(), err)
	}
	calls.Store(0)
	limited := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer limited.Close()
	if err := New().graphql(context.Background(), limited.URL, "test", "query { ok }", nil, &data); err == nil || calls.Load() != 1 {
		t.Fatalf("rate limit retried: calls=%d err=%v", calls.Load(), err)
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
