package connectors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/tommitoan/personal-ops-hub/internal/config"
	"github.com/tommitoan/personal-ops-hub/internal/core"
	"github.com/tommitoan/personal-ops-hub/internal/monitor"
)

// Collector fetches provider summaries with bounded HTTP calls.
type Collector struct {
	Client     *http.Client
	SafeClient *http.Client
}

// New returns a collector using fixed provider endpoints and a safe client for owner-configured endpoints.
func New() *Collector {
	return &Collector{Client: &http.Client{Timeout: 12 * time.Second}, SafeClient: monitor.NewChecker().Client}
}

func card(key, title, source, link string, now time.Time) core.Card {
	return core.Card{Key: key, Title: title, Status: "not configured", Summary: "Add provider configuration", Source: source, Link: link, UpdatedAt: now, ExpiresAt: now.Add(2 * time.Hour)}
}
func failed(c core.Card) core.Card {
	c.Status = "stale"
	c.Summary = "Provider request failed; open source dashboard"
	return c
}

func (c *Collector) graphql(ctx context.Context, endpoint, token, query string, variables any, dst any) error {
	payload, e := json.Marshal(map[string]any{"query": query, "variables": variables})
	if e != nil {
		return e
	}
	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "personal-ops-hub/1")
		res, err := c.Client.Do(req)
		if err != nil {
			if attempt == 0 && ctx.Err() == nil {
				if err := retryDelay(ctx); err != nil {
					return err
				}
				continue
			}
			return err
		}
		if attempt == 0 && res.StatusCode >= 500 {
			res.Body.Close()
			if err := retryDelay(ctx); err != nil {
				return err
			}
			continue
		}
		if res.StatusCode != http.StatusOK {
			res.Body.Close()
			return fmt.Errorf("provider HTTP %d", res.StatusCode)
		}
		var envelope struct {
			Data   json.RawMessage   `json:"data"`
			Errors []json.RawMessage `json:"errors"`
		}
		err = json.NewDecoder(io.LimitReader(res.Body, 2<<20)).Decode(&envelope)
		res.Body.Close()
		if err != nil {
			return err
		}
		if len(envelope.Errors) > 0 {
			return errors.New("provider GraphQL errors")
		}
		if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
			return errors.New("provider returned no data")
		}
		return json.Unmarshal(envelope.Data, dst)
	}
	return errors.New("provider retry exhausted")
}

