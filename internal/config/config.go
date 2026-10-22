package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Target is an owner-configured public endpoint.
type Target struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Domain is an owner-maintained renewal record.
type Domain struct {
	Name    string `json:"name"`
	RenewBy string `json:"renew_by"`
	URL     string `json:"url"`
}

// MetricTarget points at an owned application's Prometheus endpoint.
type MetricTarget struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	BearerToken string `json:"bearer_token,omitempty"`
}

// Config contains the runtime settings for one owner-only hub.
type Config struct {
	Port, DatabaseURL, Username, Password, SessionSecret   string
	Sites                                                  []Target
	Domains                                                []Domain
	Metrics                                                []MetricTarget
	CheckInterval                                          time.Duration
	FailureThreshold                                       int
	RailwayToken, RailwayEnvironmentID, RailwayWorkspaceID string
	CloudflareToken, CloudflareAccountID                   string
	R2Buckets                                              []string
	GitHubToken, GitHubLogin                               string
	AlertWebhookURL                                        string
}

// Load reads and validates owner-controlled environment configuration.
func Load() (Config, error) {
	c := Config{Port: env("PORT", "8080"), DatabaseURL: os.Getenv("DATABASE_URL"), Username: os.Getenv("HUB_USERNAME"), Password: os.Getenv("HUB_PASSWORD"), SessionSecret: os.Getenv("SESSION_SECRET"), RailwayToken: os.Getenv("RAILWAY_TOKEN"), RailwayEnvironmentID: os.Getenv("RAILWAY_ENVIRONMENT_ID"), RailwayWorkspaceID: os.Getenv("RAILWAY_WORKSPACE_ID"), CloudflareToken: os.Getenv("CLOUDFLARE_TOKEN"), CloudflareAccountID: os.Getenv("CLOUDFLARE_ACCOUNT_ID"), GitHubToken: os.Getenv("GITHUB_TOKEN"), GitHubLogin: os.Getenv("GITHUB_LOGIN"), AlertWebhookURL: os.Getenv("ALERT_WEBHOOK_URL")}
	if c.DatabaseURL == "" || c.Username == "" || c.Password == "" || len(c.Password) < 16 || len(c.SessionSecret) < 32 {
		return c, errors.New("DATABASE_URL, HUB_USERNAME, HUB_PASSWORD (16+ chars), and SESSION_SECRET (32+ chars) are required")
	}
	var err error
	if err = json.Unmarshal([]byte(env("SITES_JSON", "[]")), &c.Sites); err != nil {
		return c, fmt.Errorf("SITES_JSON: %w", err)
	}
	if err = json.Unmarshal([]byte(env("DOMAINS_JSON", "[]")), &c.Domains); err != nil {
		return c, fmt.Errorf("DOMAINS_JSON: %w", err)
	}
	if err = json.Unmarshal([]byte(env("METRICS_JSON", "[]")), &c.Metrics); err != nil {
		return c, fmt.Errorf("METRICS_JSON: %w", err)
	}
	c.R2Buckets = split(os.Getenv("R2_BUCKETS"))
	seconds, err := strconv.Atoi(env("CHECK_INTERVAL_SECONDS", "60"))
	if err != nil || seconds < 30 {
		return c, errors.New("CHECK_INTERVAL_SECONDS must be at least 30")
	}
	c.CheckInterval = time.Duration(seconds) * time.Second
	c.FailureThreshold, err = strconv.Atoi(env("FAILURE_THRESHOLD", "3"))
	if err != nil || c.FailureThreshold < 1 {
		return c, errors.New("FAILURE_THRESHOLD must be positive")
	}
	for _, t := range c.Sites {
		if t.Name == "" || !publicURL(t.URL) {
			return c, fmt.Errorf("invalid public site %q", t.Name)
		}
	}
	for _, t := range c.Metrics {
		if t.Name == "" || !publicURL(t.URL) {
			return c, fmt.Errorf("invalid public metrics target %q", t.Name)
		}
	}
	for _, d := range c.Domains {
		if d.Name == "" || d.RenewBy == "" {
			return c, errors.New("each domain requires name and renew_by")
		}
		if _, err = time.Parse("2006-01-02", d.RenewBy); err != nil {
			return c, fmt.Errorf("domain %q renew_by: %w", d.Name, err)
		}
	}
	if c.AlertWebhookURL != "" && !publicURL(c.AlertWebhookURL) {
		return c, errors.New("ALERT_WEBHOOK_URL must be public HTTPS")
	}
	return c, nil
}
func env(k, f string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return f
}
func split(v string) []string {
	if v == "" {
		return nil
	}
	out := []string{}
	for _, p := range strings.Split(v, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
func publicURL(s string) bool {
	u, e := url.Parse(s)
	if e != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil {
		return false
	}
	h := strings.ToLower(u.Hostname())
	return h != "localhost" && !strings.HasSuffix(h, ".local") && !strings.HasSuffix(h, ".internal")
}
