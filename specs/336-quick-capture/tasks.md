# Tasks: Quick Capture

**Input**: Design documents from `specs/336-quick-capture/`
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/quick-capture-api.md`, `quickstart.md`
**Feature**: `336-quick-capture`

**Tests**: Required for this feature because the specification and quickstart explicitly require regression coverage for collection counts, wishlist totals, sold totals, promotion idempotency, media ownership, existing add/edit image behavior, and frontend mobile/PWA workflow.

**Organization**: Tasks are grouped by user story so each story can be implemented and validated as an independently testable increment after shared foundation work is complete.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel with other tasks in the same phase because it touches different files or depends only on completed earlier phases
- **[Story]**: User story label for story phases only (`[US1]`, `[US2]`, `[US3]`, `[US4]`)
- Every task includes an exact repository-relative file path

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish feature-specific scaffolding, typed contract placeholders, and test locations without changing runtime behavior.

- [X] T001 Create quick-capture backend source files with package declarations in `src/api/models/quick_capture_draft.go`, `src/api/repository/quick_capture_repository.go`, `src/api/services/quick_capture_service.go`, and `src/api/handlers/quick_capture.go`
- [X] T002 [P] Create quick-capture backend test files with failing/skipped-free test skeletons in `src/api/repository/quick_capture_repository_test.go`, `src/api/services/quick_capture_service_test.go`, and `src/api/handlers/quick_capture_handler_test.go`
- [X] T003 [P] Create frontend quick-capture directories and placeholder component files in `src/web/src/components/quick-capture/QuickCaptureForm.vue`, `src/web/src/components/quick-capture/QuickCaptureImageSlots.vue`, `src/web/src/components/quick-capture/QuickCaptureDraftCard.vue`, and `src/web/src/components/quick-capture/PromotionReadinessPanel.vue`
- [X] T004 [P] Create frontend quick-capture page placeholders in `src/web/src/pages/QuickCapturePage.vue`, `src/web/src/pages/QuickCaptureDraftsPage.vue`, and `src/web/src/pages/QuickCaptureDraftPage.vue`
- [X] T005 [P] Create frontend quick-capture test files with failing/skipped-free test skeletons in `src/web/src/components/quick-capture/__tests__/QuickCaptureForm.test.ts`, `src/web/src/components/quick-capture/__tests__/QuickCaptureImageSlots.test.ts`, `src/web/src/components/quick-capture/__tests__/QuickCaptureDraftCard.test.ts`, `src/web/src/components/quick-capture/__tests__/PromotionReadinessPanel.test.ts`, `src/web/src/pages/__tests__/QuickCapturePage.test.ts`, `src/web/src/pages/__tests__/QuickCaptureDraftsPage.test.ts`, and `src/web/src/pages/__tests__/QuickCaptureDraftPage.test.ts`
- [X] T006 [P] Add Quick Capture API contract references and manual QA checklist links to `specs/336-quick-capture/quickstart.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Build shared model, persistence, validation, media, and route foundation required by all Quick Capture stories.

**⚠️ CRITICAL**: No user story implementation can begin until this phase is complete.

### Tests

- [X] T007 [P] Add model validation tests for draft statuses, image types, save identity rules, and lifecycle event safety in `src/api/models/quick_capture_draft_test.go`
- [X] T008 [P] Add migration coverage asserting `quick_capture_drafts`, `quick_capture_draft_images`, and `draft_lifecycle_events` AutoMigrate cleanly in `src/api/database/migration_test.go`
- [X] T009 [P] Add reusable image validation tests for extension, size, image type, and magic-byte/content checks shared by coin and draft uploads in `src/api/services/image_service_test.go`
- [X] T010 [P] Add media lookup repository tests for owner-scoped draft image authorization and non-owner denial in `src/api/repository/image_repository_test.go`

### Implementation