func retryDelay(ctx context.Context) error {
	timer := time.NewTimer(250 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Railway fetches billing-period usage and current resource measurements.
func (c *Collector) Railway(ctx context.Context, cfg config.Config, now time.Time) []core.Card {
	return c.railway(ctx, cfg, now, "https://backboard.railway.com/graphql/v2")
}

func (c *Collector) railway(ctx context.Context, cfg config.Config, now time.Time, endpoint string) []core.Card {
	cost := card("railway:cost", "Railway current usage", "Railway GraphQL", "https://railway.com/workspace/usage", now)
	cpu := card("railway:cpu", "Railway CPU usage", "Railway GraphQL", "https://railway.com/workspace/usage", now)
	if cfg.RailwayToken == "" || cfg.RailwayWorkspaceID == "" {
		return []core.Card{cost, cpu}
	}
	var billing struct {
		Workspace struct {
			Customer struct {
				CurrentUsage  *float64 `json:"currentUsage"`
				BillingPeriod struct {
					Start string `json:"start"`
					End   string `json:"end"`
				} `json:"billingPeriod"`
			} `json:"customer"`
		} `json:"workspace"`
	}
	const billingQuery = `query($workspaceId:String!){workspace(workspaceId:$workspaceId){customer{currentUsage billingPeriod{start end}}}}`
	if e := c.graphql(ctx, endpoint, cfg.RailwayToken, billingQuery, map[string]any{"workspaceId": cfg.RailwayWorkspaceID}, &billing); e != nil || billing.Workspace.Customer.CurrentUsage == nil {
		cost = failed(cost)
	} else {
		cost.Status = "healthy"
		cost.Unit = "USD"
		cost.Summary = fmt.Sprintf("$%.2f current usage", *billing.Workspace.Customer.CurrentUsage)
		cost.Window = billing.Workspace.Customer.BillingPeriod.Start + " → " + billing.Workspace.Customer.BillingPeriod.End
		cost.ExpiresAt = now.Add(2 * time.Hour)
	}
	if cfg.RailwayEnvironmentID == "" {
		cpu.Summary = "Add RAILWAY_ENVIRONMENT_ID"
		return []core.Card{cost, cpu}
	}
	const metricsQuery = `query($environmentId:String!,$startDate:DateTime!,$endDate:DateTime!){metrics(environmentId:$environmentId,startDate:$startDate,endDate:$endDate,groupBy:[SERVICE_ID],measurements:[CPU_USAGE,MEMORY_USAGE_GB,NETWORK_TX_GB]){measurement tags{serviceId} values{ts value}}}`
	var metrics struct {
		Metrics []struct {
			Measurement string `json:"measurement"`
			Tags        struct {
				ServiceID string `json:"serviceId"`
			} `json:"tags"`
			Values []struct {
				Value float64 `json:"value"`
			} `json:"values"`
		} `json:"metrics"`
	}
	e := c.graphql(ctx, endpoint, cfg.RailwayToken, metricsQuery, map[string]any{"environmentId": cfg.RailwayEnvironmentID, "startDate": now.Add(-time.Hour).Format(time.RFC3339), "endDate": now.Format(time.RFC3339)}, &metrics)
	if e != nil {
		cpu = failed(cpu)
	} else {
		var total float64
		found := false
		for _, m := range metrics.Metrics {
			if m.Measurement == "CPU_USAGE" && len(m.Values) > 0 {
				total += m.Values[len(m.Values)-1].Value
				found = true
			}
		}
		if !found {
			cpu = failed(cpu)
		} else {
			cpu.Status = "healthy"
			cpu.Unit = "vCPU"
			cpu.Summary = fmt.Sprintf("%.3f vCPU latest across services", total)
			cpu.Window = "last hour"
			cpu.ExpiresAt = now.Add(2 * time.Hour)
		}
	}
	return []core.Card{cost, cpu}
}

// GitHub fetches the authenticated user's contribution total for the current year.
func (c *Collector) GitHub(ctx context.Context, cfg config.Config, now time.Time) core.Card {
	return c.github(ctx, cfg, now, "https://api.github.com/graphql")
}

func (c *Collector) github(ctx context.Context, cfg config.Config, now time.Time, endpoint string) core.Card {
	result := card("github:activity", "GitHub contributions", "GitHub GraphQL", "https://github.com/"+cfg.GitHubLogin, now)
	if cfg.GitHubToken == "" || cfg.GitHubLogin == "" {
		return result
	}
	const query = `query($login:String!,$from:DateTime!,$to:DateTime!){user(login:$login){contributionsCollection(from:$from,to:$to){contributionCalendar{totalContributions}}}}`
	var data struct {
		User *struct {
			ContributionsCollection struct {
				ContributionCalendar struct {
					Total int `json:"totalContributions"`
				} `json:"contributionCalendar"`
			} `json:"contributionsCollection"`
		} `json:"user"`
	}
	from := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	if e := c.graphql(ctx, endpoint, cfg.GitHubToken, query, map[string]any{"login": cfg.GitHubLogin, "from": from.Format(time.RFC3339), "to": now.Format(time.RFC3339)}, &data); e != nil || data.User == nil {
		return failed(result)
	}
	result.Status = "healthy"
	result.Summary = fmt.Sprintf("%d contributions", data.User.ContributionsCollection.ContributionCalendar.Total)
	result.Window = fmt.Sprintf("%d year to date", now.Year())
	return result
}

// R2 fetches per-bucket operations, storage, and bandwidth from Cloudflare Analytics.
func (c *Collector) R2(ctx context.Context, cfg config.Config, now time.Time) []core.Card {
	return c.r2(ctx, cfg, now, "https://api.cloudflare.com/client/v4/graphql")
}

func (c *Collector) r2(ctx context.Context, cfg config.Config, now time.Time, endpoint string) []core.Card {
	if len(cfg.R2Buckets) == 0 {
		return []core.Card{card("r2:overview", "R2 usage", "Cloudflare GraphQL", "https://dash.cloudflare.com/", now)}
	}
	out := make([]core.Card, 0, len(cfg.R2Buckets))
	const query = `query($accountTag:string!,$startDate:Time,$endDate:Time,$bucketName:string){viewer{accounts(filter:{accountTag:$accountTag}){r2OperationsAdaptiveGroups(limit:10000,filter:{datetime_geq:$startDate,datetime_leq:$endDate,bucketName:$bucketName}){sum{requests}} r2StorageAdaptiveGroups(limit:1,filter:{datetime_geq:$startDate,datetime_leq:$endDate,bucketName:$bucketName},orderBy:[datetime_DESC]){max{payloadSize metadataSize objectCount}} r2BandwidthUsageAdaptiveGroups(limit:1000,filter:{datetime_geq:$startDate,datetime_lt:$endDate,bucketName:$bucketName}){sum{bytesUpload bytesDownload}}}}}`
	for _, bucket := range cfg.R2Buckets {
		result := card("r2:"+bucket, "R2 "+bucket, "Cloudflare GraphQL", "https://dash.cloudflare.com/", now)
		if cfg.CloudflareToken == "" || cfg.CloudflareAccountID == "" {
			out = append(out, result)
			continue
		}
		var data struct {
			Viewer struct {
				Accounts []struct {
					Ops []struct {
						Sum struct {
							Requests int64 `json:"requests"`
						} `json:"sum"`
					} `json:"r2OperationsAdaptiveGroups"`
					Storage []struct {
						Max struct {
							Payload  int64 `json:"payloadSize"`
							Metadata int64 `json:"metadataSize"`
							Objects  int64 `json:"objectCount"`
						} `json:"max"`
					} `json:"r2StorageAdaptiveGroups"`
					Bandwidth []struct {
						Sum struct {
							Upload   int64 `json:"bytesUpload"`
							Download int64 `json:"bytesDownload"`
						} `json:"sum"`
					} `json:"r2BandwidthUsageAdaptiveGroups"`
				} `json:"accounts"`
			} `json:"viewer"`
		}
		vars := map[string]any{"accountTag": cfg.CloudflareAccountID, "startDate": now.Add(-24 * time.Hour).Format(time.RFC3339), "endDate": now.Format(time.RFC3339), "bucketName": bucket}
		if e := c.graphql(ctx, endpoint, cfg.CloudflareToken, query, vars, &data); e != nil || len(data.Viewer.Accounts) == 0 {
			out = append(out, failed(result))
			continue
		}
		a := data.Viewer.Accounts[0]
		if len(a.Ops) == 0 && len(a.Storage) == 0 && len(a.Bandwidth) == 0 {
			out = append(out, failed(result))
			continue
		}
		var ops, up, down int64
		for _, v := range a.Ops {
			ops += v.Sum.Requests
		}
		for _, v := range a.Bandwidth {
			up += v.Sum.Upload
			down += v.Sum.Download
		}
		storage := "unknown"
		if len(a.Storage) > 0 {
			storage = fmt.Sprintf("%.2f GiB", float64(a.Storage[0].Max.Payload+a.Storage[0].Max.Metadata)/(1<<30))
		}
		result.Status = "healthy"
		result.Summary = fmt.Sprintf("%d operations · %s stored · %.2f GiB download · %.2f GiB upload", ops, storage, float64(down)/(1<<30), float64(up)/(1<<30))
		result.Window = "operations/bandwidth last 24h; storage latest sample"
		out = append(out, result)
	}
	return out
}

// RequestMetrics reads one owned application's Prometheus request counter.
func (c *Collector) RequestMetrics(ctx context.Context, t config.MetricTarget, now time.Time) core.Card {
	result := card("requests:"+t.Name, t.Name+" requests", "Prometheus exposition", t.URL, now)
	req, e := http.NewRequestWithContext(ctx, "GET", t.URL, nil)
	if e != nil {
		return failed(result)
	}
	if t.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+t.BearerToken)
	}
	res, e := c.SafeClient.Do(req)
	if e != nil {
		return failed(result)
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return failed(result)
	}
	body, e := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if e != nil {
		return failed(result)
	}
	total, e := ParsePrometheusRequests(string(body))
	if e != nil {
		return failed(result)
	}
	result.Status = "healthy"
	result.Unit = "requests"
	result.Summary = strconv.FormatFloat(total, 'f', 0, 64) + " total since process start"
	result.Window = "process lifetime; resets on restart"
	result.ExpiresAt = now.Add(2 * time.Hour)
	return result
}

// ParsePrometheusRequests sums http_requests_total samples, excluding comments and unrelated metrics.
func ParsePrometheusRequests(body string) (float64, error) {
	var total float64
	found := false
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "http_requests_total ") && !strings.HasPrefix(line, "http_requests_total{") {
			continue
		}
		valuePart := strings.TrimPrefix(line, "http_requests_total")
		if strings.HasPrefix(valuePart, "{") {
			end := labelEnd(valuePart)
			if end < 0 {
				return 0, errors.New("invalid request labels")
			}
			valuePart = valuePart[end+1:]
		}
		fields := strings.Fields(valuePart)
		if len(fields) == 0 {
			return 0, errors.New("request counter absent value")
		}
		v, e := strconv.ParseFloat(fields[0], 64)
		if e != nil || v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
			return 0, errors.New("invalid request counter")
		}
		total += v
		found = true
	}
	if !found {
		return 0, errors.New("request counter absent")
	}
	return total, nil
}

func labelEnd(s string) int {
	quoted, escaped := false, false
	for i, ch := range s {
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && quoted {
			escaped = true
			continue
		}
		if ch == '"' {
			quoted = !quoted
			continue
		}
		if ch == '}' && !quoted {
			return i
		}
	}
	return -1
}
