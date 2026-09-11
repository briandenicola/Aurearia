# Feature Specification: Structured Coin Storage Trays

**Feature Branch**: `feat/structured-storage-trays`  
**Created**: 2026-09-11  
**Status**: Draft  
**Input**: Preserve free-form storage locations while adding fixed-dimension, slot-addressed physical coin trays and an owner-scoped visual tray view.

## Summary

Collectors can continue to use named, unstructured **Standard Locations**, or create named **Coin Trays** with a fixed number of rows and columns. A coin assigned to a tray must occupy one exact, exclusive slot. A separate authenticated **Storage Trays** view shows the owner's physical trays exactly as configured, including empty wells and coordinates.

This feature is deliberately distinct from the existing responsive **Museum Tray**. Museum Tray continues to arrange a collection responsively and its semantics do not change. Storage Trays represent persisted physical placement: rows, columns, and coin positions must never be reflowed or repacked for the viewport. In accordance with Constitution Principle IV, the feature reuses the existing lower-level felt, well, and coin visual primitives and styling without duplicating the museum-tray visual system or adopting its responsive layout algorithm.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Configure Storage Locations and Trays (Priority: P1)

As a collector, I can keep simple named locations and create fixed-dimension coin trays so the application matches how my collection is physically stored.

**Why this priority**: Tray configuration and backward-compatible standard locations establish the storage model required by every other workflow.

**Independent Test**: In Settings, create and rename a Standard Location, create a 3-by-3 Coin Tray, and verify that each is shown with the correct type while the tray shows 0 of 9 occupied.

**Acceptance Scenarios**:

1. **Given** the collector opens storage settings, **When** they choose Standard Location and provide a unique name, **Then** a free-form location is created without rows, columns, or slots.
2. **Given** the collector chooses Coin Tray, **When** they provide a unique name, 3 rows, and 3 columns, **Then** a tray with exactly 9 addressable slots is created and its dimensions and occupancy are shown.
3. **Given** the collector has an occupied 6-by-10 tray, **When** they rename it without changing its dimensions, **Then** the name changes and all 60 positions and existing assignments remain unchanged.
4. **Given** a tray contains at least one coin, **When** the collector attempts to change its dimensions or delete it, **Then** the operation is blocked with an explanation that preserves every coin assignment.
5. **Given** an empty tray, **When** the collector changes dimensions within the allowed limits or deletes it, **Then** the operation succeeds.
6. **Given** the collector enters a duplicate name or invalid dimensions, **When** they submit the form, **Then** the form identifies the invalid fields and no partial change is saved.

---

### User Story 2 - Assign a Coin to an Exact Tray Slot (Priority: P1)

As a collector, I can place a coin into one available tray slot so its recorded location matches its physical position.

**Why this priority**: Exact and exclusive placement is the feature's core data-integrity promise.

**Independent Test**: Assign one coin to row 2, column 3 of a 3-by-3 tray, verify that another coin cannot select that slot, then move the first coin to a Standard Location and verify that the old tray slot becomes available.

**Acceptance Scenarios**:

1. **Given** a coin add or edit form has no tray selected, **When** the collector views storage fields, **Then** no slot picker is shown.
2. **Given** the collector selects a tray, **When** the tray has available positions, **Then** the slot picker shows all coordinates, marks occupied slots unavailable, and requires one available slot before save.
3. **Given** a collector edits a coin already in a tray, **When** the slot picker opens, **Then** that coin's current slot remains selectable while slots occupied by other coins remain unavailable.
4. **Given** a coin is assigned to a tray and slot, **When** its location is cleared or changed to a Standard Location or another tray, **Then** the previous slot is cleared atomically and becomes available.
5. **Given** two saves attempt to claim the same slot concurrently, **When** both are processed, **Then** exactly one succeeds and the other receives a conflict response with a user-actionable message.
6. **Given** a bulk-assignment workflow is open, **When** storage choices are shown, **Then** Coin Trays are excluded or disabled with an explicit explanation that tray assignments require per-coin slot selection.

---

### User Story 3 - View Physical Storage Trays (Priority: P1)

As a collector, I can open a read-only Storage Trays page and see every configured tray with coins in their recorded physical positions, so I can find a coin or audit empty space.

**Why this priority**: The visual inventory is the main user value enabled by structured placement.

