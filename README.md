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

Railway cards need RAILWAY_TOKEN, RAILWAY_WORKSPACE_ID, and RAILWAY_ENVIRONMENT_ID. R2 needs CLOUDFLARE_TOKEN, CLOUDFLARE_ACCOUNT_ID, and comma-separated R2_BUCKETS. GitHub needs GITHUB_TOKEN and GITHUB_LOGIN. ALERT_WEBHOOK_URL is optional and receives one JSON event per down/resolved incident.

## Local development

The Go commands are identical on macOS, Ubuntu, and Fedora once Go 1.24+ and Postgres 16 are available: run go run ./cmd/hub, go test ./..., go vet ./..., and go build ./.... For local database setup by OS, see [Railway and local setup](docs/RAILWAY.md).

Open the configured local port at /login. The app creates its tables at startup. /healthz returns 503 when Postgres is unavailable.

## Limits

- The embedded checker shares Railway's failure domain. Use an independent external monitor for this hub and critical endpoints.
- Email, calendar, Slack, Facebook, and Zalo connectors are deferred pending provider and account decisions.
- Provider calls have not been exercised against this owner's live accounts. See [source contracts](docs/SOURCES.md) and [operations](docs/OPERATIONS.md).
