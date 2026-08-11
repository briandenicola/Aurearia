# Tasks: Improved Numista Lookup

**Input**: Design documents from `specs/341-improve-numista-lookup/`
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/numista-lookup.openapi.yaml`, `quickstart.md`
**Tests**: Required. Write each listed test before its corresponding implementation and confirm that it fails for the intended reason.

## Phase 1: Setup (Shared Test Infrastructure)

**Purpose**: Establish live-provider-free fixtures for backend, migration, and Vue work.

- [x] T001 Add sanitized Numista broad-search, detail, malformed-response, and quota-response fixtures in src/api/services/testdata/numista/
- [x] T002 [P] Add typed Numista request, candidate, outcome, and selected-reference factories in src/web/src/test/numista-fixtures.ts
- [x] T003 [P] Add a pre-feature Quick Capture schema fixture with active and promoted drafts in src/api/database/testdata/pre_numista_quick_capture.sql

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Build the typed provider boundary, deterministic primitives, settings, and test seams shared by every story.

**Critical**: No user-story implementation starts until this phase is complete.

- [x] T004 [P] Add validation and JSON-contract tests for lookup paths, evidence bounds, candidates, outcomes, enrichment states, relevance reasons, cache metadata, and health summaries in src/api/models/numista_test.go
- [x] T005 Implement standard-library-only application DTOs, enums, validation helpers, canonical Numista URLs, and bounded value objects in src/api/models/numista.go
- [x] T006 [P] Add httptest and fake-RoundTripper tests for provider URL/header mapping, response-size limits, malformed fields, 400/401/403/429/5xx mapping, Retry-After, cancellation, deadlines, one eligible retry, and forbidden retries in src/api/services/numista_client_test.go
- [x] T007 Implement NumistaClient, HTTPNumistaClient, private provider DTOs, typed safe error taxonomy, context cancellation, four/three-second deadlines, and one bounded transient retry in src/api/services/numista_client.go
- [x] T008 [P] Add fake-clock tests for search/detail namespaces, independent TTLs, fresh empty outcomes, expiry deletion, bounded eviction, hashed identities, and same-key in-flight coalescing in src/api/services/numista_cache_test.go
- [x] T009 Implement injectable-clock bounded TTL caches and cancellation-safe same-key request coalescing in src/api/services/numista_cache.go
- [x] T010 [P] Add table-driven scorer tests for every weighted dimension, neutral missing data, exact-ID precedence, BCE/CE ranges, conflicts, NFKC/mixed scripts, punctuation, duplicate/long evidence, safe reasons, and stable ties in src/api/services/numista_scoring_test.go
- [x] T011 Implement versioned numista-v1 normalization, date parsing, weighted scoring, relevance bands/reasons, redaction, and deterministic tie-breaking in src/api/services/numista_scoring.go
- [x] T012 [P] Add concurrency and aggregate tests for bounded telemetry, p50/p95, status/cache/enrichment/quota counts, empty-ring behavior, and rejection of secret or user-text fields in src/api/services/numista_telemetry_test.go
- [x] T013 Implement the thread-safe bounded Numista telemetry ring, safe correlation digests, and redacted health aggregation in src/api/services/numista_telemetry.go
- [x] T014 [P] Add default, valid-range, invalid-value fallback, and live-reload tests for all Numista TTL, limit, and timeout settings in src/api/services/settings_service_test.go
- [x] T015 Add validated Numista search/detail TTL, enrichment/result limit, and timeout settings with documented defaults and safe invalid-configuration signals in src/api/services/settings_service.go

**Checkpoint**: Provider access, scoring, caching, telemetry, and configuration are typed and independently testable without Numista access.

---

## Phase 3: User Story 1 - Find relevant matches from coin details (Priority: P1) MVP

**Goal**: Let a collector edit a rich direct-lookup query, receive deterministic explained candidates, and explicitly persist only the chosen reference.

**Independent Test**: Open a coin with name, ruler, denomination, mint, date, material, and inscriptions; edit the generated query; search; inspect ranked explanations; select one candidate; and verify exactly one canonical `CoinReference` is added.

### Tests for User Story 1

- [x] T016 [P] [US1] Add broad-lookup service tests for rich evidence, exact effective-query preservation, application-owned candidate mapping, deterministic initial scoring, unusable provider rows, and empty manual evidence in src/api/services/numista_lookup_service_test.go
- [x] T017 [P] [US1] Add authenticated POST lookup and deprecated GET adapter tests for request bounds, safe failures, legacy count/types shape, and Swagger annotations in src/api/handlers/numista_test.go
- [x] T018 [P] [US1] Add pure direct-query builder tests covering every coin field, empty omission, source-text preservation, length bounds, and editable retry values in src/web/src/utils/__tests__/numistaLookup.test.ts
- [x] T019 [P] [US1] Add component tests for direct query editing, explained ranking, explicit radio selection, replace/remove, outside-latest-result retention, and add-only-on-confirm behavior in src/web/src/components/coin/__tests__/CoinNumistaPanel.test.ts

### Implementation for User Story 1

- [x] T020 [US1] Implement broad search orchestration, application mapping, effective-query preservation, initial scoring, and safe success/empty outcomes in src/api/services/numista_lookup_service.go
- [x] T021 [US1] Replace handler-owned provider HTTP with typed POST lookup and a deprecated shared-service GET compatibility adapter in src/api/handlers/numista.go
- [x] T022 [P] [US1] Add Swagger request/response aliases for Numista application contracts and the preserved legacy response in src/api/handlers/swagger_types.go
- [x] T023 [US1] Construct one client, cache, scorer, telemetry recorder, and lookup service; inject them into authenticated Numista routes in src/api/main.go
- [x] T024 [P] [US1] Add exact TypeScript lookup/evidence/candidate/outcome/relevance/cache DTOs and typed broad/legacy API calls in src/web/src/types/index.ts and src/web/src/api/client.ts
- [x] T025 [P] [US1] Implement the pure rich-query builder, canonical candidate identity, selection retention, and role-safe status mapping helpers in src/web/src/utils/numistaLookup.ts
- [x] T026 [US1] Create the reusable editable lookup/results/selection panel with textual relevance reasons, canonical links, and explicit confirmation events in src/web/src/components/numista/NumistaLookupPanel.vue
- [x] T027 [US1] Refine direct lookup to use all coin evidence and persist only the confirmed selection through the existing structured-reference API in src/web/src/components/coin/CoinNumistaPanel.vue

**Checkpoint**: Direct lookup is a complete, independently testable MVP and the legacy GET route remains usable.

---

## Phase 4: User Story 2 - Identify a coin from photos (Priority: P1)

**Goal**: Produce an editable photo-evidence query without eager Numista access, retain one explicit selection on a Quick Capture draft, and copy it transactionally and idempotently during collection or wishlist promotion.

**Independent Test**: Analyze non-NGC photos, edit and submit the proposed query, select a candidate, save and resume a draft, survive an unrelated validation failure, and promote to collection and wishlist while verifying the exact selected reference is copied once; repeat with no selection and verify no reference.

### Tests for User Story 2

- [x] T028 [P] [US2] Add additive migration and rollback-compatibility tests from the pre-feature schema for active/promoted drafts with no selected relation in src/api/database/migration_test.go
- [x] T029 [P] [US2] Add model validation tests for optional selection, fixed Numista catalog, positive canonical number, generated HTTPS URI, mismatches, and omitted/clear conflicts in src/api/models/quick_capture_draft_test.go
- [x] T030 [P] [US2] Add repository transaction tests for create/preserve/replace/remove, owner isolation, rollback, discard history, collection/wishlist copy, duplicate guards, concurrent claims, and repeated promotion in src/api/repository/quick_capture_repository_test.go
- [x] T031 [P] [US2] Add Quick Capture service/handler compatibility tests for omitted legacy inputs, additive responses, validation failure preservation, explicit clear, and authorization in src/api/services/quick_capture_service_test.go and src/api/handlers/quick_capture_handler_test.go
- [x] T032 [P] [US2] Add photo-analysis tests for no image, usable NGC-first suppression, non-NGC typed evidence/query without a provider call, noisy/empty evidence, cancellation, and additive legacy aliases in src/api/services/coin_lookup_service_test.go
- [x] T033 [P] [US2] Add photo and draft API serialization tests for selected-reference create/read/update/clear payloads in src/web/src/api/__tests__/quickCaptureNumista.test.ts
- [x] T034 [P] [US2] Add Identify Coin tests for user-initiated camera behavior, editable first/retry query, no eager request, NGC-first behavior, explicit selection, and narrow mobile layout in src/web/src/pages/__tests__/CoinLookupPage.test.ts
- [x] T035 [P] [US2] Add draft resume tests for retained/outside-results selection, replace/remove, unrelated edit, validation failure, optional readiness, and failed/repeated promotion in src/web/src/pages/__tests__/QuickCaptureDraftPage.test.ts

### Implementation for User Story 2

- [x] T036 [US2] Register the additive selected-reference table and indexes in AutoMigrate without modifying existing rows in src/api/database/database.go
- [x] T037 [US2] Add the one-to-zero-or-one QuickCaptureDraftReference model and optional draft response relation in src/api/models/quick_capture_draft.go
- [x] T038 [US2] Implement owner-scoped preload/create/upsert/delete and exact-once CoinReference copy inside existing draft write and promotion transactions in src/api/repository/quick_capture_repository.go
- [x] T039 [US2] Implement canonical selected-reference validation and preserve/replace/remove semantics while delegating persisted-reference rules to existing services in src/api/services/quick_capture_service.go
- [x] T040 [US2] Extend Quick Capture create/read/update/promote DTO binding and safe validation responses without breaking omitted legacy fields in src/api/handlers/quick_capture.go
- [x] T041 [US2] Refactor photo analysis to return bounded NumistaEvidence and proposedNumistaQuery without eager provider access while preserving NGC-first behavior in src/api/services/coin_lookup_service.go
- [x] T042 [US2] Add photo response fields and keep deprecated numistaCandidates/candidateReferences additive and free of unselected Numista results in src/api/handlers/coin_lookup.go
- [x] T043 [P] [US2] Add selected-reference, photo proposal, draft, and promotion DTOs plus typed multipart request mapping in src/web/src/types/index.ts and src/web/src/api/client.ts
- [x] T044 [US2] Integrate the shared lookup panel into Identify Coin with editable evidence queries, explicit selection, retained retries, and no automatic Numista request in src/web/src/pages/CoinLookupPage.vue
- [x] T045 [US2] Display, preserve, replace, and remove the optional selection across draft editing and promotion readiness in src/web/src/pages/QuickCaptureDraftPage.vue

**Checkpoint**: The photo-to-draft-to-coin workflow persists exactly one explicit selection or none, without changing NGC-first behavior.

---

## Phase 5: User Story 3 - Understand lookup availability (Priority: P1)

**Goal**: Distinguish all six lookup outcomes in both paths with safe, actionable, role-aware guidance while preserving the editable query and draft selection.

**Independent Test**: Simulate success, fresh/cached empty, unconfigured, 429 with Retry-After, deadline, malformed/provider failure, and caller cancellation from direct and photo lookup; verify the correct state and guidance, no secret/internal detail, and unchanged query/selection.

### Tests for User Story 3

- [x] T046 [P] [US3] Add service tests for all six domain statuses, caller cancellation, Retry-After propagation, configuration-before-cache, and generic internal failures in src/api/services/numista_lookup_status_test.go
- [x] T047 [P] [US3] Add handler tests for HTTP 200 domain outcomes, 400 validation, 401 authentication, safe 500 responses, and admin/non-admin guidance boundaries in src/api/handlers/numista_status_test.go
- [x] T048 [P] [US3] Add status-helper and panel tests for all six states, retained query/selection, retry actions, role-safe configuration links, aria-live announcements, and non-color guidance in src/web/src/components/numista/__tests__/NumistaLookupPanel.status.test.ts

### Implementation for User Story 3

- [x] T049 [US3] Complete typed error-to-domain outcome mapping, guidance codes, Retry-After handling, configuration-before-cache, and cancellation recording in src/api/services/numista_lookup_service.go
- [x] T050 [US3] Return expected domain states without leaking raw errors or configuration details and preserve legacy GET failure semantics in src/api/handlers/numista.go
- [x] T051 [P] [US3] Implement complete role-sensitive status labels, guidance, retry eligibility, and cache freshness text in src/web/src/utils/numistaLookup.ts
- [x] T052 [US3] Render explicit idle/loading/success/empty/unconfigured/quota-limited/timeout/unavailable states with focus management and retained input in src/web/src/components/numista/NumistaLookupPanel.vue
- [x] T053 [US3] Add cross-path regression coverage proving status transitions never clear edited queries or persisted draft selections in src/web/src/pages/__tests__/NumistaStatusWorkflows.test.ts

**Checkpoint**: Collectors can reliably choose whether to edit, configure, wait, or retry without seeing sensitive details.

---

## Phase 5A: User Story 6 - Canonical catalog-reference placement (Priority: P1 MVP amendment)

**Goal**: Make Catalog References the canonical saved-coin lookup surface,
retain contextual Identify Coin/Quick Capture lookup, provide an explicit
no-eager-request NGC override, reconcile labels and draft-card reference
visibility, and add no top-level navigation.

**Dependencies**: Depends on completed T001–T053 only. This amendment is part of
the P1 MVP and is independent of Phase 6 cache/telemetry and Phase 7
enrichment. It does not reopen or renumber completed work.

**Independent Test**: From a saved coin, expand `Search Numista` beside `Add
Reference`, persist one selection, and observe inline collapse plus the new
reference. From Identify Coin, verify contextual non-NGC lookup, reveal
`Also search Numista` under a usable NGC result with zero provider requests,
save a draft, and observe `Numista #<identifier>` on its list card.