- [X] T011 Define `QuickCaptureDraft`, `QuickCaptureDraftImage`, `DraftLifecycleEvent`, draft status constants, and lifecycle event constants in `src/api/models/quick_capture_draft.go`
- [X] T012 Register Quick Capture models in GORM AutoMigrate without altering existing coin migrations in `src/api/database/database.go`
- [X] T013 Implement owner-scoped quick-capture repository primitives for list/get/create/update/discard/image/lifecycle operations in `src/api/repository/quick_capture_repository.go`
- [ ] T014 Extract reusable image extension, type, size, magic-byte validation, safe-path, and file-save helpers from coin upload behavior in `src/api/services/image_service.go`
- [X] T015 Extend image repository media metadata lookup for `QuickCaptureDraftImage` ownership in `src/api/repository/image_repository.go`
- [X] T016 Extend authenticated upload resolution to authorize draft images owned by the viewer while preserving coin/avatar/showcase behavior in `src/api/services/image_service.go`
- [X] T017 Add Quick Capture TypeScript status/image/request/response types matching `contracts/quick-capture-api.md` in `src/web/src/types/index.ts`
- [X] T018 Wire Quick Capture repository, service, handler, and protected route group placeholders in `src/api/main.go`

**Checkpoint**: Foundation ready when model/migration/media validation tests fail for expected missing story behavior but compile cleanly; user story implementation can now proceed.

---

## Phase 3: User Story 1 - Save a minimal coin intake draft quickly (Priority: P1) 🎯 MVP

**Goal**: An authenticated collector can open mobile/PWA Quick Capture, attach obverse/reverse photos or identifying text, save a sparse draft, and verify it is incomplete and excluded from normal collection counts.

**Independent Test**: In a 375px mobile/PWA viewport, open Quick Capture, add obverse and reverse images or a working note, save the draft, and verify the draft is owned by the current user, marked active/incomplete, and not included in `/coins`, `/stats`, wishlist totals, sold totals, or collection health calculations.

### Tests for User Story 1

- [X] T019 [P] [US1] Add repository tests for owner-scoped create and image association of active drafts in `src/api/repository/quick_capture_repository_test.go`
- [X] T020 [P] [US1] Add service tests for partial-save validation requiring title, note, or image and rejecting invalid price/images in `src/api/services/quick_capture_service_test.go`
- [X] T021 [P] [US1] Add handler contract tests for `POST /api/quick-capture/drafts` multipart create success and validation failures in `src/api/handlers/quick_capture_handler_test.go`
- [ ] T022 [P] [US1] Add regression tests proving draft creation changes active collection count, wishlist total, sold total, stats, and health eligible counts by zero in `src/api/handlers/quick_capture_handler_test.go`
- [X] T023 [P] [US1] Add frontend API client tests for multipart create form data and validation error handling in `src/web/src/api/__tests__/client.test.ts`
- [ ] T024 [P] [US1] Add mobile/PWA component tests for photo-first obverse/reverse slots, file-upload fallback, camera unavailable messaging, and save disabled/enabled states in `src/web/src/components/quick-capture/__tests__/QuickCaptureForm.test.ts`
- [X] T025 [P] [US1] Add page test for 375px Quick Capture save workflow and incomplete draft messaging in `src/web/src/pages/__tests__/QuickCapturePage.test.ts`

### Implementation for User Story 1

