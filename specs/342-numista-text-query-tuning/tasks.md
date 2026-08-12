# Tasks: Measured Numista Text-Query Tuning

**Input**: `spec.md`, `plan.md`, `contracts/numista-query.openapi.yaml`
**Tests**: Required and written before corresponding implementation.

## Phase 1: Freeze Evidence and Contracts

- [x] T001 [P] Add at least 12 sanitized old/primary/relaxed query cases and
  frozen candidate responses in `src/api/services/testdata/numista/`.
- [x] T002 [P] Record sanitized live-query comparison evidence, observation
  date, expected candidate IDs, and ranks in
  `specs/342-numista-text-query-tuning/live-evidence.md`; include no images,
  credentials, owner data, raw slab text, or full prose.
- [x] T003 Add proposal/query-source/attempt schemas to Swagger aliases,
  `docs/openapi.json`, generated API docs, and the feature contract; confirm
  route drift fails before implementation.

## Phase 2: Canonical Go Query Builder

- [x] T004 [P] Add table-driven tests in
  `src/api/services/numista_query_test.go` for component order, exclusions,
  bounds, `reverseType`, primary/relaxed plans, Unicode normalization, exact
  `SMN`/`SMNT` aliases, unknown alias-like text, and the 32-entry ceiling.
- [x] T005 Add `reverseType` and typed proposal/source/attempt DTOs and
  validation in `src/api/models/numista.go`.
- [x] T006 Implement the pure injected `numista-query-v2` builder and exact
  mint alias map in `src/api/services/numista_query.go`.
- [x] T007 Add authenticated proposal handler tests for auth, body bounds,
  unknown fields, typed output, and zero provider/telemetry calls in
  `src/api/handlers/numista_test.go`.
- [x] T008 Implement `POST /api/numista/query-proposal`, Swagger annotations,
  route registration, and DI in `src/api/handlers/numista.go` and
  `src/api/main.go`.

## Phase 3: Generated-Only Relaxed Retry

- [x] T009 [P] Add lookup service tests for verified/stale generated markers,
  sticky-edited/manual semantics, exact query preservation, every no-fallback
  status, one relaxed retry after empty only, and distinct-query guard in
  `src/api/services/numista_lookup_service_test.go`.
- [x] T010 [P] Add cache tests proving primary/relaxed key independence,
  cached-primary-empty fallback, and effective-attempt cache metadata in
  `src/api/services/numista_lookup_cache_test.go`.
- [x] T011 [P] Add telemetry tests for safe source/attempt attribution,
  separate operations, aggregate counts, and prohibited raw text/images in
  `src/api/services/numista_lookup_telemetry_test.go`.
- [x] T012 Implement server verification, one relaxed attempt, effective-query
  reporting, and source/attempt telemetry in
  `src/api/services/numista_lookup_service.go`.
- [x] T013 Preserve one-query exact behavior for deprecated
  `GET /api/numista/search` and add regression coverage in
  `src/api/handlers/numista_test.go`.

## Phase 4: Direct, Photo, and Draft Integration

- [x] T014 [P] Add TypeScript DTO/API tests for proposal and additive lookup
  metadata in `src/web/src/api/__tests__/client.test.ts`.
- [x] T015 [P] Add panel tests for generated, sticky user-edited, manual,
  stale proposal, parent refresh, and relaxed effective-query disclosure in
  `src/web/src/components/numista/__tests__/NumistaLookupPanel*.test.ts`.
- [x] T016 [P] Add direct tests proving server proposal use and no TypeScript
  query assembly in `src/web/src/components/coin/__tests__/CoinNumistaPanel.test.ts`.
- [x] T017 [P] Add photo/NGC tests proving shared builder output, no eager
  provider call, and manual NGC override in
  `src/api/services/coin_lookup_service_test.go` and
  `src/web/src/pages/__tests__/CoinLookupPage.test.ts`.
- [x] T018 [P] Add draft tests proving proposal endpoint use, retained
  selection, and unchanged promotion in
  `src/web/src/pages/__tests__/QuickCaptureDraftPage.test.ts`.
- [x] T019 Replace frontend query assembly with evidence-only proposal loading
  in `src/web/src/utils/numistaLookup.ts`,
  `src/web/src/components/coin/CoinNumistaPanel.vue`, and
  `src/web/src/pages/QuickCaptureDraftPage.vue`.
- [x] T020 Update `NumistaLookupPanel.vue` with exact marker semantics and
  separate relaxed effective-query reporting without changing selection,
  focus, enrichment, or accessibility behavior.
- [x] T021 Reuse the injected Go builder in
  `src/api/services/coin_lookup_service.go`; remove the divergent photo query
  builder while retaining extraction and NGC suppression.

## Phase 5: Measurement and Release Gate

- [x] T022 Reconcile user-facing and developer documentation for concise
  generated queries, exact `SMN`/`SMNT` aliases, scorer-retained excluded
  evidence, generated/edited/manual behavior, generated-only fallback,
  effective-query disclosure, no image search, and measured limitations.
- [x] T023 Re-run the existing 24-known-coin benchmark and prove at least 85%
  top-three accuracy without exact-ID evidence.
- [x] T024 [P] Update `docs/features/numista-integration.md` with generated,
  edited, manual, fallback, alias, privacy, and no-image-search behavior.
- [x] T025 Run targeted Go/Vitest tests, `go build ./...`, `go vet ./...`,
  `go test ./...`, `vue-tsc --build`, `npm run build`, OpenAPI generation and
  route drift, secret scan, and `git diff --check`.
- [ ] T026 Maximus reviews the fixture/live comparison, alias allowlist,
  request ceiling, NGC/no-eager regressions, and explicit absence of image
  search before release approval.

## Dependencies

- T001–T003 freeze the baseline and contract before implementation.
- T004–T008 block lookup fallback and all frontend integration.
- T009–T013 block T015 and T020.
- T014–T021 must complete before measurement T022–T026.
