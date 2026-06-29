# Feature Specification: Wishlist Search Alerts for Acquisition Ideas

**Feature Branch**: `feature/357-wishlist-search-alerts`  
**Created**: 2026-06-29  
**Status**: Draft  
**Input**: GitHub issue #357: "F019: Add wishlist search alerts for acquisition ideas"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Define acquisition search alerts (Priority: P1)

As a collector, I want to save search criteria for coins I may want to acquire so the app can find candidate listings for future wishlist additions without mixing those searches into my existing wishlist availability checks.

**Why this priority**: Alert criteria are the foundation for discovery. Without a clear saved alert, the system cannot produce explainable acquisition candidates or keep discovery separate from availability checks for already-saved wishlist URLs.

**Independent Test**: Create, edit, disable, and delete an alert with numismatic, price, source, and cadence criteria; verify the saved alert is visible to the owner and does not create or modify wishlist items by itself.

**Acceptance Scenarios**:

1. **Given** an authenticated collector, **When** they create an alert with ruler or issuer, coin type, date range, mint, material, grade or condition, price range, optional source or domain filters, and cadence preference, **Then** the alert is saved and listed with its active state and criteria summary.
2. **Given** an existing alert, **When** the collector updates criteria to narrow or broaden the search, **Then** future alert runs use the updated criteria and previous run history remains attributable to the criteria used at run time.
3. **Given** an alert is disabled or deleted, **When** discovery runs are requested or scheduled, **Then** disabled or deleted alerts do not produce new candidates while existing review history remains understandable.
4. **Given** the collector has existing wishlist items with saved URLs, **When** they create or edit a discovery alert, **Then** no existing wishlist availability check configuration or history is changed.

---

### User Story 2 - Run an alert and review acquisition candidates (Priority: P1)

As a collector, I want to run a saved alert and review candidate listings with provenance so I can decide whether any result is worth adding to my wishlist.

**Why this priority**: The core value of the feature is discovery of acquisition ideas, but candidate results must be source-backed and explainable to avoid misleading collectors.

**Independent Test**: Run an alert manually against mocked or controlled search results; verify each candidate includes required source details, reason for match, lifecycle state, and duplicate suppression.

**Acceptance Scenarios**:

1. **Given** an active alert, **When** the collector selects Run Now, **Then** the system records a run and returns candidates with source URL, observed price when available, title, reason for match, last-seen timestamp, and provenance status.
2. **Given** source data cannot verify a candidate URL, price, title, or availability claim, **When** the result is shown, **Then** the candidate is clearly marked as unverified or partial rather than inventing missing dealer details.
3. **Given** the same candidate is found repeatedly for the same alert, **When** a new run completes, **Then** the system updates last-seen information or suppresses duplicates instead of creating noisy repeated review items.
4. **Given** a run finds no suitable matches, **When** the collector views the run, **Then** the app shows a clear no-candidates result and preserves enough run metadata to diagnose whether the criteria were too narrow.

---

### User Story 3 - Act on candidates without losing review control (Priority: P1)

As a collector, I want to dismiss poor candidates, convert promising candidates into wishlist items, or adjust alert criteria so I can keep future discovery useful and low-noise.

**Why this priority**: Candidate review must produce actionable collector outcomes while preserving explicit user control over wishlist additions.

**Independent Test**: Review candidate results, dismiss at least one candidate, convert another to a wishlist item, and adjust alert criteria; verify lifecycle transitions and wishlist creation are explicit and traceable.

**Acceptance Scenarios**:

1. **Given** a candidate is irrelevant, duplicate, too expensive, or otherwise unwanted, **When** the collector dismisses it, **Then** it leaves the active review queue for that alert and the dismissal reason is retained if provided.
2. **Given** a candidate looks promising, **When** the collector saves it as a wishlist item, **Then** a new wishlist item is created from verified candidate fields and the candidate is marked as converted.
3. **Given** a candidate has partial or uncertain source data, **When** the collector attempts conversion, **Then** the app requires the collector to review or supply missing wishlist fields before saving.
4. **Given** a run produces noisy results, **When** the collector adjusts criteria from the candidate review flow, **Then** the updated alert criteria reduce future matches without altering already-converted wishlist items.

