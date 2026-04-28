<!--
  Sync Impact Report
  ==================
  Version change: 1.0.0 → 1.1.0 (gap closure)
  Modified principles: None
  Added sections:
    - XI. Security Hardening
    - XII. Authentication & Token Policy
    - XIII. PWA / Mobile Interaction Rules
    - XIV. Social & Privacy Model
    - XV. Supply Chain & CI Integrity
    - XVI. Account Lifecycle
  Removed sections: None
  Templates requiring updates:
    - .specify/templates/plan-template.md ✅ compatible (Constitution Check section)
    - .specify/templates/spec-template.md ✅ compatible (FR/SC sections align)
    - .specify/templates/tasks-template.md ✅ compatible (phase structure aligns)
  Follow-up TODOs: None
-->

# Ancient Coins Constitution

## Core Principles

### I. Layered Architecture (Go API)

The Go API MUST follow a strict four-layer architecture:

```
Handler → Service → Repository → Database
```

- **Handlers** are thin: parse the request, call a service or repository,
  return the response. Handlers MUST NOT contain business logic or raw SQL.
- **Services** contain all business logic. Services MUST be HTTP-agnostic
  and MUST NOT reference `gin.Context` or any HTTP framework type.
- **Repositories** own all database access. Every GORM query MUST live in
  `src/api/repository/`. Repositories MUST use GORM scopes from
  `repository/scopes.go` instead of repeating `.Where()` clauses.
- **Models** (`src/api/models/`) MUST import only the Go standard library.
- Multi-step writes MUST use transactions (`r.db.Transaction()`).
- Internal errors MUST NOT leak to clients. Log server-side; return
  generic messages to the caller.

**Rationale**: Enforced layer separation prevents coupling, enables
independent testing of each layer, and keeps the codebase navigable as
feature count grows.

### II. Dependency Injection

All packages MUST receive dependencies via constructor injection
(`NewXxxHandler(repo, service)` pattern).

- **Only `main.go` may import the `database` package.** Every other
  package receives `*gorm.DB` or a repository/service interface through
  its constructor.
- DI wiring order in `main.go`: `config.Load()` → `database.Connect()`
  → construct repos → construct services → construct handlers → register
  routes.
- Three route groups exist: `api` (public auth), `protected`
  (JWT required), `admin` (JWT + admin role).

**Rationale**: Constructor injection makes dependencies explicit, enables
test doubles, and prevents hidden global state.

### III. Service Boundary Separation

The system is composed of three independently deployable services. Each
service MUST respect strict boundary rules:

| Service | Runtime | Responsibilities |
|---------|---------|-----------------|
| Go API | Go 1.26.1 / Gin | REST API, auth, data persistence, SSE proxy |
| Vue SPA | Browser | UI, state management, PWA shell |
| Python Agent | Python 3.12 / FastAPI | AI inference, LangGraph pipelines |

- The **Go API MUST contain zero LLM or agent logic**. All AI inference
  MUST be proxied to the Python agent service via `services/agent_proxy.go`.
- The **Python agent is stateless** — it MUST NOT access the database
  directly. All context (API keys, user data, prompts) MUST be passed
  per-request from the Go API.
- The **Vue SPA** communicates with the Go API exclusively via REST
  (`/api/*`). It MUST NOT call the Python agent directly.
- SSE streams flow Python → Go → Vue (Go proxies the byte stream).

**Rationale**: Hard service boundaries prevent accidental coupling
between AI logic and business logic, allow independent scaling, and
keep each codebase in its native language ecosystem.

### IV. Strict Typing & Build Parity

All code MUST pass the strictest available type checking for its
language, and local builds MUST match CI/Docker builds:

- **Go**: `go vet ./...` MUST pass with zero warnings.
- **TypeScript/Vue**: Docker builds use `vue-tsc --build`, which is
  stricter than local `vue-tsc --noEmit`. All code MUST pass the Docker
  check. Use `?.` (optional chaining) and `?? ''` / `?? 0` (nullish
  coalescing) for nullable props passed to non-nullable children.
- **Python**: `ruff check app/ tests/` MUST pass. All request/response
  schemas MUST use Pydantic models (`app/models/`).

**Rationale**: Type strictness catches bugs before runtime, and build
parity eliminates "works on my machine" failures.

### V. Design Token System

