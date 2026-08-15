# Feature Specification: Optional Nomisma.org Authority Linking for Global Mint Locations

**Feature Branch**: `343-nomisma-mint-authority-linking`
**Created**: 2026-08-14
**Status**: Implemented
**Input**: User description: "Optional Nomisma.org Authority Linking for Global Mint Locations — admin-managed linking of global MintLocation records to Nomisma.org authority concepts, with visible CC BY 4.0 attribution, live on-demand lookup, and no bulk ingestion."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Admin links a global mint to its Nomisma concept (Priority: P1)

An admin opens a global (admin-curated) `MintLocation` and wants to attach the
authoritative Nomisma.org identifier for that mint, so the mint record carries
a durable, citable authority reference instead of just a name and coordinates.

**Why this priority**: This is the entire feature — without it there is
nothing to display or attribute. It must work end-to-end before anything else
matters.

**Independent Test**: As an admin, open a global mint location's management
view, trigger a Nomisma search/match for that mint, select a candidate
concept, confirm the match, and verify the mint location now persists the
Nomisma concept URI and shows attribution text linking to that concept.

**Acceptance Scenarios**:

1. **Given** an admin viewing a global mint location with no Nomisma link,
   **When** they search Nomisma for a matching mint concept, **Then** the
   system shows candidate concepts (name, place, identifier) for the admin to
   review.
2. **Given** search candidates are shown, **When** the admin selects one and
   confirms it, **Then** the mint location persists the chosen concept's
   durable URI and the confirmation is recorded as an explicit admin action
   (not inferred automatically).
3. **Given** a mint location with a confirmed Nomisma link, **When** any user
   views that mint (e.g., mint detail, mint map popup), **Then** they see a
   visible attribution line reading "Source: Nomisma.org · CC BY 4.0" that
   links to the specific linked concept and to the CC BY 4.0 license.

---

### User Story 2 - Admin changes or removes an existing Nomisma link (Priority: P2)

An admin realizes a previously confirmed match was wrong, or Nomisma has
since split/merged the concept, and needs to re-match or unlink it without
affecting the underlying `MintLocation` record (name, coordinates, aliases).

**Why this priority**: Authority data curation is an ongoing admin
responsibility; the feature is not trustworthy if a bad match is permanent or
requires deleting/recreating the mint.

**Independent Test**: On a mint location with an existing confirmed Nomisma
link, run a new search, confirm a different candidate (or explicitly unlink),
and verify the mint location's name/coordinates/aliases/coin associations are
unchanged while only the Nomisma reference is updated or removed.

**Acceptance Scenarios**:

1. **Given** a mint location with a confirmed Nomisma link, **When** the admin
   runs a new search and confirms a different candidate, **Then** the old
   concept URI/provenance is replaced by the new one and the attribution
   updates to point at the new concept.
2. **Given** a mint location with a confirmed Nomisma link, **When** the admin
   explicitly unlinks it, **Then** the concept URI and provenance are cleared,
   the attribution line no longer appears, and the mint location otherwise
   behaves exactly as an unlinked mint.

---

### User Story 3 - Nomisma is unavailable, has no match, or match is ambiguous (Priority: P2)

An admin attempts to link a mint while Nomisma.org is unreachable, returns no
matching concept, or returns several plausible candidates with no clear best
match.

**Why this priority**: Ancient-mint attribution is inherently uncertain and
the reference is an unowned third-party service; the feature must degrade
predictably instead of guessing or blocking unrelated work.

**Independent Test**: Simulate (a) a Nomisma outage/timeout, (b) a search with
zero results, and (c) a search with multiple similarly-ranked candidates;
confirm in each case that no link is silently created, the admin sees a clear
status, and normal mint/coin CRUD continues to work without any dependency on
Nomisma succeeding.

**Acceptance Scenarios**:

1. **Given** Nomisma.org is unreachable or times out, **When** the admin
   triggers a search, **Then** the system shows a clear "lookup unavailable"
   state and the mint location remains fully usable (view/edit/assign to
   coins) with no link created.
2. **Given** a search returns zero candidates, **When** the admin views the
   results, **Then** the system shows "no match found" and leaves the mint
   location unlinked; the admin may retry with a different query or decline.
3. **Given** a search returns multiple candidates without one dominant match,
   **When** the admin views results, **Then** the system requires the admin to
   explicitly pick one (or decline) before any link is persisted — no
   candidate is auto-selected on the admin's behalf.