**Independent Test**: Configure trays of different dimensions, leave some wells empty, assign coins to non-contiguous coordinates, and verify that the page renders every tray with the exact persisted geometry and placement on desktop and mobile.

**Acceptance Scenarios**:

1. **Given** an authenticated collector has configured trays, **When** they choose Storage Trays under Collection navigation, **Then** a separate owner-scoped read-only route displays every configured tray.
2. **Given** a 3-by-3 tray has coins at (1,1) and (3,2), **When** it is displayed, **Then** it retains exactly three rows and three columns, the coins appear only at those coordinates, and all other wells remain visibly empty.
3. **Given** the page is viewed on a narrow PWA screen, **When** a tray cannot fit at its preferred size, **Then** its complete fixed geometry remains intact through whole-tray scaling or scrolling and no wells are reordered, repacked, hidden, or moved between rows.
4. **Given** a rendered coin is actionable, **When** the collector activates it by touch, pointer, or keyboard, **Then** the coin detail opens; empty wells do not expose misleading actions.
5. **Given** the collector has no configured Coin Trays, **When** the route loads, **Then** it shows a clear empty state with a path to create a tray.
6. **Given** tray data is loading or cannot be retrieved, **When** the page is displayed, **Then** it shows a non-destructive loading state or an error state with retry guidance rather than stale or invented placement.

---

### User Story 4 - Preserve Existing Data and Sibling Workflows (Priority: P1)

As an existing collector, I retain all current storage locations and coin assignments after upgrade, while imports, AI-assisted intake, duplication, and API clients obey the same placement rules.

**Why this priority**: A storage enhancement cannot risk losing existing location data or introduce alternate paths that bypass slot exclusivity.

**Independent Test**: Upgrade a collection containing existing locations and assignments, verify each location becomes Standard without assignment loss, then exercise manual creation, import, AI-assisted intake, edit, duplication, and bulk assignment against the shared storage contract.

**Acceptance Scenarios**:

1. **Given** existing user-defined storage locations and assigned coins, **When** the feature migration completes, **Then** every existing location is classified as Standard, retains its name and owner, and every existing coin retains its location without a slot.
2. **Given** a legacy client omits location type, dimensions, and slot fields, **When** it creates or updates a Standard Location or a coin in a Standard Location, **Then** the request remains backward compatible.
3. **Given** an import or AI-assisted intake supplies a Standard Location, **When** the coin is confirmed, **Then** it can save without a slot.
4. **Given** any manual, import, AI, or API path requests a Coin Tray assignment, **When** no valid available slot is supplied, **Then** the assignment is rejected or held for explicit user correction and is never silently placed.
5. **Given** a coin assigned to a tray is duplicated, **When** the duplicate is created, **Then** the duplicate has neither a storage location nor a storage slot.

### Edge Cases

- Rows and columns each accept inclusive values from 1 through 20; their product must not exceed 400.
- A 1-by-1 tray supports exactly one coin and still displays coordinate (1,1).
- Whitespace-only names are invalid; names are trimmed before owner-scoped uniqueness is evaluated.
- Name uniqueness applies across Standard Locations and Coin Trays for the same owner using case-insensitive comparison; different owners may use the same name.
- A slot at the first or last boundary coordinate maps correctly and cannot be confused with a zero-based value.
- A request containing a slot for a Standard Location is rejected rather than silently retaining or accepting the slot.
- A request selecting a tray with a slot outside that tray's persisted dimensions is rejected.
- A tray occupancy read that races with a later assignment may become stale; final save validation remains authoritative and reports a conflict.
- A coin image may be absent or unavailable; the tray still identifies the occupied well without exposing another owner's media or breaking the layout.
- Trays with 400 wells remain navigable and understandable on narrow screens without responsive reordering.
- A partial migration failure must roll back or stop safely; it must not leave existing locations or assignments in a mixed, ambiguous state.

## Requirements *(mandatory)*

### Functional Requirements

#### Storage Model and Migration

