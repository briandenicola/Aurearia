# Feature Specification: Mint ↔ Location Integration

**Feature Branch**: `338-mint-location-integration`
**Created**: 2026-07-26
**Status**: Draft
**Input**: User description: "I want to tie Mints and Locations better. Right now
locations are defined by the admin in settings and are used to map a coins mint
onto a map but it feels disconnected. I would like to make mints a drop down in
all places it's used that pulls from the defined location. The user can also
specify unknown if they are unsure or they can create a new mint that is user
specific. When a user creates a new mint then the system determines the
coordinates for the user or asks the user if it can't determine."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Pick a mint from a managed list (Priority: P1)

When adding or editing a coin, the user picks the mint from a dropdown sourced
from the admin-curated `MintLocation` list, instead of typing free text.

**Why this priority**: This is the core disconnect being fixed — today a coin's
mint and the map's location list are two unrelated pieces of data that only
happen to line up if the user's spelling matches exactly.

**Independent Test**: Add/edit a coin, open the mint field, confirm it is a
dropdown populated from existing mint locations, and saving links the coin to
the selected location.

**Acceptance Scenarios**:

1. **Given** a coin form, **When** the user opens the mint dropdown, **Then**
   they see all global (admin-curated) mint locations plus any private mints
   they've created themselves.
2. **Given** a coin already linked to a mint location, **When** the user views
   the Mint Map, **Then** the coin is grouped under that location without any
   client-side name matching.

---

### User Story 2 - Mark a mint as unknown (Priority: P1)

The user isn't sure of a coin's mint and wants to say so explicitly rather than
leaving it blank or guessing.

**Why this priority**: Ancient/unattributed coins are common in this domain;
forcing a guess would corrupt data quality.

**Independent Test**: Select "Unknown" in the mint dropdown, save, confirm the
coin shows no mint location and appears in the Mint Map's existing "unknown"
bucket.

**Acceptance Scenarios**:

1. **Given** the mint dropdown, **When** the user selects "Unknown", **Then**
   the coin is saved with no mint location linked.

---

### User Story 3 - Create a user-specific mint with coordinates (Priority: P1)

The user's coin was struck somewhere not in the admin list. They add a new
mint by name; the system looks up coordinates automatically and lets the user
confirm or adjust the location before saving.

**Why this priority**: This is what makes the dropdown viable in practice —
the admin list will never be exhaustive for every collector's coins.

**Independent Test**: From the mint dropdown, choose "Create new mint", type a
recognizable place name, confirm the system proposes a pin, adjust it if
needed, save, and confirm the new mint appears in the dropdown under "My
Mints" and is usable immediately on the current coin.

**Acceptance Scenarios**:

1. **Given** the create-mint flow, **When** the user types a name the geocoder
   recognizes, **Then** the system shows a proposed pin on a map for the user
   to confirm or drag into place before saving.
2. **Given** the create-mint flow, **When** the user types a name the geocoder
   cannot resolve, **Then** the system shows an empty map and asks the user to
   click to place the pin themselves — there is no dead end.
3. **Given** a mint the user created, **When** another user opens their own
   mint dropdown, **Then** they do not see it (private to the creator).

---

### User Story 4 - Nudge on unlinked legacy mints (Priority: P2)

A coin saved before this feature has a free-text mint that doesn't match any
mint location. The user should be nudged to reconcile it, without being
blocked.

**Why this priority**: Existing collections have free-text mint values already;
forcing immediate cleanup on every old coin would be disruptive.

**Independent Test**: Open a coin whose legacy `mint` text has no matching
mint location; confirm a non-blocking banner appears offering to link or
create a mint, and confirm the coin can still be saved/viewed without acting
on it.

**Acceptance Scenarios**:

1. **Given** a coin with an unlinked legacy mint string, **When** the user
   opens the coin's edit form, **Then** they see an inline, dismissible banner
   suggesting they link it to an existing or new mint location.