The Vue frontend MUST use the design token system defined in
`variables.css` and global classes in `main.css`. Raw values MUST NOT
be hardcoded when a token exists.

- **Never hardcode** `border-radius`, colors, spacing, or font sizes.
- **Never duplicate** chip or button CSS — use global classes (`.chip`,
  `.chip-sm`, `.badge`, `.btn`, `.btn-primary`, etc.).
- **Never invent** a new font size — pick from the typography scale
  (Cinzel for headings, Inter for body).
- **Gold (`--accent-gold`)** is reserved for: active states,
  values/prices, links, and section accents.
- All uppercase labels MUST use `letter-spacing: 0.08em`.

**Rationale**: A strict token system ensures visual consistency,
prevents design drift, and enables theme changes from a single file.

### VI. AI/Agent Isolation

All AI agent pipelines MUST follow these rules:

- Search agents MUST pass only tool-returned data downstream — never
  invented details.
- Verification agents MUST confirm every URL is live and every date is
  in the future.
- All worker agent outputs MUST conform to a defined Pydantic schema —
  no free-form text.
- The top-level supervisor (`app/supervisor.py`) MUST enforce a max
  iteration count to prevent infinite loops.
- Anthropic web search MUST use `get_search_model()` from
  `app/llm/provider.py` (which calls `bind_tools`). Use
  `get_chat_model()` for nodes that do not search.

**Rationale**: AI output is non-deterministic. Schema enforcement and
data provenance rules ensure the rest of the system can trust agent
results.

### VII. Schema-Driven Contracts

Every external-facing interface MUST have an explicit schema:

- **Go API**: Swagger annotations on all public handler methods.
- **Python Agent**: Pydantic models for all request/response payloads.
- **Vue SPA**: All API calls go through `src/web/src/api/client.ts`
  (Axios with JWT interceptor and 401 refresh queue).
  `sanitizeCoin()` normalizes `''`/`undefined` → `null` before sending.

**Rationale**: Explicit schemas are the single source of truth for
inter-service communication and enable automated validation.

### VIII. Conventional Commits & Workflow

All commits MUST use conventional prefixes: `feat:`, `fix:`, `docs:`,
`refactor:`, `chore:`.

- AI-assisted commits MUST include the co-author trailer:
  `Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>`
- Build automation uses Taskfile (`task --list` for all targets).
- Multi-stage Docker builds produce two containers: Go+Vue (app) and
  Python (agent), orchestrated via `docker-compose.yaml`.

**Rationale**: Conventional commits enable automated changelogs and
semantic versioning. Standardized build tooling reduces onboarding
friction.

### IX. UI/UX Consistency

- No emojis in UI text, prompts, or AI responses.
- Dark theme is the default.
- The app MUST be PWA-compatible — test on mobile viewports.
- Icons MUST use `lucide-vue-next`.
- Agent chat streaming uses `fetch` + manual SSE parsing, not Axios.
- CSS variables: `--accent-gold`, `--bg-card`, `--border-subtle`,
  `--text-primary` (see Design Token System for full list).

**Rationale**: Consistent UI rules prevent visual fragmentation across
features and ensure the app feels cohesive on all devices.

### X. Architecture Enforcement

Architecture rules MUST be enforced by automated tests:

- `architecture_test.go` validates package import rules at build time.
- Package import constraints:

| Package | May Import |
|---------|-----------|
| `handlers/` | `services/`, `repository/`, `models/` |
| `services/` | `repository/`, `models/` |
| `repository/` | `models/`, `gorm.io/gorm` |
| `models/` | Standard library only |
| `middleware/` | `models/`, `gorm.io/gorm` |

- `go test ./...` MUST pass before any PR is merged.
- `ruff check` and `pytest` MUST pass for agent changes.

**Rationale**: Automated enforcement catches violations at commit time,
not during code review. Rules that are only documented but not enforced
will eventually be violated.

### XI. Security Hardening

All services MUST follow these security baselines:

- **CORS**: MUST whitelist specific origins. `AllowOriginFunc` MUST NOT
  return `true` for all origins. Production MUST list only the
  application domain.
- **Input validation**: User-supplied values used in SQL MUST be
  parameterized or validated against a whitelist. GORM scopes are
  preferred over raw queries.
- **Upload validation**: File uploads MUST validate extension against an
  allowlist (`.jpg`, `.jpeg`, `.png`, `.gif`, `.webp`) and check MIME
  type from magic bytes, not just `Content-Type`.