- **FR-001**: The system MUST support two owner-defined storage location types: **Standard Location** and **Coin Tray**.
- **FR-002**: A Standard Location MUST have a user-provided name and MUST NOT have rows, columns, or coin slot values.
- **FR-003**: A Coin Tray MUST have a user-provided name and immutable-once-occupied row and column dimensions.
- **FR-004**: Storage location names MUST be non-empty after trimming and unique per owner across both location types using case-insensitive comparison.
- **FR-005**: Tray rows and columns MUST each be integers from 1 through 20 inclusive, and their product MUST NOT exceed 400.
- **FR-006**: Each coin MAY have one storage location and, only when that location is a Coin Tray, MUST have one one-based storage slot.
- **FR-007**: A tray slot MUST represent one exact row and column in row-major order: `slot = ((row - 1) × columns) + column`; displayed coordinates MUST be one-based.
- **FR-008**: No more than one coin MAY occupy a given non-empty tray location and slot combination, including during concurrent writes.
- **FR-009**: Existing storage locations MUST be migrated to Standard Locations without changing their names, ownership, identifiers, or coin associations; migrated coins MUST have no storage slot.
- **FR-010**: Migration MUST be atomic or safely resumable and MUST surface failure rather than discarding, duplicating, or ambiguously classifying existing storage data.

#### Settings and Lifecycle

- **FR-011**: Storage Settings MUST let the owner choose Standard Location or Coin Tray when creating a location.
- **FR-012**: Tray creation MUST require name, rows, and columns; Standard Location creation MUST require only name.
- **FR-013**: Settings MUST distinguish location types and show each tray's dimensions, occupied-slot count, and total capacity.
- **FR-014**: Owners MUST be able to rename an occupied tray without changing its dimensions or assignments.
- **FR-015**: In v1, owners MUST be able to change a tray's dimensions when it is empty and the new dimensions are valid; dimension changes MUST be rejected while the tray is occupied.
- **FR-016**: Location deletion MUST be rejected while any coin references it; the response MUST explain how many assignments prevent deletion or otherwise clearly identify the reference constraint.
- **FR-017**: All create, read, update, delete, occupancy, and aggregate tray operations MUST be authenticated and scoped to the current owner.

#### Coin Assignment and Sibling Paths

- **FR-018**: Coin add and edit forms MUST show a slot picker only after a Coin Tray is selected.
- **FR-019**: The slot picker MUST display coordinates within the tray's persisted dimensions, distinguish available and occupied wells, disable wells occupied by other coins, and allow the edited coin's current well.
- **FR-020**: Saving a coin to a Coin Tray MUST require an in-range, currently available slot; saving a coin to a Standard Location MUST require an empty slot value.
- **FR-021**: Clearing or changing a coin's storage location MUST clear its previous slot in the same successful operation.
- **FR-022**: Moving a coin between tray slots or locations MUST validate and apply release and claim as one operation so a failed claim leaves the prior assignment intact.
- **FR-023**: Bulk assignment MUST exclude or disable Coin Trays and MUST explain that each coin requires explicit slot placement.
- **FR-024**: Duplicating a coin MUST clear both its tray location and slot so duplication cannot violate exclusivity or imply a second physical coin occupies the same well.
- **FR-025**: Manual add/edit, imports, AI-assisted intake, and all other coin create/update paths MUST use the same location-type, ownership, boundary, and occupancy rules.
- **FR-026**: Import and AI-assisted workflows MUST NOT infer an arbitrary tray slot; a requested tray assignment without an explicit valid slot MUST remain unassigned or require user correction before persistence.

#### Physical Storage Trays View

- **FR-027**: Collection navigation MUST include **Storage Trays**, opening a separate authenticated owner-scoped read-only route distinct from Museum Tray.
- **FR-028**: The Storage Trays route MUST display every Coin Tray configured by the current owner, including empty trays.
- **FR-029**: Each tray MUST render its exact persisted row and column count, every coordinate, every empty well, and each coin at its persisted position.
- **FR-030**: Storage Trays MUST NOT use Museum Tray's responsive placement, pagination, drawer, sorting, filtering, or packing algorithm; viewport adaptation MUST preserve the tray as one fixed geometric unit without changing coordinate positions.
- **FR-031**: Museum Tray behavior, routes, responsive semantics, collection membership, pagination, and preferences MUST remain unchanged.
- **FR-032**: Storage Trays MUST reuse the shared lower-level felt, well, coin-image, sizing, and interaction primitives and the existing visual CSS used by Museum Tray where applicable; it MUST NOT duplicate the felt/well visual stylesheet. If current primitives cannot accept fixed coordinates independently, the shared primitive boundary MAY be minimally extracted without changing Museum Tray behavior.
- **FR-033**: The view MUST use the existing private-media handling for authenticated coin images and MUST show a safe occupied-well fallback when an image is missing.
- **FR-034**: The route MUST provide explicit empty, loading, partial-content, and recoverable error states. Failure to load one coin image MUST NOT remove its occupied position or fail the entire tray.
- **FR-035**: The view MUST be usable in supported desktop and mobile/PWA contexts without page-breaking overflow; any scaling or scrolling MUST preserve readable coordinates and reachable content.
- **FR-036**: Actionable occupied wells MUST support pointer, touch, and keyboard operation with visible focus, meaningful accessible names including coin and coordinate, and a minimum 44-by-44 CSS-pixel interaction target where activation is offered.
- **FR-037**: Empty wells MUST expose their coordinate to assistive technology without pretending to be actionable; visual meaning MUST not rely on color alone.
- **FR-038**: The view MUST respect reduced-motion preferences and existing application theme/design tokens.

