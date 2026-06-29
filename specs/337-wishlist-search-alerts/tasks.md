# Tasks: Wishlist Search Alerts for Acquisition Ideas

**Input**: Design documents from `specs/337-wishlist-search-alerts/`
**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/wishlist-search-alerts-api.md`, `contracts/agent-discovery-contract.md`, `quickstart.md`
**Feature**: `337-wishlist-search-alerts` on branch `feature/357-wishlist-search-alerts`

**Tests**: Required because the specification, quickstart, and contracts require owner scoping, criteria validation, provenance, duplicate suppression, explicit conversion, agent contract safety, and regression coverage proving existing wishlist availability checks remain separate.

**Organization**: Tasks are grouped by independently testable user story after shared setup/foundation work. Tests appear before implementation tasks and should be written to fail before the matching implementation lands.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel with other tasks in the same phase because it touches different files or depends only on completed earlier phases
- **[Story]**: User story label for story phases only (`[US1]`, `[US2]`, `[US3]`, `[US4]`, `[US5]`)
- Every task includes an exact repository-relative file path

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Establish feature-specific files and typed seams without changing runtime behavior.

- [X] T001 Create backend alert-domain source files with package declarations in `src/api/models/wishlist_search_alert.go`, `src/api/repository/wishlist_search_alert_repository.go`, `src/api/services/wishlist_search_alert_service.go`, and `src/api/handlers/wishlist_search_alerts.go`
- [X] T002 [P] Create backend alert-domain test files with failing/skipped-free skeletons in `src/api/models/wishlist_search_alert_test.go`, `src/api/repository/wishlist_search_alert_repository_test.go`, `src/api/services/wishlist_search_alert_service_test.go`, and `src/api/handlers/wishlist_search_alerts_handler_test.go`
- [X] T003 [P] Create agent discovery test skeletons in `src/agent/tests/test_alert_discovery_contract.py` and `src/agent/tests/test_alert_discovery_route.py`
- [ ] T004 [P] Create frontend wishlist-alert component and page test skeletons in `src/web/src/components/wishlist-alerts/__tests__/AlertForm.test.ts`, `src/web/src/components/wishlist-alerts/__tests__/AlertRunHistory.test.ts`, `src/web/src/components/wishlist-alerts/__tests__/CandidateReviewCard.test.ts`, and `src/web/src/pages/__tests__/WishlistAlertsPage.test.ts`
- [X] T005 [P] Create frontend wishlist-alert component placeholders in `src/web/src/components/wishlist-alerts/AlertForm.vue`, `src/web/src/components/wishlist-alerts/AlertRunHistory.vue`, `src/web/src/components/wishlist-alerts/CandidateReviewCard.vue`, and `src/web/src/components/wishlist-alerts/AlertCriteriaSummary.vue`
- [X] T006 [P] Create frontend wishlist-alert page placeholder in `src/web/src/pages/WishlistAlertsPage.vue`
- [ ] T007 [P] Add wishlist-search-alert API contract TODO anchors to `src/api/handlers/wishlist_search_alerts.go` referencing `specs/337-wishlist-search-alerts/contracts/wishlist-search-alerts-api.md`
- [X] T008 [P] Add agent discovery contract TODO anchors to `src/agent/app/models/requests.py`, `src/agent/app/models/responses.py`, and `src/agent/app/routes.py` referencing `specs/337-wishlist-search-alerts/contracts/agent-discovery-contract.md`
- [ ] T009 [P] Add manual QA checklist cross-references for alert CRUD, Run Now, candidate review, conversion, and availability separation in `specs/337-wishlist-search-alerts/quickstart.md`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Build shared schema, typed DTOs, ownership helpers, and proxy contracts required before any user story can be implemented.

**⚠️ CRITICAL**: No user story implementation can begin until this phase is complete.

### Tests

- [ ] T010 [P] Add model tests for alert, run, candidate, provenance, review-action enums, soft-delete fields, and `Coin.SourceAlertCandidateID` traceability in `src/api/models/wishlist_search_alert_test.go`
- [X] T011 [P] Add AutoMigrate tests for `wishlist_search_alerts`, `alert_runs`, `alert_candidates`, `candidate_provenances`, `candidate_review_actions`, and nullable coin traceability in `src/api/database/migration_test.go`
- [ ] T012 [P] Add repository ownership tests proving all alert, run, candidate, provenance, and review-action queries require `user_id` scoping in `src/api/repository/wishlist_search_alert_repository_test.go`
- [X] T013 [P] Add service validation tests for price ranges, date ranges, empty criteria, unsupported cadence, malformed source filters, and string limits in `src/api/services/wishlist_search_alert_service_test.go`
- [X] T014 [P] Add `AgentProxy` discovery DTO serialization tests for `/api/search/alerts` request/response shape in `src/api/services/agent_proxy_test.go`
- [ ] T015 [P] Add TypeScript compile-time and API-client tests for wishlist alert DTOs, enums, query params, and error categories in `src/web/src/api/__tests__/client.test.ts`
- [X] T016 [P] Add Python Pydantic contract tests requiring `extra="forbid"`, required source URL/title/reason/last-seen/provenance status, `max_candidates` caps, and nullable optional facts in `src/agent/tests/test_alert_discovery_contract.py`

### Implementation

- [X] T017 Define `WishlistSearchAlert`, `AlertRun`, `AlertCandidate`, `CandidateProvenance`, `CandidateReviewAction`, enum constants, JSON helper fields, indexes, and GORM relationships in `src/api/models/wishlist_search_alert.go`
- [X] T018 Add nullable `SourceAlertCandidateID` traceability field and GORM association to converted wishlist coins in `src/api/models/coin.go`
- [X] T019 Register alert-domain models and coin traceability migration in GORM AutoMigrate in `src/api/database/database.go`
- [X] T020 Implement owner-scoped repository primitives for alerts, runs, candidates, provenance, review actions, duplicate lookup, and transactional conversion helpers in `src/api/repository/wishlist_search_alert_repository.go`
- [X] T021 Extend coin repository duplicate lookup support for existing wishlist `reference_url` and converted candidate IDs in `src/api/repository/coin_repository.go`
- [X] T022 Implement shared criteria validation, source/domain filter normalization, canonical URL normalization, normalized title generation, duplicate-key calculation, result-cap constants, and sanitized error types in `src/api/services/wishlist_search_alert_service.go`
- [X] T023 Add typed alert discovery request/response DTOs and `DiscoverAlertCandidates` proxy method using the existing non-streaming proxy pattern in `src/api/services/agent_proxy.go`
- [X] T024 Add TypeScript alert, run, candidate, provenance, review-action, conversion, and pagination types to `src/web/src/types/index.ts`
- [ ] T025 Add typed wishlist alert API client methods for all contract endpoints to `src/web/src/api/client.ts`

**Checkpoint**: Foundation is ready when schema/DTO/proxy/validation tests compile and fail only for story behavior that has not been implemented yet.

---

## Phase 3: User Story 1 - Define acquisition search alerts (Priority: P1) 🎯 MVP

**Goal**: An authenticated collector can create, view, edit, disable, and delete owner-scoped search alerts with validated numismatic, price, source, keyword, notes, and cadence criteria, without creating or modifying wishlist items.

**Independent Test**: Create, edit, disable, re-enable, and delete an alert with the supported criteria; verify only the owner can see it, invalid criteria are rejected, previous run snapshots are not touched, and no wishlist coin or availability history is changed.

### Tests for User Story 1

- [X] T026 [P] [US1] Add repository tests for owner-scoped alert create/list/get/update/soft-delete, active filter, pagination, and non-owner invisibility in `src/api/repository/wishlist_search_alert_repository_test.go`
- [ ] T027 [P] [US1] Add service tests for create/update/disable/delete behavior, criteria summary, cadence storage, and unchanged existing run snapshots in `src/api/services/wishlist_search_alert_service_test.go`
- [X] T028 [P] [US1] Add handler contract tests for `GET /api/wishlist/search-alerts`, `POST /api/wishlist/search-alerts`, `GET /api/wishlist/search-alerts/:alertId`, `PUT /api/wishlist/search-alerts/:alertId`, and `DELETE /api/wishlist/search-alerts/:alertId` in `src/api/handlers/wishlist_search_alerts_handler_test.go`
- [ ] T029 [P] [US1] Add authorization tests proving Owner A cannot list, read, update, delete, or infer Owner B alerts in `src/api/handlers/wishlist_search_alerts_handler_test.go`
- [X] T030 [P] [US1] Add regression tests proving alert CRUD creates zero `Coin`, `AvailabilityRun`, and `AvailabilityResult` rows and does not mutate existing `Coin.ListingStatus` in `src/api/services/wishlist_search_alert_service_test.go`
- [ ] T031 [P] [US1] Add frontend API-client tests for alert CRUD payloads, validation errors, active filter, pagination, and typed response mapping in `src/web/src/api/__tests__/client.test.ts`
- [ ] T032 [P] [US1] Add alert form component tests for all criteria fields, source filter validation messages, cadence display, active toggle, save disabled states, and delete confirmation in `src/web/src/components/wishlist-alerts/__tests__/AlertForm.test.ts`
- [ ] T033 [P] [US1] Add alert list/page tests for owner alert list, empty state, create/edit/delete flows, criteria summary, and distinct "search alerts" copy in `src/web/src/pages/__tests__/WishlistAlertsPage.test.ts`

### Implementation for User Story 1

- [X] T034 [US1] Implement alert create/list/get/update/soft-delete repository methods with `OwnedBy`/`OwnedByID` user scoping in `src/api/repository/wishlist_search_alert_repository.go`
- [X] T035 [US1] Implement `CreateAlert`, `ListAlerts`, `GetAlert`, `UpdateAlert`, `SetAlertActive`, and `DeleteAlert` service methods with validation and sanitized errors in `src/api/services/wishlist_search_alert_service.go`
- [X] T036 [US1] Implement alert CRUD handler request binding, response DTO mapping, pagination parsing, owner-scoped 404 behavior, and Swagger annotations in `src/api/handlers/wishlist_search_alerts.go`
- [X] T037 [US1] Wire `WishlistSearchAlertRepository`, `WishlistSearchAlertService`, `WishlistSearchAlertHandler`, and protected CRUD routes under `/api/wishlist/search-alerts` in `src/api/main.go`
- [X] T038 [US1] Add alert CRUD response/request Swagger helper types to `src/api/handlers/swagger_types.go`
- [X] T039 [US1] Implement `listWishlistSearchAlerts`, `createWishlistSearchAlert`, `getWishlistSearchAlert`, `updateWishlistSearchAlert`, and `deleteWishlistSearchAlert` client methods in `src/web/src/api/client.ts`
- [X] T040 [US1] Implement `AlertCriteriaSummary` with compact criteria, active state, cadence, and last-run display in `src/web/src/components/wishlist-alerts/AlertCriteriaSummary.vue`
- [X] T041 [US1] Implement `AlertForm` with numismatic fields, price/date/source validation, cadence metadata, active toggle, notes/keywords, and user-facing validation errors in `src/web/src/components/wishlist-alerts/AlertForm.vue`
- [X] T042 [US1] Implement `WishlistAlertsPage` alert list, create/edit modal or panel, delete/disable/re-enable actions, loading/error/empty states, and clear separation from saved wishlist availability checks in `src/web/src/pages/WishlistAlertsPage.vue`
- [X] T043 [US1] Add authenticated `/wishlist/search-alerts` route in `src/web/src/router/index.ts`
- [X] T044 [US1] Add a distinct Wishlist Search Alerts navigation entry or action from the wishlist page without replacing the existing Check Availability workflow in `src/web/src/pages/WishlistPage.vue`

**Checkpoint**: US1 is complete when alert CRUD works end-to-end, criteria validation is enforced, owner scoping is proven, and no wishlist items or availability history are created or changed by alert management.

---

## Phase 4: User Story 2 - Run an alert and review acquisition candidates (Priority: P1) 🎯 MVP

**Goal**: An authenticated collector can manually run an active alert, receive source-backed candidates with provenance, see run history, handle partial/failure/rate-limited results, and avoid duplicate review noise.

**Independent Test**: Run an active alert against mocked agent results, verify run metadata and criteria snapshot are stored, candidates include provenance and lifecycle state, duplicate repeat results are updated or suppressed, and no source facts are invented.

### Tests for User Story 2

- [ ] T045 [P] [US2] Add Python route tests for `POST /api/search/alerts` success, result cap, warnings, partial status, and validation failures in `src/agent/tests/test_alert_discovery_route.py`
- [ ] T046 [P] [US2] Add Python safe outbound regression tests proving alert discovery reuses `safe_get()` and `validate_public_outbound_url()` for public fetches and redirects in `src/agent/tests/test_outbound_validation.py`
- [ ] T047 [P] [US2] Add Go proxy tests for successful discovery, agent timeout, agent unavailable, malformed response, and sanitized error mapping in `src/api/services/agent_proxy_test.go`
- [X] T048 [P] [US2] Add service tests for manual Run Now criteria snapshots, run status transitions, result counts, non-secret warnings/errors, rate-limit status, and disabled/deleted alert rejection in `src/api/services/wishlist_search_alert_service_test.go`
- [X] T049 [P] [US2] Add service tests for duplicate suppression using canonical URL strongest plus normalized title, observed price, and alert ID in `src/api/services/wishlist_search_alert_service_test.go`
- [ ] T050 [P] [US2] Add repository tests for run creation/completion, candidate/provenance persistence, last-seen updates, suppressed duplicate relationships, and run-history pagination in `src/api/repository/wishlist_search_alert_repository_test.go`
- [ ] T051 [P] [US2] Add handler contract tests for `POST /api/wishlist/search-alerts/:alertId/run`, `GET /api/wishlist/search-alerts/:alertId/runs`, `GET /api/wishlist/search-alerts/:alertId/runs/:runId`, and `GET /api/wishlist/search-alerts/:alertId/candidates` in `src/api/handlers/wishlist_search_alerts_handler_test.go`
- [ ] T052 [P] [US2] Add frontend API-client tests for Run Now, run list/detail, candidate list filters, partial warnings, and rate-limit/user-facing error responses in `src/web/src/api/__tests__/client.test.ts`
- [ ] T053 [P] [US2] Add run history component tests for status badges, criteria snapshot display, counts, partial warnings, empty results, and failed/rate-limited messages in `src/web/src/components/wishlist-alerts/__tests__/AlertRunHistory.test.ts`
- [ ] T054 [P] [US2] Add candidate review card tests for source URL, title, price unknown state, reason for match, provenance status, last-seen timestamp, suppressed/duplicate labels, and no invented dealer details in `src/web/src/components/wishlist-alerts/__tests__/CandidateReviewCard.test.ts`

### Implementation for User Story 2

- [X] T055 [US2] Add Pydantic alert discovery request, criteria snapshot, candidate, provenance, and response models with `extra="forbid"` in `src/agent/app/models/requests.py` and `src/agent/app/models/responses.py`
- [X] T056 [US2] Implement stateless alert discovery orchestration that reuses existing coin search capabilities and returns only source-backed candidate facts in `src/agent/app/teams/coin_search.py`
- [X] T057 [US2] Implement `POST /api/search/alerts` route with validation, max-candidate cap, partial warnings, timeout/rate-limit friendly responses, and no persistence in `src/agent/app/routes.py`
- [X] T058 [US2] Preserve safe outbound validation and redirect checks for any alert discovery fetch path in `src/agent/app/tools/search.py`
- [X] T059 [US2] Implement `AgentProxy.DiscoverAlertCandidates` HTTP call, timeout handling, response validation, and sanitized error mapping in `src/api/services/agent_proxy.go`
- [X] T060 [US2] Implement run-limit checks, active alert guard, criteria snapshot serialization, and run start/finalize helpers in `src/api/services/wishlist_search_alert_service.go`
- [X] T061 [US2] Implement candidate ingestion, provenance persistence, duplicate-key computation, last-seen update, suppression, count aggregation, and partial warning handling in `src/api/services/wishlist_search_alert_service.go`
- [X] T062 [US2] Implement run and candidate repository methods for create/update/list/detail/filter and provenance preloading in `src/api/repository/wishlist_search_alert_repository.go`
- [X] T063 [US2] Implement Run Now, run list/detail, and candidate list handlers with owner-scoped 404/403 behavior and Swagger annotations in `src/api/handlers/wishlist_search_alerts.go`
- [X] T064 [US2] Register authenticated Run Now, run history, run detail, and candidate-list routes with read/write rate limits in `src/api/main.go`
- [X] T065 [US2] Add run and candidate Swagger helper types to `src/api/handlers/swagger_types.go`
- [X] T066 [US2] Implement `runWishlistSearchAlert`, `listWishlistSearchAlertRuns`, `getWishlistSearchAlertRun`, and `listWishlistSearchAlertCandidates` client methods in `src/web/src/api/client.ts`
- [X] T067 [US2] Implement `AlertRunHistory` with run statuses, counts, criteria snapshots, partial warnings, rate-limit messages, failed run details, and no-results state in `src/web/src/components/wishlist-alerts/AlertRunHistory.vue`
- [X] T068 [US2] Implement `CandidateReviewCard` source-backed field display, provenance badge, unknown field display, duplicate/suppressed labels, and accessible source link in `src/web/src/components/wishlist-alerts/CandidateReviewCard.vue`
- [X] T069 [US2] Integrate Run Now button, run history, candidate filters, active/needs-review queue, loading states, and retry-safe disabled states into `src/web/src/pages/WishlistAlertsPage.vue`

**Checkpoint**: US2 is complete when a manual run creates auditable alert-run history and candidates with provenance, duplicate repeat results are suppressed or updated, and partial/failure/rate-limited outcomes are visible without leaking internals.

---

## Phase 5: User Story 3 - Act on candidates without losing review control (Priority: P1) 🎯 MVP

**Goal**: A collector can dismiss, restore, convert, and use candidates to adjust criteria while conversion to a wishlist item is explicit, traceable, duplicate-aware, and based only on reviewed/source-backed fields.

**Independent Test**: Dismiss and restore a candidate, convert a verified candidate into one normal wishlist item, attempt partial conversion and see required field review, acknowledge duplicate warnings only explicitly, and adjust alert criteria without changing converted wishlist items.

### Tests for User Story 3

- [X] T070 [P] [US3] Add service tests for dismiss, restore, review-action history, allowed dismissal reasons, and idempotent state transitions in `src/api/services/wishlist_search_alert_service_test.go`
- [ ] T071 [P] [US3] Add service tests for conversion required-field checks, source-backed prefill rules, partial/unverified candidate review requirements, and no invented wishlist fields in `src/api/services/wishlist_search_alert_service_test.go`
- [ ] T072 [P] [US3] Add repository transaction tests for candidate conversion, `Coin.SourceAlertCandidateID`, `AlertCandidate.ConvertedCoinID`, review actions, and rollback on coin creation failure in `src/api/repository/wishlist_search_alert_repository_test.go`
- [ ] T073 [P] [US3] Add duplicate warning tests for matching existing wishlist `reference_url`, previously converted candidate, and acknowledge/retry behavior in `src/api/services/wishlist_search_alert_service_test.go`
- [ ] T074 [P] [US3] Add handler contract tests for candidate dismiss, restore, convert, and criteria adjustment endpoints in `src/api/handlers/wishlist_search_alerts_handler_test.go`
- [ ] T075 [P] [US3] Add frontend API-client tests for dismiss, restore, convert, duplicate warning acknowledgement, field-error mapping, and criteria-adjustment payloads in `src/web/src/api/__tests__/client.test.ts`
- [ ] T076 [P] [US3] Add candidate review UI tests for dismiss reason, restore action, conversion form, missing required field prompts, duplicate warning acknowledgement, and converted state in `src/web/src/components/wishlist-alerts/__tests__/CandidateReviewCard.test.ts`
- [ ] T077 [P] [US3] Add alert page tests for criteria adjustment from review context and preservation of already-converted wishlist items in `src/web/src/pages/__tests__/WishlistAlertsPage.test.ts`

### Implementation for User Story 3

- [X] T078 [US3] Implement repository methods for candidate state update, review-action insert, conversion transaction, duplicate wishlist lookup, and criteria-adjustment action logging in `src/api/repository/wishlist_search_alert_repository.go`
- [X] T079 [US3] Implement `DismissCandidate`, `RestoreCandidate`, and review-action audit logic with owner scoping and sanitized errors in `src/api/services/wishlist_search_alert_service.go`
- [X] T080 [US3] Implement source-backed wishlist conversion prefill, required field validation, duplicate warning detection, explicit acknowledgement handling, and transaction orchestration in `src/api/services/wishlist_search_alert_service.go`
- [x] T081 [US3] Integrate normal coin creation/validation for converted candidates without bypassing existing coin rules in `src/api/services/coin_service.go`
- [X] T082 [US3] Implement criteria-adjustment service logic that updates alert criteria, records referenced candidate review actions, and leaves previously converted wishlist items unchanged in `src/api/services/wishlist_search_alert_service.go`
- [X] T083 [US3] Implement dismiss, restore, convert, and criteria-adjustment handlers with Swagger annotations and field-error/duplicate-warning responses in `src/api/handlers/wishlist_search_alerts.go`
- [X] T084 [US3] Register authenticated candidate review and criteria-adjustment routes with write rate limits in `src/api/main.go`
- [X] T085 [US3] Add conversion and review-action Swagger helper types to `src/api/handlers/swagger_types.go`
- [X] T086 [US3] Implement `dismissWishlistSearchAlertCandidate`, `restoreWishlistSearchAlertCandidate`, `convertWishlistSearchAlertCandidate`, and `adjustWishlistSearchAlertCriteria` client methods in `src/web/src/api/client.ts`
- [X] T087 [US3] Extend `CandidateReviewCard` with dismiss reason selector, restore action, conversion form, missing/uncertain field prompts, duplicate warning acknowledgement, and converted wishlist link in `src/web/src/components/wishlist-alerts/CandidateReviewCard.vue`
- [X] T088 [US3] Extend `WishlistAlertsPage` with candidate review queue actions, criteria adjustment flow, post-conversion refresh, and page-context preservation after reviewing 20 candidates in `src/web/src/pages/WishlistAlertsPage.vue`

**Checkpoint**: US3 is complete when candidate actions are explicit and auditable, conversion creates exactly one normal wishlist coin linked to the candidate, duplicate warnings are enforced, and criteria edits do not alter converted wishlist items.

---

## Phase 6: User Story 4 - Preserve existing wishlist availability checks (Priority: P1) 🎯 MVP Regression

**Goal**: Existing wishlist availability checks continue to validate already-saved wishlist URLs and remain separate from alert runs, candidates, review actions, and provenance.

**Independent Test**: With a saved wishlist item URL and a separate search alert, run the existing availability check and the alert; verify availability runs/results/listing status and alert runs/candidates stay separate, and converted candidates only participate in later availability checks as normal wishlist items.

### Tests for User Story 4

- [ ] T089 [P] [US4] Add availability service regression tests proving `CheckWishlistForUser` only reads wishlist coins with saved URLs and never creates alert runs or candidates in `src/api/services/availability_service_test.go`
- [ ] T090 [P] [US4] Add availability handler regression tests proving `POST /api/wishlist/check-availability` writes only `AvailabilityRun`/`AvailabilityResult` history and does not touch alert tables in `src/api/handlers/availability_handler_test.go`
- [X] T091 [P] [US4] Add alert run regression tests proving Run Now creates no `AvailabilityRun`/`AvailabilityResult` rows and does not mutate `Coin.ListingStatus`, `ListingCheckedAt`, or `ListingCheckReason` in `src/api/services/wishlist_search_alert_service_test.go`
- [ ] T092 [P] [US4] Add repository duplicate relationship tests proving existing wishlist URL matches are surfaced as `matching_wishlist_coin_id` without automatically converting candidates in `src/api/repository/wishlist_search_alert_repository_test.go`
- [ ] T093 [P] [US4] Add converted-candidate availability test proving future availability checks operate on the converted coin `reference_url` and record availability history rather than alert history in `src/api/services/availability_service_test.go`
- [ ] T094 [P] [US4] Add frontend wishlist page regression tests preserving the existing Check Availability button, status display, and copy distinct from search alerts in `src/web/src/pages/__tests__/WishlistPage.test.ts`
- [ ] T095 [P] [US4] Add frontend alert page tests showing matching existing wishlist item warnings and preventing accidental duplicate wishlist additions in `src/web/src/pages/__tests__/WishlistAlertsPage.test.ts`

### Implementation for User Story 4

- [ ] T096 [US4] Fix any availability service regressions revealed by US4 tests while keeping alert logic out of `src/api/services/availability_service.go`
- [ ] T097 [US4] Fix any availability repository regressions revealed by US4 tests while keeping alert tables out of `src/api/repository/availability_repository.go`
- [X] T098 [US4] Ensure alert candidate duplicate matching populates `matching_wishlist_coin_id` and duplicate warnings without creating wishlist items in `src/api/services/wishlist_search_alert_service.go`
- [X] T099 [US4] Preserve existing wishlist availability UI behavior and add only a distinct navigation path to alert discovery in `src/web/src/pages/WishlistPage.vue`
- [ ] T100 [US4] Ensure converted wishlist coins display and behave as normal wishlist items for later availability checks without exposing alert internals in `src/web/src/pages/WishlistPage.vue`

**Checkpoint**: US4 is complete when existing availability checks and new search-alert discovery have independent histories, statuses, UI copy, and persistence, with integration only after explicit candidate conversion.

---

## Phase 7: User Story 5 - Prepare for scheduled review without overbuilding notifications (Priority: P2)

**Goal**: Alerts store cadence preferences and manual review surfaces candidate activity without claiming push, email, digest, or scheduled execution in v1.

**Independent Test**: Save cadence metadata, run alerts manually, verify in-app run/candidate review works, and verify no scheduled run or notification is implied or emitted when cadence would be due.

### Tests for User Story 5

- [ ] T101 [P] [US5] Add service tests proving cadence values are stored and returned but do not enqueue scheduled alert execution or notification delivery in `src/api/services/wishlist_search_alert_service_test.go`
- [ ] T102 [P] [US5] Add scheduler regression tests proving `AvailabilityScheduler` still schedules only saved wishlist URL availability checks and not search alerts in `src/api/services/availability_scheduler_test.go`
- [ ] T103 [P] [US5] Add handler tests proving alert responses expose cadence metadata without scheduled-run or notification delivery claims in `src/api/handlers/wishlist_search_alerts_handler_test.go`
- [ ] T104 [P] [US5] Add frontend tests for cadence selector/help copy that explicitly states v1 supports manual Run Now and in-app review only in `src/web/src/components/wishlist-alerts/__tests__/AlertForm.test.ts`
- [ ] T105 [P] [US5] Add frontend page tests proving no push/email/digest notification copy appears after Run Now or candidate discovery in `src/web/src/pages/__tests__/WishlistAlertsPage.test.ts`

### Implementation for User Story 5

- [X] T106 [US5] Ensure cadence metadata is saved, returned, and included in criteria snapshots without scheduling side effects in `src/api/services/wishlist_search_alert_service.go`
- [ ] T107 [US5] Ensure `AvailabilityScheduler` remains scoped to existing wishlist availability checks and has no search-alert registration in `src/api/services/availability_scheduler.go`
- [ ] T108 [US5] Add user-facing cadence response fields and manual-only copy to alert handler DTO mapping in `src/api/handlers/wishlist_search_alerts.go`
- [X] T109 [US5] Add cadence selector helper text explaining manual Run Now and deferred scheduling in `src/web/src/components/wishlist-alerts/AlertForm.vue`
- [X] T110 [US5] Add in-app-only review copy and avoid notification promises in alert run/candidate UI in `src/web/src/pages/WishlistAlertsPage.vue`

**Checkpoint**: US5 is complete when cadence is future-proof metadata only, manual review remains the v1 behavior, and availability scheduling/notifications are not expanded.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, generated API artifacts, full validation, and blast-radius checks across Go API, Python agent, and Vue app.

- [ ] T111 [P] Update API contract examples and any implementation notes discovered during development in `specs/337-wishlist-search-alerts/contracts/wishlist-search-alerts-api.md`
- [ ] T112 [P] Update agent discovery contract examples and safe outbound notes discovered during development in `specs/337-wishlist-search-alerts/contracts/agent-discovery-contract.md`
- [ ] T113 [P] Update manual verification instructions for completed routes, UI paths, and regression checks in `specs/337-wishlist-search-alerts/quickstart.md`
- [X] T114 Regenerate Swagger/OpenAPI artifacts after all handler annotations land in `src/api/docs/docs.go`, `src/api/docs/swagger.json`, and `src/api/docs/swagger.yaml`
- [X] T115 Run route/OpenAPI drift tests and fix missing documented routes in `src/api/route_openapi_drift_test.go`
- [X] T116 Run Go unit, handler, repository, and service tests with `go test -v ./...` from `src/api`
- [X] T117 Run Go static checks with `go vet ./...` from `src/api`
- [x] T118 Run Python lint with `ruff check app/ tests/` from `src/agent`
- [x] T119 Run Python agent tests with `pytest tests/ -v` from `src/agent`
- [X] T120 Run Vue production build and strict type checks with `npm run build` from `src/web`
- [x] T121 Run repository-level validation commands `task build` and `task test` from repository root `C:\Users\brian.denicolafamily\Code\AncientCoins`
- [ ] T122 Perform manual quickstart validation for alert CRUD, Run Now, duplicate suppression, candidate dismissal/restore/conversion, cadence metadata, and availability separation using `specs/337-wishlist-search-alerts/quickstart.md`
- [x] T123 Review logs, API responses, persisted errors, and frontend state for leaked API keys, internal agent configuration, stack traces, private URLs, or private collection data in `src/api/services/wishlist_search_alert_service.go`, `src/api/services/agent_proxy.go`, and `src/agent/app/routes.py`
- [x] T124 Run a final layered-architecture review proving handlers are thin, services own business decisions, repositories own persistence, Vue never calls Python directly, and Python remains stateless in `src/api/handlers/wishlist_search_alerts.go`, `src/api/services/wishlist_search_alert_service.go`, and `src/agent/app/routes.py`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 Setup**: No dependencies; can start immediately.
- **Phase 2 Foundation**: Depends on Phase 1; blocks all user stories.
- **Phase 3 US1 Alert CRUD**: Depends on Phase 2 and is the first MVP slice.
- **Phase 4 US2 Manual Run + Candidates**: Depends on Phase 2; practically uses US1 alert records but can be built with seeded alerts.
- **Phase 5 US3 Candidate Actions + Conversion**: Depends on US2 candidate persistence and should follow US2 for the normal MVP path.
- **Phase 6 US4 Availability Separation**: Depends on Phase 2; regression tests can start in parallel with US1/US2, fixes depend on affected implementation.
- **Phase 7 US5 Cadence Metadata**: Depends on US1 alert persistence; can run after US1 or in parallel with US2/US3 once cadence fields exist.
- **Phase 8 Polish**: Depends on all desired MVP stories being implemented.

### User Story Dependencies

- **US1 (P1)**: Independent after foundation; delivers alert CRUD and validation.
- **US2 (P1)**: Needs an alert record to run; can use seeded alert fixtures and is independently testable for run/candidate behavior.
- **US3 (P1)**: Needs persisted candidates from US2; delivers review actions and conversion.
- **US4 (P1)**: Regression slice proving separation; tests can be written early and must pass before MVP is accepted.
- **US5 (P2)**: Depends on alert cadence persistence from US1; no dependency on notifications or scheduling.

### Dependency Graph

```text
Phase 1 Setup
  └── Phase 2 Foundation
        ├── US1 Alert CRUD
        │     └── US5 Cadence Metadata
        ├── US2 Manual Run + Candidate Review
        │     └── US3 Candidate Actions + Conversion
        └── US4 Availability Separation Regression
              └── Phase 8 Polish & Validation
