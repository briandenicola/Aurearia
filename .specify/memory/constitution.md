<!--
  Sync Impact Report
  ==================
  Version change: 3.1.0 → 4.0.0 (MAJOR — Accepted ADR 0019)
  Activation: owner approved and merged PR #734 into beta on 2026-09-21;
    merge commit ca85f7830349a4472cb5409c082b9485e7e34c44.
  Modified principles: IX (gate applicability references section 17)
  Added sections: None
  Removed sections: None
  Modified operational sections:
    - §0: accepted ADR authority and explicit work selection
    - §17–§21: proportional validation, current views, bounded agents,
      evidence-backed acceptance, and separate software/delivery audits
    - §22–§23: amendment activation and proposed revision record
  Synchronized consumers:
    - .github/copilot-instructions.md and .github/pull_request_template.md
    - .github/agents/ and .github/prompts/
    - .specify/templates/
    - .squad/routing.md, ceremonies.md, and agent charters
    - CONTRIBUTING.md, docs/testing.md, docs/adr/README.md
  Follow-up TODOs:
    - P2 archival, P3 executable checks, P4 native integration, P5 live controls
    - Historical review/spec lifecycle reconciliation remains evidence-gated
    - See ADR 0019's P1 consumer inventory and explicit context-budget exceptions
-->

# Ancient Coins Constitution

> **This document is the non-negotiable contract for how this project is built.**
> Every AI agent session must read this file first. Every PR must comply.
> Deviations require an explicit, documented waiver (ADR) under §22.

**Project**: Ancient Coins (self-hosted personal collection PWA)
**Version**: 4.0.0
**Ratified**: 2026-04-28
**Last Amended**: 2026-09-21
**Amendment Prepared**: 2026-09-21 — ADR 0019

> **Effective:** The owner approved and merged amendment PR #734 into `beta`
> on 2026-09-21 at `ca85f7830349a4472cb5409c082b9485e7e34c44`.
> ADR 0019 is Accepted and constitution 4.0.0 governs new work.
> This acceptance does not itself authorize P2 execution, tool installations,
> deployments, releases, or clearance of historical reviewer blocks.
> The previous 3.1.0 text is available from baseline commit
> `cacc177c2ecd85d62899507413184cfc395a7f6a` at this path.

## §0. Hierarchy of Authority

When two artifacts conflict, the higher-authority document wins. Lower-authority
artifacts MUST be updated to match within the same PR, or the conflict MUST be
escalated to amend the higher artifact (see §22).

Ordered list of governing artifacts, highest authority first:

1. **This Constitution** — `.specify/memory/constitution.md`
2. **Product Requirements** — `docs/prd.md`
3. **Applicable Accepted ADRs** — `docs/adr/`; bounded by their recorded scope
4. **Active Feature Spec** — `specs/NNN-*/spec.md`
5. **Active Implementation Plan** — `specs/NNN-*/plan.md`
6. **Active Task List** — `specs/NNN-*/tasks.md`
7. **Backlog Card** — `specs/_backlog/F0NN-*.md`
8. **Active Decisions** — `.squad/decisions.md`
9. **Agent Judgment** (lowest) — MUST be voiced in the PR description or in
   `.squad/decisions/inbox/`; never silently assumed.

If a lower document contradicts a higher one, stop and raise it through the
Amendment Process (§22) or — for non-constitutional artifacts — via
`.squad/decisions/inbox/`.

Framework instructions, charters, templates, skills, and session state implement
this hierarchy; none may override it. An ADR changes the constitution only
through §22. Conflicting peer ADRs require explicit supersession or owner
resolution, not a newest-file rule. Proposed records are not accepted policy.

Select work explicitly from the owner's request or a validated current-work
pointer. Ordinary development stays on `beta`; do not infer a feature from the
branch or highest-numbered spec. A bounded issue or approved process plan may
authorize a small repair or governance change without a new feature spec.
Owner-approved scope changes must update the authorizing artifact before work;
they do not implicitly waive architecture, security, or validation requirements.

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
- Applicable Go changes MUST pass `go test ./...` before acceptance or merge;
  validation scope and release checks are defined in §17.
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