#### API and Contract Requirements

- **FR-039**: Public application contracts MUST expose storage location type and nullable dimensions, and coin contracts MUST expose a nullable one-based storage slot.
- **FR-040**: The authenticated storage-location create/update contract MUST accept:
  - Standard Location: `name`, `type=standard`, with `rows` and `columns` absent or null.
  - Coin Tray: `name`, `type=tray`, with required integer `rows` and `columns`.
- **FR-041**: Authenticated coin create/update contracts MUST accept a nullable storage location identifier and nullable storage slot; the service boundary MUST validate that the location belongs to the owner and that location type, slot presence, slot range, and occupancy agree.
- **FR-042**: The system MUST provide an authenticated owner-scoped aggregate read contract that returns all trays with identifier, name, dimensions, occupancy, and positioned minimal coin render data, including coin identifier, display name, nullable diameter, slot, and minimal private image reference.
- **FR-043**: The system MUST provide an authenticated owner-scoped occupancy contract, or equivalent data within an existing location response, sufficient for add/edit forms to list available slots and identify the current coin's permitted slot without returning unrelated coin details.
- **FR-044**: Contract responses MUST distinguish validation failures from concurrent or reference conflicts: malformed/type/range combinations return HTTP 400 validation errors; occupied-slot races, occupied-tray dimension changes, and referenced-location deletion return HTTP 409 conflicts; inaccessible or cross-owner resources return HTTP 404 without revealing whether they exist.
- **FR-045**: All new or modified public contracts MUST be documented in Swagger/OpenAPI, including authentication, request/response schemas, nullability, one-based slot semantics, dimension bounds, owner scoping, and validation/conflict responses.
- **FR-046**: Legacy requests that omit the new fields MUST continue to represent Standard Locations and unslotted Standard assignments; legacy response consumers MUST receive additive fields without a breaking reinterpretation of existing fields.

### API Contract Summary

The following resource-level contracts are normative; exact URL naming may follow existing repository conventions:

| Operation | Required behavior | Principal responses |
|-----------|-------------------|---------------------|
| List/manage storage locations | Return only visible owner-appropriate locations; create/update type-specific fields; include tray occupancy | Success, validation error, not found, conflict |
| Read tray occupancy | Return dimensions and occupied slot numbers, optionally recognizing the edited coin's current slot | Success, not found |
| Read all physical trays | Return all owner trays and minimal positioned coin/image render data; include empty trays | Success |
| Create/update coin | Validate owner, type/slot consistency, range, and exclusivity at final write | Success, validation error, not found, conflict |
| Delete location | Reject any referenced location | Success, not found, conflict |

Conflict responses MUST be stable enough for the client to distinguish at least `slot_occupied`, `tray_occupied`, and `location_referenced` and show an appropriate recovery message. Validation responses MUST identify the invalid field or invalid field combination without exposing internal error details.

### Key Entities

- **Storage Location**: An owner-scoped named place with a type of Standard or Coin Tray. A tray additionally has fixed rows and columns; a Standard Location has neither.
- **Coin**: An existing owner-scoped collection item with an optional Storage Location and an optional one-based Storage Slot. The slot is required only for tray assignments.
- **Tray Position**: A derived coordinate within one tray, identified by one-based row, column, and row-major slot number. It is empty or occupied by exactly one coin.
- **Tray Aggregate**: Read-only presentation data containing tray identity, exact dimensions, capacity/occupancy, and minimal coin render data keyed by slot.

