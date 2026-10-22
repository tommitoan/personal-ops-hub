# Operations

## Alerts

Each site opens an incident after FAILURE_THRESHOLD consecutive non-healthy probes. One healthy probe closes it. Each transition emits one webhook event if ALERT_WEBHOOK_URL is set. Delivery is best effort and is not queued for retry. Monitor this hub separately.

## Retention and backup

Observations older than 30 days are deleted daily. Latest cards and incident state remain. Back up Railway Postgres using the provider backup facility and test restoration before depending on historical data.

## Secrets and access

Serve the dashboard only over HTTPS. Put credentials in Railway variables, not Git or docs. Rotate a provider token, redeploy, then confirm its card recovers. Rotating SESSION_SECRET invalidates sessions. Avoid public caching of /api/cards.
The owner session cookie is HttpOnly, SameSite Strict, and Secure by default. Logout clears the browser cookie; rotate SESSION_SECRET to invalidate every previously issued cookie immediately.

## Troubleshooting

- /healthz is 503: inspect Postgres reachability and DATABASE_URL.
- not configured: set the provider's required variables.
- stale: check provider token scope, status, schema, and rate limits.
- Site down: inspect URL, DNS, TLS, and the actual health endpoint.
- Requests absent: expose http_requests_total and configure METRICS_JSON. The target must be public HTTPS.

## Verification record

2026-10-22: Go build, vet, and unit tests passed. Local Postgres smoke passed for /healthz, unauthenticated redirect, login, and /api/cards. Live provider calls and Railway deployment were not run.
