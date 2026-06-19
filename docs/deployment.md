# Production Deployment Guide

This guide covers deploying the Ancient Coins application to a production environment. For local development setup, see the [README](../README.md).

## Architecture Overview

Ancient Coins runs as **two Docker containers** orchestrated via `docker-compose.yaml`:

| Container | Image | Port | Purpose |
|---|---|---|---|
| `app` | `<user>/ancient-coins:latest` | 8080 | Go API + Vue SPA |
| `agent` | `<user>/ancient-coins-agent:latest` | 8081 | Python LangGraph agent service |

The **app** container uses a 3-stage Dockerfile (`Dockerfile`) that builds through:

1. **Node 24** — builds the Vue frontend (`npm run build`)
2. **Go 1.26.4** — compiles the API binary and embeds the Vue dist
3. **Alpine 3.21** — minimal runtime (~40 MB final image)

The **agent** container uses a separate Dockerfile (`src/agent/Dockerfile`) to build the Python LangGraph service.
Python agent dependencies are installed from `src/agent/uv.lock` with uv 0.11.22 so CI and Docker builds use the same locked resolution.

The app container contains:

| Path | Description |
|---|---|
| `/app/ancient-coins-api` | Go binary (API + SPA server) |
| `/app/wwwroot/` | Vue SPA static assets |
| `/app/data/` | SQLite database directory |
| `/app/uploads/` | Uploaded coin images |

Data is stored in a **SQLite database** (via GORM, WAL mode enabled) and coin images are stored directly on the filesystem.

Both production images run as a non-root runtime user with UID/GID `10001:10001`. The app image owns `/app`, `/app/data`, and `/app/uploads`; the agent image owns `/app` and has no persistent writable volume by default.

---

## Quick Start (Docker Compose)

The fastest way to run in production. The repository includes a `docker-compose.yaml` that runs both containers:

```yaml
services:
  app:
    image: ${DOCKERHUB_USERNAME:-briandenicola}/ancient-coins:latest
    environment:
      - JWT_SECRET=${JWT_SECRET:?Set JWT_SECRET in .env (min 32 chars)}
      - DB_PATH=/app/data/ancientcoins.db
      - PORT=8080
      - AGENT_SERVICE_URL=http://agent:8081
      - AGENT_INTERNAL_SERVICE_TOKEN=${AGENT_INTERNAL_SERVICE_TOKEN:?Set AGENT_INTERNAL_SERVICE_TOKEN in .env}
      - AGENT_INTERNAL_CALLBACK_URL=http://app:8080
    ports:
      - "8080:8080"
    volumes:
      - db-data:/app/data
      - uploads:/app/uploads
    depends_on:
      agent:
        condition: service_healthy
    restart: unless-stopped

  agent:
    image: ${DOCKERHUB_USERNAME:-ancient-coins}/ancient-coins-agent:latest
    environment:
      - AGENT_SEARXNG_URL=${SEARXNG_URL:-}
      - AGENT_LOG_LEVEL=${AGENT_LOG_LEVEL:-INFO}
      - AGENT_DEBUG=false
      - AGENT_INTERNAL_SERVICE_TOKEN=${AGENT_INTERNAL_SERVICE_TOKEN:?Set AGENT_INTERNAL_SERVICE_TOKEN in .env}
      - AGENT_TRUSTED_OUTBOUND_ORIGINS=${AGENT_TRUSTED_OUTBOUND_ORIGINS:-http://app:8080}
      - AGENT_ALLOW_LOCAL_OUTBOUND=${AGENT_ALLOW_LOCAL_OUTBOUND:-false}
    expose:
      - "8081"
    healthcheck:
      test: ["CMD", "python", "-c", "import urllib.request; urllib.request.urlopen('http://localhost:8081/health')"]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 15s
    restart: unless-stopped

volumes:
  db-data:
  uploads:
```