### Tests for User Story 6

- [x] T087 [P] [US6] [Brutus] Add canonical placement tests for compact peer actions, inline disclosure semantics/focus, manual reference compatibility, selected-reference persistence refresh, and collapse-after-success in src/web/src/components/coin/__tests__/CoinReferencesSection.test.ts
- [x] T088 [P] [US6] [Brutus] Add transition tests proving Actions renders no full Numista panel and at most one compact contextual link to Catalog References without adding a route/navigation destination in src/web/src/components/coin/__tests__/CoinActionsPanel.test.ts and src/web/src/pages/__tests__/CoinDetailPage.test.ts
- [x] T089 [P] [US6] [Brutus] Add Identify Coin tests for retained non-NGC contextual lookup, `Also search Numista` under usable NGC results, editable reveal with zero eager Numista requests, `Analyze Photos`, retained `Save as Draft`, keyboard disclosure, and 375 px layout in src/web/src/pages/__tests__/CoinLookupPage.test.ts
- [x] T090 [P] [US6] [Brutus] Add owner-scoped draft-list contract tests proving selectedNumistaReference is preloaded/serialized when present and omitted or null otherwise in src/api/repository/quick_capture_repository_test.go, src/api/services/quick_capture_service_test.go, and src/api/handlers/quick_capture_handler_test.go
- [x] T091 [P] [US6] [Brutus] Add draft-card tests for exact `Numista #<identifier>` chip text, absence without selection, wrapping, and accessible link/card behavior in src/web/src/components/quick-capture/__tests__/QuickCaptureDraftCard.test.ts and src/web/src/pages/__tests__/QuickCaptureDraftsPage.test.ts

