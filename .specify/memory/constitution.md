# Aurearia Constitution

**Version**: 6.0.0 | **Ratified**: 2026-04-28 | **Last Amended**: 2026-11-11

These are the engineering rules for Aurearia, a self-hosted personal coin
collection app: a Go API, a Vue/TypeScript PWA and a stateless Python agent
service. The owner decides scope and merges. Everything here exists to keep
the code correct, safe and consistent, not to generate process.

## Core Principles

### I. Clear Layered Architecture

Go follows Handler → Service → Repository → Database. Handlers parse/delegate/
respond, with no business logic or raw SQL. HTTP-agnostic services own business
logic; repositories own GORM queries and reuse `repository/scopes.go`. Models
import only the standard library. Constructor injection is explicit; only
`main.go` may import `database`. Multi-step writes use transactions. Log
internal errors server-side; return generic errors to clients.

### II. Service Boundary Separation

Go owns auth, durable state, database access and SSE proxying, with zero LLM
logic. Python remains stateless with no direct database access; keys, prompts
and context arrive per request. Vue calls only Go `/api/*`, not Python. SSE
flows Python → Go → Vue. AI pipelines preserve tool-data provenance, Pydantic
worker schemas and supervisor iteration limits.

### III. Strict Types and Explicit Contracts

Keep Go compilation/vet, zero-warning frontend lint, strict `vue-tsc --build`
(not the weaker `--noEmit`), and locked-environment Ruff clean. Do not mask nullable
props; use typed optional/nullish handling. Python requests/responses use Pydantic.
Public Go handlers require Swagger annotations and synchronized generated Go
Swagger/OpenAPI snapshots. Vue API access uses `src/web/src/api/client.ts`.

### IV. Simple Complete Changes

Prefer direct typed code and existing patterns. Fix the real workflow and
directly related sibling paths, not only the first failure. Keep scope proportional
unless evidence proves a broader cause. Simplicity never waives architecture,
security, privacy, typing or data-contract safeguards.

### V. Security, Auth, and Privacy by Default

- Whitelist CORS origins; never allow all origins in production. Parameterize SQL
  or validate against allowlists. Validate upload allowlisted extensions
  (`.jpg`, `.jpeg`, `.png`, `.gif`, `.webp`) and magic-byte MIME, not headers alone.
- Rate-limit auth; cap multipart and JSON bodies (recommended 10 MB / 1 MB).
  Do not expose internal errors. Production containers run non-root.
- JWT access expiry is 15 minutes with `JWT_SECRET`; production should refuse an
  unset/low-entropy secret. Refresh tokens have 30-day rolling lifetime,
  `rt_` + 32 random hex bytes, SHA-256-only storage and one-time rotation/revocation.
  Client localStorage and concurrent 401 refresh/replay remain the accepted model.
- WebAuthn uses platform authenticators, configured production RP ID/origin and
  challenge TTL (recommended five minutes). API keys are hashed and revocable.
  First registered user becomes admin. New registration requires validated
  username/password/email; legacy missing-email handling retains `/auth/me`
  `emailMissing` and dismissible seven-day localStorage snooze.
- Accepted followers see only allowed gallery data, never private coins,
  price/value or AI analysis. `isPublic=false` permanently deletes followers.
  Preserve ownership checks and privacy/data-safety failure paths.

### VI. Consistent User Experience

Reuse design tokens (`variables.css`), global classes (`main.css`), local
components and navigation. Dark theme and `lucide-vue-next` are defaults.
No emojis in UI, prompts or AI responses. Desktop changes preserve mobile/PWA;
the offline app shell loads without connectivity, while API calls need network.

### VII. Commits, CI and Releases

Use Conventional Commits (`feat`, `fix`, `docs`, `refactor`, `chore`). Pin
GitHub Actions by SHA. CI runs each area's tests once, on the pull request into
`beta`; merging builds the beta images, and promoting `beta` to `main` re-tags
those same images. Security scans and CodeQL run weekly. Do not add CI jobs,
gates or verifiers unless they catch a class of bug the existing tests cannot.

### VIII. Decisions

Write an ADR only for a decision that is hard to reverse: a service or security
boundary, a significant new dependency, a data migration or a framework choice.
Everything else is explained in the PR description.

### IX. Tests Over Ceremony

Prefer tests, types, lint and architecture tests (`architecture_test.go`) to
written rules and checklists. A rule that matters enough to repeat should
become a test.

## Working Rules

- **Test what you change.** A bug fix comes with a test that fails without the
  fix. New behavior comes with tests for its main path and its error paths.
- **Never weaken a check to pass it.** No skipped or deleted tests, lint
  suppression, `any`, `@ts-ignore` or `nolint` to get green. If a check is
  wrong, fix or remove it openly and say why.
- **Say only what you ran.** Report the commands you actually ran and their
  results. Anything you could not run is "not verified", never "passed".
- **Keep changes proportional.** Fix the real cause and its sibling code paths;
  do not widen scope or add process the change does not need.
- **The owner decides.** Agents do not merge, release, deploy or change
  repository settings unless the owner asks for that specific action.
- **Keep docs true.** When a change makes a doc, instruction or comment wrong,
  update it in the same change. `docs/testing.md` is the command reference.

## Changing This Document

Edit it in a normal PR that explains why, and bump the version: major for a
removed or renumbered principle, minor for an added rule, patch for wording.

## Revision History

| Version | Date | Summary |
|---------|------|---------|
| 1.0.0 – 5.0.0 | 2026-04-28 – 2026-10-19 | Earlier versions, with full governance process. See git history. |
| 6.0.0 | 2026-11-11 | Kept Principles I–VI. Shortened VII–IX to match the simplified CI. Replaced §0 and §17–§23 (authority hierarchy, review lanes, evidence ledgers, amendment ceremony) with six working rules. |
