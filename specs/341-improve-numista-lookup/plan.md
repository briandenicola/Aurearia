# Implementation Plan: Improved Numista Lookup

**Branch**: `341-improve-numista-lookup` | **Date**: 2026-08-11 | **Spec**: [spec.md](./spec.md)  
**Input**: Feature specification from `specs/341-improve-numista-lookup/spec.md`

## Summary

Replace the two independent Numista integrations with one injected, typed Go
capability that owns provider HTTP, normalization, status mapping, TTL caches,
request coalescing, quota/health signals, two-stage search/detail enrichment,
and deterministic application scoring. Direct coin-detail and photo-assisted
flows will both submit an editable query and typed evidence to this capability,
render the same outcome/candidate DTOs, and persist only an explicit selection.
Photo analysis will propose evidence and a query but will not perform a Numista
search before the collector can edit it. Quick Capture will retain its selected
reference in a one-to-one draft-reference row and copy it into
`coin_references` in the existing promotion transaction. The amended saved-coin
UX makes Catalog References the canonical lookup surface, retains lookup only
where it is contextual to Identify Coin and Quick Capture, and adds no
top-level navigation or eager NGC-path request.

## Technical Context

**Language/Version**: Go 1.26.5 API; Vue 3 with strict TypeScript; no Python agent changes  
**Primary Dependencies**: Gin 1.12, GORM 1.31, `net/http`, standard-library synchronization/crypto; Vue/Vite/axios/lucide-vue-next  
**Storage**: SQLite for Quick Capture selected references and existing settings/references; bounded in-memory disposable TTL caches and rolling telemetry  
**Testing**: Go `testing` with `httptest`, repository/service/handler tests; Vitest + Vue Test Utils; strict `vue-tsc --build`; OpenAPI route-drift tests  
**Target Platform**: Self-hosted Linux container API and browser/PWA clients, including mobile capture  
**Project Type**: Three-service web application; this feature changes only Go API and Vue SPA  
**Performance Goals**: p95 uncached broad search/terminal status under 5 seconds; p95 fresh-cache response under 1 second; broad results paint before enrichment; default at most five detail calls  
**Constraints**: Shared provider allowance; explicit collector choice; NGC-first behavior; no provider payload leakage; no key/full inscription telemetry; context cancellation; rollback-safe additive schema  
**Scale/Scope**: One instance-level credential, bounded cache (default 500 searches/5,000 details), bounded telemetry (last 500 operations), direct lookup and Quick Capture/photo workflows  

There are no unresolved technical clarifications. Decisions and rejected
alternatives are recorded in [research.md](./research.md).

## Constitution Check

*GATE: evaluated before research and re-evaluated after design.*

### Pre-design evaluation

| Authority | Gate | Result |
|---|---|---|
| Principle I — Clear Layered Architecture | Handler → service → repository/database; HTTP-agnostic business services; explicit constructors; transactional multi-write | PASS. Handlers bind DTOs only. `NumistaLookupService` orchestrates client/cache/scorer/telemetry. Quick Capture repository owns the draft-reference SQL and promotion transaction. |
| Principle II — Service Boundary Separation | Vue calls Go only; no LLM logic in Go; Python stays stateless | PASS. Photo vision continues through the existing Go proxy; Numista is a Go-owned external catalog integration. |
| Principle III — Strict Types and Explicit Contracts | Typed Go/TS DTOs, Swagger, `client.ts`, strict checks | PASS. Provider structs are private, application DTOs are explicit, new/changed routes are in OpenAPI, and Vue calls remain in `api/client.ts`. |
| Principle IV — Simple Complete Changes | Complete sibling workflows without oversized rewrite | PASS. One shared capability removes duplication while retaining existing routes during transition; no local catalogue or background worker is added. |
| Principle V — Security, Auth, Privacy | Safe validation/errors, protected/admin routes, no secrets or sensitive logs | PASS. Provider key is server-only; query bounds and canonical URLs are validated; telemetry redacts user text. |
| Principle VI — Consistent User Experience | Tokens, dark/PWA/mobile, no emoji, accessible operation | PASS by design. Existing surfaces are extended with token-based responsive controls and keyboard/radio semantics. |
| Principle VII / §17 — Release Integrity and Quality Gate | Build, test, contract, regression, and documentation paths identified | PASS at planning. Exact commands and coverage are in quickstart and Phase 2. |
| Principle VIII — Documented Decisions | Material migration/provider decisions recorded | PASS. Plan/research/data model are durable; implementation must add an ADR because shared external-client/cache semantics and a data migration are material. |
| Principle IX — Automated Enforcement | Rules covered by automated tests | PASS by design. Live Numista is replaced by interfaces/`httptest`; route drift, migration, ownership, scoring, and UI paths get automated coverage. |
| §21 Definition of Done | Workflow blast radius, config contracts, new-service tests, Swagger, secrets | PASS at planning. All affected sibling paths are enumerated below. |