---

### User Story 4 - Preserve existing wishlist availability checks (Priority: P1)

As a collector, I want existing wishlist availability checks to continue validating already-saved wishlist URLs, while search alerts discover new acquisition candidates separately.

**Why this priority**: The issue explicitly requires discovery alerts not to be conflated with existing availability checks. Regression here would make both workflows less trustworthy.

**Independent Test**: Start with an existing wishlist item that has a saved source URL; run its availability check and a separate discovery alert; verify histories, statuses, and review actions remain separate.

**Acceptance Scenarios**:

1. **Given** an existing wishlist item with a saved URL, **When** its availability check runs, **Then** the check validates that saved URL only and does not create acquisition candidates.
2. **Given** a search alert discovers a candidate listing, **When** the candidate is reviewed, **Then** it remains an alert candidate until the collector explicitly converts it into a wishlist item.
3. **Given** a candidate is converted to a wishlist item, **When** future availability checks run, **Then** checks operate on the saved wishlist URL and are recorded in wishlist availability history rather than alert run history.
4. **Given** the same source URL appears in an existing wishlist item and an alert candidate, **When** results are displayed, **Then** the app identifies the relationship and prevents accidental duplicate wishlist additions.

---

### User Story 5 - Prepare for scheduled review without overbuilding notifications (Priority: P2)

As a collector, I want alerts to store a cadence preference and support manual review now, so scheduled runs and notifications can be enabled later without changing the collector-facing alert model.

**Why this priority**: Cadence is part of the issue's criteria, but v1 should avoid overbuilding push or digest notifications unless existing scheduler patterns make them straightforward.

**Independent Test**: Save cadence metadata, run alerts manually, and verify in-app review surfaces candidates without requiring push, email, or digest notification delivery.

**Acceptance Scenarios**:

1. **Given** a collector creates an alert, **When** they choose a cadence preference, **Then** the preference is stored and visible even if v1 only supports manual Run Now and in-app candidate review.
2. **Given** scheduled execution is not enabled, **When** an alert is due according to cadence, **Then** the app does not imply a notification was sent or a scheduled run occurred.
3. **Given** future scheduling is enabled, **When** automated runs are introduced, **Then** they use the same run history and candidate lifecycle as manual runs.

### Edge Cases

