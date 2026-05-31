# Feature Specification: Refine Coin Details Page for PWA and Desktop

**Feature Branch**: `219-refine-coin-details-page`  
**Created**: 2026-05-31  
**Status**: Draft  
**Input**: GitHub issue #219 + user direction for streamlined/elegant detail UI inspired by provided reference images

## User Scenarios & Testing *(mandatory)*

### User Story 1 - See both coin sides immediately in a cleaner overview (Priority: P1)

As a collector, I want the coin detail overview to show obverse and reverse by default so I can inspect both sides immediately without extra interaction.

**Why this priority**: This is the primary visual/UX goal and the first impression of the detail page.

**Independent Test**: Open any coin detail page with obverse/reverse images and verify both sides are visible in the default overview layout on desktop and PWA.

**Acceptance Scenarios**:

1. **Given** a coin has both obverse and reverse images, **When** detail overview loads, **Then** both sides are displayed by default in the hero/media section.
2. **Given** one side is missing, **When** detail overview loads, **Then** existing available side is shown and missing side is represented with deterministic fallback text/placeholder.
3. **Given** the user is in PWA mode, **When** detail overview loads, **Then** layout remains non-sticky and mobile-optimized per existing PWA rules.

---

### User Story 2 - Read coin metadata in a consistent table format (Priority: P1)

As a collector, I want coin facts presented in a clean table/list-row format instead of boxed cards so the detail page is more consistent and easier to scan.

**Why this priority**: Replacing card fragmentation with one consistent information pattern is central to the requested redesign.

**Independent Test**: Open detail overview and verify metadata renders in table rows with consistent labels/values using design tokens, replacing current detail info boxes.

**Acceptance Scenarios**:

1. **Given** coin metadata fields are present, **When** overview renders, **Then** each field appears as a standardized label/value table row.
2. **Given** optional fields are absent, **When** overview renders, **Then** empty-value handling remains consistent and does not create broken row styling.
3. **Given** desktop and PWA layouts, **When** metadata table renders, **Then** typography/spacing remains consistent with design-token rules in both modes.

---

### User Story 3 - Navigate deep detail sections from overview links (Priority: P1)

As a collector, I want Journal, Notes, Actions, and AI Analysis moved to dedicated pages linked from overview rows (settings-style) so the main page stays focused and elegant.

**Why this priority**: Reducing overview density is required to streamline the experience while preserving all existing capabilities.

**Independent Test**: From coin overview, navigate through settings-style link rows (`Journal >`, `Notes >`, `Actions >`, `AI Analysis >`) and verify each destination page shows its section content and preserves coin context.

**Acceptance Scenarios**:

1. **Given** the overview page, **When** the user taps/clicks a section link row, **Then** app routes to the corresponding coin subpage.
2. **Given** a coin subpage is loaded directly by URL, **When** route resolves, **Then** it loads the same coin context and section data.
3. **Given** user returns from a section page, **When** navigating back, **Then** overview state and media context are preserved.

### Edge Cases

- Coin has no images: overview must render gracefully with placeholders and no broken layout.
- Coin has only one side image: dual-side section still renders with explicit missing-side indicator.
- Notes/journal/analysis are empty: section pages must render clear empty-state messaging.
- Deep links to section pages with invalid coin ID must follow existing not-found/error handling path.
- Desktop sticky behavior must not be introduced into PWA mode (per constitution mobile rules).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST redesign `CoinDetailPage` overview to show obverse and reverse media by default.
- **FR-002**: System MUST replace boxed detail info presentation with a standardized table/list-row metadata format.
- **FR-003**: System MUST move Journal content from overview to a dedicated coin journal page.
- **FR-004**: System MUST move Notes content from overview to a dedicated coin notes page.
- **FR-005**: System MUST move Actions panel from overview to a dedicated coin actions page.
- **FR-006**: System MUST move AI Analysis surface from overview to a dedicated coin analysis page.
- **FR-007**: System MUST expose settings-style link rows on overview for `Journal`, `Notes`, `Actions`, and `AI Analysis`.
- **FR-008**: System MUST add authenticated coin-detail subroutes for each moved section and preserve existing auth guards.
- **FR-009**: System MUST preserve all existing section capabilities after move (add/delete journal entries, notes visibility, action workflows, AI analysis operations).
- **FR-010**: System MUST keep desktop and PWA layout behavior compliant with existing mode rules (desktop sticky allowed, PWA non-sticky).
- **FR-011**: System MUST implement new visuals using existing design tokens and global classes from `variables.css`/`main.css`.
- **FR-012**: System MUST maintain existing coin detail data loading behavior without introducing cross-user data access changes.

### Key Entities *(include if feature involves data)*

- **CoinDetailOverviewViewModel**: Presentation model for hero media (obverse/reverse), title block, and condensed metadata rows.
- **CoinDetailSectionLink**: Navigation item representing a destination section page (`journal`, `notes`, `actions`, `analysis`).
- **CoinDetailSectionPageContext**: Shared route-level context carrying coin ID, coin identity fields, and section-specific data surface.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of coin detail overview loads show dual-side media when both images exist.
- **SC-002**: 100% of metadata in scope for overview is displayed through table/list-row format (no legacy boxed info cards in overview).
- **SC-003**: 100% of moved sections (Journal, Notes, Actions, AI Analysis) are reachable through overview link rows and direct routes.
- **SC-004**: Existing section functionality remains operational after migration (no regressions in journal CRUD, actions execution, or analysis refresh flows).
- **SC-005**: PWA and desktop modes pass visual behavior checks for this feature without violating mode-specific sticky/non-sticky rules.

## Assumptions

- Existing coin detail API payload shape is sufficient for overview + section pages; no backend schema change is required.
- Existing coin detail subcomponents can be reused or lightly adapted in dedicated pages rather than rewritten from scratch.
- User-provided reference images are design inspiration for hierarchy and spacing, not exact visual cloning requirements.