**Gate result**: PASS. No waiver or Complexity Tracking entry is required.

### Post-design re-evaluation

The Phase 1 model and contracts preserve the same results:

- the only durable write added to lookup is repository-owned
  `quick_capture_draft_references`, and promotion creates the coin, images,
  selected reference, value snapshot, draft state, and lifecycle event in one
  transaction (Principle I);
- the shared service accepts application DTOs and `context.Context`, not
  `gin.Context`; provider payloads never cross the service boundary
  (Principles I and III);
- compatibility aliases are read-only and time-bounded, avoiding a flag-day
  frontend/backend deployment while new writes use the canonical contract
  (Principles III and IV, §21.7);
- telemetry contains path/status/count/duration/cache/enrichment/quota timing
  and a truncated SHA-256 correlation key, never API keys, images, full
  inscriptions, label text, or raw provider errors (Principle V);
- the two-call broad/enrich contract makes the latency requirement testable
  and keeps partial detail failure from erasing broad candidates (Principles
  IV and IX).

**Post-design gate result**: PASS. No unresolved clarification or justified
constitutional violation remains.

## Component Ownership and Dependency Flow

```text
Vue direct/photo views
  └─ src/web/src/api/client.ts (typed REST only)
       ├─ POST /api/numista/lookup       ─┐
       ├─ POST /api/numista/enrich       ├─ NumistaHandler (bind/map only)
       └─ GET  /api/numista/search       ┘  deprecated compatibility adapter
                                              │
                                              v
                                      NumistaLookupService
                                 ┌────────────┼─────────────┐
                                 v            v             v
                         NumistaClient   NumistaScorer  NumistaTelemetry
                         (interface)     (pure service) (bounded/redacted)
                              │
                     HTTPNumistaClient
                    net/http + provider DTOs
                              │
                    TTLCache search/detail

POST /api/coins/lookup
  CoinLookupHandler → CoinLookupService → AgentProxy
                                      └─ typed evidence/query proposal only;
                                         NGC match suppresses proposal by default

Quick Capture handlers → QuickCaptureService → QuickCaptureRepository → SQLite
                                                   ├─ draft reference
                                                   └─ promotion transaction

GET /api/admin/numista/health
  Admin NumistaHealthHandler → NumistaTelemetry snapshot (no credentials/text)
```

### Backend ownership

- `models/numista.go`: application-owned lookup/evidence/candidate/outcome
  types and enums; standard library only.
- `services/numista_client.go`: `NumistaClient` interface, provider error
  taxonomy, HTTP implementation, private provider DTOs, retry/cancellation.
- `services/numista_cache.go`: injectable clock/cache interfaces, bounded TTL
  cache, independent search/detail namespaces, same-key in-flight coalescing.
- `services/numista_scoring.go`: normalization, date parsing, weighted scoring,
  deterministic ordering, explanations.
- `services/numista_lookup_service.go`: orchestration, settings validation,
  broad lookup, bounded enrichment, graceful degradation, telemetry.
- `services/numista_telemetry.go`: thread-safe rolling events and aggregates.
- `handlers/numista.go`: authenticated typed lookup/enrichment and legacy GET.
- `handlers/admin_numista.go`: admin-only redacted health summary.
- `repository/quick_capture_repository.go`: owner-scoped selected-reference
  create/replace/remove/load and transactional promotion copy.
- Existing `CoinReferenceService` remains authoritative for direct selection
  validation/deduplication.

### Dependency rules and test seams

- `main.go` constructs one `HTTPNumistaClient`, cache, scorer, telemetry, and
  lookup service and injects the service into both lookup handlers/services.