2. **Given** the same coin, **When** the user ignores the banner and saves
   other changes, **Then** the save succeeds and the legacy mint text is left
   untouched.

---

### Edge Cases

- What happens when a user's typed new-mint name partially matches an
  existing global mint (e.g. alias collision)? → Block creation of a private
  mint with the same normalized name/alias as any global mint; suggest picking
  the existing global one instead.
- What happens when the geocoder returns multiple ambiguous candidates for a
  name (e.g. "Antioch")? → Show all candidates on the map for the user to
  pick from, defaulting to none pre-selected.
- What happens to the Mint Map's existing "unattributed" bucket once most
  coins are linked via FK? → It becomes a fallback path for legacy/unlinked
  free-text coins only, not the primary grouping mechanism.
- What happens if a user deletes a private mint that's still linked to one of
  their coins? → Reject the delete (mirrors existing `StorageLocation`
  in-use protection) until the coin is reassigned.
- What happens if the geocoding service is unreachable? → Treat it the same as
  "no results found" and fall back to manual pin placement.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST present mint entry, everywhere it appears (coin
  form, wishlist search-alert criteria, smart-set criteria), as a selection
  from mint locations rather than free text.
- **FR-002**: System MUST offer an explicit "Unknown" option in every mint
  dropdown, distinct from an unlinked legacy value.
- **FR-003**: Users MUST be able to create a new mint location from within the
  dropdown, scoped privately to themselves.
- **FR-004**: System MUST attempt to determine coordinates for a new
  user-created mint automatically (geocoding by name).
- **FR-005**: System MUST let the user confirm or manually adjust/place the
  coordinates before the new mint is saved, regardless of whether geocoding
  succeeded.
- **FR-006**: System MUST NOT expose a user-created private mint to any other
  user's dropdown, mint map, or aggregate data.
- **FR-007**: System MUST prevent a private mint from being created with a
  name/alias that collides with an existing global mint.
- **FR-008**: System MUST continue to support existing global (admin-curated)
  mint locations exactly as today, with no change to the admin CRUD workflow.
- **FR-009**: System MUST surface a non-blocking nudge on coins whose legacy
  free-text mint doesn't match any mint location, offering to link or create
  one, without blocking save/view of the coin.
- **FR-010**: System MUST prevent deletion of a mint location (global or
  private) that is still referenced by a coin.
- **FR-011**: System MUST backfill existing coins' mint text against known
  mint locations (global + owner's private) on rollout, linking matches
  automatically and leaving non-matches as unlinked legacy text (per FR-009).

### Key Entities

- **MintLocation**: A named, geocoded place (existing entity). Gains an
  optional owner (`UserID`) — absent means global/admin-curated, present means
  private to that user.
- **Coin**: Gains an optional link to a `MintLocation`. Retains its existing
  free-text mint field as a denormalized display value / legacy fallback.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A new coin saved through the updated form always has either a
  linked mint location or an explicit "Unknown" — never an arbitrary free-text
  string that could silently fail to match anything.
- **SC-002**: Creating a user-specific mint, from typing a name to it being
  usable in the dropdown, takes no more than a few seconds when geocoding
  succeeds, and never dead-ends when it doesn't.
- **SC-003**: The Mint Map groups the large majority of coins via direct
  FK lookup rather than fuzzy string matching after the backfill runs.
- **SC-004**: No user-created private mint is ever visible to, or affects the
  aggregate map/statistics of, another user.

## Assumptions

- Geocoding is performed server-side against OpenStreetMap Nominatim (no API
  key required), sending only the typed mint name — never coin, collection,
  or account data — consistent with the app's existing OSM/Leaflet usage and
  privacy posture.
- The geocoding result is always presented for user confirmation/adjustment on
  a map before being saved (hybrid approach), rather than auto-accepted or
  requiring fully manual entry in all cases.
- Promoting a user's private mint to the global list is a useful future
  capability but is explicitly out of scope for this pass.
- Existing global mint locations and the admin CRUD workflow for them are
  unchanged by this feature.
