# Implementation Plan: Collector Curator, Watchlist, and Provenance

**Branch**: `beta` (existing; no create/switch) | **Date**: 2026-09-18 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `specs/363-collector-curator-watchlist-provenance/spec.md`

## Summary

Deliver five independently shippable, default-off slices: a private
owner-scoped collector profile; read-only curator recommendations composed from
existing collection/portfolio/gap services; read-only watchlist evaluation
composed from wishlist availability, auctions, and eligible Coin Copilot market
results; cautious evidence-backed provenance/documentation findings; and an
explicit, confirm-gated Add to Wishlist action.

Go remains the sole authority for identity, persistence, snapshots,
eligibility, validation, idempotency, transactions, audit, and writes. Python
stays stateless and read-only and may only infer over bounded Go-supplied
contracts. Vue starts mutation only from an explicit result-card control and
reviews a Go-created immutable stage before a second confirmation. The design
reuses `CoinService.CreateCoin`, the existing `Coin` wishlist representation,
wishlist availability/search-alert conversion patterns, auction services,
Deep Analysis projections, Coin Copilot checkpoints/specialists, and the
existing Settings and chat-result surfaces.

## Technical Context

**Language/Version**: Go 1.26.1; Python 3.12; TypeScript/Vue 3
**Primary Dependencies**: Gin, GORM, SQLite; FastAPI, Pydantic,
LangGraph/LangChain; Vue Router, Pinia, axios, Vite/PWA
**Storage**: SQLite via GORM AutoMigrate; additive `collector_profiles`,
`collecting_goals`, `wishlist_action_stages`, and
`wishlist_action_outcomes`; existing `Coin`, journal, Coin Copilot
run/checkpoint/event, availability, auction, alert-candidate, and Deep Analysis
records remain authoritative
**Testing**: Go `testing`, integration/architecture tests, race detector and
`go vet`; Python pytest/ruff and architecture guards; Vitest/Vue Test Utils,
strict `vue-tsc --build`, browser workflows; OpenAPI drift, security and image
workflows
**Target Platform**: Self-hosted single-node Docker deployment; desktop and
mobile/installed PWA browsers
**Project Type**: Existing three-service web application (Go API + Vue SPA/PWA
+ stateless Python agent)
**Performance Goals**: Profile reads/updates use one owner-scoped transaction;
no duplicate provider fan-out; recommendation/evaluation payloads remain
inside existing Coin Copilot 32 KiB tool/64 KiB event limits; confirmation
creates at most one wishlist item under concurrency
**Constraints**: Owner isolation; five default-off flags; no autonomous
buying/bidding; no generic browsing/new provider/bulk scraping; no Python
persistence or write; no raw provider payload; no implicit currency conversion;
no forensic or accusatory claims; no hardcoded deployment address
**Scale/Scope**: Personal self-hosted deployment, fewer than 10 concurrent
users; at most 20/20/50 preference values and 50 goals per owner; existing
Coin Copilot item/tool/run bounds apply

There are no unresolved `NEEDS CLARIFICATION` items.

## Constitution Check

*GATE: Passed before Phase 0 and rechecked after Phase 1 design.*

| Authority/gate | Design response | Result |
|---|---|---|
| §0 hierarchy | Constitution 3.1.0, Feature 363, and Features 012/337/344/351/352/353/359/361/362 were treated as ordered authorities. | PASS |
| Principle I: layered Go | Thin handlers call HTTP-agnostic profile/composition/action services; repositories own every GORM query and transaction variant; only composition root wires dependencies. | PASS |
| Principle II: service boundaries | Vue calls Go only. Go owns all durable state and writes. Python receives bounded snapshots and exposes no DB, write, approval, arbitrary HTTP, or generic action tool. | PASS |
| Principle III: contracts | Go strict decoding, Pydantic `extra="forbid"`, TypeScript discriminated unions, closed enums, bounds, source allowlists, shared tamper fixtures, Swagger and OpenAPI drift tests are required. | PASS |
| Principle IV: proportional reuse | Existing collection, portfolio/gap, availability, auction, specialist, Deep Analysis, `CoinService.CreateCoin`, Settings, and chat-card paths are composed rather than rebuilt. Four small tables are the minimum honest durable state. | PASS |
| Principle V: privacy/security | Owner is derived server-side; foreign equals unknown; profile/private collection facts never enter public/follower/log surfaces; URL/SSRF, replay, cancellation, prompt-injection, body-size and sanitized-error controls fail closed. | PASS |
| Principle VI: UX/PWA | Extend Settings and `CopilotRunProgress`; use design tokens/Lucide, semantic controls, focus trap/restore, keyboard operation, 44px touch targets, responsive review layout, screen-reader status, and offline-safe disabled states. | PASS |
| Principles VII/IX and §§17/21 | Exact Go/Python/Vue contract, race, security, browser, OpenAPI, CodeQL/default scanning, container-image, and regression gates are specified in `quickstart.md`. | PASS |
| Principle VIII | The durable confirmation boundary and schema decision are recorded in these artifacts. An implementation ADR is required because this is a semantic migration and cross-service action contract. | PASS |

