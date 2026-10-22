package monitor

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/tommitoan/personal-ops-hub/internal/config"
	"github.com/tommitoan/personal-ops-hub/internal/core"
)

// Checker probes only public HTTPS targets and rejects private resolved addresses.
type Checker struct{ Client *http.Client }

// NewChecker creates a bounded client with private-network and redirect protection.
func NewChecker() *Checker {
	dialer := &net.Dialer{Timeout: 4 * time.Second}
	tr := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}, DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, e := net.SplitHostPort(address)
		if e != nil {
			return nil, e
		}
		ips, e := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
		if e != nil {
			return nil, e
		}
		for _, ip := range ips {
			if !publicIP(ip) {
				return nil, fmt.Errorf("target resolves to a private address")
			}
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("target has no address")
		}
		return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
	}}
	return &Checker{Client: &http.Client{Timeout: 8 * time.Second, Transport: tr, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return fmt.Errorf("too many redirects")
		}
		if r.URL.Scheme != "https" || r.URL.User != nil {
			return fmt.Errorf("redirect must use HTTPS")
		}
		return nil
	}}}
}
func publicIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return false
	}
	for _, block := range []string{"100.64.0.0/10", "192.0.0.0/24", "198.18.0.0/15", "2001:db8::/32", "fc00::/7"} {
		if netip.MustParsePrefix(block).Contains(ip) {
			return false
		}
	}
	return true
}

// Probe checks HTTP status, latency, and the endpoint's TLS certificate.
func (c *Checker) Probe(ctx context.Context, t config.Target, interval time.Duration) core.Card {
	now := time.Now().UTC()
	card := core.Card{Key: "site:" + t.Name, Title: t.Name, Status: "down", Source: "HTTPS probe", Link: t.URL, UpdatedAt: now, ExpiresAt: now.Add(2 * interval)}
	u, e := url.Parse(t.URL)
	if e != nil || u.Scheme != "https" {
		card.Summary = "invalid HTTPS URL"
		return card
	}
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, t.URL, nil)
	if e != nil {
		card.Summary = "invalid request"
		return card
	}
	req.Header.Set("User-Agent", "personal-ops-hub/1")
	start := time.Now()
	resp, e := c.Client.Do(req)
	if e != nil {
		card.Summary = "connection or TLS check failed"
		return card
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
	ms := time.Since(start).Milliseconds()
	card.Unit = "ms"
	card.Summary = fmt.Sprintf("HTTP %d · %d ms", resp.StatusCode, ms)
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		card.Status = "healthy"
	} else if resp.StatusCode < 500 {
		card.Status = "degraded"
	}
	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		days := int(time.Until(resp.TLS.PeerCertificates[0].NotAfter).Hours() / 24)
		card.Summary += fmt.Sprintf(" · TLS %d d", days)
		if days < 14 && card.Status == "healthy" {
			card.Status = "degraded"
		}
	}
	if !strings.EqualFold(resp.Request.URL.Hostname(), u.Hostname()) {
		card.Summary += " · redirected"
	}
	return card
}
