<!--
  Sync Impact Report
  ==================
  Version change: 2.0.0 → 3.0.0 (MAJOR — consolidated and renumbered principles)
  Modified principles: Consolidated Principles I–XVII into Principles I–IX
  Added sections:
    - Principle IV. Simple Complete Changes
  Removed sections:
    - Separate standalone principles for DI, architecture enforcement, AI isolation,
      schema contracts, auth, social/privacy, account lifecycle, and supply chain;
      their binding content is consolidated into the nine-principle set.
  Templates requiring updates:
    - ✅ .github/copilot-instructions.md — references Principle IV
    - ✅ .github/pull_request_template.md — adds Principle IV self-check
    - ✅ .specify/templates/plan-template.md — compatible
    - ✅ .specify/templates/spec-template.md — compatible
    - ✅ .specify/templates/tasks-template.md — compatible
    - ✅ .specify/templates/agent-file-template.md — compatible
  Follow-up TODOs: None
-->

# Ancient Coins Constitution

> **This document is the non-negotiable contract for how this project is built.**
> Every AI agent session must read this file first. Every PR must comply.
> Deviations require an explicit, documented waiver (ADR) under §22.

**Project**: Ancient Coins (self-hosted personal collection PWA)
**Version**: 3.0.0
**Ratified**: 2026-04-28
**Last Amended**: 2026-06-09

## §0. Hierarchy of Authority

When two artifacts conflict, the higher-authority document wins. Lower-authority
artifacts MUST be updated to match within the same PR, or the conflict MUST be
escalated to amend the higher artifact (see §22).

Ordered list of governing artifacts, highest authority first:

1. **This Constitution** — `.specify/memory/constitution.md`
2. **Product Requirements** — `docs/prd.md`
3. **Active Feature Spec** — `specs/NNN-*/spec.md`
4. **Active Implementation Plan** — `specs/NNN-*/plan.md`
5. **Active Task List** — `specs/NNN-*/tasks.md`
6. **Backlog Card** — `specs/_backlog/F0NN-*.md`
7. **Project Decisions Ledger** — `.squad/decisions.md`
8. **Agent Judgment** (lowest) — MUST be voiced in the PR description or in
   `.squad/decisions/inbox/`; never silently assumed.

If a lower document contradicts a higher one, stop and raise it through the
Amendment Process (§22) or — for non-constitutional artifacts — via
`.squad/decisions/inbox/`.

## Core Principles

### I. Clear Layered Architecture

The Go API MUST keep responsibilities separated:

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
- Dependencies MUST be explicit through constructor injection
  (`NewXxxHandler(repo, service)` pattern). Only `main.go` may import the
  `database` package.
- Multi-step writes MUST use transactions.
- Internal errors MUST NOT leak to clients. Log server-side; return
  generic messages to the caller.

**Rationale**: Enforced layer separation prevents coupling, enables
independent testing of each layer, and keeps the codebase navigable as
feature count grows.

### II. Service Boundary Separation

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
- AI agent pipelines MUST preserve tool-data provenance, use Pydantic
  schemas for worker outputs, and enforce a supervisor iteration limit.

**Rationale**: Hard service boundaries prevent accidental coupling
between AI logic and business logic, allow independent scaling, and
keep each codebase in its native language ecosystem.

### III. Strict Types and Explicit Contracts

All code MUST pass the strictest available type checking for its
language, and external-facing interfaces MUST have explicit schemas:

- **Go**: `go vet ./...` MUST pass with zero warnings.
- **TypeScript/Vue**: Docker builds use `vue-tsc --build`, which is
  stricter than local `vue-tsc --noEmit`. All code MUST pass the Docker
  check. Use `?.` (optional chaining) and `?? ''` / `?? 0` (nullish
  coalescing) for nullable props passed to non-nullable children.
- **Python**: `ruff check app/ tests/` MUST pass. All request/response
  schemas MUST use Pydantic models (`app/models/`).
- **Go API contracts**: Swagger annotations are required on all public
  handler methods.
- **Vue API access**: All API calls go through `src/web/src/api/client.ts`.

**Rationale**: Type strictness and explicit contracts catch bugs before
runtime and make service boundaries testable.

### IV. Simple Complete Changes

Every change MUST be simple, complete, and proportional.

- **Simple**: prefer direct, typed, human-readable code over clever
  abstractions or hidden mutation.
- **Complete**: fix the real user workflow and directly related sibling
  paths, not only the first observed failure.
- **Proportional**: keep small bugs and property changes small unless the
  investigation proves a broader root cause.
- Simplicity MUST NOT override architecture, security, typing,
  data-contract, or privacy requirements.