- Interfaces are consumer-facing and narrow: provider transport, cache, clock,
  telemetry recorder. Tests inject fakes or an `httptest.Server`; no live key.
- The client receives a base URL and `http.Client` through configuration.
  Production base URL is fixed to HTTPS Numista; tests may use localhost.
- All client methods accept `context.Context`; handlers pass request context.
  Cancellation aborts retries and enrichment work.
- Services never import Gin. Models never import GORM helpers outside struct
  tags or non-standard packages.

## Provider Client, Errors, Retry, and Configuration

`NumistaClient` exposes:

```go
Search(ctx context.Context, key string, query NumistaProviderSearch) (ProviderSearchResult, error)
GetType(ctx context.Context, key string, typeID int) (ProviderTypeDetail, error)
```

`HTTPNumistaClient` uses `GET /v3/types` then `GET /v3/types/{id}`. Responses
are size-limited before JSON decode. Only application-needed fields are mapped.
The key is added only as `Numista-API-Key`; URLs, cache keys, and logs exclude
it. Error kinds are `invalid_request`, `unauthorized`, `quota_limited`,
`timeout`, `unavailable`, `malformed_response`, and `cancelled`, carrying an
optional parsed `Retry-After` but no raw body.

One bounded retry is allowed only for connection resets and 502/503/504, with
context-aware 100–300 ms jittered backoff. 400, 401/403, and malformed JSON are
not retried. 429 is never automatically retried, to conserve allowance.
Caller cancellation is returned immediately and is not counted as provider
unavailability. Search uses a configurable four-second default timeout;
individual detail calls use three seconds, concurrency two.

Validated settings and defaults:

| Key | Default | Validation |
|---|---:|---|
| `NumistaSearchTTLHours` | 24 | integer 1–720 |
| `NumistaDetailTTLHours` | 168 | integer 1–2160 |
| `NumistaEnrichmentLimit` | 5 | integer 1–10 |
| `NumistaSearchResultLimit` | 20 | integer 1–50 |
| `NumistaSearchTimeoutSeconds` | 4 | integer 1–10 |
| `NumistaDetailTimeoutSeconds` | 3 | integer 1–10 |

Invalid stored settings fall back to documented defaults and emit a safe
warning/health flag. Changing credentials or numeric settings is read on each
operation. Cache content is keyed independently of credentials, but the
service checks configuration before any cache read, so cached data cannot hide
`unconfigured`. TTL changes affect new writes; existing entries retain their
recorded expiry and remain disposable.

## Query, Scoring, and Two-Stage Matching

Both UIs build a visible source query from non-empty fields in this order:
title/name, ruler/issuer, denomination, mint, date/range, material, obverse
inscription, reverse inscription/label text. The collector may edit the exact
`query` before every search. The request also carries bounded structured
evidence; the effective query is returned verbatim, while normalization is
internal.

Normalization applies Unicode NFKC, case folding, whitespace/punctuation
collapse, token deduplication, and a conservative 100-token/500-character
limit. Source text is not rewritten in the UI. Dates parse signed astronomical
years plus BCE/BC and CE/AD ranges; unparseable dates are `unavailable`, not a
conflict.

The injectable versioned `NumistaScoringConfigV1` uses:

| Dimension | Weight |
|---|---:|
| exact Numista identifier | 35 |
| title | 15 |
| issuer/ruler | 12 |
| denomination | 12 |
| mint | 10 |
| date compatibility | 8 |
| material | 5 |
| inscriptions/visible text | 3 |

For each request-present dimension, similarity is `[-1,1]`: exact/overlap and
token similarity are positive; explicit categorical/date conflicts are
negative; absent candidate evidence is zero and labeled unavailable. Score is
`clamp(round(50 + 50 * Σ(weight×similarity) / Σ(applicable weights)), 0, 100)`.
No request evidence means score 50 and provider order is only the final
tie-break. Bands are `strong` (>=80), `possible` (60–79), and `weak` (<60),
always labeled decision support rather than attribution.

Sort order is score descending, exact-ID match first, enrichment completeness
descending, normalized title ascending, numeric Numista ID ascending, then
original provider position. Explanations are structured reason codes plus
short safe labels for meaningful matches, conflicts, and unavailable fields;
the server does not echo full inscription or label text.

