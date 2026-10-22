# Railway production service inventory — 2026-10-22

Read-only Railway GraphQL `project.services` and `serviceInstance` queries supplied this snapshot. It can change after deployments or domain updates. No endpoint was probed by this audit. `Active` counts active deployments; an empty list does not prove an HTTP failure.

| Service | Active | Public domain for candidate HTTPS check |
|---|---:|---|
| Tommi Pocketbase | 1 | tommi-pocketbase-production.up.railway.app |
| interview-prep | 0 | interview-prep-production-fa91.up.railway.app |
| TuanLightning Admin | 1 | tuanlightning-admin-production.up.railway.app |
| PhanHuynh UI | 1 | phanhuynhsolar.com |
| Trala UI | 1 | tralatimcay.com |
| REPLY-assistant | 0 | reply-assistant-production-5d58.up.railway.app |
| TuanLightning Web | 1 | tuanlightning-web-production.up.railway.app |
| Tommitoan UI (DEV) | 1 | tommitoan-ui-dev-production.up.railway.app |
| PhanHuynh Admin | 1 | admin.phanhuynhsolar.com |
| Mercury Tools UI | 1 | m-tools.tommitoan.com |
| Hindsight | 1 | hindsight-production-6817.up.railway.app |
| Tommi bazi UI (DEV) | 1 | tommi-bazi-ui-dev-production.up.railway.app |
| Tommitoan UI | 1 | tommitoan.com |
| Trala Backend | 1 | No public domain |
| Redis | 1 | No public domain |
| Tommi bazi UI | 1 | bazi.tommitoan.com |
| Pressroom | 1 | No public domain |
| Museum UI | 0 | museum-ui-production.up.railway.app |

The Railway API also lists Railway-generated domains for PhanHuynh UI, Trala UI, PhanHuynh Admin, and Mercury Tools UI. The table uses the custom domains as the primary candidate. The three services with no active deployment are `interview-prep`, `REPLY-assistant`, and `Museum UI`; investigate intended sleep/stop policy before classifying them as incidents.

## Health contract

- For every Railway service, collect deployment state with a collection timestamp and preserve `no active deployment` as its own state. The Phase 2 connector now creates one card per service when RAILWAY_PROJECT_ID is configured.
- For public HTTP services, configure an owner-approved HTTPS URL and expected response. A domain alone does not establish a valid `/health` path; authenticated or sleeping apps may reject or delay probes. The current checker expects an HTTP success response.
- For Trala Backend, Redis, and Pressroom, use Railway deployment status until a reachable, safe health source is chosen. Do not probe Railway private addresses through the public checker.
- Application request counts require instrumentation or an authorized analytics source for each app. Railway CPU/network metrics are not request counts.

## Audit Log

- 2026-10-22: Phase 2 implemented the single-query service inventory collector. The same query shape was revalidated read-only against the selected project; it returned 18 services, no next page, and three without active deployments.