**Rationale**: The codebase should stay understandable to a human maintainer
while avoiding hopeful patches that leave the same bug in nearby paths.
Source: ADR 0005.

### V. Security, Auth, and Privacy by Default

Security-sensitive behavior MUST be explicit and safe by default:

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
- **Email**: Required for all new registrations. Legacy users without
  email see a dismissible modal (7-day snooze via `localStorage`).
  `GET /auth/me` includes `emailMissing` flag.
- **Registration**: Username + password + email. Validated format.
- **Social access**: accepted followers can view only allowed gallery
  data. Private coins, pricing/value, and AI analysis MUST NOT be exposed
  to followers.
- **Profile privacy**: setting `isPublic=false` permanently deletes
  followers.

**Rationale**: Security, authentication, and privacy rules prevent data
leakage, common attacks, and silent security degradation.
Source: `docs/security-principles.md`, `docs/threat-model.md`,
`docs/authentication.md`, `docs/social-feature.md`.

### VI. Consistent User Experience

The Vue frontend MUST preserve a consistent desktop and PWA/mobile
experience.

- Use the design token system from `variables.css` and global classes in
  `main.css`; do not hardcode visual values when a token exists.
- No emojis in UI text, prompts, or AI responses.
- Dark theme is the default and icons MUST use `lucide-vue-next`.
- The app MUST be PWA-compatible. Desktop layout changes MUST NOT break
  mobile/PWA layouts.
- Offline support requires the app shell to load without connectivity;
  API calls still require network.

**Rationale**: Consistent UI rules prevent visual fragmentation and keep
the app usable across desktop and mobile/PWA contexts.

### VII. CI, Supply Chain, and Release Integrity

Every change MUST preserve build, test, lint, and release integrity.

- Commits MUST use conventional prefixes: `feat:`, `fix:`, `docs:`,
  `refactor:`, `chore:`.
- AI-assisted commits MUST include the co-author trailer:
  `Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>`.
- Build automation uses Taskfile (`task --list` for all targets).
- GitHub Actions MUST pin action versions by SHA, not mutable tags.
- Docker base images SHOULD pin to specific digests for production
  builds.
- Branch protection MUST be enabled on `main`.

**Rationale**: Standardized workflow and supply-chain safeguards keep
changes reviewable, reproducible, and safe to release.

### VIII. Documented Decisions

Material design choices MUST be documented where future contributors can
find them.

- Constitution changes, service-boundary changes, security posture
  changes, new third-party services, and semantic data-model migrations
  require ADRs.
- Lower-authority artifacts MUST be updated when they conflict with this
  constitution.
- Agent judgment MUST be voiced in the PR description or in
  `.squad/decisions/inbox/`; never silently assumed.

**Rationale**: Durable decisions prevent drift and reduce reliance on chat
history or memory.

### IX. Automated Enforcement Over Manual Memory

Rules that can be enforced automatically SHOULD be enforced by tests, type
checks, linters, schemas, or CI.

- `architecture_test.go` validates Go package import rules.
- `go test ./...` MUST pass before any PR is merged.
- `ruff check` and `pytest` MUST pass for agent changes.
- Manual review should focus on judgment calls such as proportionality,
  clarity, and whether the real workflow was tested.

**Rationale**: Automated enforcement catches repeatable violations early;
reviewers should spend attention on decisions automation cannot judge.

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

## §17. Quality Gate

Every PR MUST pass the following checklist before merge. Items marked
"Phase 3" are not yet configured in CI — they become blocking when the
relevant tooling lands.

- [ ] `go vet ./...` clean
- [ ] `go test ./...` green (includes `architecture_test.go` from
      Principles I and IX)
- [ ] `vue-tsc --build` clean (Docker-equivalent strictness — see
      Principle III)
- [ ] `npm run build` green
- [ ] `ruff check app/ tests/` clean (when agent code is touched)
- [ ] `pytest tests/ -v` green (when agent code is touched)
- [ ] `gitleaks` scan clean *(Phase 3 — once `.gitleaks.toml` lands)*
- [ ] `trivy` container scan: no High/Critical
      *(Phase 3 — once `security-scan.yml` lands)*
- [ ] Conventional Commits format (see Principle VII)
- [ ] `Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>`
      trailer present when AI-assisted
- [ ] Constitution self-check noted in PR description — cite the
      relevant Principle or section
- [ ] Definition of Done checklist (§21) checked in the PR
- [ ] Principle IV self-check: change is simple, complete, and
      proportional

**Signed commits are NOT required.** This is a single-developer hobby
project; the Conventional Commits format and Co-authored-by trailer are
the workflow signals we rely on.

## §18. AI Agent Operating Rules

### 18.1 Always

