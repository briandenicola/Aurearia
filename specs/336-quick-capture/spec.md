# Feature Specification: Quick Capture

**Feature Branch**: `336-quick-capture`  
**Created**: 2026-06-29  
**Status**: Draft  
**Input**: Backlog card `specs/_backlog/F017-quick-capture.md`: "Add quick capture for new coins"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Save a minimal coin intake draft quickly (Priority: P1)

As a collector handling a newly acquired coin on a phone, I want to open Quick Capture and save a minimal, incomplete draft with photos and purchase context so I do not lose details before I have time for full cataloging.

**Why this priority**: Fast draft creation is the core workflow accelerator and delivers value before promotion, enrichment, or desktop follow-up exists.

**Independent Test**: Open Quick Capture from a narrow mobile/PWA viewport, attach obverse and reverse photos, enter the minimum capture fields, save, and verify the item is stored as an incomplete draft owned by the current user.

**Acceptance Scenarios**:

1. **Given** an authenticated collector is using the mobile/PWA navigation, **When** they choose Quick Capture, **Then** they see a compact capture flow optimized for photo-first intake and sparse entry.
2. **Given** the collector provides an obverse photo, an optional reverse photo, or enough identifying text, **When** they save the capture, **Then** the system creates a resumable draft marked incomplete rather than a normal collection coin.
3. **Given** the collector chooses the merged Find Coin / Quick Add AI path, **When** they capture or upload at least one coin/slab image, **Then** the system runs a quick minimum-detail analysis and offers to save the result as a Quick Capture draft.
4. **Given** an NGC certification number is visible in a captured/uploaded image, **When** quick AI analysis completes, **Then** the NGC coin/certification number is captured as structured draft data and shown for review.
5. **Given** the collector enters title, date range or era, acquisition source, price, and notes, **When** they save the capture, **Then** those values are preserved on the draft for later review.
6. **Given** the collector saves a draft, **When** they return to the main collection, **Then** the draft does not increase the normal collection count.

---

### User Story 2 - Resume and finish captured drafts (Priority: P1)

As a collector, I want a Quick Capture drafts view where I can find unfinished captures, continue editing them, and decide when each draft is ready to become a normal coin record.

**Why this priority**: Quick capture is only useful if incomplete work is recoverable and clearly separated from completed collection records.

**Independent Test**: Create multiple drafts, leave the flow, reopen the drafts view, edit a draft, and verify updates persist without affecting normal coins.

**Acceptance Scenarios**:

1. **Given** the collector has saved quick capture drafts, **When** they open the Quick Capture drafts view, **Then** they see their own drafts with incomplete status, preview photo when available, working title, last-updated date, and enough context to choose the right draft.
2. **Given** the collector opens an existing draft, **When** they edit photos, title, date range or era, source, price, or notes and save, **Then** the draft updates in place and remains incomplete.
3. **Given** a draft lacks key cataloging details, **When** it is shown in the drafts view, **Then** the UI clearly indicates it is not yet part of the main collection.

---

### User Story 3 - Promote a draft to a normal coin record (Priority: P1)

As a collector, I want to promote a completed quick capture draft into a normal coin record so it joins my collection only after I intentionally confirm it.

**Why this priority**: Promotion is the boundary between temporary intake and the authoritative collection, preserving count accuracy and existing coin workflows.

**Independent Test**: Complete a draft, promote it, and verify one normal coin is created with captured data and images while the draft lifecycle prevents duplicate promotion.

**Acceptance Scenarios**:

1. **Given** a draft has the minimum information required for normal coin creation, **When** the collector selects Promote, **Then** the system creates a normal coin record owned by that collector using the captured fields and attached images.
2. **Given** promotion succeeds, **When** the collector views the main collection, **Then** the promoted coin appears in normal collection views and the collection count increases by one.
3. **Given** promotion succeeds, **When** the collector views Quick Capture drafts, **Then** the original draft is no longer shown as an active incomplete draft and is marked promoted or linked to the created coin.
4. **Given** the collector attempts to promote the same draft again, **When** the request is processed, **Then** no duplicate coin is created and the collector receives a clear message.

---

### User Story 4 - Preserve existing collection workflows (Priority: P2)

As a collector who already uses full coin entry, wishlist, sold flags, and image management, I want Quick Capture to add a faster intake path without changing existing collection behavior.

