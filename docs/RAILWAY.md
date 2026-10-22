# Railway and local setup

## Identical steps on macOS, Ubuntu, and Fedora

1. Have Go 1.24+ and a local Postgres 16 instance or isolated local Docker Postgres.
2. Set required private environment values from .env.example. Do not store populated values in Markdown.
3. Run go test ./..., go vet ./..., and go build ./....
4. For local HTTP only, set COOKIE_SECURE=false, run go run ./cmd/hub, and open /login on the configured PORT.

## OS-specific database setup

- macOS only: Use Docker Desktop or an existing local Postgres. If using Docker, publish its port to 127.0.0.1 only.
- Ubuntu only: Use an existing local Postgres or Docker Engine. If installing packages, follow the official Ubuntu PostgreSQL or Docker instructions and confirm the local port.
- Fedora only: Use an existing local Postgres or Podman/Docker. If installing packages, follow the official Fedora PostgreSQL or Podman instructions and confirm the local port.

Docker or package installation is optional; the hub only needs reachable Postgres.

## Railway handoff

Create one Railway service from this product folder using its Dockerfile and one Railway Postgres service. Configure DATABASE_URL with the Postgres private-network value, plus HUB_USERNAME, HUB_PASSWORD, and SESSION_SECRET. Set optional provider variables after checking their scopes. The service listens on Railway's PORT; railway.toml configures /healthz. Add an HTTPS domain and confirm login, cards, and an independent external uptime check. No deployment was performed by this task.

Prefer a workspace-scoped Railway token if it can read the required usage and metrics fields. If not, leave the card unconfigured until an access decision is made. Compare the cost card with Railway's usage dashboard before relying on it for budget decisions.