- Read this constitution at session start.
- Check `.squad/decisions.md` for recent project-wide decisions.
- Check the active feature spec (`specs/NNN-*/spec.md`,
  `plan.md`, `tasks.md`) before writing code.
- Read your own agent charter (`.squad/agents/<name>/charter.md`) and
  recent `history.md` entries.
- Check `.copilot/skills/` for any skill that matches the task.
- Cite this constitution by Principle or section when making a design
  decision.
- Run the full Quality Gate (§17) before declaring a task done.
- Apply Principle IV: choose the simplest complete proportional change.

### 18.2 Never

- Invent facts, file paths, package names, APIs, or library symbols.
  When uncertain, read the file or run a search.
- Modify locked files: agent charters (`.squad/agents/*/charter.md`),
  append-only logs retroactively (`.squad/log/**`,
  `.squad/agents/*/history.md`, `.squad/decisions.md`,
  `.squad/orchestration-log/**`).
- Bypass a reviewer rejection (Strict Lockout — see
  `.squad/agents/brutus/charter.md`). A rejected PR is rejected until
  the reviewer explicitly clears it.
- Disable lint rules, weaken tests, or use `any` / `@ts-ignore` /
  `nolint` without an inline justification comment.
- Ship a hopeful patch that fixes only the first observed failure.
- Add clever abstractions or oversized rewrites for small bugs without
  proving a broader root cause.
- Commit secrets, `.env` files, or generated build artifacts.

### 18.3 Context Discipline

- Load only the files needed for the current task. Reference larger
  documents by path; do not paste their contents into chat unless
  required.
- Record cross-cutting decisions in `.squad/decisions/inbox/`, not in
  chat replies that will be lost.
- One feature per session — do not juggle multiple specs.
- Prefer `grep` / `glob` over reading whole files when looking for a
  symbol.

### 18.4 Drift Recovery

If work begins to diverge from the active spec:

1. **Stop** — do not silently re-scope.
2. Commit any working code as a WIP commit on the feature branch.
3. Write a current-state note to `.squad/decisions/inbox/` describing
   the drift and your proposed adjustment.
4. Wait for Lead (Maximus) acknowledgment before continuing.

### 18.5 Session Handoff

Session handoff is owned by **Scribe** via `.squad/log/` and per-agent
`.squad/agents/<name>/history.md`. Agents MUST NOT introduce
`SESSION-NOTES.md`, `.copilot-state.md`, or any other flat session-log
file — that pattern is explicitly superseded by the Squad ceremony
system documented in `.squad/ceremonies.md`.

## §19. Documentation Requirements

The following documents constitute the canonical documentation surface.
✅ exists today; ⏳ Phase 3 = scheduled deliverable, not yet present.

| Document | Path | Status | Owner |
|----------|------|--------|-------|
| Product Requirements (PRD) | `docs/prd.md` | ⏳ Phase 3 | Lead |
| Architecture overview | `docs/ARCHITECTURE.md` | ✅ exists | Lead |
| Software Design Document | `docs/SDD.md` | ✅ exists | Lead |
| Architecture Decision Records | `docs/adr/NNNN-*.md` | ✅ exists (0001–0005) | Lead |
| Security principles | `docs/security-principles.md` | ✅ exists | Lead |
| Threat model | `docs/threat-model.md` | ✅ exists | Lead |
| Incident response playbook | `docs/incident-response.md` | ✅ exists | Lead |
| Testing strategy | `docs/testing.md` | ✅ exists | Tester |
| External references index | `docs/references.md` | ✅ exists | Lead |
| API reference | `docs/api-reference.md` + `docs/openapi.json` | ✅ exists (generated via `task openapi`) | Backend |
| Authentication design | `docs/authentication.md` | ✅ exists | Backend |
| Deployment runbook | `docs/deployment.md` | ✅ exists | Lead |
| Getting started / onboarding | `docs/getting-started.md` | ✅ exists | Lead |
| Feature surface | `docs/features.md` | ✅ exists | Product |
| Testing strategy | `docs/testing.md` | ⏳ Phase 3 (extracted from copilot-instructions) | Lead |
| References / prior art | `docs/references.md` | ⏳ Phase 3 | Lead |
| Changelog | `docs/CHANGELOG.md` | ✅ exists | Lead |
| Operational runbooks | `docs/runbooks/` | ⏳ Phase 3 stretch | Lead |

ADRs use the Nygard format (Context / Decision / Status / Consequences).
ADR `0001` will retroactively record this constitution as the governing
contract when Phase 3 begins.

## §20. Audit & Continuous Improvement

### 20.1 Cadence

- **Weekly**: run `/audit` (the `speckit.analyze` prompt). Maximus
  drives the analysis; Brutus reviews findings. File issues for any
  High/Critical drift between constitution and code.