### Post-design re-evaluation

PASS. No waiver is required. Admin `AppSetting` remains appropriate only for
global rollout flags; collector preferences are not forced into it. Existing
Coin Copilot checkpoints safely retain bounded read-only profile/evidence
snapshots, while immutable staging/outcome rows are necessary because
checkpoints are prunable and cannot authorize a write.

## Project Structure

### Documentation (this feature)

```text
specs/363-collector-curator-watchlist-provenance/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/
    ├── collector-workflows.md
    ├── wishlist-action.openapi.yaml
    └── openapi-impact.md
```

No `tasks.md` is created by this planning work.

### Source Code (repository root)

```text
src/api/
├── models/                    # profile, goal, immutable stage/outcome models
├── repository/                # owned profile/action queries and WithTx variants
├── services/
│   ├── collector_profile_service.go
│   ├── collector_workflow_service.go
│   ├── wishlist_action_service.go
│   ├── coin_service.go        # canonical CreateCoin path reused
│   ├── collection_tools_service.go
│   ├── coin_copilot_*.go
│   └── agent_proxy.go
├── handlers/                  # thin authenticated profile/workflow/action handlers
├── database/database.go       # ordered additive AutoMigrate
├── routes_protected.go
└── architecture_test.go

src/agent/
├── app/models/{requests.py,responses.py}
├── app/teams/{coin_copilot.py,specialist_contracts.py}
├── app/tools/copilot_collection_tools.py
└── tests/

src/web/src/
├── api/endpoints/{agent.ts,wishlist.ts,collectorProfile.ts}
├── types/{agent.ts,wishlist.ts,collectorProfile.ts}
├── components/settings/       # Collector Profile section
├── components/chat/CopilotRunProgress.vue
├── components/wishlist/       # stage review + confirmation modal
├── composables/useCoinCopilot.ts
└── pages/SettingsPage.vue
```

**Structure Decision**: Extend only current composition points. Do not add a
service, agent platform, provider module, marketplace, generic tool router, or
separate recommendation database. The only new Vue surface is a Settings
section plus review modal/drawer anchored to existing chat cards.

## Existing Symbol Reuse Map

| Current symbol/path | Planned reuse |
|---|---|
| `AddCoinPage.vue` → `stores/coins.ts:addCoin` → `api.createCoin` | Behavioral baseline for an ordinary manual wishlist create; assisted creation must converge on the same Go service, not submit this broad form payload. |
| `handlers/coins.go:Create` and `CoinService.CreateCoin`, `prepareCoinForCreate`, `createPreparedCoinInTx` | Canonical wishlist validation/create path. Add an internal transaction-aware service entry accepting only the mapped allowlist; preserve reference validation and value snapshot behavior. |
| `handlers/coins.go:Update` / `CoinService.updateCoin` | Regression proof that omitted/manual fields remain untouched; Feature 363 confirmation never invokes broad update. |
| `CoinJournal`, journal repository, `RecordValueSnapshot` | Append the existing assisted-create journal entry in the confirmation transaction; do not repurpose journal rows as action authorization. |
| `wishlist_search_alert_service.go:NormalizeSourceFilters`, `CanonicalSourceURL`, candidate conversion patterns | Reuse canonical source normalization, duplicate lookups, and explicit conversion precedent; do not reuse its broader/manual field mapping. |
| availability and auction repositories/services/routes | Read authoritative listing/run/lot state only; never replace their state machines. |
| `AgentRepository` portfolio summary and `CollectionToolsService` portfolio/gap methods | Sole collection facts for curator/gap reasoning. |
| Coin Copilot run/checkpoint/event models, `AuthorizeToolCall`, cancellation/replay, `CoinCopilotSpecialistResult` | Existing bounded durable read-only harness and eligible market-result evidence. |
| `specialist_contracts.py`, `validate_outbound_url`, `safe_get` | Existing provider, URL, redirect, provenance, prompt-injection and bounds policy. |
| Deep Analysis persisted report/proposal projection and Feature 362 handoff | Attribution-rich evidence only after 362; never copy raw JSON or apply authority. |
| `SettingsPage.vue`, `SettingsAccountSection.vue`, `useSettingsProfile.ts` | Add a dedicated Collector Profile section without extending admin settings or unrelated user columns. |
| `CopilotRunProgress.vue` specialist-result block | Add the explicit Vue-rendered control only for validated eligible dealer/auction items. |
| `CategoryEraConfirmModal.vue` | Interaction/accessibility precedent; action review uses a purpose-built modal because evidence and mapping are materially richer. |