### Implementation for User Story 6

- [x] T092 [US6] [Aurelia] Compose CoinNumistaPanel as an inline disclosure in Catalog References beside the existing manual Add Reference action, pass complete coin evidence from src/web/src/pages/CoinDetailPage.vue, and collapse only after confirmed persistence plus refresh in src/web/src/components/coin/CoinReferencesSection.vue and src/web/src/pages/CoinDetailPage.vue
- [x] T093 [US6] [Aurelia] Remove CoinNumistaPanel from Actions and add at most one compact contextual row/link targeting the saved coin's Catalog References section, without changing router or top-level navigation definitions, in src/web/src/components/coin/CoinActionsPanel.vue and src/web/src/pages/CoinDetailActionsPage.vue
- [x] T094 [US6] [Aurelia] Preserve contextual non-NGC lookup, add the explicit NGC `Also search Numista` disclosure without eager lookup, rename the initial action to `Analyze Photos`, and retain `Save as Draft` in src/web/src/pages/CoinLookupPage.vue
- [x] T095 [US6] [Aurelia] Render the retained `Numista #<identifier>` chip from selectedNumistaReference without new API calls in src/web/src/components/quick-capture/QuickCaptureDraftCard.vue
- [x] T096 [US6] [Maximus] Reconcile canonical placement, contextual surfaces, NGC override/no-eager behavior, labels, draft-card chip, no-navigation scope, and transition compatibility in docs/features/numista-integration.md and docs/quick-capture.md; run the targeted Go/Vitest checks plus npm run type-check and npm run build from specs/341-improve-numista-lookup/quickstart.md

