# Source contracts and provenance

| Source | Contract | Verification |
|---|---|---|
| Railway public API | workspace.customer.currentUsage, billingPeriod, and CPU metrics | Query shapes checked against [Railway CLI source](https://docs.rs/crate/railwayapp/latest/source/src/commands/usage.rs) and [API docs](https://docs.railway.com/integrations/api). Read-only CLI session returned live values; intended service token scope remains unverified. |
| Railway service state | `project.services` and `serviceInstance.activeDeployments` in the selected environment | Read-only CLI session returned 18 services; three had no active deployment. This source is documented for Phase 2 and is not yet collected by the app. |
| Railway request count | Not provided by [Railway resource metrics](https://docs.railway.com/observability/metrics) | Requires app instrumentation. |
| Cloudflare R2 | Operations, storage, and bandwidth GraphQL datasets | Query shapes from [Cloudflare R2 docs](https://developers.cloudflare.com/r2/platform/metrics-analytics/). Live bucket response unverified. Small transfers can be excluded from bandwidth data. |
| GitHub | Contribution calendar total | Query shape from [GitHub GraphQL docs](https://docs.github.com/en/graphql/reference/users). Live account response unverified. |
| Owned app traffic | Prometheus http_requests_total | Parser unit-tested. Daily totals need persisted delta logic or a time-series backend. |
| Domain renewal | Manually configured date | This is a reminder, not verified registrar data. |

GraphQL HTTP 200 responses with an errors array count as failures. The UI stores no raw provider response. Railway account or workspace tokens can have broader permissions than this app needs. Use the narrowest available scope and verify it against the actual account.

Site checks use the configured interval. Provider cards refresh every 15 minutes and become stale after two hours. A provider error never becomes numerical zero.

## Audit Log

- 2026-10-22: Read-only Railway CLI queries confirmed one billing-period response and 45 nonempty resource metric series. See [capability audit](20261022-102706-source-validation.md).
- 2026-10-22: Read-only Railway CLI queries confirmed the selected production service instances, active deployments, and public domains. See [service inventory](20261022-103123-railway-inventory.md).
