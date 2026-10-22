package connectors

import "testing"

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