Create a `.env` file with your JWT secret (see [Generating a JWT Secret](#generating-a-jwt-secret)), then start the app:

```sh
docker compose up -d
```

The app is now available at `http://localhost:8080`.

> **⚠️ Multi-container networking:** The API and agent communicate bidirectionally:
> - `AGENT_SERVICE_URL` (API → agent): used for all AI features
> - `AGENT_INTERNAL_CALLBACK_URL` (agent → API): used only by collection chat (#217)
>
> In containerized deployments, both **must** use Docker service names (e.g., `http://app:8080`, `http://agent:8081`) instead of `localhost`. The agent service is not published on a host port by default; all non-health endpoints require `AGENT_INTERNAL_SERVICE_TOKEN` from the Go API. The `AGENT_INTERNAL_CALLBACK_URL` defaults to `localhost:8080` for local `task up-all` development; **set it explicitly** in compose/k8s environments or collection chat will fail with "All connection attempts failed".

---

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `JWT_SECRET` | `dev-secret-key-change-in-production-min32chars` | JWT signing key (**min 32 chars, must change for production**) |
| `DB_PATH` | `./ancientcoins.db` | SQLite database file path |
| `PORT` | `8080` | HTTP server port |
| `UPLOAD_DIR` | `./uploads` | Directory for uploaded coin images |
| `WEBAUTHN_RP_ID` | `localhost` | WebAuthn Relying Party ID (your domain) |
| `WEBAUTHN_ORIGIN` | `http://localhost:8080` | WebAuthn origin URL (supports comma-separated list) |
| `AGENT_SERVICE_URL` | `http://agent:8081` | Python agent service URL (API → agent) |
| `AGENT_INTERNAL_CALLBACK_URL` | `http://localhost:8080` | URL the Python agent uses to call back into the Go API for collection-chat tools (#217). **In multi-container deployments, must be set to the API container's network address** (e.g., `http://app:8080`), **not `localhost`**, or collection chat fails with "All connection attempts failed" |
| `AGENT_INTERNAL_SERVICE_TOKEN` | — | Shared API → agent credential required by every Python agent endpoint except `/health` and `/ready`. Required in Docker Compose and production-like deployments. |
| `CORS_ORIGINS` | *(WebAuthn origins + localhost)* | Comma-separated list of allowed CORS origins. Falls back to `WEBAUTHN_ORIGIN` values plus `http://localhost:5173` and `http://localhost:8080` |
| `AGENT_LOG_LEVEL` | `INFO` | Python agent log level |
| `AGENT_DEBUG` | `false` | Enable debug mode on the agent container (exposes `/docs` endpoint) |
| `AGENT_SEARXNG_URL` | — | SearXNG instance URL for Ollama web search (required when using Ollama provider) |
| `AGENT_TRUSTED_OUTBOUND_ORIGINS` | `http://app:8080` in Compose | Comma-separated exact origins the Python agent may call for Ollama, SearXNG, and collection tools. |
| `AGENT_ALLOW_LOCAL_OUTBOUND` | `false` | Set `true` only for explicit local development when trusted origins include localhost/private service endpoints. |

### Generating a JWT Secret

The `task init` command generates a `.env` file containing only `JWT_SECRET` (set to a random value via `openssl rand -base64 48`). To generate one manually:

```sh
openssl rand -base64 48
```

Copy the output into your `.env` file:

```sh
JWT_SECRET=your-generated-secret-here
```

> **⚠️ Important:** Never use the default JWT secret in production. A weak or default secret compromises all authentication tokens.

---

## Docker Volumes

Two paths **must** persist across container restarts:

| Volume Mount | Purpose |
|---|---|
| `/app/data` | SQLite database — all collection data, user accounts, and settings |
| `/app/uploads` | Uploaded coin images referenced by the database |

If these volumes are lost, you lose your entire collection. Use named Docker volumes (as shown in the Compose file) or bind mounts to a backed-up directory:

```yaml
volumes:
  - /opt/ancient-coins/data:/app/data
  - /opt/ancient-coins/uploads:/app/uploads
```

The app container writes to both mounted paths as UID/GID `10001:10001`. Docker named volumes are initialized from the image ownership on first use. For bind mounts, create the host directories and make them writable by UID/GID `10001:10001` before starting the container:

```sh
sudo mkdir -p /opt/ancient-coins/data /opt/ancient-coins/uploads
sudo chown -R 10001:10001 /opt/ancient-coins/data /opt/ancient-coins/uploads
```

The agent container also runs as UID/GID `10001:10001`. It does not require a mounted writable path in the default deployment; if you add one for diagnostics or custom tooling, grant write access to UID/GID `10001:10001`.

---

## Building from Source

Use Taskfile commands to build the Docker image locally:

```sh
task docker-build  # builds with commit SHA tag + latest
```

The agent image can be built directly after refreshing or reviewing the lock:

```sh
docker build -f src/agent/Dockerfile -t ancient-coins-agent:local src/agent
```

The build injects two build arguments:

| Build Arg | Description |
|---|---|
| `APP_VERSION` | Git commit SHA — combined with the root `VERSION` file for app UI version display |
| `BUILD_DATE` | Build timestamp — injected into Vite for version display |

Refresh Python agent dependencies intentionally from `src/agent`:

```sh
pip install uv==0.11.22
uv lock --upgrade
uv sync --locked --extra dev
```

---

## WebAuthn / Passkeys in Production

Ancient Coins supports passwordless authentication via WebAuthn. In production, two environment variables **must** be configured correctly:

### `WEBAUTHN_RP_ID`

Set this to your **bare domain** — no scheme, no port:

```sh
WEBAUTHN_RP_ID=coins.example.com    # ✅ correct
WEBAUTHN_RP_ID=https://coins.example.com  # ❌ wrong
WEBAUTHN_RP_ID=coins.example.com:443      # ❌ wrong
```

### `WEBAUTHN_ORIGIN`

Set this to the **full URL with scheme** that users see in their browser:

```sh
WEBAUTHN_ORIGIN=https://coins.example.com          # ✅ single origin
WEBAUTHN_ORIGIN=https://coins.example.com,https://www.coins.example.com  # ✅ multiple origins
```

> **📝 Note:** The server auto-detects the request origin as a fallback, but explicit configuration is strongly recommended for production.

> **🔒 HTTPS Required:** Browsers enforce HTTPS for WebAuthn/passkey registration and authentication. Passkeys will not work over plain HTTP (except `localhost` for development).

---

## Reverse Proxy Setup

The Go app serves HTTP on the configured `PORT` (default `8080`). In production, place a reverse proxy in front for TLS termination.

Key considerations:
- **Proxy all traffic** to the app on `:8080` — the app handles its own routing, including SPA fallback for Vue routes
- **Set forwarding headers** — `X-Forwarded-Proto` and `Host` headers are required for WebAuthn origin detection
- **WebSocket/SSE support** — the AI search agent uses Server-Sent Events (SSE) streaming; ensure your proxy supports long-lived connections

### nginx

```nginx
server {
    listen 443 ssl;
    server_name coins.example.com;

    ssl_certificate     /etc/letsencrypt/live/coins.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/coins.example.com/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # SSE support for AI agent streaming
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 300s;
    }
}
```

### Caddy

Caddy provides automatic HTTPS via Let's Encrypt with zero configuration:

```
coins.example.com {
    reverse_proxy localhost:8080
}
```

Caddy automatically sets the correct forwarding headers and handles SSE streaming without additional configuration.

---

## HTTPS

HTTPS is **required** for WebAuthn/passkeys to function in production. The Go app itself serves HTTP only — TLS termination happens at the reverse proxy.

Options:
- **Caddy** — automatic HTTPS with Let's Encrypt, zero config (recommended for simplicity)
- **nginx + certbot** — manual setup with Let's Encrypt certificate renewal
- **Traefik** — automatic HTTPS with Docker label-based configuration
- **Cloud load balancer** — AWS ALB, GCP Load Balancer, Azure Application Gateway, etc.

---

## First Launch

After deploying, complete the initial setup:

1. **Register the first account** — navigate to your app URL and click **Register**. The first user to register automatically becomes the **admin**.
2. **Configure AI** — go to **Admin → AI Configuration** and select your AI Provider. Choose **Anthropic** (recommended) and enter your API key, or select **Ollama** for self-hosted models.
3. **System Settings** — go to **Admin → System Settings** to set the log level and Numista API key for catalog lookups.
4. **Start adding coins** — see the [Getting Started Guide](getting-started.md) for details on adding coins and using AI analysis.

---

## CI/CD

GitHub Actions publish Docker images only after the release branch Quality Gate succeeds for the exact commit SHA being published.

**Triggers:**
- `.github/workflows/ci.yml` (`Quality Gate`) runs on pull requests and pushes to `main` and `beta`.
- `.github/workflows/docker-publish.yml` runs after a successful `Quality Gate` push run on `main`.
- `.github/workflows/docker-publish-beta.yml` runs after a successful `Quality Gate` push run on `beta`.
- `.github/workflows/security-scan.yml` runs on pull requests, pushes to `main` and `beta`, weekly schedule, and manual dispatch.

Manual Docker publishing is intentionally disabled so `latest` and `beta` tags cannot bypass the Quality Gate.

**Required GitHub repository settings:**
- Protect `main` and `beta`.
- Require pull requests before merging unless an explicit emergency release process is approved.
- Require the `Quality Gate` workflow jobs (`Go API`, `Vue Web`, `Python Agent`) for both release branches.
- Require the desired `Security Scan` checks when the repository is ready to make scan findings blocking.
- Block force pushes and branch deletion on both release branches.

**Image Tags:**
| Tag | Example |
|---|---|
| Full SHA | `<user>/ancient-coins:sha-a1b2c3d4e5f6...` |
| Short SHA | `<user>/ancient-coins:sha-a1b2c3d` |
| `latest` | `<user>/ancient-coins:latest` from `main` |
| `beta` | `<user>/ancient-coins:beta` from `beta` |

Two images are published per build:
- `<user>/ancient-coins` (app)
- `<user>/ancient-coins-agent` (agent)

**Required Repository Secrets:**

| Secret | Description |
|---|---|
| `DOCKERHUB_USERNAME` | Docker Hub username for authentication |
| `DOCKERHUB_TOKEN` | Docker Hub access token |

**Workflow action pin maintenance:** Workflow `uses:` entries in `.github/workflows/` are pinned to commit SHAs. When updating an action version, resolve the new tag to a commit SHA (for example: `git ls-remote https://github.com/<owner>/<repo> refs/tags/<tag>`) and keep the version comment (for example `# v4`) beside the SHA.

**Reviewed tool and base-image pins:** CI-installed Go tools are pinned in the workflows and matching Taskfile target (`swag` for OpenAPI generation, `govulncheck` for Go vulnerability scans). Refresh these pins monthly or when a security advisory lands by reviewing upstream release notes, updating workflow and Taskfile installs together, then running `task openapi`, `cd src/api; go test ./...`, and `govulncheck ./...`. Dockerfiles use reviewed tag-plus-OCI-index-digest references (for example `image:tag@sha256:...`) so multi-arch builds remain available while production builds are reproducible; refresh the digest at the same cadence or when a base-image CVE requires it.

### Security scan gates

The `.github/workflows/security-scan.yml` workflow runs on pull requests to `main` and `beta`, on a weekly schedule, and by manual dispatch. These checks are intended to be branch-protection requirements:

| Check | Blocking threshold |
|---|---|
| `Gitleaks` | Any detected secret unless precisely allowlisted in `.gitleaks.toml` |
| `Govulncheck` | Any actionable Go vulnerability reported by `govulncheck ./...` |
| `npm audit` | Any high or critical npm advisory (`npm audit --audit-level=high`) |
| `pip-audit` | Any Python vulnerability; this is stricter than high/critical because `pip-audit` does not provide a portable severity threshold |

Temporary exceptions must be narrow, reviewed, and documented with an owner and expiration date in the scanner configuration or the linked threat-model finding. Because Docker image publish workflows run only after pushes to protected branches, these blocking PR checks are the release gate for `latest` and `beta` images.

---

## Backup & Restore

### Database

The SQLite database (at the path configured by `DB_PATH`) can be safely copied while the application is running thanks to WAL mode:

```sh
# Copy from the Docker volume
docker cp ancient-coins-app-1:/app/data/ancientcoins.db ./backup-$(date +%Y%m%d).db
```

### Uploaded Images

Back up the entire uploads directory:

```sh
docker cp ancient-coins-app-1:/app/uploads ./uploads-backup
```

### In-App Export

Navigate to **Settings → Export Collection** to download your collection as JSON. This export includes all coin metadata but **does not include images**.

### Restore

To restore from backup, stop the container, copy the database and uploads back into the volumes, and restart:

```sh
docker compose down
docker cp ./backup.db ancient-coins-app-1:/app/data/ancientcoins.db
docker cp ./uploads-backup/. ancient-coins-app-1:/app/uploads/
docker compose up -d
```

---

## Admin-Managed Settings

These settings are stored in the database and configured through the **Admin** UI — they are not environment variables.

| Setting | Default | Description |
|---|---|---|
| AIProvider | — | Explicit provider choice: `anthropic` or `ollama` (must be set before agent features work) |
| AnthropicAPIKey | — | API key for Claude models |
| AnthropicModel | — | Claude model (e.g., `claude-sonnet-4-20250514`) |
| OllamaURL | `http://localhost:11434` | Ollama server URL |
| OllamaModel | `llava` | Vision model name |
| OllamaTimeout | `300` | Request timeout in seconds |
| SearXNGURL | — | SearXNG search engine URL (required for Ollama web search) |
| NumistaAPIKey | — | Numista catalog API key |
| CoinSearchPrompt | — | System prompt for coin search agent |
| CoinShowsPrompt | — | System prompt for coin shows agent |
| ValuationPrompt | — | System prompt for value estimator |
| ObversePrompt | — | Prompt for obverse image analysis |
| ReversePrompt | — | Prompt for reverse image analysis |
| TextExtractionPrompt | — | Prompt for OCR text extraction |
| LogLevel | — | Application log level |

---

## Troubleshooting

### Passkeys not working
- Verify `WEBAUTHN_RP_ID` matches your exact domain (no scheme or port)
- Verify `WEBAUTHN_ORIGIN` includes the full URL with `https://`
- Confirm HTTPS is properly configured — passkeys require a secure context

### Database locked errors
- Ensure only one container instance accesses the database file at a time
- SQLite is not designed for multi-process concurrent writes

### AI analysis not responding
- Check that the Ollama server is reachable from the container (use `host.docker.internal` if Ollama runs on the host)
- Verify the configured model is pulled (`ollama pull llava`)
- Increase `OllamaTimeout` in Admin settings for slower hardware

### AI agent not responding
- Check that the agent container is running (`docker compose logs agent`)
- Verify `AGENT_SERVICE_URL` points to the agent container

### AI provider not configured
- Set AIProvider in Admin → AI Configuration. The agent chat shows a configuration banner when it's empty.

### Container won't start
- Check logs: `docker compose logs -f`
- Verify the `.env` file exists and `JWT_SECRET` is set (min 32 characters)
- Ensure volume mount paths exist and have correct permissions
