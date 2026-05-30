# Feature Specification: Collection Health Scorecard (v1)

**Feature Branch**: `208-collection-health-scorecard`  
**Created**: 2026-05-30  
**Status**: Draft  
**Input**: GitHub issue #208 — https://github.com/briandenicola/coin-collection-app/issues/208

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See collection health at a glance (Priority: P1)

As a collector, I want one collection-level health score (0-100), letter grade (A-F), and 30-day trend indicator so I can quickly understand overall catalog quality and whether it is improving.

**Why this priority**: This is the top-level value proposition; without the collection score and trend, the feature does not deliver dashboard insight.

**Independent Test**: With a seeded collection containing mixed-completeness coins, open dashboard and verify score, grade, and 30-day delta match backend calculations.

**Acceptance Scenarios**:

1. **Given** an authenticated user with at least one active coin, **When** they open the dashboard, **Then** they see collection score (0-100), grade (A-F), and component contributions using default weights (metadata 40%, images 20%, valuation freshness 20%, AI coverage 20%).
2. **Given** a user with historical snapshots, **When** they view collection health, **Then** a 30-day trend delta is shown and reflects current score minus score from ~30 days earlier.
3. **Given** a user with no eligible coins, **When** they open dashboard, **Then** scorecard displays an empty-state message and no misleading grade.

---

### User Story 2 - Improve low-quality coin records quickly (Priority: P1)

As a collector, I want each coin’s health score, missing-items checklist, and a Needs Attention queue ordered by worst scores so I can fix the most important gaps fast.

**Why this priority**: Per-coin visibility and remediation is required to make the score actionable.

**Independent Test**: Open coin list/detail and Needs Attention queue; confirm lowest-score coins appear first and quick actions route to existing edit/upload/valuation/analysis flows.

**Acceptance Scenarios**:

1. **Given** a coin missing required data, **When** user views coin row/detail, **Then** they see coin score, grade, and checklist of missing items by dimension.
2. **Given** a set of coins with varying health, **When** user opens Needs Attention queue, **Then** coins are sorted ascending by score, with deterministic tie-breaker by oldest update time then coin ID.
3. **Given** a queue entry with identified gaps, **When** user selects a quick action, **Then** they are taken to the relevant existing workflow (edit metadata, upload images, run valuation, run AI analysis).

---

### User Story 3 - Monitor quality across users as admin (Priority: P2)

As an admin, I want aggregate health metrics so I can understand system-wide data quality and guide cleanup priorities.

**Why this priority**: Admin aggregate visibility is important but secondary to collector-facing functionality.

**Independent Test**: Log in as admin and verify median score, low-score percentage, and top missing fields are available and reflect all active collections.

**Acceptance Scenarios**:

1. **Given** an admin user, **When** they open admin health panel, **Then** they see median score and percentage of low-score coins.
2. **Given** mixed data quality across users, **When** admin checks aggregate panel, **Then** top missing fields list reflects most frequent missing checklist items.
3. **Given** a non-admin user, **When** they access admin aggregate endpoint/panel, **Then** access is denied with existing admin authorization behavior.

### Edge Cases

- User has no active coins (all sold/wishlist or none created): collection scorecard shows empty state, no grade.
- Coin has no valuation history and no current value: valuation freshness subscore is 0 and checklist includes valuation action.
- Coin has only one side image: partial image score and checklist indicates missing opposite side image.
- Coin has generic “Other” category/material or empty metadata fields: metadata checklist marks gaps but does not crash scoring.
- 30-day trend data unavailable (new user): trend displays “insufficient history” instead of 0 delta.
- Queue tie scores: deterministic order uses oldest `updated_at`, then smallest coin ID.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST compute a per-coin health score from 0-100 using weighted subscores: metadata completeness 40%, image coverage 20%, valuation freshness 20%, AI analysis coverage 20%.
- **FR-002**: System MUST compute and return a collection-level health score and grade based on eligible coins (active, non-sold, non-wishlist by default).
- **FR-003**: System MUST map scores to grades using fixed thresholds: A (90-100), B (80-89), C (70-79), D (60-69), F (<60).
- **FR-004**: System MUST provide per-coin missing-items checklist grouped by dimensions (metadata, image, valuation, AI analysis) suitable for UI display.
- **FR-005**: System MUST expose a Needs Attention queue sorted by lowest per-coin score first, with deterministic tie-breaking.
- **FR-006**: System MUST provide quick-action metadata for each queue item to drive existing UI actions (edit metadata, upload image, run valuation, run analysis).
- **FR-007**: System MUST show a 30-day collection trend indicator derived from persisted daily health snapshots.
- **FR-008**: System MUST persist daily collection health snapshots with at least user ID, snapshot date, score, and grade distribution to support trend calculation.
- **FR-009**: System MUST provide admin-only aggregate metrics: median coin health score, low-score coin percentage (<60), and most common missing checklist fields.
- **FR-010**: System MUST document scoring formula, threshold definitions, and missing-field rules in repository docs for deterministic implementation/testing.
- **FR-011**: System MUST default to issue-defined scoring weights (40/20/20/20) and treat weight customization as out-of-scope for v1.

### Key Entities *(include if feature involves data)*

- **CoinHealthScore**: Computed per-coin projection containing total score, grade, dimension subscores, checklist items, and quick-action hints.
- **CollectionHealthSummary**: Computed aggregate for one collection containing total score, grade, dimensions, eligible coin count, and 30-day trend delta.
- **CollectionHealthSnapshot**: Persisted daily record for trend calculations with user ID, date, score, and grade distribution counts.
- **HealthMissingFieldStat**: Admin aggregate projection representing a missing-field key and frequency across low-quality coins.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Dashboard health score + grade renders in under 1.5s p95 for collections up to 500 coins on local self-hosted deployment.
- **SC-002**: Needs Attention queue returns first page (25 items) in under 2s p95 for collections up to 500 coins.
- **SC-003**: Per-coin scores in list/detail match deterministic backend calculation with 100% parity across API and UI tests.
- **SC-004**: 30-day trend indicator is available for at least 95% of users after 31 days of scheduler operation.
- **SC-005**: Admin aggregate panel metrics are accessible only to admin users (0 authorization bypasses in integration tests).

## Assumptions

- Existing coin lifecycle flags (`is_sold`, `is_wishlist`) continue to define “active collection” eligibility for v1 health calculations.
- Existing valuation and AI endpoints are reused for quick actions; this feature does not introduce new analysis/valuation engines.
- Daily snapshot generation can be executed by existing scheduler infrastructure in the Go API service.
- v1 does not include per-user custom scoring weights, custom grade boundaries, or historical backfill beyond available data.