```

### Within Each User Story

- Write tests first and verify they fail before implementing behavior.
- Models and DTOs precede repositories.
- Repositories precede services.
- Services precede handlers and routes.
- API client types precede Vue components/pages.
- Story checkpoint must pass before treating the story as complete.

---

## Parallel Execution Examples

### User Story 1

```text
Task: "Add repository tests for owner-scoped alert CRUD in `src/api/repository/wishlist_search_alert_repository_test.go`"
Task: "Add service tests for criteria validation in `src/api/services/wishlist_search_alert_service_test.go`"
Task: "Add frontend AlertForm tests in `src/web/src/components/wishlist-alerts/__tests__/AlertForm.test.ts`"
Task: "Implement alert criteria summary in `src/web/src/components/wishlist-alerts/AlertCriteriaSummary.vue`"
```

### User Story 2

```text
Task: "Add Python route tests for `POST /api/search/alerts` in `src/agent/tests/test_alert_discovery_route.py`"
Task: "Add Go proxy tests in `src/api/services/agent_proxy_test.go`"
Task: "Add run/candidate repository tests in `src/api/repository/wishlist_search_alert_repository_test.go`"
Task: "Add CandidateReviewCard display tests in `src/web/src/components/wishlist-alerts/__tests__/CandidateReviewCard.test.ts`"
```

### User Story 3

```text
Task: "Add conversion transaction tests in `src/api/repository/wishlist_search_alert_repository_test.go`"
Task: "Add duplicate warning service tests in `src/api/services/wishlist_search_alert_service_test.go`"
Task: "Add frontend conversion API-client tests in `src/web/src/api/__tests__/client.test.ts`"
Task: "Extend CandidateReviewCard with conversion UI in `src/web/src/components/wishlist-alerts/CandidateReviewCard.vue`"
```

### User Story 4

```text
Task: "Add availability service regression tests in `src/api/services/availability_service_test.go`"
Task: "Add alert run separation tests in `src/api/services/wishlist_search_alert_service_test.go`"
Task: "Add wishlist page regression tests in `src/web/src/pages/__tests__/WishlistPage.test.ts`"
```

### User Story 5

```text
Task: "Add scheduler regression tests in `src/api/services/availability_scheduler_test.go`"
Task: "Add cadence helper-copy tests in `src/web/src/components/wishlist-alerts/__tests__/AlertForm.test.ts`"
Task: "Implement manual-only cadence copy in `src/web/src/components/wishlist-alerts/AlertForm.vue`"
```

---

## Implementation Strategy

### MVP First

1. Complete Phase 1 Setup.
2. Complete Phase 2 Foundation.
3. Deliver US1 alert CRUD and validation.
4. Deliver US2 typed agent discovery, Run Now, run history, candidates, provenance, duplicate suppression.
5. Deliver US3 dismiss/restore/convert/criteria adjustment.
6. Deliver US4 regression proof that availability checks remain separate.
7. Include US5 cadence metadata if time allows; cadence storage is already part of US1 but scheduling/notifications stay deferred.
8. Complete Phase 8 validation and OpenAPI regeneration.

### Recommended MVP Slices

- **Slice A**: Foundation + US1 alert CRUD, criteria validation, owner scoping, no wishlist side effects.
- **Slice B**: US2 typed Python discovery endpoint + Go Run Now + candidate/provenance persistence + duplicate suppression.
- **Slice C**: US3 review actions + explicit conversion to wishlist coin + duplicate warnings.
- **Slice D**: US4 regression hardening + OpenAPI regeneration + full validation.
- **Slice E**: US5 manual-only cadence UX polish if not completed in Slice A.

### Parallel Team Strategy

1. Backend developer A: Go models/repositories/services/handlers for US1 and foundation.
2. Backend developer B: Agent discovery endpoint and `AgentProxy` for US2.
3. Frontend developer: TypeScript types/API client and alert/review UI after DTOs are stable.
4. QA/regression developer: US4 availability separation tests and quickstart validation from the start.

---

## Notes

- Search alerts never create wishlist items until explicit candidate conversion.
- Search alert runs never create `AvailabilityRun` or `AvailabilityResult` records.
- Existing availability checks continue to validate saved wishlist URLs only.
- Python agent remains stateless and never persists candidates or scopes users.
- Vue calls only the Go API; it must not call the Python agent directly.
- Missing or uncertain source facts must be displayed as unknown/partial/unverified, never inferred as verified.