4. **Given** any of the above outcomes, **When** the admin closes the search
   without confirming, **Then** the mint location and all existing coin data
   remain completely unaffected.

---

### User Story 4 - Private user mints are never sent to Nomisma (Priority: P1)

A regular (non-admin) user has created a private mint location for their own
coins. The system must never expose, search, or link that private mint
against Nomisma.

**Why this priority**: This is a hard privacy boundary carried over from the
mint/location feature; violating it would leak user-specific collection data
to a third-party service and contradicts the admin-only, global-only scope of
this feature.

**Independent Test**: Confirm that no Nomisma search/link UI or API path is
reachable for a private (user-owned) mint location, whether attempted by the
owning user or an admin acting on that user's private mint.

**Acceptance Scenarios**:

1. **Given** a private (user-owned) mint location, **When** any user or admin
   views it, **Then** no Nomisma search/link/attribution controls are shown.
2. **Given** a private mint location, **When** a client attempts to invoke
   the Nomisma link workflow against it directly, **Then** the system rejects
   the request and no data about the private mint is transmitted to Nomisma.

---

### Edge Cases

- What happens if an admin tries to link a mint location that has no name
  variants Nomisma recognizes (e.g., a very obscure or purely local mint)? →
  Treated as "no match found" (User Story 3, Scenario 2); admin may leave it
  unlinked indefinitely with no negative effect on the mint's normal use.
- What happens if the Nomisma concept a mint was linked to is later
  redirected, deprecated, or removed on Nomisma's side? → Out of scope for
  MVP to auto-detect (no background sync); the stored attribution link may
  become stale until an admin manually re-checks it. This is documented as a
  known limitation, not silently hidden.
- What happens if two different global mint locations are both linked to the
  same Nomisma concept? → Allowed. Nomisma concepts represent a single
  historical mint; nothing prevents two `MintLocation` records (e.g., an
  older and a corrected entry) from referencing the same authority concept.
- What happens if the admin's search query is empty or whitespace-only? →
  Rejected client-side/server-side as invalid input before any Nomisma call is
  made.
- What happens to already-confirmed Nomisma links if this feature is later
  disabled or Nomisma access is revoked? → Existing stored concept URIs and
  attribution continue to display (they are static, previously confirmed
  data); only new searches/matches stop working.
- What happens if a coin's mint is unknown or unlinked to any `MintLocation`?
  → No Nomisma attribution is ever shown; this feature only ever applies to
  mint locations that are both global and explicitly linked.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST allow an admin to search for candidate Nomisma
  authority concepts for a given global `MintLocation`, using an on-demand
  live lookup (no pre-loaded or bulk-ingested Nomisma dataset).
- **FR-002**: System MUST require an explicit admin confirmation of a
  specific candidate before persisting any Nomisma link — matches are never
  auto-applied, regardless of confidence or ranking.
- **FR-003**: System MUST persist, for each confirmed link, the durable
  Nomisma concept URI plus enough provenance (at minimum: matched label and
  confirmation timestamp) to support display and future re-verification.
- **FR-004**: System MUST display a visible attribution line reading "Source:
  Nomisma.org · CC BY 4.0" wherever a linked mint location's authority
  reference is shown, linking to the specific confirmed concept and to the CC
  BY 4.0 license text.
- **FR-005**: System MUST allow an admin to replace an existing confirmed
  Nomisma link with a different concept, or remove it entirely, without
  altering the `MintLocation`'s name, coordinates, aliases, or coin
  associations.
- **FR-006**: System MUST restrict all Nomisma search/link/unlink capability
  to global (admin-curated, non-owned) mint locations; private (user-owned)
  mint locations MUST NOT be searchable, linkable, or otherwise transmitted to
  Nomisma by any user or admin action.
- **FR-007**: System MUST treat a Nomisma outage, timeout, or error response
  as a non-blocking, clearly surfaced "lookup unavailable" state that leaves
  the mint location unlinked and fully functional for all existing mint/coin
  workflows.
- **FR-008**: System MUST treat a zero-result search as "no match found" and
  leave the mint location unlinked without error.
- **FR-009**: System MUST present multiple ambiguous/low-confidence
  candidates for explicit admin choice rather than silently selecting or
  discarding any of them.
- **FR-010**: System MUST NOT perform background synchronization, full
  Nomisma dataset ingestion, SPARQL-endpoint browsing, or ingestion of
  Nomisma partner corpora (types, hoards, specimens); lookups are strictly
  on-demand and scoped to authority concept search/match.