Stage 1 returns up to the configured broad limit immediately. The UI paints it
and then requests Stage 2 for only the service-ranked leading IDs, capped again
server-side by `NumistaEnrichmentLimit`. Detail requests use a concurrency of
two. Each candidate records `not_requested`, `enriched`, `cached`, or `failed`.
The enrichment response reranks the complete submitted broad set. A failed
detail remains selectable with its broad fields; all detail failures preserve
overall `success`. Requests above the limit or containing invalid IDs fail
validation without provider calls.

## Status, Cache, Quota, and Observability

Lookup outcomes are exactly `success`, `empty`, `unconfigured`,
`quota-limited`, `timeout`, or `unavailable`. HTTP 200 carries every expected
domain outcome; malformed client input is 400, auth is 401/403, and unexpected
application failure is generic 500. This keeps an empty result distinct from
transport state. `guidanceCode` lets Vue choose role-sensitive copy; only
admins see a configuration link.

Search cache key: SHA-256 of contract version, normalized query, normalized
evidence relevant to provider search, language, object type, and result limit.
Detail key: version, language, numeric type ID. Search caches both success and
empty outcomes; provider errors are not cached. Detail caches valid mapped
details only. Each response includes `cache.hit`, `cache.createdAt`,
`cache.expiresAt`, and `cache.ageSeconds`; stale entries are deleted and never
served as current. Same-key misses share one in-flight call.

The telemetry ring records timestamp, path (`direct`/`photo`), operation
(`search`/`detail`), status, cache hit/refreshed, elapsed milliseconds,
candidate/detail counts, enrichment failures, retry count, optional
retry-after, and the first 16 bytes of a SHA-256 correlation digest. It never
records the key, images, raw provider body/error, query, inscriptions, or label
text. Admin summary exposes configuration boolean, last outcome, rolling
counts, p50/p95 latency, cache-hit rate, last quota-limited time/retry-after,
and enrichment success/failure. Logs use the same safe fields.

## Persistence, Migration, and Transactions

Add `QuickCaptureDraftReference` with a unique `DraftID`, owner `UserID`,
fixed validated `Catalog="Numista"`, `Number`, canonical HTTPS `URI`, and
timestamps. AutoMigrate creates an additive table; existing drafts require no
backfill. Draft reads preload the optional relation. Create accepts zero or one
reference; update omission preserves it, a valid pair replaces it, and an
explicit clear removes it. Repository transactions keep draft scalar/image/
reference changes and lifecycle events atomic.

Promotion extends the existing repository transaction: after creating the
coin, create one `CoinReference` from the selected draft row using the existing
validation/dedup rules, then transfer images, snapshot value, and finalize the
draft. Any reference failure rolls back the entire promotion. The existing
compare-and-swap claim and promoted-coin response make retries idempotent; the
unique coin-reference index prevents duplicates. Both collection and wishlist
targets retain the selected reference, per this higher-authority active spec.

Rollback leaves the additive table unread by old binaries; coins and existing
references remain readable. A new binary tolerates absent selected rows.
Cached data is never authoritative and needs no migration.

## Frontend Design

- `CoinReferencesSection.vue`: canonical saved-coin surface. Place compact
  `Search Numista` beside manual `Add Reference`, expand the lookup inline,
  retain all manual create/edit/delete behavior, and collapse only after a
  confirmed selection persists and the reference list refreshes.
- `CoinNumistaPanel.vue`: initialize a multiline/combobox-style editable query
  from all coin evidence; show status/guidance, cache freshness, ranked cards,
  reason lists, enrichment progress, selection radio, clear/replace, and an
  explicit “Add selected reference” action through the existing coin-reference
  API. Never persist on result click. It is composed by Catalog References for
  saved coins, not by the full Actions panel.
- `CoinActionsPanel.vue`: remove the full lookup panel. During compatibility
  transition it may render one compact contextual row/link to the saved coin's
  Catalog References section; it does not own lookup state.
- `CoinLookupPage.vue`: after vision analysis, preserve NGC-first behavior. If
  no NGC result, show the proposed editable query and evidence, but wait for
  collector search. If NGC succeeds, show `Also search Numista`; activating it
  reveals an editable panel but still performs no provider request before
  explicit search submission. Label initial analysis `Analyze Photos`; retain
  `Save as Draft` for persistence. Preserve query and selected reference across
  retries. Broad results render before enrichment. Saving a Quick Capture draft
  sends only the selected reference.
