# Production Deployment Guide

This guide covers deploying the Ancient Coins application to a production environment. For local development setup, see the [README](../README.md).

> **Public exposure warning:** The default Docker Compose file is suitable for local development or a trusted home network. Do **not** place it directly on the public internet without TLS, a hardened reverse proxy, trusted proxy configuration, closed or invite-only registration, backups, monitoring, and firewall rules from this guide. Treat internet-facing deployment as a different threat model: anonymous attackers can brute-force auth, upload payloads, scrape public media, and probe the Python agent boundary continuously.

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
      test: ["CMD", "python", "-c", "import urllib.request; urllib.request.urlopen('http://localhost:8081/ready')"]
      interval: 10s
      timeout: 5s
      retries: 3
      start_period: 15s
    restart: unless-stopped

volumes:
  db-data:
  uploads:
```

Create a `.env` file with every required value in [Required `.env` values](#required-env-values), then start the app:

```sh
docker compose up -d
```

The app is now available at `http://localhost:8080`. Keep this binding local or home-network-only until the [Public Exposure Beta Acceptance Checklist](#public-exposure-beta-acceptance-checklist) is complete. For internet exposure, put nginx/Caddy/Traefik in front of the app and publish only ports `80` and `443`.

> **⚠️ Multi-container networking:** The API and agent communicate bidirectionally:
> - `AGENT_SERVICE_URL` (API → agent): used for all AI features
> - `AGENT_INTERNAL_CALLBACK_URL` (agent → API): used only by collection chat (#217)
>
> In containerized deployments, both **must** use Docker service names (e.g., `http://app:8080`, `http://agent:8081`) instead of `localhost`. The agent service is not published on a host port by default; every endpoint except `/health` requires `AGENT_INTERNAL_SERVICE_TOKEN` to be configured, and protected endpoints also require the Go API to send it. The `AGENT_INTERNAL_CALLBACK_URL` defaults to `localhost:8080` for local `task up-all` development; **set it explicitly** in compose/k8s environments or collection chat will fail with "All connection attempts failed".

---

## Environment Variables

### Required `.env` values

Copy `.env.example` to `.env` and replace the placeholders before starting the app. For Docker Compose, the following values are required:

```env
JWT_SECRET=<48-byte random value>
AGENT_INTERNAL_SERVICE_TOKEN=<48-byte random value shared by app and agent>
TRUSTED_PROXIES=none
```

Generate the two secrets with:

```sh
openssl rand -base64 48
```

`AGENT_INTERNAL_SERVICE_TOKEN` is **not** an Anthropic/Ollama API key. It is a private shared credential between the Go API and Python agent service. If it is missing from either process, Admin AI provider tests can still pass, but coin analysis and agent chat will fail with an internal agent service credential error.

For public HTTPS/passkey deployments, also set:

```env
WEBAUTHN_RP_ID=coins.example.com
WEBAUTHN_ORIGIN=https://coins.example.com
CORS_ORIGINS=https://coins.example.com
```

For Docker Compose/multi-container deployments, keep the agent URLs on Docker service names:

```env
AGENT_SERVICE_URL=http://agent:8081
AGENT_INTERNAL_CALLBACK_URL=http://app:8080
AGENT_TRUSTED_OUTBOUND_ORIGINS=http://app:8080
```

For local `task up-all` development, the API and agent both load the repo-root `.env`. Use localhost URLs:

```env
AGENT_SERVICE_URL=http://localhost:8081
AGENT_INTERNAL_CALLBACK_URL=http://localhost:8080
AGENT_TRUSTED_OUTBOUND_ORIGINS=http://localhost:8080
AGENT_ALLOW_LOCAL_OUTBOUND=true
```

### Full variable reference

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
| `AGENT_INTERNAL_SERVICE_TOKEN` | — | Shared API → agent credential. The Python agent `/ready` endpoint fails without it, and every endpoint except `/health` requires it in configured deployments. Required for local `task up-all`, Docker Compose, and production-like deployments. |
| `CORS_ORIGINS` | *(WebAuthn origins + localhost)* | Comma-separated list of allowed CORS origins. Falls back to `WEBAUTHN_ORIGIN` values plus `http://localhost:5173` and `http://localhost:8080` |
| `AGENT_LOG_LEVEL` | `INFO` | Python agent log level |
| `AGENT_DEBUG` | `false` | Enable debug mode on the agent container (exposes `/docs` endpoint) |
| `AGENT_SEARXNG_URL` | — | SearXNG instance URL for Ollama web search (required when using Ollama provider) |
| `AGENT_TRUSTED_OUTBOUND_ORIGINS` | `http://app:8080` in Compose | Comma-separated exact origins the Python agent may call for Ollama, SearXNG, and collection tools. |
| `AGENT_ALLOW_LOCAL_OUTBOUND` | `false` | Set `true` only for explicit local development when trusted origins include localhost/private service endpoints. |
| `TRUSTED_PROXIES` / `GIN_TRUSTED_PROXIES` | *(required in `GIN_MODE=release`)* | Comma-separated proxy IPs/CIDRs whose `X-Forwarded-*` headers the Go API should trust. Use the exact reverse-proxy IP/CIDR, `none` only when no reverse proxy is present, and never `0.0.0.0/0`. |

### Generating local secrets

The `task init` command creates `.env` if needed and adds missing `JWT_SECRET` and `AGENT_INTERNAL_SERVICE_TOKEN` values without overwriting existing secrets.

To generate either value manually:

```sh
openssl rand -base64 48
```

Copy each output into your `.env` file:

```sh
JWT_SECRET=your-generated-secret-here
AGENT_INTERNAL_SERVICE_TOKEN=your-generated-token-here
```

> **⚠️ Important:** Never use the default JWT secret in production. A weak or default secret compromises all authentication tokens. Never expose `AGENT_INTERNAL_SERVICE_TOKEN` to browsers or third-party services; it must only be shared between the Go API and Python agent service.

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

### nginx baseline for internet-facing deployments

This is the minimum nginx shape for public exposure. Adjust certificate paths and domain names only after you understand each security header.

```nginx
server {
    listen 80;
    server_name coins.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name coins.example.com;

    ssl_certificate     /etc/letsencrypt/live/coins.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/coins.example.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers off;

    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header Content-Security-Policy "default-src 'self'; img-src 'self' data: blob:; media-src 'self'; connect-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self'; object-src 'none'; base-uri 'self'; frame-ancestors 'none'; form-action 'self'; upgrade-insecure-requests" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    add_header X-Frame-Options "DENY" always;
    add_header Permissions-Policy "camera=(self), microphone=(), geolocation=(), payment=(), usb=(), serial=(), bluetooth=()" always;

    # Keep in sync with the API upload cap (10 MB multipart baseline in Constitution Principle V).
    client_max_body_size 10m;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;

        # Re-set forwarding headers at the trusted edge. Do not append user-supplied X-Forwarded-For.
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $remote_addr;
        proxy_set_header X-Forwarded-Proto https;
        proxy_set_header X-Forwarded-Port 443;
        proxy_set_header Forwarded "";

        # SSE support for AI agent streaming through the Go API.
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
    }
}
```

If you intentionally run multiple trusted proxies, append only within the trusted chain and configure the API's trusted proxy list to match. Never trust `X-Forwarded-For` from arbitrary clients.

### Caddy

Caddy provides automatic HTTPS via Let's Encrypt with zero configuration:

```
coins.example.com {
    reverse_proxy localhost:8080
}
```

Caddy automatically sets the correct forwarding headers and handles SSE streaming without additional configuration.

---

## Trusted Proxy and Client IP Handling

Internet-facing deployments must make the Go API derive the client IP only from trusted reverse proxies. This matters for failed-login logs, IP bans, account lockout, audit events, and rate limits.

Supported configuration:

```env
TRUSTED_PROXIES=127.0.0.1/32,172.18.0.0/16
```

Docker examples:

```yaml
# Host nginx terminates TLS; app is reachable only from localhost.
services:
  app:
    ports:
      - "127.0.0.1:8080:8080"
    environment:
      - TRUSTED_PROXIES=127.0.0.1/32

# Containerized nginx shares a Docker network; app has no public host port.
services:
  app:
    expose:
      - "8080"
    environment:
      - TRUSTED_PROXIES=172.18.0.0/16
```

Verification checklist:

1. Browse from a device outside the server network.
2. Open **Admin → Security → Public-Facing Readiness** (`GET /admin/security/exposure-check`). If the UI shows **API sees your IP as**, confirm the value; otherwise inspect a new failed-login/security event after the check.
3. Confirm it shows your public client IP, not the nginx container IP, Docker bridge IP, `127.0.0.1`, or a spoofed value you sent in `X-Forwarded-For`.
4. Send a request with a fake `X-Forwarded-For` header from the internet and confirm the API still records the real edge-observed IP.

Until trusted proxy configuration is verified, use home-network-only exposure or restrict access with a VPN.

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

1. **Register the first account** — navigate to your app URL and click **Register**. The first user to register automatically becomes the **admin**. Do this from a trusted network before opening the site broadly.
2. **Set registration mode** — public-facing deployments should not remain open by default. The `RegistrationMode` admin setting supports `closed`, `invite`, or `open` registration. Use `invite` for beta, `closed` for private/home deployments after admin creation, and `open` only if you intentionally accept public signups and have monitoring enabled.
3. **Configure AI** — go to **Admin → AI Configuration** and select your AI Provider. Choose **Anthropic** (recommended) and enter your API key, or select **Ollama** for self-hosted models.
4. **System Settings** — go to **Admin → System Settings** to set the log level, public app URL, and Numista API key for catalog lookups.
5. **Start adding coins** — see the [Getting Started Guide](getting-started.md) for details on adding coins and using AI analysis.

---

## Keep the Agent Service Private

The Python agent service is an internal worker, not a public API. Public traffic must flow Browser → nginx/TLS → Go API (`:8080`) → agent (`agent:8081`).

Required controls:

- Do **not** add `ports: ["8081:8081"]` to the `agent` service. Use Docker `expose` only.
- Set a high-entropy `AGENT_INTERNAL_SERVICE_TOKEN` in both containers; rotate it if logs or config are exposed.
- Keep `AGENT_DEBUG=false` outside local development so FastAPI docs are not exposed.
- Firewall public hosts to allow only `80/tcp`, `443/tcp`, and restricted admin SSH/VPN. Block `8080/tcp` unless it is bound to `127.0.0.1`; block `8081/tcp` always.
- Keep `AGENT_TRUSTED_OUTBOUND_ORIGINS` narrow. Only set `AGENT_ALLOW_LOCAL_OUTBOUND=true` for deliberate local development, never for internet-facing beta.

Quick check:

```sh
# From outside the host/network, these must fail or time out:
curl http://coins.example.com:8080/health
curl http://coins.example.com:8081/health
```

---

## Registration, Audit Events, and Abuse Controls

### Registration modes

The `RegistrationMode` admin setting includes three modes:

| Mode | Use when | Behavior |
|---|---|---|
| `closed` | Private home deployment after first admin setup | No new public self-registration. Admins manage users/invites. |
| `invite` | Beta rollout | New users need an admin-created invitation. Recommended default for public beta. |
| `open` | Intentional public community mode | Anyone can register. Use only with alerting and daily audit review. |

The first registered user remains the initial admin per Constitution Principle V. Complete first-user setup before switching DNS or firewall rules to public traffic.

### What admins should be able to see/do

The public-facing hardening branch exposes admin security endpoints under `/admin/security/*` and user unlock at `/admin/users/:id/unlock` for:

- Failed login events: `password_login_failure` and `webauthn_login_failure` include username, derived client IP, user agent, timestamp, and message where applicable.
- Successful auth events: `password_login_success` and `webauthn_login_success`; refresh/API-key failures are recorded as `refresh_failure` and `api_key_auth_failure`.
- IP bans: active banned IPs/subnets are managed via `GET/POST/DELETE /admin/security/ip-rules`; `ip_rule_created` and `ip_rule_deleted` events are recorded.
- Account lockouts: 5 failed password attempts in 15 minutes locks the account for 15 minutes; admins can unlock with `POST /admin/users/:id/unlock`; events use `account_lockout` and `account_unlock`.
- IP throttling: 20 failed password attempts from one IP in 15 minutes creates a temporary 15-minute IP rule.
- Backup status: `BackupStatus` admin setting and `/admin/security/summary` response surface whether backups are configured (`not_configured` by default).

Current admin UI sections are **Security Overview**, **Public-Facing Readiness**, **Security Events**, **Manual Bans**, and **Locked Users**. Review them daily during beta and keep registration invite-only or closed.

### Alerting baseline

At minimum, configure host or log-based alerts for:

- Failed-login bursts from one IP or against one account.
- New IP ban or repeated ban hits.
- Account lockout.
- Admin login from a new IP/device or outside the normal maintenance window.
- Registration mode changed to `open`.
- Backup job failure or missing backup for more than 24 hours during beta.
- Agent service receiving direct traffic or returning unauthorized-token errors unexpectedly.

Pushover, uptime monitors, a lightweight log scraper, or the host provider's alerting are all acceptable for beta; the key requirement is that Brian receives the alert without opening the app.

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

SQLite and uploads must be backed up together. The database contains coin metadata and image references; `/app/uploads` contains the media those records point to. A JSON export is useful for portability but is **not** a disaster-recovery backup because it does not include images, users, settings, passkeys, API keys, or scheduler history.

### Backup policy

- Back up `/app/data/ancientcoins.db` plus any `ancientcoins.db-wal` and `ancientcoins.db-shm` files if the app remains running.
- Back up the entire `/app/uploads` directory in the same backup set.
- Encrypt backups before they leave the host.
- Store at least one copy off-host (cloud storage, NAS, or external disk not mounted full-time).
- Keep enough retention to recover from accidental deletion that is noticed days later.
- During public beta, confirm the backup job every day for the first week.

### Cold backup example

A cold backup is simplest and safest for a small personal deployment:

```sh
docker compose stop app
# Copy the database directory and uploads directory from your named volume or bind mount.
# If using bind mounts, archive /opt/ancient-coins/data and /opt/ancient-coins/uploads together.
docker compose start app
```

If you use Docker named volumes, prefer a documented host backup tool that snapshots the volume contents. If you use bind mounts, make the backup path explicit (for example `/opt/ancient-coins/data` and `/opt/ancient-coins/uploads`).

### In-App Export

Navigate to **Settings → Export Collection** to download your collection as JSON. This export includes coin metadata but **does not include images** and does not replace volume-level backups.

### Restore

Restore both database and uploads from the same backup timestamp:

```sh
docker compose down
# Restore ancientcoins.db plus matching -wal/-shm files if present.
# Restore the uploads directory contents.
docker compose up -d
```

After restore:

1. Log in as admin.
2. Open several image-heavy coins and confirm media renders.
3. Confirm WebAuthn/passkey login still works on the production domain.
4. Run a small AI feature if the agent is enabled.
5. Record the restore drill date and result. The Constitution §20 cadence requires at least an annual restore drill; public beta should perform one before inviting users.

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
| PublicAppURL | — | Public `https://` base URL used for external notifications and links |
| RegistrationMode | `closed` | `closed`, `invite`, or `open`; use `invite` for public beta |
| BackupStatus | `not_configured` | Operator-maintained backup readiness/status surfaced in the Admin Security summary |


---

## Public Exposure Beta Acceptance Checklist

Use this gate before inviting beta users or pointing public DNS at the host.

### External nginx checks

- [ ] Only `80/tcp` and `443/tcp` are public; SSH is restricted by IP/VPN; `8080` is localhost-only or private; `8081` is not public.
- [ ] nginx/Caddy terminates TLS and redirects HTTP to HTTPS.
- [ ] TLS allows TLS 1.2/1.3 only.
- [ ] HSTS, CSP, `X-Content-Type-Options`, `Referrer-Policy`, `X-Frame-Options`/`frame-ancestors`, and `Permissions-Policy` are present.
- [ ] `client_max_body_size` matches the API upload cap.
- [ ] SSE buffering is off for proxied app traffic.
- [ ] Forwarding headers are stripped/re-set at the edge; the API sees the real client IP.

### WebAuthn HTTPS and registration

- [ ] First admin account created from a trusted network.
- [ ] Registration is `invite` for beta or `closed` for private deployment; never left accidentally `open`.
- [ ] **WebAuthn HTTPS** uses the real production domain: `WEBAUTHN_RP_ID=coins.example.com`, `WEBAUTHN_ORIGIN=https://coins.example.com`.
- [ ] Test password login, passkey registration, passkey login, logout, and refresh after TLS is active.
- [ ] Invite only a small initial group and expand gradually.

### Private media

- [ ] **Private media**: confirm authenticated `/uploads/*` requests require a valid token, private coins/profile/follower views do not expose private images, and public showcase images only load through `/api/showcase/:slug/uploads/*`.
- [ ] Open public showcase URLs in a logged-out/private browser and verify only intended data is visible.
- [ ] Confirm PWA/service worker logout clears private media caches on shared devices.

### Operations

- [ ] **Agent port privacy**: agent service is private (`expose`, no host `ports` for 8081), is not published, and requires `AGENT_INTERNAL_SERVICE_TOKEN`.
- [ ] Backups cover SQLite plus uploads, are encrypted, and have an off-host copy.
- [ ] **Backup restore drill** has succeeded before beta invites.
- [ ] Alerts exist for failed-login bursts, bans, account lockouts, admin logins, direct agent traffic, and backup failures.
- [ ] Review security/audit logs daily for the first beta week.
- [ ] Security scan gates have run for the exact image/commit being deployed; High/Critical findings are triaged before public rollout.

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
