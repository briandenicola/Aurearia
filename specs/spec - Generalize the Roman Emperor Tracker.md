# Feature Specification: Tracker Sets (Registry Set Primitive)

**Feature Branch**: `010-tracker-sets`
**Created**: 2026-07-26
**Status**: Draft
**Input**: User description: "Generalize the Roman Emperor Tracker into a reusable, data-driven set type. A Tracker Set has a fixed roster of slots, its own link under the Sets menu, goal-based progress tracking (filled / total), and a museum tray display where every slot always renders — owned coins in filled slots, simple silhouette placeholders in unfilled slots."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - View a Tracker Set with placeholders (Priority: P1)

A collector opens the Sets menu and sees a dedicated link for each of their Tracker Sets (e.g., "Emperors", "Wheat Cents"). Opening one shows the full roster rendered in the museum tray: slots matched to owned coins display the coin's image and label; unfilled slots display a silhouette placeholder with the slot label (e.g., "1914-D"). A progress indicator shows "42 of 145 filled (29%)".

**Why this priority**: This is the core value of the primitive — visualizing a complete goal set with gaps — and everything else (migration, dynamic creation) depends on it.

**Independent Test**: Seed a Tracker Set with a roster via the API, add coins matching some slots, and verify the tray renders all slots (filled + placeholder) with correct progress.

**Acceptance Scenarios**:

1. **Given** a Tracker Set with 10 slots and 3 matched coins, **When** the user opens the set detail page, **Then** the tray renders 10 slots in roster order, 3 showing owned coin images and 7 showing silhouette placeholders with slot labels, and progress reads "3 of 10 (30%)".
2. **Given** a Tracker Set exists, **When** the user opens the Sets menu, **Then** a navigation entry for that Tracker Set appears under a Trackers section at `/sets/trackers/:slug`.
3. **Given** a Tracker Set with grouped slots (e.g., by decade or dynasty), **When** the tray renders, **Then** slots appear grouped with group headings in the configured order.

---

### User Story 2 - Slots fill automatically as the collection changes (Priority: P1)

A collector adds a new coin (or edits an existing one). Any Tracker Set slot whose match criteria the coin now satisfies fills automatically without manual assignment. If the coin is later deleted or edited so it no longer matches, the slot reverts to its placeholder.

**Why this priority**: Automatic matching is what distinguishes a tracker from a manually curated open set; without it the roster goes stale immediately.

**Independent Test**: Create a Tracker Set with a slot matching "1943 steel cent"; add a matching coin; verify the slot fills. Edit the coin's year; verify the slot reverts to placeholder.

**Acceptance Scenarios**:

1. **Given** an unfilled slot with criteria (year=1943, denomination=cent), **When** a coin matching those criteria is added to the collection, **Then** the slot shows that coin on next view and progress increments.
2. **Given** a filled slot, **When** the matched coin is deleted or edited to no longer match, **Then** the slot reverts to its placeholder and progress decrements.
3. **Given** two owned coins both match one slot, **When** the tray renders, **Then** exactly one coin fills the slot (deterministic selection, e.g., highest grade then earliest acquisition), and the user can override which coin occupies the slot.
4. **Given** one coin matches two different slots in the same set, **When** matching runs, **Then** the coin fills only one slot per set (deterministic assignment; user can override).

---

### User Story 3 - Manage Tracker Sets manually (Priority: P2)

A collector creates a Tracker Set by hand: names it, defines slots (label, match criteria, group, order), reorders or edits slots, and deletes the set. This provides full lifecycle management independent of any AI-assisted creation.

**Why this priority**: Manual CRUD is required for corrections and for collectors who don't use AI features, and it is the API surface a future dynamic-creation feature will target.

**Independent Test**: Create, edit, reorder, and delete a Tracker Set entirely through the UI and verify nav entries and tray output update accordingly.

**Acceptance Scenarios**:

1. **Given** the set creation wizard, **When** the user selects type "Tracker" and defines at least one slot, **Then** the set is created, appears in the Sets menu, and its tray renders.
2. **Given** an existing Tracker Set, **When** the user adds/edits/removes/reorders slots, **Then** the tray and progress update and matching recalculates for changed slots.
3. **Given** an existing Tracker Set, **When** the user deletes it, **Then** the nav entry disappears and no coin records are modified or deleted.

---

### User Story 4 - Emperor Tracker migrated onto the primitive (Priority: P2)

The existing Emperor Tracker becomes the first Tracker Set. Its nav entry, progress numbers, and coverage behavior are preserved for users who had it enabled, with no user action required.

**Why this priority**: Migration proves the primitive is genuinely general and removes the bespoke code path, but the feature is valuable even before migration completes.

**Independent Test**: On a database with Emperor Tracker enabled and partial coverage, run the migration and verify identical figure coverage counts pre/post at the new route, with the old route redirecting.

**Acceptance Scenarios**:

1. **Given** a user with Emperor Tracker enabled and N figures covered, **When** migration completes, **Then** a "Emperors" Tracker Set exists with the same roster and N filled slots.
2. **Given** the legacy `/sets/emperors` route, **When** visited post-migration, **Then** the user is redirected to the new Tracker Set route.
3. **Given** a user with Emperor Tracker disabled in settings, **When** migration completes, **Then** no Emperors Tracker Set appears in their nav.

---

### Edge Cases

- Roster with a very large slot count (e.g., 500+): tray must paginate or lazy-render; define a per-set slot ceiling (see FR-014).
- Slot criteria that match no possible coin (contradictory criteria): slot remains a permanent placeholder; UI flags "no coin can match" during editing preview.
- Duplicate slugs when two sets normalize to the same slug: system appends a suffix and guarantees uniqueness per user.
- Wishlist/sold coins: matching considers only owned collection coins by default; sold coins vacate their slot.
- Set privacy: a private Tracker Set never appears in public showcase or follower views, including its placeholders.
- Concurrent edit: a coin edit during a roster edit must not corrupt assignments (matching recalculation is idempotent).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST support a new set type "tracker" alongside open, defined, goal, and smart sets.
- **FR-002**: A Tracker Set MUST consist of an ordered roster of slots; each slot has a label, match criteria, optional group name, and sort order.
- **FR-003**: Slot match criteria MUST support at minimum: year/date range, mint mark, denomination, ruler/figure, material, era/category, and structured catalog reference — combinable with AND logic.
- **FR-004**: System MUST automatically evaluate slot matching when coins are created, updated, or deleted, and on demand via a full re-match action.
- **FR-005**: Exactly one owned coin MAY occupy a slot; assignment MUST be deterministic when multiple coins match, and users MUST be able to manually pin a specific owned coin to a slot (pin wins over automatic assignment while the pinned coin still matches).
- **FR-006**: The museum tray for a Tracker Set MUST render every slot, using the owned coin image for filled slots and a silhouette placeholder for unfilled slots. v1 placeholders are simple generic silhouettes selected by denomination/shape; no external reference images.
- **FR-007**: Placeholder slots MUST display the slot label and MUST be visually distinct from owned coins (e.g., muted/embossed rendering on the tray felt).
- **FR-008**: Each Tracker Set MUST appear as its own link in the Sets navigation, addressable at a stable per-user slug route.
- **FR-009**: System MUST display progress as filled/total and percentage, and include Tracker Sets in the existing snapshot, trends, and milestone-notification mechanisms.
- **FR-010**: Users MUST be able to create, edit (metadata and roster), reorder, and delete Tracker Sets; deletion MUST NOT affect coin records.
- **FR-011**: System MUST migrate the Emperor Tracker to a Tracker Set preserving roster, coverage counts, per-user enablement, and redirecting the legacy route.
- **FR-012**: Tracker Sets MUST respect existing set privacy controls and per-coin privacy toggles.
- **FR-013**: All Tracker Set endpoints MUST require authentication with no cross-user access, consistent with existing set APIs.
- **FR-014**: System MUST enforce an admin-configurable maximum slots per Tracker Set (default: [NEEDS CLARIFICATION: proposed 250 — confirm ceiling given tray rendering performance]).
- **FR-015**: Roster editing MUST provide a preview showing which owned coins would fill which slots before saving.

### Key Entities

- **TrackerSet**: A set of type "tracker" — name, slug, description, icon/color, privacy flag, group ordering, relationship to owner and to its slots. Reuses existing set metadata, snapshot, and trend records.
- **TrackerSlot**: One roster position — label, match criteria, group, sort order, optional pinned coin reference, current assignment state.
- **SlotAssignment**: The link between a slot and the owned coin currently occupying it, including whether it was pinned or auto-assigned (may be modeled as fields on TrackerSlot).
- **PlaceholderStyle**: A small enumerated mapping from coin shape/denomination class to a bundled silhouette asset (static app data, not user content).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A collector can create a 50-slot Tracker Set manually and see correct fills and placeholders in the tray in under 5 minutes of effort.
- **SC-002**: Adding or editing a coin reflects in affected Tracker Set progress within one page refresh (no manual re-match needed) for collections up to 5,000 coins.
- **SC-003**: Emperor Tracker migration produces identical coverage counts for 100% of migrated users, with zero data loss.
- **SC-004**: Tray view of a 250-slot Tracker Set renders without visible jank on a mid-range mobile device (initial paint under 2 seconds on a warm cache).

## Assumptions

- The existing Coin Sets data model (sets, snapshots, trends, milestones) can be extended rather than replaced; Tracker Sets reuse those mechanisms.
- The museum tray component (Tray View) can render non-coin placeholder items with modest extension.
- Matching criteria reuse the field vocabulary already established by Smart Set rules and Defined Set targets; no new coin metadata fields are required for v1.
- Placeholder silhouettes are a small bundled asset library (roughly: round-large, round-small, and a default), shipped with the app; per-slot reference images are out of scope for v1.
- Dynamic/AI-assisted creation of Tracker Sets is out of scope for this spec and covered by the companion spec `011-dynamic-set-builder`, which depends on this one.
- Existing `/api/stats/emperors` consumers (if any beyond the UI) will be preserved or shimmed during migration.