- `QuickCaptureDraftPage.vue` and draft card/forms: display retained selection,
  allow replace/remove, and preserve it when unrelated edits or validation
  failures occur. Promotion readiness identifies that a reference is optional.
  `QuickCaptureDraftCard.vue` renders `Numista #<identifier>` from the existing
  selected-reference list projection.
- Shared `NumistaLookupPanel.vue` and small pure query/status helpers avoid
  divergent behavior while keeping page ownership explicit.
- Use native buttons/radio semantics, visible focus, `aria-live` for status,
  `aria-expanded`/`aria-controls` for disclosure controls, textual score
  explanations, descriptive image alt text, 44 px mobile tap targets, and no
  color-only meaning. Cards and header actions wrap at 375 px without
  horizontal overflow. Existing design tokens and Lucide icons apply.

### Approved placement amendment and authority

Feature 341 is active and unmerged, so Constitution §0 permits editing its
specification package in place. The amendment does not alter landed Feature
214 or Feature 336 artifacts, reopen completed T001–T053, or require a
constitutional/ADR amendment. It is a proportional Principle IV reconciliation
of the active spec, plan, contract projection, quickstart, and tasks. Phase 6
cache/telemetry and Phase 7 enrichment scope remain unchanged.

## API Compatibility and Rollout

1. Add database table, services, new typed POST routes, admin health route, and
   additive fields to photo/Quick Capture DTOs behind the existing auth.
2. Keep `GET /api/numista/search?q=` for one release as a deprecated adapter
   backed by the shared service and preserving its `{count,types}` shape.
3. During the same release, `POST /api/coins/lookup` adds
   `proposedNumistaQuery`, `numistaEvidence`, and `numistaLookup`; retain
   deprecated `numistaCandidates` and `candidateReferences`, but never populate
   them from unselected Numista results. NGC references remain unchanged.
4. Deploy backend before frontend. New Vue uses only the typed POST contract;
   an old frontend continues to use legacy GET/photo aliases.
5. Update Swagger artifacts, `docs/openapi.json`, API reference, Numista
   feature docs, Quick Capture docs, admin settings/deployment docs, and add an
   ADR. Announce legacy adapter removal separately; do not remove it here.
6. Observe status mix, p95 search latency, cache hit rate, 429s, and detail
   failures. Roll back UI independently if needed; additive data remains safe.
7. Move saved-coin composition from Actions to Catalog References without
   removing the Actions route, manual reference controls, legacy lookup routes,
   or draft APIs. No new top-level route or navigation entry is introduced.

## Project Structure

### Documentation (this feature)

```text
specs/341-improve-numista-lookup/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
└── contracts/
    └── numista-lookup.openapi.yaml
```

### Source code affected during implementation

```text
src/api/
├── models/
│   ├── numista.go                         # new application DTO/value types
│   └── quick_capture_draft.go             # selected reference relation
├── services/
│   ├── numista_client.go                  # new provider boundary
│   ├── numista_cache.go                   # new TTL/coalescing cache
│   ├── numista_scoring.go                 # new pure scorer
│   ├── numista_lookup_service.go          # new shared orchestration
│   ├── numista_telemetry.go               # new rolling health signals
│   ├── coin_lookup_service.go             # analysis/query proposal only
│   ├── quick_capture_service.go
│   └── settings_service.go
├── repository/
│   └── quick_capture_repository.go
├── handlers/
│   ├── numista.go
│   ├── admin_numista.go
│   ├── coin_lookup.go
│   ├── quick_capture.go
│   └── swagger_types.go
├── database/database.go
├── main.go
└── *_test.go / package-local tests

src/web/src/
├── api/client.ts
├── types/index.ts
├── components/coin/CoinNumistaPanel.vue
├── components/coin/CoinReferencesSection.vue
├── components/coin/CoinActionsPanel.vue
├── components/numista/NumistaLookupPanel.vue
├── components/quick-capture/QuickCaptureDraftCard.vue
├── components/admin/AdminSystemSection.vue
├── pages/CoinLookupPage.vue
├── pages/CoinDetailPage.vue
├── pages/QuickCaptureDraftPage.vue
├── pages/QuickCaptureDraftsPage.vue
└── corresponding __tests__/

docs/
├── features/numista-integration.md
├── quick-capture.md
├── api-reference.md
├── deployment.md
├── openapi.json
└── adr/NNNN-shared-numista-lookup.md
```

