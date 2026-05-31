# Feature Specification: Agentic Collection Updates (Epic F012)

**Feature Branch**: `012-agentic-collection-updates`  
**Created**: 2026-05-30  
**Status**: Draft  
**Input**: Backlog epic `specs/_backlog/F012-agentic-collection-updates.md`

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Structured catalog references (Priority: P1)

As a collector, I want normalized coin catalog references (RIC/RPC/Sear/KM/etc.) so records are searchable, portable, and machine-usable.

**Why this priority**: It is foundational data quality work that materially improves the other three features.

**Independent Test**: Create and edit a coin with multiple references from different catalogs; verify normalized entries validate and render consistently in API and UI.

**Acceptance Scenarios**:

1. **Given** an authenticated owner editing a coin, **When** they add references, **Then** each reference is stored as typed catalog data (not only free text).
2. **Given** a reference with a malformed identifier for a known catalog, **When** it is submitted, **Then** validation fails with a clear field-level error.
3. **Given** a coin with references, **When** coin details are fetched, **Then** normalized references and authority URIs are returned.

---

### User Story 2 - Agentic coin entry draft + confirm (Priority: P1)

As a collector, I want AI to draft a fully populated coin record from photos/OCR/lookups so I can review and confirm instead of entering everything manually.

**Why this priority**: This is the highest direct UX gain for day-to-day data entry.

**Independent Test**: Upload photos to intake flow, review draft fields/confidence, confirm save, and verify coin + journal entries persist.

**Acceptance Scenarios**:

1. **Given** image uploads and an intake prompt, **When** the intake team runs, **Then** it returns a structured draft with confidence and source citations.
2. **Given** a draft with uncertain fields, **When** user reviews it, **Then** uncertain values are clearly marked and editable before commit.
3. **Given** user confirms draft, **When** commit executes, **Then** coin is created via Go API write path and journal records AI-assisted creation.

---

### User Story 3 - Collection chat over owned data (Priority: P1)

As a collector, I want to ask conversational questions about my collection and apply safe updates through confirm-gated proposals.

**Why this priority**: It turns existing collection data into actionable conversational workflows.

**Independent Test**: Ask read queries and write-intent prompts in chat; verify scoped query results and two-phase update behavior.

**Acceptance Scenarios**:

1. **Given** an authenticated user, **When** they ask collection questions, **Then** answers come from user-scoped collection tools (`get_coin`, `query_coins`, `aggregate`) with structured payloads.
2. **Given** a write-intent message, **When** chat proposes an update, **Then** no DB write occurs until explicit confirmation.
3. **Given** confirmed update, **When** commit runs, **Then** only allowlisted fields are changed and journal includes source `collection_chat`.

---

### User Story 4 - External collection tool server parity (Priority: P2)

As a power user, I want external clients (OpenWebUI/Ollama and Claude Desktop) to query and update my collection through the same guarded tool layer.

**Why this priority**: It expands utility beyond in-app chat while reusing the same validated logic.

**Independent Test**: Execute read and write flows from an external client using API key auth, confirm read/write parity with in-app tool behavior.

**Acceptance Scenarios**:

1. **Given** a valid scoped API key, **When** an external client calls read tools, **Then** results match in-app chat tool responses for same user.
2. **Given** external write proposal, **When** commit token is missing/expired, **Then** commit is rejected and nothing is changed.
3. **Given** confirmed external commit, **When** update succeeds, **Then** journal entry includes source `external_tool_server`.

### Edge Cases

- Conflicting catalog entries on the same coin (duplicate catalog+number pairs) must deduplicate or reject deterministically.
- Intake image/OCR failures return partial drafts rather than silent errors.
- Ambiguous chat target coin (multiple matches) must trigger disambiguation, not guessed updates.
- External clients attempting cross-user access via crafted IDs must be denied server-side.
- Proposal tokens must expire and become unusable after commit/revoke.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST introduce a normalized `CoinReference` model linked to a coin with catalog type, identifier, display label, and optional authority URI.
- **FR-002**: System MUST preserve backward compatibility for existing `referenceText` and `referenceURL` fields while introducing structured references.
- **FR-003**: System MUST validate known catalog identifier formats with per-catalog rules.
- **FR-004**: System MUST provide CRUD APIs for structured references through existing authenticated coin workflows.
- **FR-005**: System MUST provide an AI intake pipeline (`coin_intake`) that returns structured draft payloads, confidence buckets, and source evidence.
- **FR-006**: System MUST require user confirmation before persisting intake drafts.
- **FR-007**: System MUST implement a transport-agnostic collection tool layer in Go for reads: `get_coin`, `query_coins`, `aggregate`.
- **FR-008**: System MUST implement two-phase write tools: `propose_update` then `commit_update`.
- **FR-009**: System MUST enforce server-side user scoping for all tool calls; user identity cannot be client-supplied.
- **FR-010**: System MUST support a collection chat mode in the existing app chat surface.
- **FR-011**: System MUST expose external access to the same tool layer for supported clients with read and write parity.
- **FR-012**: System MUST enforce protocol-level confirm gating for all external writes (proposal token required for commit).
- **FR-013**: System MUST restrict conversational/external updates to an allowlist (`grade`, `currentValue`, `notes`, `tags`, `structured references`, `referenceText`, `referenceURL`) in v1.
- **FR-014**: System MUST journal all AI/external initiated commits with source tags and diff metadata.
- **FR-015**: System MUST publish interface contracts for collection tool operations and auth expectations.

### Key Entities *(include if feature involves data)*

- **CoinReference**: Normalized reference row for a coin (catalog, identifier, authority URI).
- **CoinIntakeDraft**: Structured AI-generated draft for new coin creation before user commit.
- **CollectionToolRequest**: Normalized internal request envelope for read/update tool operations.
- **CollectionUpdateProposal**: Proposed mutation with diff, allowlist validation status, and expiry.
- **CollectionUpdateCommit**: Confirmed proposal audit/commit event and journal metadata.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 95%+ of newly created coins via intake include at least one structured reference when source evidence is present.
- **SC-002**: In-app collection chat read tool responses return within 2s p95 for collections up to 500 active coins.
- **SC-003**: 100% of write attempts from chat/external without valid proposal token are rejected in integration tests.
- **SC-004**: External server read/write operations match in-app tool behavior with schema parity in contract tests.
- **SC-005**: All committed AI/external updates produce journal records with source and changed-field metadata.

## Assumptions

- Existing Go API remains the only write path; Python agent stays stateless and DB-disconnected.
- Existing auth and API-key infrastructure is reused for external server access.
- First delivery targets single-user ownership scope (no follower/social collection editing).
- External clients are enabled through explicit user configuration and API key permissions.
