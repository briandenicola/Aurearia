# Feature Specification: Critical Workflow Hardening

**Feature Branch**: `220-critical-workflow-hardening`  
**Created**: 2026-06-09  
**Status**: Draft  
**Input**: Backlog card F013 and Agentic Excellence Roadmap AE001-AE028

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Safely create and edit coins through explicit contracts (Priority: P1)

As a collector, I want coin create and edit requests to use explicit fields so routine collection changes do not accidentally overwrite related data.

**Why this priority**: Coin create/edit is the foundation for manual collection management and future agentic write paths.

**Independent Test**: Submit create/update requests for one-field edits and association edits, then verify only intended fields and related records changed.

**Acceptance Scenarios**:

1. **Given** an authenticated collector editing one scalar coin field, **When** the update is saved, **Then** only that field changes and existing associations remain intact.
2. **Given** a collector updates storage location, tags, sets, references, legacy/custom era fields, or current value, **When** the update is saved, **Then** the service applies the intended business rule and preserves unrelated sibling data.
3. **Given** a request contains unknown or broad model payload fields, **When** the handler binds the request, **Then** only allowlisted typed request fields can affect the mutation.

---

### User Story 2 - Reuse golden fixture coins for workflow regressions (Priority: P1)

As a developer, I want representative coin fixtures so backend and frontend workflow tests exercise realistic collection states.

**Why this priority**: Critical workflow tests need shared, recognizable data rather than one-off ad hoc records.

**Independent Test**: Build tests from the fixture set and verify it includes the required representative coin states.

**Acceptance Scenarios**:

1. **Given** tests need a representative collection, **When** fixture builders are used, **Then** they can create Roman, Greek, Byzantine, wishlist, sold, private, tagged, set-member, storage-location, image-heavy, and legacy/custom-era coins.
2. **Given** backend tests use fixture builders, **When** a fixture is persisted, **Then** ownership and association data are valid for service/repository checks.
3. **Given** frontend workflow tests use fixture data, **When** the test session starts, **Then** the expected coins and relationships are available deterministically.

---

### User Story 3 - Run deterministic browser tests for critical collection workflows (Priority: P1)

As a developer, I want repeatable browser workflow tests so login, add, edit, image, search, and mobile regressions are caught before Brian finds them manually.

**Why this priority**: Browser-level coverage is the safety net for the app's most valuable workflows and the handoff point to later AI-driven exploration in F011.

**Independent Test**: Run the critical workflow command locally and verify the scripted browser flows pass against a seeded test collection.

**Acceptance Scenarios**:

1. **Given** the app is running with test fixtures, **When** the critical workflow test command runs, **Then** login, add coin, edit one field, edit storage location, edit tags/sets, upload/delete image, search/filter, and mobile edit workflows pass.
2. **Given** a tested workflow fails, **When** the command exits, **Then** failure output identifies the broken workflow and enough context to reproduce it.
3. **Given** F011 later adds AI-driven exploration, **When** it needs seed data and target workflows, **Then** it reuses this feature's fixture and workflow definitions instead of inventing new ones.

### Edge Cases

- Update request omits fields that already have values; omitted fields must not be zeroed.
- Update request sends empty values intentionally; empty strings and explicit nullable scalar `null` values clear their corresponding allowlisted fields while omitted fields preserve existing values.
- Storage location is changed to another owned location, cleared, or set to an invalid/non-owned location.
- Tags, sets, and references are added, removed, or left unchanged in the same edit flow.
- Legacy/custom era values coexist with structured era/category fields.
- Current value changes create or preserve value-history side effects according to existing service rules.
- Image-heavy coins remain usable in browser tests without relying on production files.
- Mobile viewport tests must not depend on desktop-only controls.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Coin create/update handlers MUST bind explicit typed request DTOs rather than broad `models.Coin` mutation payloads.
- **FR-002**: Typed request DTOs MUST map to service-layer inputs without bypassing existing business rules.
- **FR-003**: Service logic MUST preserve storage location, era, reference, value history, tag, and set behavior while narrowing mutation contracts.
- **FR-004**: Repository update helpers MUST have one obvious association-safe mutation path, or each remaining helper MUST have a distinct documented purpose.
- **FR-005**: Backend regression tests MUST cover one-field edits, storage-location edits, tags, sets, references, legacy/custom era values, and value snapshots.
- **FR-006**: Repository tests MUST cover association-safe updates for representative related data.
- **FR-007**: Golden fixtures MUST cover Roman, Greek, Byzantine, wishlist, sold, private, tagged, set-member, storage-location, image-heavy, and legacy/custom-era coins.
- **FR-008**: Deterministic browser workflow tests MUST cover login, add coin, edit one field, edit storage location, edit tags/sets, upload/delete image, search/filter, and mobile viewport edit.
- **FR-009**: The repository MUST expose a local Taskfile command for the critical workflow suite.
- **FR-010**: `docs/testing.md` MUST document fixture setup and the critical workflow command.
- **FR-011**: Public API documentation MUST be regenerated if create/update request or response contracts change.

### Key Entities *(include if feature involves data)*

- **CoinCreateRequest**: Explicit handler input for creating a coin through the existing service path.
- **CoinUpdateRequest**: Explicit handler input for updating allowlisted coin fields and related data.
- **GoldenCollectionFixture**: Deterministic test data shape covering representative coin categories, states, and associations.
- **CriticalWorkflowTest**: Browser-level deterministic test for a named user workflow.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of coin create/update handler mutation inputs use typed DTOs instead of binding broad model payloads.
- **SC-002**: Regression tests prove scalar edits do not unintentionally clear storage location, tags, sets, references, images, or era/value data.
- **SC-003**: Golden fixtures cover every required representative coin state listed in FR-007.
- **SC-004**: One documented local command runs the deterministic critical workflow browser suite.
- **SC-005**: Future F011 AI-driven browser testing can reference this feature's fixture and workflow definitions as its baseline.

## Assumptions

- F013 hardens existing manual and API workflows; it does not introduce new agentic write behavior.
- Deterministic browser tests in this feature use conventional scripted automation; AI-driven exploratory browser testing remains F011 scope.
- Existing auth, ownership, and layered API boundaries remain in force.
- The first implementation slice should finish typed mutation contracts and backend regressions before browser automation expands.