### Ownership and Integrity Rules

- An owner can see or mutate only their own storage locations, trays, assignments, occupancy, and private coin media.
- A coin cannot reference another owner's location, even if an identifier or name is guessed.
- Slot exclusivity is enforced by an authoritative persistence constraint for non-null tray slots and by service validation that produces a readable conflict.
- The recommended persistence shape is additive: Storage Location gains type and nullable rows/columns; Coin gains nullable one-based storage slot; uniqueness applies to non-null `(storage location, storage slot)` pairs.
- Validation and ownership checks belong at the shared service boundary so HTTP handlers, import, AI-assisted, duplication, and other sibling workflows cannot diverge.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of pre-feature storage locations and their coin associations remain present after migration, with every migrated location classified as Standard and every migrated coin unslotted.
- **SC-002**: In assignment tests, 100% of attempts to save a tray assignment without an in-range slot are rejected, and 100% of attempts to assign two coins to one slot result in exactly one occupant.
- **SC-003**: Under simultaneous claims for the same slot, exactly one request succeeds and every losing request receives a recoverable conflict without corrupting either coin's prior assignment.
- **SC-004**: For trays from 1-by-1 through the 400-slot maximum, the Storage Trays view displays exactly `rows × columns` wells and every seeded coin appears at its persisted coordinate on desktop and mobile/PWA.
- **SC-005**: Across supported viewport sizes, 100% of tested tray coordinates retain the same row and column; no responsive adaptation reorders, repacks, omits, or paginates physical wells.
- **SC-006**: A collector can create a tray, assign a coin to a specific slot, and find that coin in the Storage Trays view in no more than three primary workflow steps after entering the relevant Settings or coin form.
- **SC-007**: At least 90% of representative usability-test participants can correctly identify an available slot, an occupied slot, and a coin's coordinate on their first attempt without assistance.
- **SC-008**: 100% of tested cross-owner tray, occupancy, assignment, and private-image requests return no other owner's storage or coin information.
- **SC-009**: Standard Location create/edit and all existing Museum Tray scenarios continue to pass without changed user-visible semantics.
- **SC-010**: Manual, import, AI-assisted, duplication, and bulk workflows each satisfy the same tray-placement rules in contract tests, with no path able to create an unslotted or duplicate tray assignment.
- **SC-011**: All new controls and actionable wells are keyboard operable, have visible focus and meaningful accessible names, and all non-actionable empty wells expose coordinates without false controls.
- **SC-012**: Empty, loading, image-failure, aggregate-error, validation-error, and conflict states each present a clear next action in 100% of defined acceptance tests.
- **SC-013**: The published API description contains all new fields, bounds, nullability rules, authentication requirements, response shapes, and validation/conflict cases with no undocumented public contract.

## Assumptions

- The existing authentication, collection ownership, coin detail navigation, Settings surface, and private-image mechanisms remain the authority for access and navigation behavior.
- Storage Trays is read-only for placement changes; collectors edit placement through coin add/edit workflows.
- Tray coordinates and slot numbers are one-based and use row-major order, starting at the upper-left and proceeding left-to-right, then top-to-bottom.
- Empty, valid trays may be resized in v1; resizing is blocked as soon as any slot is occupied.
- Storage names share one owner-scoped, case-insensitive namespace across Standard Locations and Coin Trays to avoid ambiguous selectors.
- The aggregate tray response contains only the coin fields needed to render and navigate; full financial, provenance, AI-analysis, and other private coin details are outside this contract.
- On narrow devices, preserving physical geometry takes precedence over fitting every well simultaneously without scrolling.

## Out of Scope

- Responsive repacking, automatic placement, sorting, filtering, or pagination of physical tray wells.
- Drag-and-drop placement or rearrangement within the Storage Trays view.
- Changing dimensions of an occupied tray, including automatic remapping of occupied slots.
- Multi-coin bulk placement into trays.
- Sharing physical tray layouts with followers or unauthenticated users.
- Changes to Museum Tray semantics, layout, routes, preferences, or collection membership.
- Multiple coins per well, irregular tray shapes, blocked wells, custom well sizes, or coordinates beyond a rectangular row-by-column grid.
- Automatic physical labels, printing, barcode/QR workflows, or inventory scanning.