## Delivery Phases

### Phase 1 — Collector profile foundation (independent, pre-362)

1. Add profile/goal models, owner indexes, repository transaction, validation,
   optimistic versioning, safe missing-profile defaults, and profile REST API.
2. Add **Settings > Collector Profile** with bounded lists, budget/currency,
   goal CRUD in one atomic save, clear-to-default, version-conflict recovery,
   accessible errors, and desktop/PWA layouts.
3. Add the default-off `collector_profile_enabled` global flag. Disabled reads
   return neutral defaults; stored rows remain inert/recoverable.

**Gate**: full boundary fixtures, rollback/migration tests, cross-user/admin
isolation, atomic stale-write tests, and zero admin-settings mutation.

### Phase 2 — Read-only curator recommendations (independent, pre-362)

1. Add a fixed `collector_curator` read-only capability composed from existing
   collection summary, portfolio review and gap analysis outputs.
2. Go snapshots one profile version and bounded collection facts; Python
   returns strict observed facts, strengths, themes, gaps and acquisition ideas.
3. Render typed recommendation cards in existing Coin Copilot progress/result
   surfaces with evidence, confidence, why-it-matters, profile effects and
   limitations.

**Gate**: no route/tool with write semantics is registered; view/rerun/replay/
cancel/dismiss produces zero database mutation outside existing run durability.

### Phase 3 — Read-only watchlist evaluation (independent, pre-362)

1. Add fixed evaluation input unions for owned wishlist coins, active profile
   goals, and retained eligible Feature 361 dealer/auction results only.
2. Go resolves ownership, source freshness, availability/auction authority,
   duplicates, collection coverage, and one profile snapshot. Python ranks
   bounded candidates but cannot ingest direct URLs or saved searches.
3. Display input origin, comparability/unknowns, stale listing state, duplicate
   relationships, confidence and limitations.

**Gate**: direct URLs/saved searches fail as deferred; no provider work is
triggered for replay/evaluation; no wishlist/listing/auction state changes.

### Phase 4 — Evidence-backed provenance/documentation findings

**4A baseline (pre-362)**: missing provenance/documentation, broken approved
links through existing bounded availability/URL capabilities, unverified or
conflicting listing claims, and duplicate-image findings only when an existing
defensible comparison record is supplied.

**4B enriched (post-362 only)**: consume Feature 362's validated persisted Deep
Analysis projection, preserving its provider coverage, citations, conflicts,
confidence and limitations. If unavailable, emit the closed
`attribution_evidence_unavailable` gate—never infer from absence.

**Gate**: every finding is one allowed kind/tier and literally says “needs
review”; factual/claim/recommendation fields remain separated; accusation,
fraud and authenticity language tamper fixtures fail closed.

### Phase 5 — Confirmed Add to Wishlist (pre-362 when Feature 361 is present)

1. Vue shows **Add to Wishlist** only on validated retained dealer/auction
   result cards. The click, not model prose/tool choice, calls Go `stage`.
2. Go owner-resolves the retained result, revalidates provider/URL/provenance/
   freshness/run state, maps only wishlist-valid fields, computes duplicate
   warnings, and stores an immutable expiring stage. No coin is created.