- **Rate limiting**: Auth endpoints (`/api/auth/*`) MUST be rate-limited
  to prevent brute-force attacks.
- **Body size limits**: `MaxMultipartMemory` and JSON body size MUST be
  capped (recommended: 10 MB for multipart, 1 MB for JSON).
- **Error responses**: Internal error details MUST NOT leak to clients.
  Log server-side; return generic messages.
- **Containers**: Production Docker images MUST run as non-root users.

**Rationale**: Security defaults prevent common attack vectors (XSS,
CSRF, injection, DoS) and reduce the blast radius of vulnerabilities.
Source: `docs/security-analysis.md`.

### XII. Authentication & Token Policy

The application uses a multi-method auth stack. Each method MUST follow
these rules:

- **JWT access tokens**: 15-minute expiry. Signed with `JWT_SECRET`
  environment variable. The application SHOULD refuse to start if
  `JWT_SECRET` is unset or below minimum entropy in production.
- **Refresh tokens**: 30-day rolling lifetime. Format: `rt_` prefix +
  32 random hex bytes. Server stores SHA-256 hash only. Old refresh
  tokens MUST be revoked on each refresh (one-time use).
- **Client token storage**: `localStorage` on the frontend. The axios
  interceptor handles 401 → refresh → replay automatically with a
  concurrent-request queue.
- **WebAuthn/FIDO2**: Platform authenticators only (Face ID, Touch ID,
  fingerprint). `WEBAUTHN_RP_ID` and `WEBAUTHN_ORIGIN` MUST be set
  for production. WebAuthn challenge sessions MUST have a TTL
  (recommended: 5 minutes) to prevent memory leaks.
- **API keys**: Used for programmatic access. Keys MUST be stored hashed
  and MUST be revocable.
- **First user**: The first registered user is auto-assigned admin role.

**Rationale**: Explicit token policies prevent silent security
degradation and ensure consistent auth behavior across deployments.
Source: `docs/authentication.md`.

### XIII. PWA / Mobile Interaction Rules

The application MUST maintain two distinct UI modes:

- **PWA/mobile mode** (`display-mode: standalone`):
  - Hamburger menu with popover for filters, sort, navigation, and
    logout.
  - Default gallery view is swipe carousel (315 × 399 px cards).
  - Pull-to-refresh on gallery when scrolled to top.
  - Camera capture button on image uploads (rear camera).
  - "My Collection" title hidden for compact header.
  - No page-level pagination in swipe mode.
  - NO sticky positioning on detail page images or action bars.
- **Desktop mode** (standard browser):
  - Inline toolbar with filters, sort, and view controls.
  - Default gallery view is grid.
  - Sticky image sidebar and sticky action bar on detail pages.
  - Full pagination visible.

- Offline: Service worker caches static assets. API calls require
  network. The app shell MUST load without connectivity.
- Settings (default view, sort) persist in `localStorage`.
- Desktop layout changes MUST NOT break PWA layout. Use
  `@media (min-width: 769px)` for desktop-only CSS.

**Rationale**: PWA and desktop are the two primary consumption modes.
Regressions in either degrade user experience significantly.
Source: `docs/pwa-guide.md`.

### XIV. Social & Privacy Model

Social features MUST enforce these rules:

- **Follow workflow**: pending → accepted / blocked. Only accepted
  followers can view a user's gallery.
- **Blocked users**: Cannot re-request until explicitly unblocked.
- **Public/private profiles**: Only `isPublic=true` users appear in
  search and can receive follow requests. Setting a profile to private
  PERMANENTLY DELETES all existing followers (destructive action).
- **Private coins**: Individual coins marked `isPrivate` are hidden from
  all followers, even accepted ones.
- **Gallery access**: Read-only. Limited to images and essential details.
  Pricing/value and AI analysis MUST NOT be shown to followers.
- **Comments & ratings**: Accepted followers can comment and rate (1–5
  stars). Both commenter and coin owner can delete comments.
- **Avatars**: Locally uploaded images (no Gravatar dependency). Default
  avatar is the Ed-Mar coin logo.

**Rationale**: Privacy and access control are critical for a personal
collection app. These rules prevent data leakage and give users control.
Source: `docs/social-feature.md`.

### XV. Supply Chain & CI Integrity