See `docs/testing.md` §6 for the current CI-aligned recipes and Windows
invocation notes. Existing Taskfile targets are conveniences, not proof that
every gate ran. Shared executable gate targets are a pending P3 deliverable;
do not invoke a proposed command or silently skip a missing tool.

## §17. Quality Gate

Use three validation tiers:

1. **Fast feedback:** targeted checks while implementing.
2. **Completion:** all checks applicable to the affected layers, shared contracts,
   and sibling workflows, with exact-path regression evidence.
3. **CI/release:** all applicable configured jobs and full release checks on the
   candidate commit, including runner-only checks.

| Change surface | Required completion evidence |
|----------------|------------------------------|
| Go API | Build, vet, full package tests including architecture; contract/OpenAPI consistency when applicable |
| Vue | Zero-warning lint, strict type check, full tests, production build; affected browser/mobile workflows |
| Python agent | Locked-environment Ruff and full tests; affected cross-service contracts |
| Shared contracts/migrations | Producer/consumer compatibility, sibling workflows, and representative upgrade/recovery evidence |
| Pure documentation | Document/link/consistency review and existing documentation checks; no application builds |
| Executable prompts/policy/workflows/checks | Relevant behavior or fixture evidence; not automatically docs-only because the file is Markdown |

Current executable recipes are in `docs/testing.md` §6 and the workflow/package
definitions it cites. P3 will consolidate shared entry points without weakening
coverage. Preserve Go race detection, security scans, container checks, and
compatibility jobs. A runner-only check stays pending until its matching CI
evidence exists.

Every change MUST identify its lane (§18.1), acceptance criteria, affected
workflows, applicable checks, and any approved exceptions. Missing tools,
unauthorized execution, or unavailable evidence mean **incomplete**, not passed.
Dependency setup, builds, containers, and other machine operations require the
owner authorization applicable to the session; do not install implicitly.

Before acceptance, verify conventional commit hygiene, required AI co-author
trailers, absence of secrets, constitution citations, and §21. Test results must
identify the tested commit or dirty-tree identity. Changed implementation
invalidates applicable prior verification and review. Green CI does not prove a
workflow it never exercised, and a beta validation push is not release approval.

**Signed commits are NOT required.** This is a single-developer hobby
project; the Conventional Commits format and Co-authored-by trailer are
the workflow signals we rely on.

## §18. AI Agent Operating Rules

### 18.1 Always

- Read this constitution, relevant active decisions, and the explicit work
  artifact. Load a charter/history only when acting in that role.
- Choose the smallest sufficient lane:
  - **Bug/small change:** reproduction, cause, expected outcome, non-goals,
    sibling paths, regression evidence, and review.
  - **Feature:** approved outcome/non-goals, criteria, short plan, bounded tasks,
    usable-slice evidence, and review.
  - **High risk:** either lane plus the required ADR, compatibility/recovery
    evidence, independent review, and explicit owner approval.
- Treat auth, data semantics, migrations, service boundaries, external providers,
  and delivery-control changes as high risk regardless of diff size.
- Discover relevant skills in the supported native locations; until P4 migration,
  also inspect applicable `.squad/skills/` entries. Do not claim native discovery
  of a wrapper-specific skill.
- Cite the governing Principle/section; apply §17 before claiming verification.
- Default to one implementation owner. Any delegated work needs a bounded
  objective, allowed paths, evidence, checkpoint/effort lease, and stop condition.
  No speculative fan-out or nested delegation without approval.
- Keep mandatory review independent of the implementation author.

### 18.2 Never

- Invent facts, file paths, package names, APIs, or library symbols.
  When uncertain, read the file or run a search.
- Change the constitution, locked charters, or landed specifications without
  explicit amendment authority. Accepted ADR bodies, session logs, orchestration
  logs, and archived evidence are immutable.
- Bypass a reviewer rejection. For new reviews after ADR 0019 acceptance, the
  original author may normally repair rejected work; the reviewer may require an
  independent revision owner. Only that reviewer clears the block. If unavailable,
  the owner may explicitly appoint an independent successor, who must re-review.
  Reassignment is not clearance. Existing blocks and author restrictions retain
  their original terms until explicitly resolved.
- Disable lint rules, weaken tests, or use `any` / `@ts-ignore` /
  `nolint` without an inline justification comment.