**Checkpoint**: The P1 MVP matches the collector mental model “find catalog
reference”; saved-coin lookup is canonical under Catalog References, contextual
draft lookup remains available, and no Phase 6/7 behavior changed.

---

## Phase 6: User Story 4 - Conserve quota without hiding freshness (Priority: P2)

**Goal**: Reuse fresh equivalent work across paths, expose freshness, and give administrators redacted configuration, quota, latency, cache, and enrichment health.

**Independent Test**: Repeat normalized direct/photo searches concurrently, verify one provider call and visible freshness metadata, expire entries with a fake clock, remove configuration, and inspect complete redacted admin health aggregates.

### Tests for User Story 4

- [x] T054 [P] [US4] Add shared-service cache tests for normalized cross-path reuse, fresh empty results, expiry refresh, concurrent coalescing, setting changes, removed credentials, and independent detail/search TTLs in src/api/services/numista_lookup_cache_test.go
- [x] T055 [P] [US4] Add telemetry integration tests for every status, cache hit/refresh, broad/detail counts, retry/quota timing, latency percentiles, bounded retention, and absence of keys/query/inscriptions/labels/raw errors in src/api/services/numista_lookup_telemetry_test.go
- [x] T056 [P] [US4] Add admin-only health endpoint tests for empty and populated summaries, invalid configuration, 401/403 boundaries, and redacted JSON in src/api/handlers/admin_numista_test.go
- [x] T057 [P] [US4] Add admin component tests for validated settings, status counts, p50/p95, cache/enrichment/quota signals, no estimated remaining quota, and no sensitive text in src/web/src/components/admin/__tests__/AdminSystemSection.numista.test.ts

