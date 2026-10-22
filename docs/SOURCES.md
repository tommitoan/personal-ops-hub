# Source contracts and provenance

| Source | Contract | Verification |
|---|---|---|
| Railway public API | workspace.customer.currentUsage, billingPeriod, and CPU metrics | Query shapes checked against [Railway CLI source](https://docs.rs/crate/railwayapp/latest/source/src/commands/usage.rs) and [API docs](https://docs.railway.com/integrations/api). Read-only CLI session returned live values; intended service token scope remains unverified. |
| Railway service state | `project.services.serviceInstances.activeDeployments` in the selected environment | Exact single-query shape validated read-only against the project: 18 services, three without active deployment. The app now collects and displays these states when RAILWAY_PROJECT_ID is set. |
| Railway request count | Not provided by [Railway resource metrics](https://docs.railway.com/observability/metrics) | Requires app instrumentation. |
| Cloudflare R2 | Operations, storage, and bandwidth GraphQL datasets | Query shapes from [Cloudflare R2 docs](https://developers.cloudflare.com/r2/platform/metrics-analytics/). Live bucket response unverified. Small transfers can be excluded from bandwidth data. |
| GitHub | Contribution calendar total | Query shape from [GitHub GraphQL docs](https://docs.github.com/en/graphql/reference/users). Live account response unverified. |
| Owned app traffic | Prometheus http_requests_total | Parser unit-tested. Daily totals need persisted delta logic or a time-series backend. |
| Domain renewal | Manually configured date | This is a reminder, not verified registrar data. |

GraphQL HTTP 200 responses with an errors array count as failures. The UI stores no raw provider response. Railway account or workspace tokens can have broader permissions than this app needs. Use the narrowest available scope and verify it against the actual account.

Site checks use the configured interval. Provider cards refresh every 15 minutes and become stale after two hours. A provider error never becomes numerical zero.

GraphQL calls make at most one retry for network or HTTP 5xx errors. HTTP 429 and authorization failures are not retried within the same poll. Empty Railway CPU or R2 datasets produce stale cards rather than a measured zero. GitHub activity is a year-to-date contribution count; recent event details require a separate authorized source and are not shown.
The Prometheus reader sums finite, nonnegative `http_requests_total` samples from one configured endpoint. Its value is a process-lifetime counter; no daily request total is inferred from it.

## Audit Log

- 2026-10-22: Read-only Railway CLI queries confirmed one billing-period response and 45 nonempty resource metric series. See [capability audit](20261022-102706-source-validation.md).
- 2026-10-22: Read-only Railway CLI queries confirmed the selected production service instances, active deployments, and public domains. See [service inventory](20261022-103123-railway-inventory.md).
- 2026-10-22: Phase 2 added project service-state cards and mock-verified Railway, R2, GitHub, empty-response, and bounded-retry behavior. The deployed service token and Cloudflare/GitHub API responses remain unverified.