**Why this priority**: Quick Capture touches shared workflow surfaces; regressions to existing collection counts, flags, images, and edit flows would undermine trust.

**Independent Test**: Exercise existing add/edit, image upload, collection count, wishlist, and sold workflows before and after creating/promoting drafts; verify their contracts remain unchanged except for the intentional promoted coin.

**Acceptance Scenarios**:

1. **Given** existing normal coins, wishlist coins, and sold coins, **When** a quick capture draft is created or edited, **Then** normal collection counts and existing flags remain unchanged.
2. **Given** a promoted quick capture coin, **When** the collector opens the normal edit flow, **Then** the promoted coin can be edited using the same workflow as any other coin.
3. **Given** existing image upload behavior for coins, **When** Quick Capture accepts photos, **Then** supported file types, validation feedback, ownership, and displayed image behavior are consistent with existing coin image handling.

### Edge Cases

- Camera capture is unavailable, permission is denied, or the browser does not support direct camera input; file upload remains available.
- Network connectivity is lost before save; unsaved local input is not silently represented as persisted and the collector receives a clear retry path.
- A required photo or field is missing at draft save time; the system allows a partial draft when at least a working title or note exists, and clearly marks missing information.
- A required normal-coin field is missing at promotion time; promotion is blocked with field-specific guidance while the draft remains editable.
- Uploaded files have unsupported extensions, invalid image signatures, excessive size, or failed processing; the draft is not saved with invalid images and the collector receives safe validation feedback.
- Two save or promote actions are submitted quickly because of double-tap or retry; drafts and promoted coins are not duplicated.
- A collector attempts to view, edit, delete, or promote another user's draft; access is denied without leaking draft details.
- A draft references an image that was removed or failed to load; the draft remains accessible and highlights the missing image.
- The draft list is empty; the view explains how to start a new Quick Capture instead of showing a broken state.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide an authenticated Quick Capture entry point from mobile/PWA navigation and an accessible desktop route or navigation path.
- **FR-002**: System MUST allow authenticated users to create separate quick capture draft/intake records before any normal coin record is created.
- **FR-003**: System MUST support camera capture and file upload for an obverse photo, an optional reverse photo, and optional detail/slab photos using the same supported image constraints as normal coin image handling.
- **FR-004**: System MUST allow users to capture and persist a working title, date range or era, acquisition source, price, and freeform notes on a draft.
- **FR-005**: System MUST allow saving a partial draft when the collector provides at least enough information to identify it later, such as a working title, note, or image.
- **FR-006**: System MUST mark quick capture drafts as incomplete until they are intentionally promoted or otherwise closed.
- **FR-007**: System MUST provide a Quick Capture drafts view that lists only the authenticated user's active drafts with status, preview, key identifying text, and last-updated information.
- **FR-008**: System MUST allow the authenticated owner to resume and update an active draft without creating or modifying a normal coin record.
- **FR-009**: System MUST provide an explicit promote action that creates a normal Coin record from a draft only after user confirmation.
- **FR-010**: System MUST require the normal coin minimum field rules before promotion, and MUST keep the draft editable when promotion validation fails.
- **FR-011**: System MUST transfer draft photos and captured fields into the promoted Coin record without requiring the collector to re-enter successfully saved draft data.
- **FR-012**: System MUST make successful promotion idempotent from the user's perspective so repeated promote attempts do not create duplicate Coin records.
- **FR-013**: System MUST exclude incomplete quick capture drafts from main collection counts, normal collection views, wishlist totals, sold totals, and collection health calculations until promotion.
- **FR-014**: System MUST include promoted quick capture coins in normal collection counts and workflows exactly once after successful promotion.
- **FR-015**: System MUST preserve existing full add/edit coin workflows, wishlist and sold flags, image handling, and collection count behavior except for the intentional addition of a promoted coin.
- **FR-016**: System MUST enforce authenticated user ownership for draft create, read, update, delete/close, and promote operations.
- **FR-017**: System MUST provide clear user-facing validation and error messages for missing promotion fields, invalid images, unavailable camera, failed save, failed promote, and unauthorized access.
- **FR-018**: System MUST merge Find Coin and Quick Capture by allowing captured/uploaded images to run through a quick AI analysis that extracts only minimum draft details and saves the reviewed result as a Quick Capture draft rather than creating a normal Coin automatically.
- **FR-019**: System MUST provide a way to discard or close an unwanted draft without creating a normal Coin record.
- **FR-020**: System MUST record enough lifecycle information to distinguish active, promoted, and discarded drafts and to link a promoted draft to its created Coin.
- **FR-021**: System MUST capture NGC coin/certification number, NGC lookup URL, grade, and visible label text as structured draft data when present in Find Coin / Quick Add AI analysis.