### Implementation for User Story 4

- [x] T058 [US4] Integrate normalized hashed cache identities, fresh success/empty reuse, expiry refresh, and same-key coalescing into lookup operations in src/api/services/numista_lookup_service.go
- [x] T059 [US4] Record redacted broad/detail/status/cache/retry/quota/enrichment events and expose aggregate snapshots in src/api/services/numista_lookup_service.go and src/api/services/numista_telemetry.go
- [x] T060 [US4] Implement the admin-only redacted Numista configuration and rolling health handler with Swagger annotations in src/api/handlers/admin_numista.go
- [x] T061 [US4] Register the health route under the existing admin authorization group and inject the shared telemetry/settings dependencies in src/api/main.go
- [x] T062 [P] [US4] Add Numista settings and health summary TypeScript contracts plus admin API calls in src/web/src/types/index.ts and src/web/src/api/client.ts
- [x] T063 [US4] Add bounded Numista configuration controls and redacted operational health cards to the existing system settings surface in src/web/src/components/admin/AdminSystemSection.vue

**Checkpoint**: Equivalent work conserves provider allowance, stale data is never shown as fresh, and administrators have safe operational visibility.

---

## Phase 7: User Story 5 - Review useful details without excessive requests (Priority: P2)

**Goal**: Paint broad candidates first, enrich only a server-selected bounded leading subset, rerank deterministically, and retain broad candidates when details fail.

**Independent Test**: Search for more candidates than the configured limit, verify broad results render before enrichment, at most five details are requested by default with concurrency two, enriched reasons update ranking, explicit selection remains stable, and partial/all detail failures retain every broad candidate.

### Tests for User Story 5