- [X] T026 [US1] Implement draft create request parsing, multipart image extraction, validation responses, and Swagger annotations in `src/api/handlers/quick_capture.go`
- [X] T027 [US1] Implement `CreateDraft` service orchestration with partial identity validation, image validation reuse, lifecycle event creation, and safe user-facing errors in `src/api/services/quick_capture_service.go`
- [X] T028 [US1] Implement repository create, image insert, lifecycle insert, and owner preload behavior used by draft creation in `src/api/repository/quick_capture_repository.go`
- [X] T029 [US1] Register `POST /api/quick-capture/drafts` behind authenticated write-rate-limited routes in `src/api/main.go`
- [X] T030 [US1] Add typed `createQuickCaptureDraft` API client method using `FormData` and typed responses in `src/web/src/api/client.ts`
- [X] T031 [US1] Implement reusable obverse/reverse/detail photo slots with preview, remove, file input fallback, accepted-image hints, and validation display in `src/web/src/components/quick-capture/QuickCaptureImageSlots.vue`
- [X] T032 [US1] Implement sparse mobile-first Quick Capture form fields, save state, retry messaging, and incomplete-draft confirmation in `src/web/src/components/quick-capture/QuickCaptureForm.vue`
- [X] T033 [US1] Implement Quick Capture page composition and post-save navigation/confirmation in `src/web/src/pages/QuickCapturePage.vue`
- [X] T034 [US1] Add `/quick-capture` authenticated route in `src/web/src/router/index.ts`
- [X] T035 [US1] Add Quick Capture mobile/PWA and desktop sidebar navigation entry using existing tokens/icons without disrupting Add Coin navigation in `src/web/src/App.vue`
- [X] T036 [US1] Add navigation regression test for Quick Capture entry visibility and existing Add Coin entry preservation in `src/web/src/__tests__/AppNavigation.test.ts`

**Checkpoint**: US1 is complete when a mobile user can save an active incomplete draft with images or identifying text, all US1 tests pass, and normal collection/wishlist/sold/stats/health counts remain unchanged.

---

## Phase 4: User Story 2 - Resume and finish captured drafts (Priority: P1)

**Goal**: A collector can list their active drafts, identify drafts by preview/context/status/updated time, resume editing, update photos/fields, and discard unwanted drafts without creating or modifying a normal coin.

**Independent Test**: Create multiple drafts, leave the flow, open the drafts view, verify only current-user active drafts appear, edit a draft's fields/images, discard another draft, and verify updates persist while normal coins remain unchanged.

### Tests for User Story 2

- [X] T037 [P] [US2] Add repository tests for active draft pagination, status filtering, last-updated ordering, non-owner invisibility, update, and discard idempotency in `src/api/repository/quick_capture_repository_test.go`
- [X] T038 [P] [US2] Add service tests for update-in-place, image replacement/removal ownership, active-only edits, and discard lifecycle events in `src/api/services/quick_capture_service_test.go`
- [X] T039 [P] [US2] Add handler contract tests for `GET /api/quick-capture/drafts`, `GET /api/quick-capture/drafts/:id`, `PUT /api/quick-capture/drafts/:id`, and `POST /api/quick-capture/drafts/:id/discard` in `src/api/handlers/quick_capture_handler_test.go`
- [X] T040 [P] [US2] Add frontend API client tests for list/get/update/discard methods, pagination params, and multipart update payloads in `src/web/src/api/__tests__/client.test.ts`
- [X] T041 [P] [US2] Add draft card component tests for preview image, incomplete label, context text, updated timestamp, empty-image fallback, and owner-safe media URLs in `src/web/src/components/quick-capture/__tests__/QuickCaptureDraftCard.test.ts`
- [X] T042 [P] [US2] Add drafts page and resume page tests for empty state, list rendering, edit persistence, validation errors, and discard confirmation in `src/web/src/pages/__tests__/QuickCaptureDraftsPage.test.ts` and `src/web/src/pages/__tests__/QuickCaptureDraftPage.test.ts`

### Implementation for User Story 2