- Ship a hopeful patch that fixes only the first observed failure.
- Add clever abstractions or oversized rewrites for small bugs without
  proving a broader root cause.
- Commit secrets, `.env` files, or generated build artifacts.
- Treat changed files, an empty agent response, or checked tasks as success.
- Automatically stage unrelated files, commit, stash, push, install, or deploy
  as a side effect of a handoff or audit.

### 18.3 Context Discipline

- Load only the files needed for the current task. Reference larger
  documents by path; do not paste their contents into chat unless
  required.
- Record cross-cutting decisions in `.squad/decisions/inbox/`, not in
  chat replies that will be lost.
- One feature per session — do not juggle multiple specs.
- Prefer `grep` / `glob` over reading whole files when looking for a
  symbol.

After ADR 0019 acceptance, current decisions/history views may be curated by the
owner or an explicitly delegated maintainer. First preserve original material
in append-only archives with an inventory and checksum evidence; link the
current decision to its source/supersession. Do not delete a pending block or
change the meaning of historical evidence while summarizing.

Maintenance warning budgets: 150 lines of automatic repository instructions,
20 KiB active decisions, 12 KiB per-agent history, 100 lines of current-work
pointer, and 6,000 words of initial policy/context reading excluding selected
feature artifacts. Consolidate or record an exception; never truncate binding
constraints to meet a budget.

Use `.squad/identity/now.md` as a pointer to the authoritative issue/spec/tasks,
review blocks, evidence, and next action, not a duplicate task ledger.

### 18.4 Drift Recovery

If work begins to diverge from the active spec:

1. **Stop** — do not silently re-scope.
2. Preserve unrelated work and report the affected files; do not automatically
   commit or stash.
3. Write an authorized current-state note to `.squad/decisions/inbox/` describing
   the drift and your proposed adjustment.
4. Wait for owner-approved scope/authority changes before continuing. An agent
   lead may advise but cannot invent product authorization.

### 18.5 Session Handoff

The implementation owner is responsible for persisting handoff evidence via
`.squad/log/` and the current-work pointer; Scribe is optional delegated help.
Record completed/incomplete work, relevant evidence identity, decisions,
unresolved blocks, and the exact next action. Wait for required recording to
finish before declaring the batch complete. Commit only explicitly approved
paths when authorized. Agents MUST NOT introduce
`SESSION-NOTES.md`, `.copilot-state.md`, or any other flat session-log
file as an alternative source of task state.

## §19. Documentation Requirements

The following documents constitute the canonical documentation surface.
Keep current metadata accurate; do not rewrite accepted historical bodies.

| Document | Path | Status | Owner |
|----------|------|--------|-------|
| Product Requirements (PRD) | `docs/prd.md` | ✅ exists | Lead |
| Architecture overview | `docs/ARCHITECTURE.md` | ✅ exists | Lead |
| Software Design Document | `docs/SDD.md` | ✅ exists | Lead |
| Architecture Decision Records | `docs/adr/NNNN-*.md` | See `docs/adr/README.md` | Lead |
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
| Changelog | `docs/CHANGELOG.md` | ✅ exists | Lead |

ADRs use the Nygard format (Context / Decision / Status / Consequences).
ADR 0001 records the decision lifecycle. An index mirrors source status; a
merged implementation or completed task list cannot silently promote a Proposed
ADR. Document unresolved lifecycle conflicts without inferring acceptance.

## §20. Audit & Continuous Improvement

### 20.1 Cadence

- **Major/high-risk work and release readiness**: run the software quality audit
  against the exact changeset and approved requirements.
- **After each owner-designated major release**: run the separate agentic-delivery
  audit, including the disposition of previous findings. Record invocation and
  evidence in release closeout; a skill definition alone is not an automatic trigger.
- **On detected drift or recurring failure**: investigate the concrete issue.
  Use deterministic CI checks rather than mandatory weekly broad agent ceremonies.
- **Per-release**: regenerate SBOM, re-review the threat model.
- **Quarterly**: PRD review — verify what we are building still matches
  the documented product intent.
- **Annually**: full dependency major-version review and restore drill.

### 20.2 Artifacts

- Audit reports are appended to `docs/audits/YYYY-MM-DD.md`
  (create the folder when the first audit runs).