- [ ] T064 [P] [US5] Add provider detail tests for validated IDs, application-needed field mapping, canonical/image URL safety, malformed optional fields, cacheability, timeout, cancellation, and transient retry policy in src/api/services/numista_client_detail_test.go
- [ ] T065 [P] [US5] Add enrichment tests for server-side reranking before selection, unique IDs, 1-50 bounds, default/configured cap, concurrency two, cached details, cancellation, deterministic rerank, and partial/all failure retention in src/api/services/numista_enrichment_test.go
- [ ] T066 [P] [US5] Add authenticated enrichment contract tests for invalid/reordered/duplicated client candidates, provider-call suppression on 400, full broad-set responses, safe failures, and route documentation in src/api/handlers/numista_enrichment_test.go
- [ ] T067 [P] [US5] Add component tests proving broad-first paint, enrichment progress, cached/enriched/failed labels, reason updates, stable explicit selection, keyboard operation, image alt text, and mobile stacking in src/web/src/components/numista/__tests__/NumistaLookupPanel.enrichment.test.ts

### Implementation for User Story 5

- [ ] T068 [US5] Implement typed GET /types/{id} mapping, HTTPS image validation, response bounds, cancellation, and retry rules in src/api/services/numista_client.go
- [ ] T069 [US5] Implement server-capped two-stage enrichment with concurrency two, independent detail caching, all-candidate retention, reranking, and safe partial degradation in src/api/services/numista_lookup_service.go
- [ ] T070 [US5] Implement authenticated POST enrichment validation and full-outcome mapping with Swagger annotations in src/api/handlers/numista.go
- [ ] T071 [P] [US5] Add typed enrichment request/response calls and cancellation support in src/web/src/types/index.ts and src/web/src/api/client.ts
- [ ] T072 [US5] Trigger enrichment only after broad paint, display progressive states/reasons, and avoid silently changing explicit selection in src/web/src/components/numista/NumistaLookupPanel.vue

**Checkpoint**: Detail data improves comparison without delaying discovery or multiplying provider usage.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Prove security, compatibility, performance, contracts, documentation, rollout readiness, and the complete quality gate.

- [ ] T073 [P] Add backend workflow integration tests for direct selected-only persistence and photo draft-to-collection/wishlist promotion with no-selection and repeated-promotion cases in src/api/integration/numista_workflows_test.go
- [ ] T074 [P] Add compatibility regression tests for legacy GET clients, additive photo fields, old draft inputs, NGC references, ownership, structured-reference deduplication, and rollback-readable records in src/api/integration/numista_compatibility_test.go
- [ ] T075 [P] Add API-key secrecy and privacy tests covering headers only, errors, logs, responses, cache keys, telemetry, scorer explanations, canonical links, auth/admin boundaries, and oversized input/body rejection in src/api/integration/numista_security_test.go
- [ ] T076 [P] Add deterministic fake-provider workload tests for 5-second uncached p95, 1-second fresh-cache p95, at least 80% broad-call reduction, top-three scoring fixtures, and default five-detail ceiling in src/api/integration/numista_performance_test.go
- [ ] T077 Regenerate Swagger with task openapi and reconcile route/schema drift in src/api/docs/docs.go, src/api/docs/swagger.json, src/api/docs/swagger.yaml, and docs/openapi.json
- [ ] T078 [P] Reconcile explicit-selection, editable-query, status, caching, enrichment, and NGC-first behavior in docs/features/numista-integration.md and docs/quick-capture.md
- [ ] T079 [P] Document typed endpoints, compatibility rollout, settings defaults/ranges, redacted health signals, backend-first deployment, observation, and rollback in docs/api-reference.md and docs/deployment.md
- [ ] T080 [P] Record the shared client/cache/scoring/telemetry boundary and additive selected-reference migration as a Nygard ADR in docs/adr/0007-shared-numista-lookup.md
- [ ] T081 Run go build ./..., go vet ./..., go test ./... -count=1, and go test -run TestArchitecture ./... from src/api/ and resolve all failures in src/api/
- [ ] T082 Run npm run test -- --run, npm run type-check, vue-tsc --build, and npm run build from src/web/ and resolve all failures in src/web/
- [ ] T083 [P] Run ruff check app/ tests/ and pytest tests/ -v from src/agent/ to prove the untouched service remains green, recording any environment-only exception in specs/341-improve-numista-lookup/quickstart.md
- [ ] T084 Run task --list and applicable OpenAPI/documentation checks from Taskfile.yml, then complete every direct/photo/status/cache/enrichment/admin/manual rollout walkthrough in specs/341-improve-numista-lookup/quickstart.md
- [ ] T085 Run gitleaks against the repository and Trivy against the production container, requiring no leaked secrets and no High/Critical findings; resolve configuration findings in .gitleaks.toml and Dockerfile without weakening policy
- [ ] T086 Verify conventional-commit/co-author expectations for the future PR, Constitution Principles I-IX, §17, §21, workflow/blast-radius evidence, no generated build artifacts, and the full Definition of Done in .github/pull_request_template.md

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies.
- **Phase 2 (Foundational)**: Depends on Phase 1 and blocks all stories.
- **Phase 3 (US1)**: Depends on Phase 2; establishes the MVP shared lookup and panel.
- **Phase 4 (US2)**: Depends on Phase 2 and reuses US1's lookup service/panel; its persistence slice is independently testable through API tests.
- **Phase 5 (US3)**: Depends on US1's shared service/panel and should follow US2 to prove selection preservation across both paths.
- **Phase 5A (US6 amendment)**: Depends on completed T001–T053 and closes the
  P1 MVP UX. It has no dependency on Phase 6 or Phase 7 and must not change
  their cache/telemetry or enrichment scope.