- **FR-011**: System MUST bound and time-limit any caching of Nomisma lookup
  results so cached data cannot be served indefinitely if the upstream
  concept changes; cached results MUST NOT replace the durably persisted
  confirmed link.
- **FR-012**: System MUST leave existing `MintLocation` aliases, coordinates,
  Mint Map behavior, and legacy free-text mint reconciliation (per
  `338-mint-location-integration`) unchanged by this feature; Nomisma linking
  is strictly additive metadata.
- **FR-013**: System MUST NOT treat a Nomisma link as a substitute for or
  replacement of `MintLocation` records, Numista catalog data, coin type
  attribution, valuation data, structured catalog references (per
  `214-structured-numismatic-catalog-references`), or AI-attribution output;
  it MUST only ever be presented as supplementary authority metadata for
  global mints.
- **FR-014**: System MUST restrict initiating, confirming, replacing, or
  removing a Nomisma link to admin users; non-admin users MUST only ever see
  the resulting attribution, never the search/match controls.
- **FR-015**: System MUST continue to allow full mint/coin create, read,
  update, and delete operations regardless of whether Nomisma is reachable,
  configured, or linked for any given mint.

### Key Entities

- **MintLocation (existing entity, extended)**: Gains optional Nomisma
  authority fields, populated only for global (non-owned) records: a durable
  concept URI, the matched label/name captured at confirmation time, and a
  confirmation timestamp. Absent fields mean "not linked." Name, coordinates,
  aliases, and ownership are unchanged by this feature.
- **Nomisma Concept (external reference, not locally owned)**: A third-party
  authority record identified by a durable URI, representing a single
  historical mint. The system stores only a reference to it (URI + minimal
  display/provenance fields) — never a copy of Nomisma's broader dataset,
  partner corpora, or mixed-license type/hoard/specimen records.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: An admin can go from opening an unlinked global mint location to
  a confirmed Nomisma link, when a clear match exists, in a single guided
  workflow with no more than one search and one confirmation step.
- **SC-002**: 100% of mint locations displaying Nomisma attribution show the
  exact "Source: Nomisma.org · CC BY 4.0" text linked to the specific
  confirmed concept and license — never an unattributed or generic reference.
- **SC-003**: Zero private (user-owned) mint locations are ever searchable,
  linkable, or transmitted to Nomisma, verified across every code path that
  reaches the feature.
- **SC-004**: 100% of Nomisma outages, timeouts, zero-result searches, or
  ambiguous-match situations leave existing mint/coin data fully readable and
  editable, with no failed or partial mint/coin save attributable to Nomisma.
- **SC-005**: Zero Nomisma links are created without a recorded, explicit
  admin confirmation action — no match is ever silently auto-applied.

## Assumptions

- Only admin users manage Nomisma linking; regular users see attribution
  read-only if present, consistent with the existing admin-only ownership of
  global `MintLocation` records.
- Nomisma's reconciliation service is the intended on-demand lookup
  mechanism for planning and implementation (see Clarifications). Planning
  and implementation proceed on this assumption without a separate upfront
  API validation gate; ordinary reconciliation timeouts, errors, or no-match
  responses are handled as the normal non-blocking outcomes already described
  in User Story 3 and FR-007/FR-008, not as a reason to pause or rescope.
- "Bounded short-lived caching" means a cache with a short, fixed expiry used
  only to avoid redundant identical lookups in quick succession (mirroring
  the existing Numista lookup cache pattern) — it is not a substitute for the
  durably persisted, admin-confirmed link.
- Attribution display applies wherever a linked mint's name/details currently
  surface (e.g., mint detail views, Mint Map popups) without requiring new
  dedicated pages.
- This feature does not introduce any new user-facing (non-admin) settings or
  toggles; it is entirely an admin data-curation capability plus a read-only
  attribution display for everyone else.
- Nomisma partner-contributed corpora (coin types, hoards, specimens under
  mixed licenses) are never fetched, stored, or displayed by this feature —
  only Nomisma's own CC BY 4.0 authority concepts (e.g., mint identifiers).

## Clarifications

### Session 2026-08-14

- Q: Should the plan require a short API validation gate before scheduling UI
  and persistence work, pausing/rescoping if Nomisma's supported lookup
  method or usage terms cannot be confirmed? → A: No — plan implementation
  assuming reconciliation works. Nomisma's reconciliation service is the
  intended on-demand lookup mechanism for planning and implementation; normal
  timeout/error/no-match behavior remains non-blocking per User Story 3 and
  FR-007/FR-008, and no separate upfront validation gate is required.
