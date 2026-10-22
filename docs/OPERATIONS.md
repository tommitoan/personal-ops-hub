# Operations

## Routine review and ownership

The owner checks the dashboard and independent uptime monitor daily. Assign the external monitor's alert destination before deploying. The hub runs on Railway alongside monitored services, so its own failure may hide their cards. Railway deployment health checks do not continuously monitor a running service. Keep at least one critical endpoint and the hub under an external monitor with a separate failure domain.

Review Railway usage against the USD 5 monthly target in Railway's billing view. The card is a delayed observation, not a billing limit or budget alert. Confirm whether that target covers this hub alone or the whole workspace before treating it as a threshold.

## Site incidents and alerts

Each configured site opens an incident after `FAILURE_THRESHOLD` consecutive non-healthy probes and closes after one healthy probe. One JSON webhook event is sent on each down/resolved transition when `ALERT_WEBHOOK_URL` is configured. Delivery is best effort and is not queued for retry. Record the person who owns the webhook destination and check delivery with a controlled local or sandbox incident before depending on it.

On a down alert: verify the target URL independently, then inspect DNS, TLS expiry, response code, and latency. Compare the relevant Railway deployment state and logs. If only the hub is unavailable, use the external monitor and Railway service logs. On recovery, confirm both the target and the resolved event; missed webhook delivery does not change stored incident state.

## Retention, backup, and restore

The app prunes observations older than 30 days every 24 hours while it runs. Latest card snapshots and incident state remain. The first prune is after startup, so a long outage can delay deletion. Backups may retain older observations beyond the app's 30-day window according to their own retention schedule.

Enable Railway PostgreSQL volume backups and review the configured schedule and retention. Periodically create a logical dump for an offsite copy if the history matters beyond a project or volume loss. Test restore to a separate local or sandbox database, start the hub against that copy, and confirm cards, history, and incident state. Do not overwrite the production database for a drill. Railway's [backup and restore guide](https://docs.railway.com/guides/postgres-backups-restores) describes the available backup layers and restore paths.

## Secrets and access

Serve the dashboard over HTTPS. Store `DATABASE_URL`, owner credentials, session key, provider tokens, and webhook destination only in Railway variables or a private secret manager. Never commit a populated `.env` file. Use least-privilege read scopes for providers. Rotate a provider token in its source system, update the Railway variable, deploy, and confirm that the corresponding card refreshes. Rotate `HUB_PASSWORD` and `SESSION_SECRET` together when owner access may be exposed; changing `SESSION_SECRET` invalidates all existing sessions. Verify signed-out `/api/cards` and `/api/history` redirect to `/login` after rotation.

The owner cookie is HttpOnly, SameSite Strict, and Secure by default. `COOKIE_SECURE=false` is for local HTTP only. Responses use `Cache-Control: no-store`. Avoid sharing a populated dashboard URL or screenshots that expose private activity.

## Troubleshooting

| Symptom | First checks |
|---|---|
| `/healthz` is 503 or startup exits | PostgreSQL service readiness, private `DATABASE_URL` reference, database logs, connection limit. |
| Card says `not configured` | Required variable group in `.env.example`; verify values are in the correct environment. |
| Card says `stale` | Provider token scope/expiry, source outage, schema change, rate limit, and collection time. HTTP 429 waits for the next 15-minute cycle. |
| Site says `down` | Public HTTPS health URL, DNS, TLS, expected status, and the service's own logs. |
| Request count absent | Owned app exports `http_requests_total`; `METRICS_JSON` points to a public HTTPS endpoint. Counter is process lifetime and may reset. |
| Domain card wrong | Owner-maintained `DOMAINS_JSON` renewal date and registrar record. |
| No alert arrived | Webhook configuration, receiver logs, hub delivery log, and whether an incident transition actually occurred. No retry queue exists. |

## Verification record

See the timestamped Phase 4 verification report in `docs/` for local test commands and results. Railway deployment, provider-account reconciliation, external monitor, and backup schedule are owner handoff steps until performed in the target environment.