- [X] T043 [US2] Implement list, get, update, and discard handler methods with owner-scoped 404 behavior and Swagger annotations in `src/api/handlers/quick_capture.go`
- [X] T044 [US2] Implement `ListDrafts`, `GetDraft`, `UpdateDraft`, and `DiscardDraft` service methods with active/promoted/discarded state checks in `src/api/services/quick_capture_service.go`
- [X] T045 [US2] Implement repository methods for list pagination, single owner lookup, field updates, image removal/replacement, and idempotent discard in `src/api/repository/quick_capture_repository.go`
- [X] T046 [US2] Register authenticated `GET`, `PUT`, and `POST discard` quick-capture draft routes with correct read/write rate limits in `src/api/main.go`
- [X] T047 [US2] Add typed `listQuickCaptureDrafts`, `getQuickCaptureDraft`, `updateQuickCaptureDraft`, and `discardQuickCaptureDraft` API client methods in `src/web/src/api/client.ts`
- [X] T048 [US2] Implement draft card UI with `AuthenticatedImage`, incomplete status chip, context summary, updated timestamp, and accessible action target in `src/web/src/components/quick-capture/QuickCaptureDraftCard.vue`
- [X] T049 [US2] Implement Quick Capture Drafts page with owner draft list, pagination-ready state, empty state, loading/error states, and start-new link in `src/web/src/pages/QuickCaptureDraftsPage.vue`
- [X] T050 [US2] Implement Quick Capture Draft page with resume editing, image replace/remove controls, save/discard actions, and validation feedback in `src/web/src/pages/QuickCaptureDraftPage.vue`
- [X] T051 [US2] Add `/quick-capture/drafts` and `/quick-capture/drafts/:id` authenticated routes in `src/web/src/router/index.ts`

**Checkpoint**: US2 is complete when active drafts can be listed/resumed/updated/discarded by owner only, discarded/promoted drafts are hidden from the default active list, and no normal coin rows or counts change.

---

## Phase 5: User Story 3 - Promote a draft to a normal coin record (Priority: P1)

**Goal**: A collector can intentionally promote a valid draft into exactly one normal `Coin` record with captured fields/images transferred, and repeated promote attempts cannot create duplicates.

**Independent Test**: Complete a draft with required normal coin fields, promote it, verify one normal coin is created with images and collection count increases by exactly one, then repeat promotion and verify the existing promoted coin is returned with no duplicate rows.

### Tests for User Story 3

- [X] T052 [P] [US3] Add repository transaction tests for promotion claim, promoted coin link, promoted status, and repeated promoted lookup in `src/api/repository/quick_capture_repository_test.go`
- [X] T053 [P] [US3] Add service tests for missing promotion fields, field-level validation, draft-to-coin field mapping, image transfer, lifecycle events, transaction rollback, and idempotent repeated promotion in `src/api/services/quick_capture_service_test.go`
- [X] T054 [P] [US3] Add handler contract tests for `POST /api/quick-capture/drafts/:id/promote` success, repeated success with `alreadyPromoted`, validation `400`, non-owner `404`, and concurrent/conflict `409` responses in `src/api/handlers/quick_capture_handler_test.go`
- [X] T055 [P] [US3] Add regression tests proving promotion increments active collection count exactly once and leaves wishlist/sold totals unchanged for Quick Capture v1 coins in `src/api/handlers/quick_capture_handler_test.go`
- [X] T056 [P] [US3] Add frontend API client tests for promote request/response typing and field-error propagation in `src/web/src/api/__tests__/client.test.ts`
- [X] T057 [P] [US3] Add promotion readiness component tests for missing required fields, confirmation requirement, success message, idempotent already-promoted message, and disabled double-submit state in `src/web/src/components/quick-capture/__tests__/PromotionReadinessPanel.test.ts`

### Implementation for User Story 3

- [X] T058 [US3] Implement promote handler request binding, confirmation enforcement, typed field-error response, idempotent response, and Swagger annotations in `src/api/handlers/quick_capture.go`
- [X] T059 [US3] Implement transactional `PromoteDraft` service flow using normal coin validation, draft claim, coin creation, image transfer, lifecycle events, rollback on failure, and idempotent repeated promotion in `src/api/services/quick_capture_service.go`
- [X] T060 [US3] Implement repository transaction helpers for promotion claim, promoted link update, promoted draft lookup, and draft-image-to-coin-image conversion in `src/api/repository/quick_capture_repository.go`
- [X] T061 [US3] Expose or reuse normal coin minimum validation needed by promotion without bypassing existing create rules in `src/api/services/coin_service.go`
- [X] T062 [US3] Register authenticated write-rate-limited `POST /api/quick-capture/drafts/:id/promote` route in `src/api/main.go`
- [X] T063 [US3] Add typed `promoteQuickCaptureDraft` API client method and promotion field-error type handling in `src/web/src/api/client.ts`
- [X] T064 [US3] Implement promotion readiness panel with required-field guidance, override inputs, confirm action, success/idempotency messaging, and retry-safe disabled state in `src/web/src/components/quick-capture/PromotionReadinessPanel.vue`
- [X] T065 [US3] Integrate promotion panel into draft resume page and navigate/link to promoted coin after success in `src/web/src/pages/QuickCaptureDraftPage.vue`

