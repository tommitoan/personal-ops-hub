# Personal Ops Hub

Private Railway-hosted overview for owned sites, infrastructure usage, GitHub activity, and instrumented request counters. One Go web process uses Postgres for history and serves an owner-only dashboard. Provider tokens stay on the server.

## Data shown

| Card | Source | Refresh |
|---|---|---|
| Site status, latency, TLS expiry | HTTPS probe | Configured interval |
| Railway current workspace usage and CPU | Railway GraphQL | 15 minutes |
| R2 operations, storage, bandwidth | Cloudflare Analytics GraphQL | 15 minutes |
| GitHub contributions | GitHub GraphQL | 15 minutes |
| Owned app request counter | Prometheus http_requests_total | 15 minutes |
| Domain renewal | Owner-maintained date | 15 minutes |

The request card is a process-lifetime counter and can reset when an app restarts. Railway does not collect application request counts. Provider cost data can lag the billing dashboard. Missing configuration and provider errors are shown explicitly.

## Configuration

Use .env.example as the field list. Never commit populated values. Required: DATABASE_URL, HUB_USERNAME, HUB_PASSWORD of at least 16 characters, and SESSION_SECRET of at least 32 characters. COOKIE_SECURE defaults to true; set false only for local HTTP testing.

SITES_JSON example: [{"name":"site-a","url":"https://example.com/health"}]. DOMAINS_JSON example: [{"name":"example.com","renew_by":"2027-01-01","url":"https://example.com"}]. METRICS_JSON example: [{"name":"api","url":"https://example.com/metrics"}]. The optional bearer_token field belongs only in private environment values. The hub rejects private network destinations after DNS resolution.

Railway cost/CPU cards need RAILWAY_TOKEN, RAILWAY_WORKSPACE_ID, and RAILWAY_ENVIRONMENT_ID. Add RAILWAY_PROJECT_ID to show deployment-state cards for all services in that project and environment. A service with no active deployment is marked degraded for review; this does not establish that its HTTP endpoint is down. R2 needs CLOUDFLARE_TOKEN, CLOUDFLARE_ACCOUNT_ID, and comma-separated R2_BUCKETS. GitHub needs GITHUB_TOKEN and GITHUB_LOGIN. ALERT_WEBHOOK_URL is optional and receives one JSON event per down/resolved incident.

## Local development

The Go commands are identical on macOS, Ubuntu, and Fedora once Go 1.24+ and Postgres 16 are available: run go run ./cmd/hub, go test ./..., go vet ./..., and go build ./.... For local database setup by OS, see [Railway and local setup](docs/RAILWAY.md).

Open the configured local port at /login. The app creates its tables at startup. /healthz returns 503 when Postgres is unavailable.

Postgres stores the latest cards, observation history, and incident state across app restarts. A new database starts empty: it does not discover Railway services or create sample activity by itself. Add `SITES_JSON` for public HTTPS health endpoints and provider variables from `.env.example` for Railway, R2, and GitHub. The dashboard shows setup guidance until the first configured signal is collected. `PORT` is configurable; for example, set `PORT=18084` when 8080 is occupied. A hosted Postgres connection such as Neon can be supplied through `DATABASE_URL` privately.

For Railway setup, database backup, alert ownership, and rollback, follow [Railway handoff](docs/RAILWAY.md) and [operations](docs/OPERATIONS.md). The [Phase 4 verification record](docs/20261022-151709-phase-4-verification.md) lists local checks and deployment checks still for the owner.

## Dashboard

The owner-only dashboard groups site health, Railway, R2, GitHub, request traffic, and domain cards. Each card shows its status, source, collection time, and measurement window when available. Its History link shows recent observations; `/api/cards` and `/api/history` provide the same data as JSON behind the same login. The attention count includes down, degraded, stale, and not-configured cards. Sessions last 12 hours.

## Limits

- The embedded checker shares Railway's failure domain. Use an independent external monitor for this hub and critical endpoints.
- Email, calendar, Slack, Facebook, and Zalo connectors are deferred pending provider and account decisions.
- Railway billing, resource metrics, and service inventory queries were exercised using the authenticated local CLI. The future service token and R2/GitHub API access remain unverified. Empty provider datasets are shown as stale. Transient network/5xx failures get one retry; HTTP 429 waits for the next 15-minute collection. See [source contracts](docs/SOURCES.md), [production service inventory](docs/20261022-103123-railway-inventory.md), [decisions](docs/DECISIONS.md), and [operations](docs/OPERATIONS.md).
