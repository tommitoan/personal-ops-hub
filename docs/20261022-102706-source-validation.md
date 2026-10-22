# Provider capability audit — 2026-10-22

## Method and evidence

This audit used the already-authenticated local Railway CLI for read-only schema introspection and GraphQL queries. No Railway mutation, deployment, GitHub CLI command, or credential inspection was performed. Provider IDs and financial values are intentionally omitted from this report.

| Check | Result | Scope of conclusion |
|---|---|---|
| Railway CLI authentication | Pass | Local CLI session can issue read-only API queries. |
| Workspace/project inventory | Pass | One accessible workspace, one project, one production environment, and 18 services were returned. |
| Billing-period query | Pass | The exact workspace.customer.currentUsage and billingPeriod fields used by the connector returned a numeric usage value and start/end timestamps. |
| Metrics query | Pass | The exact metrics field, environment ID argument, grouping by SERVICE_ID, and CPU_USAGE/MEMORY_USAGE_GB/NETWORK_TX_GB measurements returned 45 nonempty series with no GraphQL errors. |
| Service token scope | Unverified | CLI authentication does not establish access for a future workspace-scoped service token. |
| Cloudflare R2 | UI identity checked; API unverified | The owner-provided account and `museum` bucket are visible in Cloudflare R2. No API token is configured. |
| GitHub GraphQL | UI identity checked; API unverified | Public profile `tommitoan` is visible. No GraphQL token is configured; GitHub CLI is not used under repository rules. |
| Railway service inventory | Pass | Production environment returned 18 services, 15 with active deployments and 15 with public domains. See the timestamped inventory. |
| Site, domain, and app request sources | Partial | Owner supplied `tommitoan.com`; registrar/renewal date, health paths, and Prometheus endpoints are unknown. |

## Metric contract

- Railway currentUsage is shown as current billing-period USD usage, using the billingPeriod start/end timestamps returned with the value.
- Railway resource card uses the latest CPU_USAGE sample per service from a one-hour query. It is a resource metric, not HTTP traffic.
- R2 operations and bandwidth are intended as last-24-hour totals; storage is the latest sample in that query. This still needs a live bucket check.
- GitHub is intended as year-to-date contribution total. It does not imply a count of all events or private activity unless the selected token can access them.
- An owned app must expose Prometheus http_requests_total for the current request card. This is process-lifetime data and can reset.

## Next validation

Provide IDs and target names without credentials in chat. Put credentials in private local/Railway environment values. Then run one read-only query per provider with the intended service token and compare one card to its authoritative dashboard.

## Audit Log

- 2026-10-22: Owner supplied the Railway project, Cloudflare account and `museum` bucket, GitHub login, domain, and USD 5 monthly target. Browser UI confirmed R2 bucket and public GitHub identity. Railway read-only API supplied the service/domain inventory. No credentials were inspected or stored.