- ADRs are preserved indefinitely in `docs/adr/`.
- Squad ceremony logs in `.squad/log/` provide institutional memory.
- Audits are read-only by default; save reports or create issues only when
  authorized. A finding closes with corrective-action evidence, not a written
  retrospective alone. An audit verdict is not release authorization.

## §21. Definition of Done

A task is **done** only when applicable evidence below is present. Mark an item
N/A only with a reason; missing execution is not N/A. Mirror the disposition in
the PR or approved work record.

| State | Meaning |
|-------|---------|
| Implemented | Scoped work exists; verification may be pending |
| Verified | Applicable gates and acceptance paths have evidence |
| Accepted | Independent review blocks are cleared and required owner approval exists |
| Released | The owner-authorized candidate was promoted/published; deployment is recorded separately |

1. **Scope and builds**: approved lane/non-goals are recorded and applicable
   builds pass per §17. Installing dependencies is not a compilation check.
2. **Architecture tests green**: applicable architecture/contract tests pass;
   use actual test names, not an assumed selector that might run zero tests.
3. **Tests pass**: applicable Go, full web package, and locked-environment agent
   suites pass using the recipes in `docs/testing.md` §6.
4. **Type checks pass**: `vue-tsc --build` and Go's compiler are clean
   (Principle III).
5. **Linters clean**: applicable Go vet, zero-warning frontend lint, and
   locked-environment Ruff are clean.
6. **Regression coverage**: every bug fix includes a targeted regression
   test for the exact failing user path or a documented reason automation
   is deferred.
7. **Workflow contracts**: if a change touches shared forms, settings,
   validation, API DTOs, collection counts, wishlist/sold flags, set
   membership, AI intake, or other shared workflow surfaces, the PR lists
   the affected sibling workflows and proves the relevant contract with
   automated tests where practical.
8. **Config contracts**: any value a user/admin can configure in the UI
   MUST be accepted by every API path the UI can submit it to, or the UI
   MUST prevent the invalid submission with an explicit message.
9. **Test coverage**: every new service method has ≥ 1 unit test.
10. **Swagger**: every new or modified public handler has Swagger
   annotations (Principle III).
11. **API contract sync**: if the API surface changed, generated Go Swagger
   artifacts and `docs/openapi.json` are synchronized using `task openapi`.
12. **ADR**: if a material design choice was made, an ADR is added in
   `docs/adr/`.
13. **Task state reconciled**: the active task list/issue reflects implemented,
    verified, accepted, and deferred work without treating checkboxes as proof.
14. **Decisions captured**: any cross-cutting decision is written to
    `.squad/decisions/inbox/`.
15. **Simple Complete Changes**: the change is simple, complete, and
    proportional (Principle IV).
16. **Secrets scan clean**: no credentials, tokens, or API keys in the
    diff.
17. **Commit hygiene**: Conventional Commit prefix and (when
    AI-assisted) `Co-authored-by: Copilot` trailer present.
18. **Acceptance and release evidence**: the PR/work record cites the relevant
    Principles and DoD, tested/reviewed commit or tree, unresolved exceptions,
    and reviewer verdict. Main/release actions need explicit owner authorization
    for the candidate; successful CI or an agent-authored checkbox is insufficient.

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

Owner approval to prepare local changes is not approval to publish or accept
them. Keep the amendment marked Proposed until the required PR is approved and
merged. Apply prospective operational changes only after acceptance; preserve
existing reviewer restrictions. Synchronize active instruction consumers in the
same amendment, and record separately staged automation as pending rather than
claiming it already enforces policy.

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
| 3.1.0 | 2026-06-11 | Brian | Added workflow-contract, blast-radius, configurable-value, and exact regression coverage gates to reduce repeated user-flow regressions. | ADR 0006 |
| 4.0.0 | 2026-09-21 | Copilot; approved and merged by owner in PR #734 | Reconciles authority, proportional gates, current views, bounded agents, review restrictions, evidence-backed completion, and audit cadence. Accepted at ca85f7830349a4472cb5409c082b9485e7e34c44; historical review restrictions remain unchanged. | ADR 0019 |

**Version**: 4.0.0 | **Ratified**: 2026-04-28 | **Last Amended**: 2026-09-21