- Alert criteria are too broad and return many candidates; the system must cap review volume, explain truncation or filtering, and encourage criteria refinement.
- Alert criteria are contradictory or incomplete, such as minimum price above maximum price or an invalid date range.
- A source URL redirects, canonicalizes differently, becomes unavailable, or changes listing title or price between runs.
- A candidate lacks observed price, grade, mint, or other optional details; missing values must be shown as unknown rather than guessed.
- A dealer or source domain filter excludes all results.
- A candidate appears for multiple alerts owned by the same collector; each alert keeps its own review state while duplicate relationships are visible.
- A candidate is already represented by an existing wishlist item or previously converted candidate.
- Search or agent execution times out, is rate-limited, or returns partial results.
- Provenance data conflicts across sources; the system preserves each source-backed claim and avoids merging incompatible facts silently.
- A dismissed candidate reappears with a changed URL, title, or price; duplicate detection should prevent obvious repeats while allowing materially changed listings to be reviewed.
- Private collection data, pricing preferences, and wishlist information must remain scoped to the authenticated owner.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST allow authenticated collectors to create, read, update, disable, and delete wishlist search alerts.
- **FR-002**: Alert criteria MUST support ruler or issuer, coin type, date range, mint, material, grade or condition, price range, dealer/source preference, optional source or domain filters, cadence preference, and free-text notes or keywords.
- **FR-003**: System MUST validate alert criteria before saving, including price ranges, date ranges, empty criteria, unsupported cadence values, and malformed source or domain filters.
- **FR-004**: Alert criteria and run history MUST be scoped to the owning collector and inaccessible to other users.
- **FR-005**: System MUST support manual Run Now for an active alert in v1.
- **FR-006**: System MUST store cadence preference data for alerts, but v1 MUST NOT claim push, email, or digest notification delivery unless a working scheduled delivery path is implemented and verified.
- **FR-007**: Search discovery MUST use the existing search-agent capability behind the application-owned service boundary for web/dealer discovery in v1; per-dealer scrapers are out of scope for v1.
- **FR-008**: The application-owned service boundary MUST own alert persistence, user scoping, run history, candidate lifecycle, duplicate suppression, rate-limit decisions, and conversion to wishlist items.
- **FR-009**: Agent/search outputs MUST preserve provenance for source URL, observed price, title, source/dealer name when present, last-seen timestamp, and reason for match.
- **FR-010**: System MUST NOT invent dealer details, availability claims, prices, titles, or source facts that are not present in source-backed results.
- **FR-011**: Each alert run MUST record alert identity, run trigger type, run status, started time, completed time when available, criteria snapshot, result counts, errors or partial-result warnings, and rate-limit status when applicable.
- **FR-012**: Each candidate result MUST include source URL, observed price when available, title, reason for match, last-seen timestamp, provenance status, lifecycle state, and the alert/run that produced it.
- **FR-013**: System MUST allow collectors to dismiss a candidate and SHOULD allow a dismissal reason such as irrelevant, duplicate, price too high, poor provenance, or other.
- **FR-014**: System MUST allow collectors to convert a candidate into a wishlist item only through an explicit save action.
- **FR-015**: Candidate conversion MUST prefill wishlist fields only from source-backed candidate data and MUST require collector review or input for required wishlist fields that are missing or uncertain.
- **FR-016**: System MUST allow collectors to adjust alert criteria from candidate review context without modifying previously converted wishlist items.
- **FR-017**: System MUST keep search alert data, run history, candidate lifecycle, and candidate provenance separate from wishlist availability check history.
- **FR-018**: Existing wishlist availability checks MUST continue to validate already-saved wishlist URLs and MUST NOT be treated as discovery alert runs.
- **FR-019**: Converted candidates MUST link back to their source alert candidate for traceability while becoming normal wishlist items for subsequent availability checks.
- **FR-020**: Duplicate suppression MUST use canonical source URL, normalized title, observed price, and alert identity, with canonical source URL as the strongest identity signal.
- **FR-021**: System MUST detect and warn when a candidate appears to match an existing wishlist item or previously converted candidate before allowing duplicate wishlist creation.
- **FR-022**: Candidate review surfaces MUST show verified, partial, and unverified source data distinctly so collectors can judge provenance quality.
- **FR-023**: System MUST handle search failures, source rate limits, timeouts, and partial responses with clear run statuses and non-secret user-facing messages.
- **FR-024**: System MUST enforce reasonable per-user and per-alert run limits before scheduled runs are enabled to avoid excessive source traffic or noisy candidate queues.
- **FR-025**: System MUST provide enough auditability to answer which alert criteria, source URL, and observed facts led to a candidate or converted wishlist item.

### Key Entities

- **WishlistSearchAlert**: Collector-owned saved discovery criteria for acquisition ideas, including numismatic filters, price/source filters, cadence preference, active state, and timestamps.
- **AlertCriteriaSnapshot**: Immutable copy of criteria used for a specific run so results remain explainable after the alert is edited.
- **AlertRun**: Execution record for a manual or future scheduled discovery attempt, including trigger type, status, timing, result counts, errors, and rate-limit/partial-result indicators.
- **AlertCandidate**: Source-backed acquisition idea produced by an alert run, including source URL, title, observed price, reason for match, last-seen timestamp, provenance status, lifecycle state, duplicate key, and relationship to alert/run.
- **CandidateProvenance**: Evidence record for observed facts such as URL, title, price, dealer/source label, availability text, timestamp, and confidence/verification state.
- **CandidateReviewAction**: Collector action on a candidate, such as dismissed, converted, restored, or criteria-adjusted, with timestamp and optional reason.
- **WishlistItem**: Existing wishlist record; may be created from a converted candidate and later participates in existing wishlist availability checks.
- **WishlistAvailabilityCheck**: Existing validation workflow for already-saved wishlist URLs; remains separate from discovery alert runs and candidate review.