3. Vue reviews source, mapping, omitted fields, evidence, confidence,
   price/currency/listing state, limitations and duplicates. Editable fields
   are restricted to destination-valid allowlisted values and are incorporated
   into a new exact stage version rather than mutating a prior stage.
4. A second UI confirmation sends stage id/version and a fresh idempotency key.
   Go rechecks owner, expiry, cancellation, stage fingerprint, eligibility and
   duplicates; one transaction calls the canonical `CoinService` create seam,
   writes the immutable outcome, links the coin journal, and returns a stable
   replay result.
5. Python has no stage/confirm route, tool, credential, DTO or approval state.

**Gate**: cancellation/unconfirmed/expired stages create zero coins; same
payload replay returns the original result; key/payload mismatch conflicts;
concurrent tabs create at most one source-linked wishlist item; manual fields
remain empty/unmodified.

### Phase 6 — Operational hardening and release

Run every contract, tamper, owner-isolation, migration/rollback, race,
cancellation/replay, privacy/logging, fallback, accessibility, browser,
OpenAPI, security, CodeQL/default-code-scanning, and container/image gate in
`quickstart.md`. Roll out flags in phase order to a small owner cohort. Run the
`post-major-work-qc-audit` skill after implementation and before merge/release;
resolve High/Critical findings.

## Architecture and Route Guards

- Public mutation surface is exactly
  `POST /api/collector/wishlist-actions/stages` and
  `POST /api/collector/wishlist-actions/stages/{id}/confirm`; both require user
  auth, JSON caps and server-derived owner.
- No `/api/internal/copilot/tools/*` stage/confirm route exists. Add tests that
  enumerate internal callbacks and reject `wishlist_*`, `create_*`, `save_*`,
  `stage_*` and `confirm_*`.
- Python `COPILOT_ALLOWED_TOOLS`, `CALLBACK_TOOLS`, `ARG_MODELS`, and
  `RESULT_MODELS` remain read-only; extend
  `test_coin_copilot_architecture.py` to prove no write/approval imports,
  endpoints or models.
- Vue API calls remain under `src/web/src/api/client.ts`; add a guard forbidding
  browser references to the Python agent base URL and require the result-card
  click event before stage calls.
- Go architecture tests require Handler → Service → Repository → DB and forbid
  handlers/raw SQL/direct DB access. Confirmation transaction starts in a
  repository transaction boundary supplied to both action and coin services.
- Recommendation/risk response schemas contain no action URL or model-chosen
  approval flag. The only action route is constructed by Vue's typed API client.

## Operational, Privacy, and Rollout Rules

Five existing-style global `AppSetting` flags default false:
`CollectorProfileEnabled`, `CollectorCuratorEnabled`,
`CollectorWatchlistEvaluationEnabled`, `CollectorRiskReviewEnabled`, and
`CollectorWishlistActionEnabled`. Each later preflight requires its
foundations; 4B separately requires Feature 362. Disablement leaves existing
rows readable/inert and legacy paths unchanged.

Log only safe event code, request/action id, actor owner id (server log only),
profile version/digest, source provider id, source identity digest, outcome,
duration, counts, truncation, and created coin id. Never log profile text,
budgets, collection facts, result titles/descriptions, source URLs, evidence
text, raw prompts/provider payloads, credentials, stage JSON, or internal
exceptions. Audit retention follows existing owner/action retention policy;
profile deletion/export must include these private records.

## Complexity Tracking

No constitutional violation or waiver exists. The additive complexity is
justified:

| Addition | Why required | Simpler alternative rejected |
|---|---|---|
| Profile + goal tables | Dedicated private owner profile, bounded goal identities and atomic optimistic updates | Admin settings are global; `User` columns would be unbounded/schema-heavy and conflate account identity with collector intent |
| Immutable stage + outcome tables | Exact review snapshot, expiry, idempotency, source uniqueness, atomic audited confirmation and durable replay | Coin Copilot checkpoints are prunable/read-only; coin journal is post-write and cannot authorize or deduplicate a write |

## Stop Point

Planning stops after Phase 2 design artifacts. No tasks, production code,
migration execution, deployment, commit or branch operation is performed.
