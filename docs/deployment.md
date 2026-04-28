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
2. **Go 1.26.1** — compiles the API binary and embeds the Vue dist
3. **Alpine 3.21** — minimal runtime (~40 MB final image)

The **agent** container uses a separate Dockerfile (`src/agent/Dockerfile`) to build the Python LangGraph service.

The app container contains:

| Path | Description |
|---|---|
| `/app/ancient-coins-api` | Go binary (API + SPA server) |
| `/app/wwwroot/` | Vue SPA static assets |
| `/app/data/` | SQLite database directory |
| `/app/uploads/` | Uploaded coin images |

Data is stored in a **SQLite database** (via GORM, WAL mode enabled) and coin images are stored directly on the filesystem.

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
    ports:
      - "8081:8081"
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
| `AGENT_SERVICE_URL` | `http://agent:8081` | Python agent service URL |
| `CORS_ORIGINS` | *(WebAuthn origins + localhost)* | Comma-separated list of allowed CORS origins. Falls back to `WEBAUTHN_ORIGIN` values plus `http://localhost:5173` and `http://localhost:8080` |
| `AGENT_LOG_LEVEL` | `INFO` | Python agent log level |
| `AGENT_DEBUG` | `false` | Enable debug mode on the agent container (exposes `/docs` endpoint) |
| `AGENT_SEARXNG_URL` | — | SearXNG instance URL for Ollama web search (required when using Ollama provider) |

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

---

## Building from Source

Use Taskfile commands to build the Docker image locally:

```sh
task docker-build  # builds with commit SHA tag + latest
```

The build injects two build arguments:

| Build Arg | Description |
|---|---|
| `APP_VERSION` | Git commit SHA — displayed in the app UI |
| `BUILD_DATE` | Build timestamp — injected into Vite for version display |

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

The GitHub Actions workflow at `.github/workflows/docker-publish.yml` automates image builds and publishing.

**Triggers:**
- Push to `main` branch
- Manual dispatch (workflow_dispatch)

**Image Tags:**
| Tag | Example |
|---|---|
| Full SHA | `<user>/ancient-coins:a1b2c3d4e5f6...` |
| Short SHA | `<user>/ancient-coins:a1b2c3d` |
| `latest` | `<user>/ancient-coins:latest` |

Two images are published per build:
- `<user>/ancient-coins:latest` (app)
- `<user>/ancient-coins-agent:latest` (agent)

**Required Repository Secrets:**

| Secret | Description |
|---|---|
| `DOCKERHUB_USERNAME` | Docker Hub username for authentication |
| `DOCKERHUB_TOKEN` | Docker Hub access token |

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
