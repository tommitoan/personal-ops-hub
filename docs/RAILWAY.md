# Railway and local setup

## Local prerequisites

- Identical on macOS, Ubuntu, and Fedora: install Go 1.24 or newer and provide a local PostgreSQL 16 instance. Copy the variable names from `.env.example` into your private environment; never commit populated values. Run `go test ./...`, `go vet ./...`, and `go build ./...` from the product folder.
- macOS only: use Docker Desktop or a local PostgreSQL installation. Bind a Docker database port to `127.0.0.1` only.
- Ubuntu only: use local PostgreSQL or Docker Engine; follow the official Ubuntu installation instructions for your chosen runtime. Bind a Docker database port to `127.0.0.1` only.
- Fedora only: use local PostgreSQL or Podman/Docker; follow the official Fedora installation instructions for your chosen runtime. Bind a container database port to `127.0.0.1` only.
- Identical on all three OSes: set `COOKIE_SECURE=false` only for local HTTP, start `go run ./cmd/hub`, then visit `/login` on the configured port. The app creates its tables at startup. `/healthz` returns 503 when PostgreSQL is unreachable.

## Railway owner handoff

1. In the intended project and production environment, create a PostgreSQL service and a web service from this repository. Set the web service root directory to the product folder if the GitHub repository contains other projects. Use the checked-in `Dockerfile` and `railway.toml`.
2. In the web service Variables tab, add `DATABASE_URL` as a Railway reference variable to the PostgreSQL service's `DATABASE_URL`. Use its private connection, not a public TCP proxy. Set `HUB_USERNAME`, a unique `HUB_PASSWORD` of at least 16 characters, and a random `SESSION_SECRET` of at least 32 characters as private variables. Keep `COOKIE_SECURE=true`. Railway supplies `PORT`; the app listens on it. No start-command override is needed.
3. Set only the optional provider variables you can validate: `RAILWAY_TOKEN`, `RAILWAY_WORKSPACE_ID`, `RAILWAY_ENVIRONMENT_ID`, `RAILWAY_PROJECT_ID`; `CLOUDFLARE_TOKEN`, `CLOUDFLARE_ACCOUNT_ID`, `R2_BUCKETS`; `GITHUB_TOKEN`, `GITHUB_LOGIN`; and `SITES_JSON`, `METRICS_JSON`, `DOMAINS_JSON`, `ALERT_WEBHOOK_URL` as appropriate. Leave incomplete connectors unset so their cards show `not configured`. Never paste actual secrets into documentation or Git.
4. Review the staged configuration and deploy. The configured `/healthz` must return 2xx after the database is ready. Railway health checks gate a new deployment; they do not continuously monitor it after activation. Generate an HTTPS domain, open `/login`, sign in, and verify `/api/cards` redirects when signed out. Confirm the cookie is Secure in the browser.
5. Compare one Railway usage, R2, and GitHub card against its source dashboard, including units and time windows. Check the Railway usage dashboard against the USD 5 monthly target; the target's scope still needs an owner decision. No cost alert is implemented. Set up an independent external uptime monitor for this hub and critical service endpoints, with a destination the owner checks.
6. Enable database backups and perform a restore drill using a separate local or sandbox database before relying on history. Record the alert owner, provider token expiry/rotation date, domain registrar/renewal date, and backup review cadence privately. See [operations](OPERATIONS.md).

Do not point every service health card at a generic Railway deployment status. Configure each public HTTPS health URL in `SITES_JSON` only after confirming its endpoint and expected behavior. Railway's deployment cards show deployment state, not HTTP health.

## Rollback and failure checks

If a deploy fails health checks, inspect the web and PostgreSQL logs, `DATABASE_URL` reference, required variables, and the listening `PORT`. The app exits when PostgreSQL is unavailable at startup; retry the web deployment after the database is ready. If a live version has an issue, use Railway's deployment controls to return to the last known good image after reviewing database compatibility. Schema creation is idempotent; this project has no rollback migration.

Provider cards marked stale or not configured do not make `/healthz` fail. Investigate the corresponding token scope, provider status, and rate limit separately. See [operations](OPERATIONS.md) for incident handling.

## Railway references

- [PostgreSQL and private connections](https://docs.railway.com/databases/postgresql)
- [Reference variables](https://docs.railway.com/variables)
- [Deployment health checks](https://docs.railway.com/deployments/healthchecks)
- [PostgreSQL backup and restore](https://docs.railway.com/guides/postgres-backups-restores)
