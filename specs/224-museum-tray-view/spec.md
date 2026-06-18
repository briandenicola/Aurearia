# Spec: Museum Coin Tray View

**Feature Branch**: `224-museum-tray-view-rerun`
**Created**: 2026-06-18
**Last Updated**: 2026-06-18 (tightened for accuracy)
**Status**: Draft
**Input**: Historical spec from `224-museum-tray-view` branch, adapted for current beta state

---

## Update History

| Date | Change |
|------|--------|
| 2026-06-18 | Tightened spec to match actual codebase (store is `useCoinsStore()`, route is `/coin/:id`); clarified responsive behavior without specific class names; simplified scope to current loaded collection only. |

## Summary

Add a read-only **Tray view** display mode that presents the collection as physical coins seated in a museum exhibit tray. Coins rest in felt-lined circular wells on a textured felt background with soft shadows, conveying the collection as tangible objects rather than data cards. The tray is responsive, paginates large collections into multiple drawers, and scales coins proportionally by their diameter when available.

---

## User Scenarios & Testing *(mandatory)*

### User Story 1: Enter Tray View (Priority: P1)

As a collector, I open the "Tray" view from the collection page and see my coins seated in felt-lined wells, so the collection reads as physical objects.

**Why this priority**: Core feature. Without this, the tray doesn't exist. Enables all other stories.

**Independent Test**: Able to navigate from collection page to tray view and see coins rendered in wells. Empty tray shows friendly message if collection is empty.

**Acceptance Scenarios**:

1. **Given** I am on the collection page with coins loaded, **When** I click/tap the "Tray" view button, **Then** I navigate to `/tray` and see coins seated in felt-lined wells with soft shadows.
2. **Given** I am on an empty collection, **When** I navigate to `/tray`, **Then** I see a friendly empty state and option to return to collection or add coins.
3. **Given** I am viewing the tray, **When** I return to the collection page, **Then** my filters and sort order are preserved.

---

### User Story 2: Size-Accurate Coin Display (Priority: P2)

As a collector with a varied collection, I see coins sized relative to each other based on their actual diameter, so the tray honestly reflects their physical scale.

**Why this priority**: Enhances visual authenticity. Depends on diameter data availability. If diameter is missing, fallback gracefully.

**Independent Test**: Can verify coins with different diameters (small obol 8mm, large sestertius 35mm) render at proportionally different sizes; coins without diameter use safe default size; all coins remain tappable (min tap target met).

**Acceptance Scenarios**:

1. **Given** I have coins with `diameterMm` populated, **When** I view the tray, **Then** coins are scaled proportionally (larger coins appear visibly larger than smaller coins).
2. **Given** I have coins with missing or zero `diameterMm`, **When** they render in the tray, **Then** they use a sensible default size and remain fully tappable.
3. **Given** coin diameters range from 8mm to 40mm, **When** displayed in the tray, **Then** render sizes are clamped to safe min/max bounds (e.g., 40px–120px) so small coins remain visible and large coins don't dominate.

---

### User Story 3: Responsive Tray Layout (Priority: P2)

As a collector on mobile, tablet, or desktop, the tray grid reflows sensibly for my viewport, wells remain finger-friendly on small screens, and pagination controls are compact and discoverable.

**Why this priority**: Core UX requirement for PWA. Tray is useless if it doesn't adapt to device orientation and size.

**Independent Test**: On mobile (375px), tablet (768px), and desktop (1024px+), verify wells are appropriately sized, pagination controls are visible, and no horizontal scroll is needed.

**Acceptance Scenarios**:

1. **Given** I am on a mobile device (375px), **When** I view the tray, **Then** wells stack in a 2–3 column grid, remain finger-friendly, and pagination is compact and accessible.
2. **Given** I am on a tablet (768px), **When** I view the tray, **Then** wells stack in a 4–5 column grid with appropriate gaps.
3. **Given** I am on desktop (1024px+), **When** I view the tray, **Then** wells stack in a 6–8 column grid with balanced spacing.
4. **Given** any viewport, **When** I rotate the device, **Then** the grid reflows smoothly and content remains accessible.

