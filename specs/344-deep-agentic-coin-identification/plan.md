# Implementation Plan: Deep Agentic Coin Identification

**Branch**: `344-deep-agentic-coin-identification` | **Date**: 2026-08-15 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/344-deep-agentic-coin-identification/spec.md`

**Companion artifacts**: [research.md](./research.md) · [data-model.md](./data-model.md) ·
[quickstart.md](./quickstart.md) ·
[contracts/deep-identification.openapi.yaml](./contracts/deep-identification.openapi.yaml) ·
[contracts/sse-events.md](./contracts/sse-events.md) ·
[contracts/agent-internal-contract.md](./contracts/agent-internal-contract.md)

## Summary

Add an explicit, opt-in **Deep Analysis** path beside the existing synchronous
Identify Coin lookup (`POST /api/coins/lookup`, unchanged and still the
default). Deep Analysis is a persisted, owner-scoped background job
(`DeepIdentificationJob`) with an append-only, replayable event log
(`DeepIdentificationEvent`) — a **new sibling domain**, not a mutation of the
coin-bound `models.AIJob`.

Go owns everything stateful: auth, job/event persistence, image artifacts, SSE
replay, provider HTTP calls (reusing the shipped Numista/Nomisma clients), and
confirmed writes. Python remains stateless and owns only inference: a bounded
LangGraph pipeline (`prepare_evidence → router → bounded provider fan-out →
contradiction/provenance evaluator → typed synthesis`) streaming typed events
back to Go over an authenticated internal SSE call.

Providers at MVP: **Nomisma** and **Numista** automated (via existing Go
clients exposed to Python as internal tools); **NGC** is OCR cert extraction +
link-out only (its Terms of Use prohibit automated access); **OCRE** and **RPC
Online** ship as `not_automated`/`unavailable` behind explicit validation gates.
No provider is ever faked as "no match".

Output is a narrative report with citations, typed proposed coin fields with
per-field confidence/evidence, disagreements, unresolved questions, provider
coverage, and a partial-success marker. Nothing is ever auto-written: new-intake
confirmation seeds the existing Quick Capture draft/promote path, saved-coin
confirmation flows through the existing coin-edit write path.

## Technical Context

**Language/Version**: Go 1.26.1 (Gin, GORM, SQLite) · TypeScript/Vue 3 (Vite, Pinia, PWA) · Python 3.12 (FastAPI, LangGraph, LangChain, Pydantic v2)
**Primary Dependencies**: existing only — `gin`, `gorm`, `lucide-vue-next`, `axios`, `langgraph`, `langchain-anthropic`/`langchain-ollama`, `httpx`, `tenacity`, `pydantic-settings`. **No new third-party dependency is introduced by this feature.**
**Storage**: SQLite via GORM; additive `AutoMigrate` in `src/api/database/database.go:36`; image artifacts on disk under `<UploadDir>/deep-jobs/job-<id>/`
**Testing**: `go test ./...` (incl. `architecture_test.go`, `route_openapi_drift_test.go`, `database/migration_test.go`), `pytest` + `ruff` (`src/agent`), `vitest` + `vue-tsc --build` (`src/web`), Playwright specs in `src/web/e2e/workflows/`
**Target Platform**: self-hosted single-node Docker deployment (Go API + Vue static + Python agent), desktop + mobile PWA client
**Project Type**: three-service web application (`src/api`, `src/web`, `src/agent`) — Constitution Principle II
**Performance Goals**: job accepted in < 1 s (SC-001); 95% of runs terminal within 5 min (SC-002, budget table in research.md §9); zero measurable regression on `POST /api/coins/lookup` (SC-008)
**Constraints**: ≤ 4 providers/job, ≤ 2 concurrent provider calls, 300 s hard ceiling; Python stateless with no DB and no provider API keys; hint images deleted on every terminal state; no coin write without explicit owner confirmation; owner-scoped everything
**Scale/Scope**: single primary user + invited friends; expected ≤ tens of deep jobs/day; 4 new Go models, ~6 new Go files, 1 new Python package + 1 route, 1 new Vue page + ~6 components/composables

**NEEDS CLARIFICATION**: none remaining. Ten open technical unknowns were
identified while drafting this Technical Context and all ten are resolved in
[research.md §12](./research.md) (provider execution boundary, SSE resumability,
event retention, cancellation propagation, draft bridging, MVP provider set,
5-minute budget allocation, image transport/cleanup, frontend streaming
approach, EXIF stance).

## Constitution Check

*GATE: evaluated before Phase 0, re-evaluated after Phase 1 design (see
"Post-Design Constitution Check" below). Constitution v3.1.0.*

| Principle / Section | Requirement | Plan compliance |
|---|---|---|
| **I. Clear Layered Architecture** | Handler → Service → Repository → Database; models stdlib-only; DI via constructors; transactions for multi-step writes | New `handlers/deep_identification.go` (thin), `services/deep_identification_service.go` (business logic, HTTP-agnostic), `repository/deep_identification_repository.go` (all GORM), `models/deep_identification_*.go` (stdlib only). Wiring in `main.go` follows the `repo → service → handler` pattern at `main.go:246-249`. Terminal settle + event append run in one repository transaction. **PASS** |
| **II. Service Boundary Separation** | Go holds zero LLM logic; Python stateless, no DB; Vue talks only to `/api/*`; SSE flows Python → Go → Vue; Pydantic schemas; supervisor iteration limit | Router/evaluator/synthesis LLM logic lives only in `src/agent/app/teams/deep_identification/`. Python gets all context per request and holds no DB handle and no provider keys (research.md §4). Vue calls only Go. SSE still flows Python → Go → Vue, with Go persisting before re-serving (documented deviation, see Complexity Tracking). `recursion_limit` bound set explicitly. **PASS with one recorded deviation** |
| **III. Strict Types and Explicit Contracts** | `go vet` clean; `vue-tsc --build` clean; `ruff` clean; Pydantic models for all agent schemas; swaggo annotations on public handlers; Vue API access only via `client.ts` | All request/response DTOs typed in Go and mirrored as `StrictRequestModel`/response models in `app/models/`. Every new handler gets swaggo annotations (`route_openapi_drift_test.go` enforces). All new frontend calls added to `src/web/src/api/client.ts` with types in `src/web/src/types/index.ts`. **PASS** |
| **IV. Simple Complete Changes** | Simple, complete, proportional; no clever abstractions; fix sibling paths | New sibling job domain rather than a risky retrofit of `AIJob` (research.md §3). **No third coin-write path** — confirmation reuses `QuickCaptureService.PromoteDraft` and `CoinService.UpdateCoinWithFields` (research.md §8). No new dependencies, no generic job framework. Sibling workflows explicitly covered: fast lookup, quick capture promote, coin edit, AI job polling. **PASS** |
| **V. Security, Auth, and Privacy by Default** | Upload allowlist + magic bytes; body/multipart caps; rate limits; owner scoping; no internal error leakage; no secrets in logs | Reuses `services.ValidateImageData`/`NormalizeImageExt`/`MaxImageUploadBytes`; start/cancel/retry/apply behind `writeRateLimit`; every query owner-scoped, non-owner ⇒ 404; typed `failureCode` + generic `failureMessage`; owner notes / hint-derived context / provider query strings never logged (FR-036); provider API keys never leave Go; Python receives only a short-lived minted token. **PASS** |
| **VI. Consistent User Experience** | Design tokens, no emojis, dark default, `lucide-vue-next`, PWA-safe, desktop change must not break mobile | Deep UI built from `variables.css` tokens and existing global classes; icons from `lucide-vue-next`; mobile-first capture reuses `InlineCameraCapturePanel`/`QuickCaptureImageSlots`; new page decomposed into components to stay well under the de-facto 500–700 line page ceiling. **PASS** |
| **VII. CI, Supply Chain, and Release Integrity** | Conventional Commits + Copilot trailer; Taskfile; CI gates green | Work sequenced so each phase leaves `task test`, `npm run build`, `ruff`, `pytest` green; `task openapi` re-run whenever handlers change (CI diffs `docs/openapi.json`). **PASS** |
| **VIII. Documented Decisions** | ADRs for service-boundary/security/new third-party/semantic data-model changes | **ADR 0010 required** (deep identification job/event domain, Go-owned persisted SSE, provider-calls-in-Go boundary, provider staging & licensing). A further ADR is deferred until gate G-OCRE actually opens. **PASS (ADR planned)** |
| **IX. Automated Enforcement** | `architecture_test.go`, `go test`, `ruff`, `pytest` | Layering enforced by the existing architecture test; migration additivity by a new `TestDeepIdentificationModelsAutoMigrate`; route/Swagger sync by `route_openapi_drift_test.go`; provider-status honesty and citation validation are unit-tested invariants. **PASS** |
| **§17 Quality Gate** | Workflow-contract check: identify workflows, shared contracts, targeted regression tests | Workflows touched: Identify Coin (fast), Quick Capture draft/promote, coin edit, coin images, AI job UI neighbourhood. Targeted regression tests listed under Testing Strategy, including an explicit fast-path no-regression test. **PASS** |
| **§21 Definition of Done** | Builds, arch tests, unit tests, type checks, linters, regression coverage, workflow/config contracts, ≥1 test per new service method, Swagger, OpenAPI sync, ADR, decisions captured | All enumerated in the phase plan; every new service method gets ≥ 1 unit test; `task openapi` and ADR 0010 are explicit deliverables; a cross-cutting decisions note goes to `.squad/decisions/inbox/`. **PASS** |
| **PRD alignment** (`docs/prd.md`) | Non-goals: no catalogue replication, no forensic authentication, no bulk import | Provider access is on-demand and per-job only; the report links out rather than mirroring catalogues; output is explicitly a proposal for owner review, not an appraisal or authentication verdict. **PASS** |

**Initial gate result: PASS** (one documented deviation, tracked in Complexity
Tracking; no unjustified violation).

## Project Structure

### Documentation (this feature)

```text
specs/344-deep-agentic-coin-identification/
├── plan.md                                   # This file
├── research.md                               # Phase 0 output
├── data-model.md                             # Phase 1 output
├── quickstart.md                             # Phase 1 output
├── contracts/
│   ├── deep-identification.openapi.yaml      # REST contract (Vue ↔ Go)
│   ├── sse-events.md                         # SSE envelope + replay contract (Go → Vue)
│   └── agent-internal-contract.md            # Go ↔ Python + provider tool contract
├── checklists/requirements.md                # Existing spec-quality checklist
└── tasks.md                                  # NOT created by /speckit.plan
```

### Source Code (repository root)

```text
src/api/                                       # Go API — owns state, auth, SSE, writes
├── models/
│   ├── deep_identification_job.go             # NEW
│   ├── deep_identification_event.go           # NEW
│   ├── deep_identification_provider_run.go    # NEW
│   └── deep_identification_artifact.go        # NEW
├── repository/
│   └── deep_identification_repository.go      # NEW (all GORM for the domain)
├── services/
│   ├── deep_identification_service.go         # NEW (worker pool, state machine, cleanup)
│   ├── deep_identification_broker.go          # NEW (in-process SSE fan-out + cancel registry)
│   ├── deep_identification_proposal.go        # NEW (field allowlist, apply via existing write paths)
│   ├── deep_identification_provider_tools.go  # NEW (Go-side provider tool operations)
│   ├── agent_proxy.go                         # CHANGED (+ StreamDeepIdentification callback consumer)
│   ├── coin_lookup_service.go                 # UNCHANGED (reused for quick evidence)
│   ├── numista_client.go / numista_cache.go   # UNCHANGED (reused behind internal tools)
│   ├── nomisma_client.go / nomisma_cache.go   # UNCHANGED (reused behind internal tools)
│   ├── image_service.go                       # CHANGED (job-scoped artifact save/delete helpers)
│   ├── quick_capture_service.go               # UNCHANGED (draft seed + promote)
│   ├── coin_service.go                        # UNCHANGED (UpdateCoinWithFields)
│   └── settings_service.go                    # CHANGED (+ SettingDeepIdentification* keys)
├── handlers/
│   ├── deep_identification.go                 # NEW (create/get/list/cancel/retry/proposal/apply/SSE)
│   └── internal_tools.go                      # CHANGED (+ numista_search/detail, nomisma_search)
├── database/database.go                       # CHANGED (AutoMigrate additions)
├── database/migration_test.go                 # CHANGED (+ TestDeepIdentificationModelsAutoMigrate)
└── main.go                                    # CHANGED (wiring, routes, StartWorkers, janitor)

src/agent/                                     # Python — stateless inference only
├── app/routes.py                              # CHANGED (+ POST /api/deep-identify/stream)
├── app/models/requests.py                     # CHANGED (+ DeepIdentifyRequest family)
├── app/models/responses.py                    # CHANGED (+ ProviderEvidence, DeepSynthesis)
├── app/teams/deep_identification/             # NEW package
│   ├── graph.py  state.py  router.py  evaluator.py  synthesis.py  merge.py
│   └── providers/{nomisma.py,numista.py,ngc.py,ocre.py,rpc.py}
├── app/tools/provider_tools.py                # NEW (Go internal tool client, mirrors collection_tools.py)
└── app/config.py                              # CHANGED (+ AGENT_DEEP_* bounds)

src/web/                                       # Vue SPA
├── src/pages/CoinLookupPage.vue               # CHANGED (one secondary "Deep Analysis" CTA only)
├── src/pages/DeepAnalysisPage.vue             # NEW (run + review shell, thin)
├── src/components/deep-identification/        # NEW
│   ├── DeepAnalysisStartPanel.vue             # roles, notes, hint images, provider override
│   ├── DeepAnalysisProgressTimeline.vue       # event timeline + connection indicator
│   ├── DeepProviderCoverageList.vue           # per-provider status + attribution/link-out
│   ├── DeepReportPanel.vue                    # narrative, disagreements, citations
│   ├── DeepProposalEditor.vue                 # per-field edit/accept, AI-vs-owner diff
│   └── DeepAnalysisEntryButton.vue            # shared CTA (intake + saved coin)
├── src/composables/useDeepIdentification.ts   # NEW (job lifecycle, cancel/retry, resume-on-mount)
├── src/composables/useDeepIdentificationStream.ts # NEW (fetch+ReadableStream SSE reader with since/resume)
├── src/api/client.ts                          # CHANGED (typed wrappers + stream reader)
├── src/types/index.ts                         # CHANGED (deep identification types)
├── src/router/index.ts                        # CHANGED (route /deep-analysis/:jobId?)
└── src/components/coin/CoinActionsPanel.vue   # CHANGED (saved-coin entry point)

docs/
├── adr/0010-deep-agentic-coin-identification.md   # NEW (required)
├── openapi.json                                   # REGENERATED via `task openapi`
├── ARCHITECTURE.md                                # CHANGED (job/SSE/provider-tool sections)
├── features/ai-analysis.md                        # CHANGED (deep vs fast path)
└── quick-capture.md                               # CHANGED (deep proposal → draft seeding)
```

**Structure Decision**: the existing three-service layout is used unchanged
(`src/api`, `src/agent`, `src/web`). All new Go code follows the shipped
`models → repository → service → handler → main.go` wiring; the Python addition
is a self-contained `app/teams/deep_identification/` package plus one route; the
Vue addition is one thin page plus a component/composable set, deliberately
decomposed so no page approaches the repository's de-facto ~500–700 line ceiling
(`SetDetailPage.vue` 720, `CoinLookupPage.vue` 507).

## State Machines (summary; full detail in data-model.md)

| Machine | States / transitions | Where enforced |
|---|---|---|
| **Job** | `queued → running → {completed \| partial \| failed \| cancelled}`; terminal is one-way | Conditional UPDATE `WHERE status IN ('queued','running')` in the repository; `RowsAffected` decides the winner |
| **Provider run** | `pending → running → {contributed \| no_match \| failed \| timed_out \| skipped}`; or `pending → {not_automated \| unavailable}` without a call | Python emits typed evidence; Go persists per `(job_id, provider)` |
| **Event replay** | first connect / resume from `since` / truncated / terminal snapshot / gone (410) | Go SSE handler + `(job_id, seq)` unique index |
| **Retry lineage** | terminal job → new job with `retry_of_job_id`, depth ≤ 3; originals never mutated | Retry handler + repository |
| **Cancel vs complete race** | `cancel_requested_at` recorded; first conditional terminal UPDATE wins; loser gets 409 with the settled state; exactly one terminal event | Repository `SettleTerminal` transaction |
| **Stale restart recovery** | `running` with stale heartbeat → `failed:stale_restart` at boot and every 60 s | Service janitor (`RecoverStaleJobs` pattern from `ai_job_repository.go`) |
| **Expiration / cleanup** | hint artifacts deleted at any terminal state; events pruned 24 h post-terminal; report/job pruned at 90 d | Terminal hook + hourly janitor + startup sweep |
| **Draft / proposal application** | `proposed → (edited) → applied-to-draft \| applied-to-coin`; `already_applied` and `source_coin_missing` guarded | Proposal service via `QuickCaptureService.PromoteDraft` / `CoinService.UpdateCoinWithFields` |

## Implementation Phases

Effort is rough, in ideal focused sessions. Each phase must leave the repo
green (`task test`, `ruff`, `pytest`, `npm run build`).

| # | Phase | Depends on | Deliverables | Effort | Deployable? |
|---|---|---|---|---|---|
| 1 | **Job/event/storage foundation (Go)** | — | 4 models + AutoMigrate + migration test; repository (create/idempotent-find/claim/heartbeat/append-event/list-since/settle-terminal/prune/recover-stale); service skeleton with worker pool + cancel registry + janitor; artifact save/validate/delete; settings keys + feature flag; create/get/list/cancel/retry handlers with swaggo; `task openapi` | L (3–4) | Yes (flag off) |
| 2 | **Internal streaming graph (Python + Go proxy)** | 1 | `DeepIdentifyRequest`/`ProviderEvidence`/`DeepSynthesis` models; `app/teams/deep_identification/` graph with bounded fan-out, evaluator, synthesis; `POST /api/deep-identify/stream`; `AgentProxy.StreamDeepIdentification` callback consumer; Go persists translated events; image-evidence path | L (3–4) | Yes (flag off) |
| 3 | **MVP provider adapters** | 2 | Go internal tools `numista_search`/`numista_detail`/`nomisma_search` reusing existing clients + per-job budgets; Python `provider_tools.py`; provider nodes for Nomisma/Numista (automated), NGC (`not_automated` + OCR cert + link-out), OCRE/RPC (`not_automated`/`unavailable`); citation host validation; attribution strings | M (2–3) | Yes (flag on for dogfooding) |
| 4 | **SSE + resume UI** | 1–3 | Go SSE endpoint with `since`/`Last-Event-ID`/truncation/heartbeat/terminal-close; `client.ts` wrappers + stream reader; `useDeepIdentification*` composables; start panel, progress timeline, coverage list; cancel/retry; resume-on-mount; mobile/PWA/a11y pass | L (3–4) | Yes |
| 5 | **Report, proposal editor, confirm/apply** | 4 | `DeepReportPanel`, `DeepProposalEditor` (per-field accept/edit, AI-vs-owner distinction); `PATCH /proposal`; `POST /apply` → Quick Capture draft seed or `UpdateCoinWithFields`; saved-coin review entry point; deleted-coin and already-applied handling | M (2–3) | Yes — **MVP complete here** |
| 6 | **Hardening, docs, rollout** | 5 | Full test matrix (below); observability counters; safe-log audit; `docs/ARCHITECTURE.md` + feature docs; ADR 0010; `.squad/decisions/inbox/` note; rollout/rollback runbook | M (2) | Yes |
| 7 | **Gate G-OCRE (post-MVP)** | 6 | ODbL/attribution review recorded; OCRE adapter over the Nomisma SPARQL endpoint; flip `SettingDeepIdentificationOCREEnabled`; **no contract change required** | S–M (1–2) | Independent |
| 8 | **Gate G-RPC (post-MVP, blocked)** | 6 | Requires written permission or a documented API from the RPC project; until then RPC stays `unavailable` and no adapter is written | Blocked | Independent |

### MVP vs later-provider split

- **MVP (phases 1–6)**: Nomisma + Numista automated, NGC OCR/link-out,
  OCRE/RPC visible but `not_automated`/`unavailable`. Fully usable end to end.
- **Later**: OCRE (gate G-OCRE), RPC (gate G-RPC). Both are already first-class
  in the job/event/report contracts, so enabling them requires an adapter plus a
  settings flip — **no respecification** (spec Assumptions).

### Risk register

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| SSE replay bugs (duplicate/missed events) | Medium | High (SC-003) | `(job_id, seq)` unique index; sequence assigned in the state-change transaction; dedicated drop/reconnect tests |
| Cancel/complete race leaves ambiguous state | Medium | High (FR-019) | Single conditional terminal UPDATE + same-transaction terminal event; explicit concurrency test |
| 5-minute budget overrun | Medium | Medium (SC-002) | Explicit phase budgets, `asyncio.wait_for`, Go hard ceiling, partial synthesis on expiry |
| Numista free-tier quota exhaustion | Medium | Medium | ≤ 4 calls/job budget, existing TTL cache + coalescing, `quota_limited` status surfaced honestly |
| Numista "no persistent caching" clause vs persisted report | Low | Medium (legal) | Store N#/URL/claim text only; rely on the private-personal-project carve-out; recorded in ADR 0010 and re-reviewed if deployment model changes |
| Hint image leaks into coin images | Low | High (SC-004) | Separate artifact table with `Role='hint'`, no `CoinImage` write path, deletion tested on all three terminal paths |
| Oversized Vue page regression | Medium | Low | Component/composable split defined up front |
| Nomisma/OCRE upstream instability (no SLA, observed 500s) | Medium | Low | Typed `unavailable` status, partial success, no retry storms |

## Testing Strategy (deterministic, no live providers in CI)

**Go** (`src/api`)

- Repository: owner scoping (cross-user access returns not-found); idempotent
  duplicate submit returns the in-flight job; `(job_id, seq)` uniqueness under
  concurrent appends; terminal-settle race (parallel cancel + complete ⇒ exactly
  one terminal state and exactly one terminal event); stale recovery flips
  `running` → `failed:stale_restart`; event pruning sets `EventsPrunedAt` and
  preserves the report.
- Service: worker bounds/queue backpressure; per-user active limit; hard timeout
  ⇒ `partial`/`failed`, never hung; hint-artifact deletion asserted on **all
  three** terminal paths (SC-004); orphan/restart janitor; proposal field
  allowlist rejects non-writable fields; apply routes through
  `QuickCaptureService`/`CoinService` fakes (proving no direct coin write);
  apply on a deleted coin ⇒ `source_coin_missing`.
- Handler: multipart validation (type/magic bytes/size/count, missing role,
  hint-in-coin-role); auth/ownership 404s; SSE replay from `since`, truncation
  frame, heartbeat, terminal `end`, reconnect with no duplicates/gaps;
  feature-flag off blocks new starts but not in-flight jobs; swagger drift test.
- Provider tools: `httptest`-backed Numista/Nomisma clients (as
  `numista_client_test.go`/`nomisma_client_test.go` do today); budget
  enforcement returns `quota_limited`; internal-token middleware rejects
  unauthenticated calls.
- Architecture + migration: `TestArchitecture` green;
  `TestDeepIdentificationModelsAutoMigrate` additive.
- **Fast-path regression**: `POST /api/coins/lookup` contract test asserting the
  unchanged response shape and that zero `DeepIdentificationJob` rows are created
  (FR-001 / SC-008).

**Python** (`src/agent`)

- Router selects only from the supplied catalog, honours `provider_override`,
  and never selects an `automatable: false` provider for a live call.
- Fan-out respects `max_concurrency`/`max_providers`; per-provider timeout yields
  `timed_out`; one failing provider still reaches synthesis (partial success) —
  all via fake tool callables, no network.
- Deterministic merge ordering (same inputs ⇒ same prompt ordering).
- Contradiction detection surfaces both claims; nothing silently dropped.
- Citation host validation drops off-allowlist URLs so the LLM cannot introduce
  arbitrary URLs (SC-006).
- `not_automated` providers emit the correct status and never `no_match`.
- SSE envelope shape and terminal-frame invariants using the existing
  `FakeGraph`/`collect_sse` harness; token-shaped strings sanitized.

**Vue** (`src/web`)

- `useDeepIdentificationStream`: resume with `since`, duplicate suppression,
  truncation handling, terminal close, reconnect indicator.
- Start panel validation (missing obverse/reverse, hint count cap, saved-image
  reuse) and provider override.
- Coverage list renders `not_automated`/`unavailable` distinctly from
  `no_match`, with attribution and link-out.
- Proposal editor: per-field accept/reject, owner edit visibly distinct from the
  AI value, confirm gating (nothing written without confirm).
- Mobile viewport + a11y checks (labels, focus order, keyboard-accessible
  cancel) in component tests; one Playwright workflow spec (start → progress →
  reconnect → review) under `src/web/e2e/workflows/`.

**Never in CI**: real Numista/Nomisma/NGC/OCRE/RPC HTTP calls; real LLM calls.

## Operability: flags, limits, observability, rollout

- **Feature flag**: `SettingDeepIdentificationEnabled` (default `false`,
  admin-controlled, consistent with existing AI settings). When disabled, only
  **new starts** are blocked (403); running jobs finish, results stay readable,
  the fast path and coin CRUD are untouched (FR-008).
- **Limits**: worker count 2, queue depth 32 (→ `503 job_queue_full`), 1 active
  job/user, ≤ 4 providers, ≤ 2 concurrent provider calls, 300 s hard ceiling,
  Numista ≤ 4 calls/job, Nomisma ≤ 3 calls/job (research.md §9).
- **Observability**: counters/gauges through the existing logger and admin
  surface — jobs by terminal status, partial-success rate, p50/p95 duration,
  per-provider status counts and latency, active SSE streams,
  reconnect/truncation counts, queue depth, hint-deletion success/failure,
  janitor sweeps. Numista/Nomisma call volume reuses the existing redacted
  telemetry (F341).
- **Health**: existing `/health` and `/ready` remain the probes; a degraded agent
  surfaces as job `failed:agent_unavailable`, never as an API 5xx on job
  endpoints.
- **Safe logs**: log job id, user id, status, provider, typed error kind, and
  durations only. Never log notes, hint-derived context, provider query strings,
  report text, tokens, or API keys (FR-036).
- **Rollout**: ship dark (flag off) through phases 1–3 → enable for the admin
  user during phases 4–5 → enable generally after phase 6.
  **Rollback**: flip the flag off; in-flight jobs drain; tables and files are
  inert and read by no other feature; no schema rollback needed (additive only).

## Required ADRs, docs, migrations, release sequencing

- **ADR 0010 — Deep Agentic Coin Identification** (`docs/adr/0010-…md`, Nygard
  format). Must cover: (a) the new sibling job/event domain vs `AIJob`;
  (b) Go-persisted, Go-served SSE with bounded event retention;
  (c) provider HTTP calls remaining in Go, invoked by Python via internal tools;
  (d) provider staging and licensing posture (NGC no-automation, OCRE ODbL gate,
  RPC blocked, Numista attribution/caching constraints).
- **Docs**: `docs/ARCHITECTURE.md` (background jobs, AI integration, SSE),
  `docs/features/ai-analysis.md` (fast vs deep), `docs/quick-capture.md` (deep
  proposal seeding a draft), `docs/testing.md` (deterministic agent seams),
  `docs/openapi.json` regenerated via `task openapi`.
- **Migrations**: additive `AutoMigrate` only; no backfill; no destructive
  change; covered by a new migration test.
- **Decisions ledger**: cross-cutting note to `.squad/decisions/inbox/` (never
  edit `.squad/decisions.md` directly, §18.2).
- **Release sequencing**: phases 1→6 in order, each independently mergeable
  behind the flag; provider gates 7–8 are separate releases with their own ADR
  amendments if enabled.

## Post-Design Constitution Check

*Re-evaluated after Phase 1 (data model, contracts, quickstart) — Constitution v3.1.0.*

| Principle / Section | Post-design finding |
|---|---|
| **I. Layered Architecture** | The design keeps all GORM in the repository. The one multi-step write (terminal settle + event append) is expressed as a **repository method** `SettleTerminal(...)` running in a `WithTx` transaction, so **no new `allowedServiceGORMFiles` exception is required** and `architecture_test.go` needs no change. **PASS** |
| **II. Service Boundaries** | Contracts confirm Python receives everything per request (`llm_config`, images, quick evidence, provider catalog, bounds, minted token) and stores nothing; provider keys never leave Go; Vue never calls Python. The one deviation — Go persists and re-serves SSE rather than proxying bytes — is required by FR-009/FR-016 and is recorded in Complexity Tracking and ADR 0010. **PASS (deviation justified)** |
| **III. Strict Types** | REST contract, SSE envelope, and internal contract are explicitly typed and versioned (`schemaVersion`/`schema_version`); Pydantic strict models; swaggo + `task openapi` in the DoD. **PASS** |
| **IV. Simple Complete Changes** | The design added **no** new draft model, **no** new write path, **no** new dependency, and **no** duplicate provider client. Scope is 4 tables + 1 Python package + 1 UI flow. Sibling workflows are named with regression tests. **PASS** |
| **V. Security & Privacy** | Owner scoping is in the schema (denormalized `user_id` on events/artifacts), in the queries, and on the SSE endpoint; validation reuses shipped upload rules; hint images are ephemeral by construction with cleanup on all terminal paths plus a janitor for restart safety; the error surface is typed and generic; a log allowlist is defined. **PASS** |
| **VI. Consistent UX** | Component split defined (6 components + 2 composables + a thin page) to avoid an oversized-page regression; tokens/icons/PWA/a11y captured; provider transparency is a first-class UI element; no emojis in report or UI text. **PASS** |
| **VII. CI / Release** | Every phase leaves gates green; OpenAPI snapshot regeneration is an explicit deliverable; rollout is flag-gated with a no-op rollback. **PASS** |
| **VIII. Documented Decisions** | ADR 0010's scope is concrete (four decisions); the three no-precedent decisions (event retention, cancellation propagation, provider location) plus draft bridging are written up in research.md §§4–8. **PASS** |
| **IX. Automated Enforcement** | Invariants were chosen to be machine-checkable: `(job_id, seq)` uniqueness, single-terminal-state race, hint deletion on all terminals, provider-status honesty, citation host allowlist, fast-path no-regression. **PASS** |
| **§17 / §21** | Workflow-contract and blast-radius items enumerated; regression tests target exact paths; ≥ 1 unit test per new service method; ADR, Swagger, OpenAPI sync, and the decisions-inbox note are all in the phase plan. **PASS** |
| **PRD non-goals** | No catalogue replication (on-demand, per-job, link-out), no bulk ingestion, no forensic authentication claim, no new third-party service beyond already-approved Numista/Nomisma. **PASS** |

**Post-design gate result: PASS.** No unjustified violation; one deviation
carried in Complexity Tracking. Ready for `/speckit.tasks`.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|---|---|---|
| Go persists agent events and serves its **own** SSE endpoint instead of proxying the Python byte stream (Principle II states SSE flows Python → Go → Vue via `AgentProxy.proxySSE`) | FR-009 (job survives client disconnect), FR-015/FR-016 (gap-free resume from a last-seen sequence), FR-012 (restart recovery) and SC-003 (exactly-once replay) are impossible with a pass-through proxy bound to a single request | A pure proxy dies with the client and cannot replay; polling-only (the `CoinAIAnalysis.vue` pattern) fails "as they occur" streaming and gives poor UX on a 5-minute run; WebSockets would introduce a new transport with no precedent in this repository |
| Four new tables instead of extending `models.AIJob` | Optional coin linkage, six statuses, an append-only event log, provider runs, ephemeral artifacts, a cancellation request, retry lineage and fingerprint idempotency are all absent from `AIJob`, whose composite index encodes coin-bound idempotency used by shipped flows | Retrofitting ~10 nullable columns and changing the meaning of `idx_ai_jobs_user_coin_type_side_status` is a larger, riskier change with cross-feature blast radius (§21.7) than an additive sibling domain |
| Provider HTTP executes in Go and is invoked by Python over the internal tool channel (rather than Python calling providers directly, as it does for SearXNG/dealer pages) | Numista requires an API key held in `AppSetting` plus quota/TTL/coalescing/telemetry that already exist in Go (ADR 0007); Nomisma's typed status contract already exists in Go (ADR 0009). Keeping them in Go avoids secret spread and two divergent status vocabularies while keeping Python stateless | Duplicating both clients in Python would spread secrets to a stateless service, duplicate license/quota accounting, and create a second source of truth for provider statuses — the opposite of Principle IV |