**Checkpoint**: US3 is complete when promotion creates exactly one normal coin, transfers captured data/images, hides the draft from active drafts, links the promoted coin, and repeated promotion returns the existing coin without duplicate creation.

---

## Phase 6: User Story 4 - Preserve existing collection workflows (Priority: P2)

**Goal**: Existing full add/edit coin workflows, wishlist/sold flags, image handling, authenticated media, collection counts, and AI intake surfaces continue to behave as before except for the intentional single coin added by promotion.

**Independent Test**: Run existing add/edit/image/wishlist/sold/count workflows before and after draft create/update/promote, verify draft operations have zero effect on normal counts/totals, and verify promoted coins edit/display exactly like normal coins.

### Tests for User Story 4

- [X] T066 [P] [US4] Add/extend coin handler regression coverage for existing manual add/edit behavior and promoted-coin edit compatibility in `src/api/handlers/coin_handler_test.go`
- [X] T067 [P] [US4] Add/extend authenticated media tests proving normal coin image behavior is unchanged and draft media cannot be viewed by non-owners in `src/api/handlers/images_media_test.go`
- [X] T068 [P] [US4] Add/extend collection repository tests proving draft rows are never returned by normal `/coins` list filters and promoted coins appear once in `src/api/repository/coin_repository_test.go`
- [X] T069 [P] [US4] Add/extend stats and health service regression tests for active count, wishlist total, sold total, value snapshot, and health eligible rows before draft, after draft, and after promotion in `src/api/services/health_service_test.go` and `src/api/handlers/health_handler_test.go`
- [X] T070 [P] [US4] Add/extend frontend collection, wishlist, sold, stats, and edit page regression tests for unchanged draft behavior and promoted-coin edit/image display in `src/web/src/pages/__tests__/CollectionPage.test.ts`, `src/web/src/pages/__tests__/WishlistPage.test.ts`, `src/web/src/pages/__tests__/StatsPage.test.ts`, and `src/web/src/pages/__tests__/EditCoinPage.test.ts`
- [X] T071 [P] [US4] Add frontend PWA workflow regression covering Quick Capture navigation, Add Coin navigation, existing image upload UI, and no AI intake route expansion in `src/web/src/__tests__/AppNavigation.test.ts` and `src/web/src/pages/__tests__/QuickCapturePage.test.ts`

### Implementation for User Story 4

- [ ] T072 [US4] Fix any count/list/stats/health query regressions revealed by US4 tests without adding draft filters to normal coin code paths unnecessarily in `src/api/repository/coin_repository.go`, `src/api/services/health_service.go`, and `src/api/handlers/coins.go`
- [ ] T073 [US4] Fix any existing add/edit/image compatibility regressions revealed by US4 tests while preserving existing contracts in `src/api/handlers/coins.go`, `src/api/handlers/images.go`, `src/web/src/pages/EditCoinPage.vue`, and `src/web/src/components/CoinForm.vue`
- [ ] T074 [US4] Fix frontend navigation/mobile/PWA regressions revealed by US4 tests while preserving existing Add Coin and AI intake entry behavior in `src/web/src/App.vue`, `src/web/src/pages/AddCoinPage.vue`, and `src/web/src/router/index.ts`

