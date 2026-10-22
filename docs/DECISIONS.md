# MVP decisions

These are implementation defaults plus owner-provided targets. Unanswered fields remain open.

| Topic | Current decision | Reason / revisit condition |
|---|---|---|
| Runtime | One Go web service plus Postgres | Small Railway footprint and simple deployment; revisit if richer frontend interactions are needed. |
| Health | Embedded HTTPS checker | Avoids a second service for MVP. Add an independent external check for critical services and this hub itself. |
| Access | One owner, signed session cookie | No team role model has been requested. |
| Health cadence | 60 seconds by default | Configurable; review against number of targets and Railway cost. |
| Provider cadence | 15 minutes | Bounds API usage and permits a cached overview. |
| History | 30 days | Configurable in code later if longer retention is needed. |
| Railway workspace | The one workspace visible to the authenticated CLI session | The service token scope and intended production workspace still need confirmation. No workspace ID is stored in this document. |
| Monitored scope | All 18 services in the owner's Railway project production environment | Railway deployment state covers all services; HTTPS probes apply only to selected public HTTP endpoints. |
| R2 | Bucket `museum` in the owner-provided Cloudflare account | Account identity was matched in the Cloudflare dashboard; API token scope remains unverified. |
| GitHub | `tommitoan` | Public profile and repository were visible; GraphQL token scope remains unverified. |
| Domain | `tommitoan.com` | Renewal date and registrar remain unknown. |
| Railway budget | USD 5 per month, scope to confirm | Distinguish hub-only budget from workspace-wide budget before applying alerts. |
| Request count | Prometheus http_requests_total process-lifetime total | Daily totals require persisted deltas or a time-series backend; do not label this as daily traffic. |
| Domain renewal | Owner-entered date | Registrar API is unknown. |

## Open decisions

- Exact HTTP health paths and request-metrics endpoints for public apps. Railway service/domain inventory is recorded separately.
- Cloudflare and GitHub API token scopes and live query reconciliation.
- Alert destination, domain registrar/renewal date, and whether the USD 5 budget is hub-only or workspace-wide.
- Whether independent Uptime Kuma or another external monitor is worth the extra runtime cost.
- Whether email, calendar, team chat, Facebook Page, or Zalo Official Account joins a later phase.