**Structure Decision**: Preserve the existing Go API/Vue SPA layout. Numista
business and provider concerns live in Go services, database work remains in
the Quick Capture repository, and Vue owns only presentation/request shaping.
No Python or fourth service is introduced.

## Phase 0: Research

Completed in [research.md](./research.md). It resolves provider endpoint/error
behavior, two-stage transport, cache location, scoring, persistence,
compatibility, security/privacy, and telemetry choices. No `NEEDS
CLARIFICATION` remains.

## Phase 1: Design and Contracts

- [data-model.md](./data-model.md) specifies value objects, validation,
  relations, lifecycle, cache and telemetry records, and migration/rollback.
- [contracts/numista-lookup.openapi.yaml](./contracts/numista-lookup.openapi.yaml)
  defines direct lookup, enrichment, photo-analysis additions, Quick Capture
  selected-reference fields, and admin health.
- [quickstart.md](./quickstart.md) provides implementation order and exact
  Go/Vue/integration validation.
- Agent context is updated with the repository script after these artifacts
  are written.

## Phase 2: Implementation Planning

Implementation should proceed in dependency order:

1. **Foundation**: add application DTOs/enums, client/error mapping, injectable
   clock/cache/telemetry, settings parsing, scoring and exhaustive unit tests.
2. **Shared lookup API**: add orchestration and typed handlers, legacy adapter,
   `main.go` injection, Swagger generation, contract and `httptest` coverage.
3. **Photo integration**: make `CoinLookupService` return typed evidence/query
   proposal without eager Numista calls; add NGC-first and status regressions.
4. **Canonical-placement amendment**: after completed P1 status behavior, move
   the saved-coin panel into Catalog References, add the explicit NGC override,
   reconcile labels and draft-card chip, and prove the existing draft-list
   projection. This slice depends only on T001–T053 and is part of the P1 MVP;
   it does not depend on or modify Phase 6 caching/telemetry or Phase 7
   enrichment.
4. **Persistence**: add draft-reference model/migration/repository operations;
   extend create/update/read and promotion transaction; test ownership,
   validation rollback, collection/wishlist copy, repeated promotion, and no
   selection.
5. **Direct and photo UI**: shared panel, editable queries, broad-first
   enrichment, explanations, explicit selection, freshness/status states,
   retry/cancel, accessibility and mobile tests.
6. **Admin/operations**: typed admin health endpoint/view and settings fields;
   verify redaction, cache/quota metrics, invalid config fallback.
7. **Compatibility/docs/quality**: regenerate API artifacts, update lower
   authority docs, add ADR, run all targeted and full gates in quickstart,
   verify old GET response and old draft/photo inputs remain accepted.

Each slice must keep the broad lookup usable before adding the next layer.
Do not gate product correctness on a live Numista account.

## Testing Matrix

| Layer | Required coverage |
|---|---|
| Client | URL/header mapping, body cap, malformed/missing fields, 400/401/403/429/5xx, Retry-After, timeout, cancellation, one transient retry, no 429 retry |
| Cache | normalization equivalence, independent TTLs, fresh empty, expiry, coalescing, bounded eviction, config-before-cache |
| Scorer | every dimension, BCE/CE overlap/conflict, punctuation/mixed script, duplicate/long evidence, missing fields, equal-score stable ordering, explanation redaction |
| Service | six statuses, broad first, max-five default, partial/all detail failure, cache metadata, quota telemetry, no credential/raw text signals |
| Repository/migration | additive migration from old DB, owner scope, one reference, preserve/replace/remove, rollback, exact-once promotion to collection and wishlist |
| Handlers/contracts | auth/admin boundaries, typed 200 domain outcomes, 400 bounds, safe 500, legacy GET shape, Swagger route drift |
| Vue | editable initial/retry query, status guidance, progressive enrichment, stable selection outside new results, explicit add/remove, draft resume/promotion failure, keyboard/ARIA/mobile |
| Integration | direct selected reference only; photo → draft → edit → collection/wishlist promotion; no selection; repeated promotion; NGC-first; old frontend/backend compatibility |

## Complexity Tracking

No constitutional violations require justification.