- **Phase 6 (US4)**: Depends on Phase 2 primitives and US1 service orchestration; admin UI is independent of US2 persistence.
- **Phase 7 (US5)**: Depends on US1 broad lookup and Phase 2 client/cache/scorer; it does not depend on Quick Capture persistence.
- **Phase 8 (Polish)**: Depends on all stories selected for release.

### User Story Completion Graph

```text
Setup -> Foundational -> US1 (MVP)
                         |-> US2 -> US3
                         |          `-> US6 placement amendment (P1 MVP)
                         |-> US4
                         `-> US5
US2 + US3 + US6 + US4 + US5 -> Polish
```

### Within Each Story

1. Add the listed failing tests.
2. Implement models/services before handlers and routes.
3. Update exact Go/TypeScript contracts before consuming them in views.
4. Complete the independent test before starting the next priority slice.

## Parallel Execution Examples

- **US1**: T016, T017, T018, and T019 can run concurrently; after T020-T023, T024 and T025 can run concurrently before T026-T027.
- **US2**: T028-T035 are independent test files and can run concurrently; T041-T042 can proceed beside T036-T040; T043 can proceed before T044-T045.
- **US3**: T046, T047, and T048 can run concurrently; T051 can proceed beside backend work T049-T050.
- **US6**: T087-T091 can run concurrently; after they fail for the intended
  reasons, T092-T095 can proceed by owned surface, followed by T096.
- **US4**: T054-T057 can run concurrently; T062 can proceed beside T058-T061 before T063.
- **US5**: T064-T067 can run concurrently; T071 can proceed beside T068-T070 before T072.

## Requirements-to-Task Coverage

