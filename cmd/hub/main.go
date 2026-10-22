package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/tommitoan/personal-ops-hub/internal/config"
	"github.com/tommitoan/personal-ops-hub/internal/connectors"
	"github.com/tommitoan/personal-ops-hub/internal/core"
	"github.com/tommitoan/personal-ops-hub/internal/monitor"
	"github.com/tommitoan/personal-ops-hub/internal/store"
	"github.com/tommitoan/personal-ops-hub/internal/web"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	cfg, err := config.Load()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	st, err := store.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database unavailable", "error", err)
		os.Exit(1)
	}
	defer st.Pool.Close()
	checker := monitor.NewChecker()
	collector := connectors.New()
	go runLoop(ctx, cfg, st, checker, collector)
	srv := &http.Server{Addr: ":" + cfg.Port, Handler: web.New(cfg, st).Handler(), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()
	slog.Info("hub listening", "port", cfg.Port)
	if err = srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func runLoop(ctx context.Context, cfg config.Config, st *store.Store, checker *monitor.Checker, collector *connectors.Collector) {
	check := func() {
		for _, target := range cfg.Sites {
			card := checker.Probe(ctx, target, cfg.CheckInterval)
			if err := st.SaveCard(ctx, card); err != nil {
				slog.Error("save site card", "site", target.Name, "error", err)
			}
			event, err := st.UpdateIncident(ctx, card.Key, card.Status == "healthy", cfg.FailureThreshold, time.Now().UTC())
			if err != nil {
				slog.Error("incident update", "site", target.Name, "error", err)
			} else if event != "" {
				sendAlert(ctx, collector, cfg.AlertWebhookURL, target.Name, event)
			}
		}
	}
	provider := func() {
		now := time.Now().UTC()
		cards := []core.Card{}
		cards = append(cards, collector.Railway(ctx, cfg, now)...)
		cards = append(cards, collector.R2(ctx, cfg, now)...)
		cards = append(cards, collector.GitHub(ctx, cfg, now))
		for _, target := range cfg.Metrics {
			cards = append(cards, collector.RequestMetrics(ctx, target, now))
		}
		for _, domain := range cfg.Domains {
			due, _ := time.Parse("2006-01-02", domain.RenewBy)
			days := int(time.Until(due).Hours() / 24)
			status := "healthy"
			if days < 30 {
				status = "degraded"
			}
			if days < 0 {
				status = "down"
			}
			cards = append(cards, core.Card{Key: "domain:" + domain.Name, Title: domain.Name + " renewal", Status: status, Summary: domain.RenewBy + " · " + formatDays(days), Source: "manual registrar date", Link: domain.URL, UpdatedAt: now, ExpiresAt: now.Add(26 * time.Hour)})
		}
		for _, card := range cards {
			if err := st.SaveCard(ctx, card); err != nil {
				slog.Error("save provider card", "key", card.Key, "error", err)
			}
		}
	}
	check()
	provider()
	checkTicker := time.NewTicker(cfg.CheckInterval)
	providerTicker := time.NewTicker(15 * time.Minute)
	pruneTicker := time.NewTicker(24 * time.Hour)
	defer checkTicker.Stop()
	defer providerTicker.Stop()
	defer pruneTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-checkTicker.C:
			check()
		case <-providerTicker.C:
			provider()
		case <-pruneTicker.C:
			if err := st.Prune(ctx, 30); err != nil {
				slog.Error("prune failed", "error", err)
			}
		}
	}
}

func formatDays(d int) string {
	if d < 0 {
		return "expired"
	}
	return fmtInt(d) + " days remaining"
}
func fmtInt(n int) string { return strconv.Itoa(n) }

func sendAlert(ctx context.Context, c *connectors.Collector, endpoint, name, event string) {
	if endpoint == "" {
		return
	}
	body, _ := json.Marshal(map[string]string{"source": "personal-ops-hub", "site": name, "event": event})
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.SafeClient.Do(req)
	if err != nil {
		slog.Error("alert delivery failed", "site", name, "error", err)
		return
	}
	res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		slog.Error("alert delivery failed", "site", name, "status", res.StatusCode)
	}
}