---

### User Story 4: Pagination and Navigation (Priority: P2)

As a collector with a large collection, my tray paginates into multiple "drawers" so I can browse without overwhelming the page, and I can navigate between drawers.

**Why this priority**: Necessary for usability with 100+ coins. Drawer navigation must be intuitive and position-preserving.

**Independent Test**: With a collection of 100+ coins, verify Previous/Next drawer buttons work, drawer count updates correctly, opening a coin and returning preserves drawer position.

**Acceptance Scenarios**:

1. **Given** I have 150 coins and the tray shows 50 per drawer, **When** I view the tray, **Then** I see Drawer 1 of 3 with Previous/Next navigation.
2. **Given** I am on Drawer 2 of 3, **When** I click a coin to open detail, **Then** I open the coin detail page.
3. **Given** I am viewing a coin detail and return to the tray, **Then** I return to the same drawer and position I left.
4. **Given** I am on the last drawer, **When** I click Next, **Then** the button is disabled or no-op (no wrapping).

---

### User Story 5: Coin Interaction (Priority: P1)

As a collector in the tray, I can tap a coin to open its detail page, so I can inspect or edit it without leaving the tray view unnecessarily.

**Why this priority**: Core interaction. Tray is read-only presentation, but users must be able to access full coin details.

**Independent Test**: Tap/click a coin well, and it routes to `/coin/:id` detail page. Keyboard Enter on a focused well also opens detail. Long-press is not required.

**Acceptance Scenarios**:

1. **Given** I am viewing the tray, **When** I click/tap a coin well, **Then** I navigate to the coin's detail page (`/coin/:id`).
2. **Given** I am viewing the tray and a well is focused (keyboard), **When** I press Enter, **Then** the coin detail page opens.
3. **Given** I am viewing a coin from the tray and click back, **Then** I return to the tray (not the collection page).

---

### User Story 6: Felt Theming (Priority: P3)

As a collector with an eye for aesthetics, I can choose a felt color theme (red, green, navy) for the tray, so it matches my collection's presentation style or personal taste.

**Why this priority**: Nice-to-have visual polish. Ships after core interaction. Preference stored in localStorage.

**Independent Test**: Can click/tap felt color chips in the tray controls, tray background updates immediately, preference persists across session reloads.

**Acceptance Scenarios**:

1. **Given** I am viewing the tray, **When** I click a felt color chip (red, green, navy), **Then** the tray background theme updates immediately.
2. **Given** I have selected a felt color theme, **When** I reload the page, **Then** the selected theme persists.

---

### User Story 7: Reduced Motion Support (Priority: P3)

As an accessibility-conscious user with `prefers-reduced-motion` enabled, the tray respects my preference and avoids distracting animations.

**Why this priority**: Accessibility requirement per Principle V. Optional polish if unrelated animations exist.

**Independent Test**: With `prefers-reduced-motion: reduce`, verify hover/focus effects and any transitions are instant, not animated.

**Acceptance Scenarios**:

1. **Given** I have `prefers-reduced-motion: reduce` set in my OS, **When** I view the tray and hover/focus wells, **Then** transitions are instant (no animations).

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST render coins in a grid of circular felt-lined wells on a textured felt background.
- **FR-002**: System MUST use the current collection result set (`store.coins`) for v1; no API changes or cross-page data fetching.
- **FR-003**: System MUST scale coins proportionally by `diameterMm` within safe clamp bounds (min 40px, max 120px) when available; use a default size (e.g., 70px) when missing.
- **FR-004**: System MUST paginate the tray into drawable chunks (e.g., 50 coins per drawer) with Previous/Next navigation.
- **FR-005**: System MUST reflow the tray grid responsively: 2–3 columns on mobile (375px), 4–5 on tablet (768px), 6–8 on desktop (1024px+).
- **FR-006**: System MUST open coin detail (`/coin/:id`) when a user taps/clicks a well or presses Enter on a focused well.
- **FR-007**: System MUST preserve tray drawer position and return to the same drawer when a user navigates back from a coin detail page.
- **FR-008**: System MUST offer at least 3 felt color themes (red, green, navy) and persist the user's selection in localStorage.
- **FR-009**: System MUST respect `prefers-reduced-motion: reduce` and avoid animations on hover/focus.
- **FR-010**: System MUST show a friendly empty state if the collection is empty or the current filtered set is empty.
- **FR-011**: System MUST be reachable via a "Tray" button/chip in both desktop and PWA collection headers.
- **FR-012**: System MUST use only existing design tokens and global button/chip CSS classes; no hardcoded colors or spacing.