- **Per-release**: regenerate SBOM, re-review the threat model.
- **Quarterly**: PRD review — verify what we are building still matches
  the documented product intent.
- **Annually**: full dependency major-version review and restore drill.

### 20.2 Artifacts

- Audit reports are appended to `docs/audits/YYYY-MM-DD.md`
  (create the folder when the first audit runs).
- ADRs are preserved indefinitely in `docs/adr/`.
- Squad ceremony logs in `.squad/log/` provide institutional memory.

## §21. Definition of Done

A task is **done** only when every item below is true. Mirror this
checklist in the PR description.

1. **Code compiles**: `go build ./...`, `npm run build`, and
   `pip install -e ".[dev]"` succeed.
2. **Architecture tests green**: `go test -run TestArchitecture ./...`
   passes (Principles I and IX).
3. **Unit tests pass**: `go test ./...` and `pytest tests/` pass for
   any touched module.
4. **Type checks pass**: `vue-tsc --build` and Go's compiler are clean
   (Principle III).
5. **Linters clean**: `go vet ./...` and `ruff check app/ tests/` are
   clean.
6. **Test coverage**: every new service method has ≥ 1 unit test.
7. **Swagger**: every new or modified public handler has Swagger
   annotations (Principle III).
8. **API contract sync**: if the API surface changed, `swag` is
   regenerated AND the root `openapi.yaml` is updated (Phase 3).
9. **ADR**: if a material design choice was made, an ADR is added in
   `docs/adr/`.
10. **Tasks checked off**: the active `specs/NNN-*/tasks.md` items for
    this work are checked off.
11. **Decisions captured**: any cross-cutting decision is written to
    `.squad/decisions/inbox/`.
12. **Simple Complete Changes**: the change is simple, complete, and
    proportional (Principle IV).
13. **Secrets scan clean**: no credentials, tokens, or API keys in the
    diff.
14. **Commit hygiene**: Conventional Commit prefix and (when
    AI-assisted) `Co-authored-by: Copilot` trailer present.
15. **PR self-check**: PR description cites the relevant Constitution
    Principle(s) and lists this DoD as a checklist.

## §22. Amendment Process

This section supersedes the prior brief amendment language. Constitution
changes follow a deliberate, auditable process.

1. **Propose** — Open an ADR (`docs/adr/NNNN-*.md`) with status
   `PROPOSED` describing the change and rationale.
2. **PR** — Submit the constitution change PR alongside the ADR; the PR
   description MUST link the ADR.
3. **Semver bump** — Update the version in the file header and the
   Sync Impact Report:
   - **MAJOR**: a Principle is removed or renumbered; a
     backward-incompatible governance change; restructuring of
     operational sections.
   - **MINOR**: a Principle is added; a new operational section is
     added; existing guidance is materially expanded.
   - **PATCH**: typo, clarification, or non-semantic edit.
4. **Sync Impact Report** — Update the HTML-comment header at the top
   of this file (modified principles, added/removed sections, templates
   needing follow-up, TODOs).
5. **Revision History** — Append a row to §23.
6. **Announce** — On merge, announce the change in `.squad/decisions.md`.
7. **ADR status** — Transition the ADR from `PROPOSED` to `ACCEPTED`.

Automated enforcement (`architecture_test.go`, linters, type checkers)
is always preferred over manual review. Deviations from any Principle
MUST be explicitly justified in the PR description and tracked in the
plan's Complexity Tracking table.

## §23. Revision History

| Version | Date | Author | Summary | ADR |
|---------|------|--------|---------|-----|
| 1.0.0 | 2026-04-28 | Brian | Initial 10-principle constitution covering layered architecture, DI, service boundaries, typing, design tokens, AI isolation, schemas, commits, UI/UX, and architecture enforcement. | — |
| 1.1.0 | 2026-04-28 | Brian | Gap closure: added Principles XI–XVI (Security Hardening, Authentication & Token Policy, PWA/Mobile Rules, Social & Privacy, Supply Chain & CI, Account Lifecycle). | — |
| 2.0.0 | 2026-05-28 | Maximus (approved by Brian) | Added §0 Hierarchy of Authority, §17 Quality Gate, §18 AI Agent Operating Rules, §19 Documentation Requirements, §20 Audit & Continuous Improvement, §21 Definition of Done, §22 Amendment Process, §23 Revision History. All 16 Principles (I–XVI) preserved verbatim. | ADR 0001 (to be added in Phase 3) |
| 3.0.0 | 2026-06-09 | Brian | Consolidated 17 principles into 9 streamlined principles and made Simple Complete Changes Principle IV. | ADR 0005 |

**Version**: 3.0.0 | **Ratified**: 2026-04-28 | **Last Amended**: 2026-06-09
