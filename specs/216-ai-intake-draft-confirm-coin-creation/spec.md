# Feature Specification: AI Intake Draft + Confirm Coin Creation

**Feature Branch**: `216-ai-intake-draft-confirm-coin-creation`  
**Created**: 2026-05-30  
**Status**: Draft  
**Input**: GitHub issue #216

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Generate structured intake draft from images (Priority: P1)

As a collector, I want AI to generate a structured coin draft from uploaded photos and optional coin-card metadata so I can start from a populated form instead of manual entry.

**Why this priority**: This is the core value of issue #216 and the prerequisite for review and commit.

**Independent Test**: Submit image-driven intake request with optional coin-card upload and verify response contains structured coin candidate fields, confidence summary, and source evidence.

**Acceptance Scenarios**:

1. **Given** uploaded coin images and an optional coin-card image, **When** draft generation runs, **Then** API returns a typed draft payload.
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

---

### User Story 4 - Bypass AI and use manual entry directly (Priority: P1)

As a collector, I want to skip AI intake and use the existing manual Add Coin form so desktop-heavy workflows remain fast and familiar.

**Why this priority**: Manual entry remains a common and expected path; AI intake is optional assistance, not a required gate.

**Independent Test**: Choose manual mode from Add Coin, submit a coin, and verify no intake draft/commit flow is required.

**Acceptance Scenarios**:

1. **Given** Add Coin page is open, **When** user chooses manual entry, **Then** intake draft generation is bypassed and existing manual fields are immediately usable.
2. **Given** manual mode is selected, **When** user saves coin, **Then** coin is created via existing manual create flow.
3. **Given** user switches between manual and AI intake paths, **When** no confirm action is sent, **Then** no intake commit write occurs.

---

### User Story 5 - PWA opens directly into camera-ready AI intake (Priority: P1)

As a PWA user, I want Add Coin to open directly into the agentic capture flow with camera ready so I can quickly observe and identify a coin from my phone.

**Why this priority**: Mobile/PWA capture speed is the primary UX win for intake and should reduce friction in on-the-go entry.

**Independent Test**: Open Add Coin in PWA mode and verify camera-ready intake opens by default, upload fallback exists, and manual-mode bypass link is available.

**Acceptance Scenarios**:

1. **Given** Add Coin is opened in PWA mode and camera permission is granted, **When** the intake view loads, **Then** agentic intake opens by default with camera ready for capture.
2. **Given** camera permission is denied/unavailable, **When** intake view loads, **Then** user can still continue with image upload for intake.
3. **Given** camera intake view is visible, **When** user clicks `Use Manual Mode instead`, **Then** the existing manual Add Coin flow opens immediately without intake commit side effects.

### Edge Cases

- Draft ID is missing, invalid, expired, or already confirmed.
- Commit attempt is made by non-owner for another user's draft.
- OCR returns incomplete data for required coin fields.
- Optional coin-card upload is invalid/unsupported and must return deterministic validation errors.
- Camera permission is denied or revoked in PWA mode; upload fallback remains available.
- Camera hardware is unavailable in PWA mode; workflow degrades to upload-based intake.
- Evidence URIs are unavailable; evidence still captured as textual provenance.
- Duplicate submission of same commit request must not create duplicate coins.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST implement an AI intake team (`coin_intake`) that returns typed draft coin payloads.
- **FR-002**: System MUST include confidence and evidence metadata in draft responses, including provenance from optional coin-card input when present.
- **FR-003**: System MUST persist generated drafts in a draft store before commit.
- **FR-004**: System MUST provide `POST /api/coins/intake/draft` for authenticated users and accept optional coin-card image input.
- **FR-005**: System MUST provide `POST /api/coins/intake/commit` for authenticated users.
- **FR-006**: System MUST require explicit user confirmation payload to commit a draft.
- **FR-007**: System MUST allow user-provided field overrides at commit time.
- **FR-008**: System MUST create coins only through existing Go API write path (no direct Python DB writes).
- **FR-009**: System MUST write a coin journal entry for committed intake creations with source tag `coin_intake`.
- **FR-010**: System MUST enforce user scoping so draft and commit actions only apply to the authenticated owner's drafts.
- **FR-011**: System MUST preserve backward compatibility with existing manual coin creation flow.
- **FR-012**: System MUST provide an explicit UI path to bypass AI intake and use manual coin entry directly.
- **FR-013**: System MUST keep AI intake optional; coin creation cannot require intake draft generation.
- **FR-014**: In PWA mode, Add Coin MUST default to the agentic intake surface with camera capture initialized when permission is granted.
- **FR-015**: In PWA mode intake view, system MUST provide upload-based intake as an alternative to camera capture.
- **FR-016**: In PWA mode intake view, system MUST render a clear manual-bypass link (`Use Manual Mode instead`) below camera capture.
- **FR-017**: Camera permission denial/unavailability MUST not block intake; users can continue via image upload or manual mode.

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
- **SC-006**: Manual add-coin path remains available and functional without invoking intake draft/commit endpoints.
- **SC-007**: In PWA mode, Add Coin opens in camera-ready agentic intake when permission is granted, with visible upload and manual bypass alternatives.

## Assumptions

- Feature 1 structured references (#215) is a soft dependency; intake should attach references when available but remain functional without that feature fully shipped.
- Existing auth middleware and owner scoping in the Go API is reused.
- Add Coin page remains the host surface for intake draft/review/confirm UX and explicit manual bypass.
- PWA-specific default behavior applies only to PWA mode; non-PWA/desktop keeps explicit user choice without forced camera startup.