### Key Entities

- **Coin**: Existing entity with `id`, `name`, `diameterMm` (nullable), `images: CoinImage[]`. Reused from collection store.
- **Tray Layout**: Internal helper struct for calculating coin render sizes and grid layout based on viewport and coin diameter data.
- **TrayPreference**: LocalStorage key-value pair for felt color theme (e.g., `tray:feltColor = 'red'`).

### Data Flow

1. **CollectionPage** / **Headers** → User clicks "Tray" button.
2. **Router** → Navigate to `/tray`.
3. **TrayViewPage** → Fetch `store.coins` (already loaded from collection); if empty, show empty state.
4. **MuseumTray** → Calculate layout via `trayLayout.ts`, render wells grid.
5. **MuseumTrayWell** → Render coin image, shadow, handle click/keyboard.
6. **Route on click** → Open `/coin/:id`; preserve return path.

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can navigate to tray view from collection page in one click/tap.
- **SC-002**: Tray renders all loaded coins in the current drawer without jank (60fps on mobile, no long paint times).
- **SC-003**: 100% of coins with `diameterMm` are scaled proportionally; 100% of coins without it use safe default size.
- **SC-004**: On mobile, tray grid is 2–3 columns; on tablet 4–5; on desktop 6–8 (responsive layout verified, no hardcoded pixel values).
- **SC-005**: Users can navigate between drawers (Previous/Next) and the drawer count is accurately displayed.
- **SC-006**: Tapping a coin opens its detail page; keyboard Enter also works.
- **SC-007**: Returning from a coin detail preserves drawer position (no loss of context).
- **SC-008**: Felt color theme updates immediately on click and persists across reloads.
- **SC-009**: All component and utility tests pass: `npm run test -- tray --run`.
- **SC-010**: Type checking passes: `npm run type-check`.
- **SC-011**: Build succeeds: `npm run build`.
- **SC-012**: `prefers-reduced-motion: reduce` disables animations (visual inspection + test).

---

## Assumptions

- **Assumption 1**: The `diameterMm` field already exists on the Coin model and is populated for most coins; missing/zero values are acceptable and handled gracefully.
- **Assumption 2**: The collection store (`store.coins`) is already populated when the user navigates to `/tray`, so no additional API call is needed for v1.
- **Assumption 3**: The tray uses the **current loaded result set** (filtered, sorted, paginated) from the collection, not a separate "all coins" fetch. This simplifies the MVP and focuses on the current visible collection.
- **Assumption 4**: The default tap action is to open coin detail (`/coin/:id`). A secondary inline flip control (if 3D viewer exists) can be added but is not required for MVP.
- **Assumption 5**: Felt texture is pure CSS (gradients/layered backgrounds), not an image asset, for performance and themability.
- **Assumption 6**: Drag-and-drop tray rearrangement and saving custom layouts are out of scope and deferred to a future backlog card.
- **Assumption 7**: The tray is read-only. Editing coins, bulk actions, and selection mode do not apply in tray view.
- **Assumption 8**: PWA and desktop browsers share the same tray component; no separate mobile-only implementation is needed.
- **Assumption 9**: `localStorage` is available and secure enough for felt color preference (non-sensitive, UI-only data).
- **Assumption 10**: Coins per drawer is fixed at 50 (or adaptive based on page size) for v1; fully dynamic per-viewport sizing can be a follow-up polish task.