## Constitution-Aligned Constraints

- **Clear layered ownership**: Business decisions for alert lifecycle, candidate review, duplicate suppression, and wishlist conversion belong in the service layer; persistence belongs in repositories; user-facing handlers remain thin.
- **Service boundary separation**: The application-owned API is the source of truth for persistence and user scoping. Search-agent capability may discover candidates but must not directly access the database or bypass the API-owned review and conversion workflow.
- **Strict contracts and provenance**: Alert criteria, run results, candidates, and conversion payloads need explicit request/response contracts with typed provenance fields so missing or uncertain source facts are represented deterministically.
- **Security and privacy by default**: Alert criteria, run history, wishlist candidates, prices, and private collection context remain authenticated and owner-scoped. Errors must not expose internal details or secrets.
- **Consistent UX**: Candidate review, alert CRUD, and conversion flows must work in desktop and PWA contexts and avoid UI copy that implies unverified availability.
- **Simple, complete, proportional scope**: v1 includes alert CRUD, manual runs, in-app review, candidate lifecycle, provenance, duplicate suppression, and conversion. Per-dealer scrapers, push notifications, email digests, and broad scheduler rollout are out of scope unless existing project patterns make them low-risk.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A collector can create an alert with at least 8 supported criteria fields in under 3 minutes during usability testing or manual verification.
- **SC-002**: 100% of candidate results displayed from alert runs include source URL, title, reason for match, last-seen timestamp, lifecycle state, and provenance status.
- **SC-003**: 0 candidate results display invented dealer details, prices, titles, or availability claims when source-backed data is missing.
- **SC-004**: 100% of converted candidates require an explicit collector save action before a wishlist item is created.
- **SC-005**: Existing wishlist availability checks for already-saved wishlist URLs continue to pass their current regression coverage and remain recorded separately from alert run history.
- **SC-006**: Duplicate suppression prevents obvious repeated candidates for the same alert in at least 95% of repeat-run cases using canonical URL, normalized title, observed price, and alert identity.
- **SC-007**: Candidate review lets a collector dismiss or convert at least 20 returned candidates without losing page context or altering alert criteria unintentionally.
- **SC-008**: Manual alert runs return a completed, failed, rate-limited, or partial status with user-readable explanation within 30 seconds for normal result volumes.
- **SC-009**: Owner-scoping tests verify that collectors cannot view, run, dismiss, convert, or edit alerts and candidates owned by another user.
- **SC-010**: v1 can be released without push, email, or digest notifications while still preserving cadence preference data for future scheduled runs.

## Assumptions

- The target collector is an authenticated owner of the personal collection app.
- v1 uses the existing Python agent/search capability behind the Go API service boundary for web/dealer discovery and does not add custom per-dealer scrapers.
- v1 supports manual Run Now and in-app candidate review. Scheduled cadence metadata is captured, but push notifications, email digests, and automatic scheduled delivery are deferred unless existing scheduler patterns make them straightforward during planning.
- Optional source/domain filters are allowed so collectors can constrain discovery, but source coverage is best-effort and must be represented honestly.
- Duplicate detection uses canonical source URL plus normalized title plus observed price plus alert ID, with canonical URL as the strongest identity signal.
- A candidate can be converted to a wishlist item only after collector review; search alerts never automatically create wishlist items.
- Existing wishlist availability checks already validate saved wishlist URLs and are preserved as a separate workflow.
- Source provenance can be partial. Missing or uncertain facts are acceptable if they are clearly marked and not invented.
- Rate limits and result caps are acceptable in v1 to protect source sites and keep review queues manageable.