**Checkpoint**: US4 is complete when all targeted regression tests pass and Quick Capture has no unintended effects on existing collection, wishlist, sold, image, edit, or AI intake workflows.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Final validation, documentation, generated artifacts, and quality gates after desired stories are complete.

- [X] T075 [P] Update API contract documentation with any implementation-confirmed request/response details in `specs/336-quick-capture/contracts/quick-capture-api.md`
- [X] T076 [P] Update user-facing Quick Capture workflow documentation and manual verification notes in `docs/quick-capture.md`
- [X] T077 [P] Update release-facing feature notes for Quick Capture in `README.md`
- [X] T078 Regenerate or validate Swagger/OpenAPI output after handler annotations in `src/api/docs/docs.go`
- [X] T079 Run backend validation commands `go vet ./...` and `go test ./...` from `src/api/`
- [X] T080 Run frontend validation command `npm run build` from `src/web/`
- [ ] T081 Execute the manual quickstart verification path, including 375px mobile/PWA save/resume/promote and repeated promote checks, and record results in `specs/336-quick-capture/quickstart.md`
- [X] T082 Confirm no Python agent files were changed for Quick Capture v1 unless required by a failed existing contract in `src/agent/`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1: Setup** has no dependencies.
- **Phase 2: Foundational** depends on Phase 1 and blocks every user story.
- **Phase 3: US1 Save Minimal Draft** depends on Phase 2 and is the MVP.
- **Phase 4: US2 Resume Drafts** depends on Phase 2 and can begin after shared create/list semantics are agreed, but should be validated after US1 create works.
- **Phase 5: US3 Promote Draft** depends on Phase 2 and uses draft data from US1/US2; validate after at least US1 create is functional.
- **Phase 6: US4 Preserve Existing Workflows** depends on US1-US3 behavior for full regression coverage.
- **Phase 7: Polish** depends on all implemented story phases selected for release.

### User Story Dependencies

- **US1 (P1)**: Starts after Foundational; no dependency on other stories; MVP scope.
- **US2 (P1)**: Starts after Foundational; depends operationally on draft entities and benefits from US1 create endpoint but remains independently testable with seeded drafts.
- **US3 (P1)**: Starts after Foundational; depends on draft entities and normal coin validation; independently testable with seeded valid drafts.
- **US4 (P2)**: Depends on US1-US3 to prove no regressions across draft create/update/promote and existing workflows.

### Within Each User Story

- Write and run story tests first; confirm they fail for missing implementation.
- Implement repository/persistence before service orchestration.
- Implement service validation and lifecycle before handlers.
- Implement API client types before frontend pages/components.
- Complete each story checkpoint before proceeding to dependent validation.

---

## Parallel Opportunities

- Setup tasks T002-T006 can run in parallel after T001.
- Foundational tests T007-T010 can run in parallel with each other; implementation tasks T013-T016 can run in parallel after T011-T012 where files do not conflict.
- US1 tests T019-T025 can run in parallel; frontend tasks T031-T036 can run in parallel with backend tasks T026-T029 after API shapes are stable.
- US2 tests T037-T042 can run in parallel; T048-T050 can run in parallel with T043-T047 after response types are stable.
- US3 tests T052-T057 can run in parallel; T064-T065 can run in parallel with T058-T063 after promotion response shape is stable.
- US4 regression tests T066-T071 can run in parallel by workflow area.
- Polish documentation tasks T075-T077 can run in parallel after story behavior is stable.

---

## Parallel Example: User Story 1

```text
Task: "T019 [P] [US1] Add repository tests for owner-scoped create and image association of active drafts in src/api/repository/quick_capture_repository_test.go"
Task: "T020 [P] [US1] Add service tests for partial-save validation requiring title, note, or image and rejecting invalid price/images in src/api/services/quick_capture_service_test.go"
Task: "T021 [P] [US1] Add handler contract tests for POST /api/quick-capture/drafts multipart create success and validation failures in src/api/handlers/quick_capture_handler_test.go"
Task: "T024 [P] [US1] Add mobile/PWA component tests for photo-first obverse/reverse slots, file-upload fallback, camera unavailable messaging, and save disabled/enabled states in src/web/src/components/quick-capture/__tests__/QuickCaptureForm.test.ts"
```