### Constitution-Aligned Constraints

- **CON-001**: Quick Capture persistence and promotion MUST follow the repository's layered architecture: Handler → Service → Repository → Database, with multi-step promotion treated as a transactional workflow.
- **CON-002**: Quick Capture MUST respect service boundaries; AI analysis MUST be proxied through the Go API to the Python agent, and the Python agent MUST remain stateless with no direct database access.
- **CON-003**: New or changed user-facing contracts MUST remain explicit and typed across API and frontend boundaries.
- **CON-004**: Draft and image handling MUST be authenticated, user-scoped, validated, and safe by default; internal errors and other users' draft details MUST NOT leak to clients.
- **CON-005**: The mobile/PWA experience MUST reuse existing design tokens, global styles, icon conventions, buttons, chips, and upload patterns rather than introducing a parallel visual system.
- **CON-006**: The change MUST stay simple, complete, and proportional: implement manual quick capture, drafts, resume, and promotion before optional enrichment or advanced cataloging shortcuts.
- **CON-007**: Planning and implementation MUST identify affected sibling workflow contracts for collection counts, image handling, wishlist/sold flags, edit flows, and AI intake surfaces and prove them with targeted tests where practical.

### Key Entities

- **QuickCaptureDraft**: A user-owned, incomplete intake record containing lifecycle status, working title, date range or era, acquisition source, price, notes, optional AI/NGC metadata, image references, timestamps, and optional link to a promoted Coin.
- **DraftImage**: An obverse, reverse, or supplemental image reference attached to a draft with validation status and display ordering.
- **Coin**: Existing normal collection record created only when a draft is promoted; after promotion it participates in standard collection counts, views, flags, images, and edit workflows.
- **DraftLifecycleEvent**: Status transition evidence for draft creation, update, promotion, and discard, sufficient to prevent duplicate promotion and support troubleshooting.
- **User**: Existing authenticated account that owns drafts and promoted coins; ownership controls every draft operation.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A collector can open Quick Capture on a 375px-wide mobile viewport, add at least one photo or identifying note, and save a draft in under 60 seconds during usability testing.
- **SC-002**: 100% of saved drafts are resumable from the Quick Capture drafts view by their owner and are not visible to other users.
- **SC-003**: Creating or updating drafts changes normal collection counts, wishlist totals, and sold totals by 0 in automated regression coverage.
- **SC-004**: Promoting one valid draft creates exactly one normal Coin record and increases the owner's normal collection count by exactly 1.
- **SC-005**: Repeated promotion attempts for the same draft create 0 additional Coin records after the first successful promotion.
- **SC-006**: 100% of invalid image uploads and missing promotion-required fields produce clear user-facing validation messages without exposing internal error details.
- **SC-007**: Existing normal coin add/edit, image display, wishlist flag, sold flag, and collection count regression tests continue to pass after Quick Capture is added.
- **SC-008**: At least 90% of first-time test users can correctly identify whether a captured item is an incomplete draft or a normal collection coin from the UI labels and placement.

## Assumptions

- Quick Capture uses separate draft/intake records first, then promotes to normal Coin records after explicit user confirmation.
- Find Coin / Quick Add AI analysis is intentionally shallow and fast: enough to seed a draft, not enough to replace full cataloging or promotion review.
- Drafts do not appear in the main collection count or normal collection views until promoted, but are resumable from a Quick Capture drafts view.
- Existing authentication and user ownership rules apply to all draft and promotion operations.
- Existing normal Coin minimum field rules remain authoritative for promotion readiness.
- Existing image upload validation and display behavior should be reused for draft photos where possible.
- Mobile/PWA capture is the primary UX target, while desktop support remains functional through normal responsive layouts.
- Documentation updates are expected during implementation if user-facing workflows or API contracts change.