- GitHub Actions MUST pin action versions by SHA, not mutable tags.
- Docker base images SHOULD pin to specific digests for production
  builds.
- Branch protection MUST be enabled on `main` (require PR reviews,
  require status checks to pass).
- Dependency updates SHOULD be automated (Dependabot or equivalent).

**Rationale**: Supply chain attacks are a growing vector. Pinning and
branch protection prevent unauthorized code from reaching production.
Source: `docs/security-analysis.md`.

### XVI. Account Lifecycle

- **Email**: Required for all new registrations. Legacy users without
  email see a dismissible modal (7-day snooze via `localStorage`).
  `GET /auth/me` includes `emailMissing` flag.
- **Registration**: Username + password + email. Validated format.
- **Admin**: First registered user auto-assigned admin role.
- **Profile deletion**: Setting `isPublic=false` permanently deletes
  followers (see Social & Privacy Model).

**Rationale**: Clear account lifecycle rules prevent edge cases around
legacy data and ensure consistent onboarding.
Source: `docs/authentication.md`, `docs/social-feature.md`.

## Technology Stack

| Layer | Technology | Version | Path |
|-------|-----------|---------|------|
| Backend | Go, Gin, GORM, SQLite | Go 1.26.1 | `src/api/` |
| Frontend | Vue 3, TypeScript, Pinia, Vite, PWA | Vue 3 | `src/web/` |
| Agent | Python, FastAPI, LangGraph, LangChain | Python 3.12 | `src/agent/` |
| Build | Multi-stage Docker, Taskfile | — | `Dockerfile`, `src/agent/Dockerfile` |
| Database | SQLite (pure-Go driver) | — | Runtime volume |
| Auth | JWT (access + refresh tokens) | — | `src/api/middleware/` |

- Settings use key-value `AppSetting` model; constants and defaults
  live in `services/settings_service.go`.
- Sentinel errors in services (e.g., `ErrNotFound`,
  `ErrInvalidCredentials`).
- First registered user is auto-assigned admin role.

## Development Workflow

### Adding a New API Feature

1. Model in `src/api/models/` → add to `AutoMigrate` in
   `database/database.go`.
2. Repository in `src/api/repository/*_repository.go`.
3. Service (if business logic needed) in `src/api/services/*_service.go`.
4. Thin handler in `src/api/handlers/` with `NewXxxHandler()` constructor.
5. Wire in `src/api/main.go` (create repo → service → handler, register
   routes under correct group).
6. Run `go test ./...` to verify architecture rules pass.

### Build & Test Commands

```bash
# Go API (from src/api/)
go build ./...               # compile
go vet ./...                 # lint
go test -v ./...             # all tests

# Vue frontend (from src/web/)
npm run build                # production build (type-check + vite)

# Python agent (from src/agent/)
ruff check app/ tests/       # lint
pytest tests/ -v             # all tests

# Task runner (from repo root)
task build                   # build API + web
task test                    # Go tests
task up-all                  # all dev servers
```

### Quality Gates

- All Go architecture tests MUST pass (`architecture_test.go`).
- Docker `vue-tsc --build` MUST pass (stricter than local).
- Python `ruff check` and `pytest` MUST pass for agent changes.
- Conventional commit format MUST be used.

## Governance

1. **Supremacy**: This constitution supersedes all other development
   practices for the Ancient Coins project. In case of conflict between
   this document and any other guide, this document prevails.

2. **Runtime Guidance**: `.github/copilot-instructions.md` contains
   detailed runtime development guidance (design tokens, code
   conventions, API recipes). It MUST remain consistent with this
   constitution. If a conflict is discovered, amend the guidance file
   to match.

3. **Amendment Procedure**:
   - Any principle change MUST be documented with a version bump.
   - Adding or materially expanding a principle = MINOR bump.
   - Removing or redefining a principle = MAJOR bump.
   - Wording clarifications or typo fixes = PATCH bump.
   - Every amendment MUST update `Last Amended` date.

4. **Compliance Review**: All PRs and code reviews MUST verify
   compliance with these principles. Automated enforcement
   (`architecture_test.go`, linters, type checkers) is preferred
   over manual review.

5. **Complexity Justification**: Any deviation from these principles
   MUST be explicitly justified in the PR description and tracked in
   the plan's Complexity Tracking table.

**Version**: 1.1.0 | **Ratified**: 2026-04-28 | **Last Amended**: 2026-04-28