| Requirement | Covered by tasks |
|---|---|
| FR-001 shared typed capability | T004-T007, T020-T024 |
| FR-002 application-owned candidates | T004-T007, T016, T020 |
| FR-003 direct rich query | T018, T025-T027 |
| FR-004 photo evidence query | T032, T041-T044 |
| FR-005 edit before first search/retry | T018-T019, T026-T027, T034, T044 |
| FR-006 exact effective query | T016, T020, T048-T053 |
| FR-007 application scoring | T010-T011, T016, T020 |
| FR-008 all scoring evidence | T010-T011, T016, T065 |
| FR-009 explanations/no attribution claim | T010-T011, T019, T026, T067, T072 |
| FR-010 deterministic ranking | T010-T011, T016, T065, T069 |
| FR-011 explicit select/replace/remove | T019, T026-T027, T034-T035, T044-T045 |
| FR-012 direct selected-only persistence | T019, T027, T073 |
| FR-013 draft selection survival | T029-T031, T035, T037-T045 |
| FR-014 transactional idempotent promotion | T030-T031, T038-T040, T073 |
| FR-015 never infer or persist unselected | T019, T030-T032, T038-T044, T073-T074 |
| FR-016 six outcomes in both paths | T046-T053 |
| FR-017 safe state guidance | T046-T052, T075 |
| FR-018 broad then enrichment | T064-T072 |
| FR-019 configurable bounded subset | T014-T015, T063, T065, T069 |
| FR-020 useful details and failure retention | T064-T072 |
| FR-021 shared fresh reuse | T008-T009, T054, T058 |
| FR-022 independent configurable TTLs | T008-T009, T014-T015, T054, T058, T063 |
| FR-023 visible freshness/no stale data | T048, T051-T052, T054, T058, T067, T072 |
| FR-024 live configuration precedence | T014-T015, T046, T049, T054, T058 |
| FR-025 redacted operational signals | T012-T013, T055, T059, T075 |
| FR-026 admin health visibility | T056-T063 |
| FR-027 NGC-first behavior | T032, T041-T044, T074 |
| FR-028 auth and ownership | T017, T030-T031, T038-T040, T047, T056, T066, T074-T075 |
| FR-029 reference validation/deduplication | T029-T031, T038-T040, T073-T075 |
| FR-030 canonical Catalog References placement | T087-T088, T092-T093 |
| FR-031 inline expand/collapse after persistence | T087, T092 |
| FR-032 no full Actions panel/compact transition link | T088, T093 |
| FR-033 contextual surfaces/no top-level navigation | T088-T089, T093-T094, T096 |
| FR-034 NGC explicit override/no eager request | T089, T094 |
| FR-035 Analyze Photos/Save as Draft labels | T089, T094 |
| FR-036 retained draft-card Numista chip | T090-T091, T095 |
| FR-037 accessibility/mobile behavior | T087-T089, T091-T096 |
| FR-038 transition compatibility | T087-T090, T092-T096 |
| NFR-001/NFR-002 latency | T007, T076 |
| NFR-003 progressive bounded enrichment | T065-T072, T076 |
| NFR-009 disclosure focus/375 px overflow | T087, T089, T091-T095 |
| NFR-004 understandable uncertainty | T010-T011, T019, T026, T067, T072 |
| NFR-005/NFR-006 secrecy and redaction | T006-T013, T046-T052, T055-T060, T075 |
| NFR-007 no live-provider dependency | T001-T015, T016-T076 |
| NFR-008 accessible mobile/PWA | T019, T026, T034, T044-T045, T048, T052, T067, T072, T082 |
| SC-001 | T018-T019, T025-T027, T073 |
| SC-002 | T010-T011, T065, T069, T076 |
| SC-003 | T028-T045, T073 |
| SC-004 | T019, T027, T030-T044, T073 |
| SC-005 | T046-T053 |
| SC-006 | T008-T009, T054, T058, T076 |
| SC-007 | T064-T072, T076 |
| SC-008 | T007, T054, T058, T064-T072, T076 |
| SC-009 | T012-T013, T055-T063, T075 |
| SC-010 | T028-T045, T073-T075, T081-T085 |
| SC-011 | T087-T088, T092-T093, T096 |
| SC-012 | T089, T094, T096 |
| SC-013 | T090-T091, T095-T096 |
| SC-014 | T087-T089, T091-T096 |

## Implementation Strategy

### MVP First

1. Historical initial MVP: Setup, Foundational, and US1 (completed T001–T027).
2. Completed P1 workflow expansion: US2 and US3 (completed T028–T053).
3. Current MVP UX reconciliation: execute US6 T087–T096 test-first, then
   validate canonical placement, contextual photo/draft access, labels, and
   draft-card reference visibility.
4. Stop and validate the amended P1 MVP before starting or changing Phase 6
   caching/telemetry or Phase 7 enrichment.

### Incremental Delivery

1. **US1**: Shared broad direct lookup and explicit reference selection.
2. **US2**: Photo proposal and durable Quick Capture selection/promotion.
3. **US3**: Complete status guidance across both workflows.
4. **US4**: Shared quota conservation and admin operations.
5. **US5**: Progressive bounded detail enrichment.
6. **Polish**: Security, compatibility, performance, docs, rollout, and all quality gates.

### Safe Rollout

1. Deploy additive migration and backend contracts before the new SPA.
2. Keep the deprecated GET adapter and additive photo/draft fields for one release.
3. Observe status mix, p95 latency, cache hit rate, 429s, and enrichment failures.
4. Roll back the SPA independently if needed; old binaries ignore the additive table and persisted `CoinReference` rows remain readable.
