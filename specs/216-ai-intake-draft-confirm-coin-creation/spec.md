# Feature Specification: AI Intake Draft + Confirm Coin Creation

**Feature Branch**: `216-ai-intake-draft-confirm-coin-creation`  
**Created**: 2026-05-30  
**Status**: Draft  
**Input**: GitHub issue #216

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Generate structured intake draft from images (Priority: P1)

As a collector, I want AI to generate a structured coin draft from uploaded photos/OCR/lookups so I can start from a populated form instead of manual entry.

**Why this priority**: This is the core value of issue #216 and the prerequisite for review and commit.

**Independent Test**: Submit image-driven intake request and verify response contains structured coin candidate fields, confidence summary, and source evidence.

**Acceptance Scenarios**:

1. **Given** uploaded images and optional intake prompt, **When** draft generation runs, **Then** API returns a typed draft payload.
2. **Given** uncertain extractions, **When** draft is returned, **Then** uncertain fields are called out with confidence and evidence metadata.
3. **Given** partial extraction failure, **When** draft generation completes, **Then** system returns partial draft with unresolved field list (not silent failure).

---

### User Story 2 - Review and edit draft before save (Priority: P1)

As a collector, I want to review confidence/evidence and edit draft values so I remain in control before any coin is persisted.

**Why this priority**: Confirm-gated writes require a clear review/edit stage before persistence.

**Independent Test**: Open draft review panel, inspect confidence/evidence, edit multiple fields, and stage override payload for commit.

**Acceptance Scenarios**:

1. **Given** a drafted intake result, **When** review UI opens, **Then** confidence and evidence are visible for user inspection.
2. **Given** uncertain fields, **When** user edits values, **Then** edits replace draft values in staged commit payload.
3. **Given** user cancels review, **When** no commit is submitted, **Then** no coin write occurs.

---

### User Story 3 - Explicitly confirm draft to create coin (Priority: P1)

As a collector, I want coin creation to occur only after explicit confirmation so AI suggestions never auto-write to my collection.

**Why this priority**: Explicit confirmation is the safety control required by the issue and epic.

**Independent Test**: Confirm an edited draft and verify coin creation, draft status transition, and coin journal source tagging.

**Acceptance Scenarios**:

1. **Given** a valid draft and user overrides, **When** commit endpoint is called, **Then** coin is created through existing Go write path.
2. **Given** successful commit, **When** operation completes, **Then** draft status transitions to `confirmed`.
3. **Given** successful commit, **When** journal is recorded, **Then** source is tagged as AI intake (`coin_intake`).

### Edge Cases

- Draft ID is missing, invalid, expired, or already confirmed.
- Commit attempt is made by non-owner for another user's draft.
- OCR returns incomplete data for required coin fields.
- Evidence URIs are unavailable; evidence still captured as textual provenance.
- Duplicate submission of same commit request must not create duplicate coins.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST implement an AI intake team (`coin_intake`) that returns typed draft coin payloads.
- **FR-002**: System MUST include confidence and evidence metadata in draft responses.
- **FR-003**: System MUST persist generated drafts in a draft store before commit.
- **FR-004**: System MUST provide `POST /api/coins/intake/draft` for authenticated users.
- **FR-005**: System MUST provide `POST /api/coins/intake/commit` for authenticated users.
- **FR-006**: System MUST require explicit user confirmation payload to commit a draft.
- **FR-007**: System MUST allow user-provided field overrides at commit time.
- **FR-008**: System MUST create coins only through existing Go API write path (no direct Python DB writes).
- **FR-009**: System MUST write a coin journal entry for committed intake creations with source tag `coin_intake`.
- **FR-010**: System MUST enforce user scoping so draft and commit actions only apply to the authenticated owner's drafts.
- **FR-011**: System MUST preserve backward compatibility with existing manual coin creation flow.

### Key Entities *(include if feature involves data)*

- **CoinIntakeDraft**: Persisted structured draft payload and metadata prior to confirmation.
- **IntakeEvidenceItem**: Source/evidence item attached to draft confidence and field provenance.
- **CoinJournal**: Existing audit entity extended with AI intake source tag on commit.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of intake draft responses include structured payload + confidence metadata.
- **SC-002**: 100% of successful commits require explicit draft confirmation (no auto-persist path).
- **SC-003**: 100% of successful intake commits create a coin journal entry tagged `coin_intake`.
- **SC-004**: Review UI allows editing of all commit-eligible draft fields before save.
- **SC-005**: Invalid/expired/non-owned draft commit attempts are rejected with deterministic API errors.

## Assumptions

- Feature 1 structured references (#215) is a soft dependency; intake should attach references when available but remain functional without that feature fully shipped.
- Existing auth middleware and owner scoping in the Go API is reused.
- Add Coin page remains the host surface for intake draft/review/confirm UX.