## Parallel Example: User Story 2

```text
Task: "T037 [P] [US2] Add repository tests for active draft pagination, status filtering, last-updated ordering, non-owner invisibility, update, and discard idempotency in src/api/repository/quick_capture_repository_test.go"
Task: "T040 [P] [US2] Add frontend API client tests for list/get/update/discard methods, pagination params, and multipart update payloads in src/web/src/api/__tests__/client.test.ts"
Task: "T041 [P] [US2] Add draft card component tests for preview image, incomplete label, context text, updated timestamp, empty-image fallback, and owner-safe media URLs in src/web/src/components/quick-capture/__tests__/QuickCaptureDraftCard.test.ts"
```

## Parallel Example: User Story 3

```text
Task: "T052 [P] [US3] Add repository transaction tests for promotion claim, promoted coin link, promoted status, and repeated promoted lookup in src/api/repository/quick_capture_repository_test.go"
Task: "T053 [P] [US3] Add service tests for missing promotion fields, field-level validation, draft-to-coin field mapping, image transfer, lifecycle events, transaction rollback, and idempotent repeated promotion in src/api/services/quick_capture_service_test.go"
Task: "T057 [P] [US3] Add promotion readiness component tests for missing required fields, confirmation requirement, success message, idempotent already-promoted message, and disabled double-submit state in src/web/src/components/quick-capture/__tests__/PromotionReadinessPanel.test.ts"
```

## Parallel Example: User Story 4

```text
Task: "T066 [P] [US4] Add/extend coin handler regression coverage for existing manual add/edit behavior and promoted-coin edit compatibility in src/api/handlers/coin_handler_test.go"
Task: "T067 [P] [US4] Add/extend authenticated media tests proving normal coin image behavior is unchanged and draft media cannot be viewed by non-owners in src/api/handlers/images_media_test.go"
Task: "T070 [P] [US4] Add/extend frontend collection, wishlist, sold, stats, and edit page regression tests for unchanged draft behavior and promoted-coin edit/image display in src/web/src/pages/__tests__/CollectionPage.test.ts, src/web/src/pages/__tests__/WishlistPage.test.ts, src/web/src/pages/__tests__/StatsPage.test.ts, and src/web/src/pages/__tests__/EditCoinPage.test.ts"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 and Phase 2.
2. Complete Phase 3 (US1) tests and implementation.
3. Stop and validate the 375px mobile/PWA save-draft workflow.
4. Confirm draft creation changes normal collection count, wishlist total, sold total, stats, and health counts by zero.
5. Demo or release only if the MVP checkpoint passes.

### Incremental Delivery

1. Deliver US1 for fast minimal draft capture.
2. Deliver US2 so drafts are resumable, editable, and discardable.
3. Deliver US3 so validated drafts can become exactly one normal coin.
4. Deliver US4 regression hardening before release to protect existing collection workflows.
5. Complete Polish quality gates and documentation.

### Validation Gates

- Backend: `go vet ./...` and `go test ./...` from `src/api/`.
- Frontend: `npm run build` from `src/web/`.
- Manual: follow `specs/336-quick-capture/quickstart.md` on a 375px mobile/PWA viewport.
- Regression: verify automated coverage for collection counts, wishlist totals, sold totals, promotion idempotency, media ownership, existing add/edit image behavior, and frontend mobile/PWA workflow.

---

## Notes

- Quick Capture v1 is deterministic/manual only; do not expand Python agent or AI intake scope.
- Drafts live in dedicated quick-capture tables and must not be stored as incomplete `coins` rows.
- All draft operations must be authenticated, owner-scoped, and safe by default.
- Promotion must be transactional and idempotent from the user's perspective.
- Use existing design tokens, buttons, chips, icons, upload patterns, and `AuthenticatedImage`.
