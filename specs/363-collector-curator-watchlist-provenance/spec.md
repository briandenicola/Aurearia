# Feature Specification: Collector Curator, Watchlist, and Provenance Workflows

**Feature Branch**: `beta` (existing working branch; no feature branch created)
**Created**: 2026-09-18
**Status**: Draft - Analyzer blockers resolved; required ADR and release gates pending
**Input**: Promote backlog card F015 as Feature 363 using the product-owner-set
collector-profile, watchlist-input, tiered-risk, and confirmed-wishlist
decisions dated 2026-09-18.

## Scope Authority

Feature 363 is a suite of bounded collector-facing workflows composed from
existing capabilities. It does not create a parallel agent platform:

- Feature 012 and the shipped collection tools remain the authority for
  owner-scoped collection reads, confirm-gated writes, and journal behavior.
- Features 337 and 353 remain the authorities for acquisition-candidate
  discovery and saved-wishlist URL availability/run history.
- Features 344, 351, and 352 remain the authorities for Deep Analysis evidence,
  confidence, contradictions, proposals, and confirm-gated application.
- Features 359 and 361 remain the authorities for Coin Copilot durability,
  specialists, source validation, bounded execution, replay, cancellation,
  privacy-safe logs, feature rollout, and fallback.
- Feature 362 is the dependency for evidence-rich attribution handoff and
  conversational use of persisted Deep Analysis attribution evidence.
- Existing portfolio review, collection gap analysis, wishlist availability,
  auction tracking, and market specialist workflows remain canonical. Feature
  363 composes them through existing application services and does not bypass
  or duplicate them.

The Go application remains the authority for authentication, owner scope,
collector-profile persistence, durable state, reads, writes, validation,
idempotency, and audit metadata. Python remains stateless and database-free and
may only reason over bounded data supplied through approved capabilities.

The suite is delivered as independently shippable slices in this order:

1. collector profile foundation;
2. curator read-only recommendations;
3. watchlist evaluation;
4. provenance and documentation risk;
5. confirmed Add to Wishlist from eligible Coin Copilot market results.

The profile foundation introduces no autonomous behavior. The first
collector-facing recommendation slice is read-only. Each later slice can be
disabled or rolled back without disabling earlier slices.

## Clarifications

### Session 2026-09-18

- Q: What happens when an owner edits a Wishlist Action during review? → A: `immutable_revision`
- Q: What does explicit Cancel do to a Wishlist Action stage? → A: `revoke_stage`
- Q: What may owners do with existing stages while Wishlist Action is disabled? → A: `read_revoke_no_confirm`

## Settled Product Decisions

1. **Preference storage — `collector_profile`**: budgets, favorite periods,
   disliked categories, preferred dealers, and collecting goals live in a
   dedicated owner-scoped collector profile, not admin application settings.
2. **Watchlist inputs — `wishlist_goals_results`**: initial evaluation inputs
   are existing wishlist coins, active manual collecting goals, and eligible
   Coin Copilot dealer/auction market results. Direct auction-URL and
   saved-search ingestion are deferred unless a result is already represented
   by one of those surfaces.
3. **Risk threshold — `tiered_review`**: all defensible provenance and
   documentation findings are surfaced by tier with evidence, confidence,
   “why this matters,” limitations, and explicit “needs review” language.
   Low-confidence findings are not hidden and are never presented as facts.
4. **Mutation boundary**: Coin Copilot may recommend and display or stage
   eligible result data, but it cannot approve a mutation or call an arbitrary
   write tool. Add to Wishlist begins only from an explicit UI control and
   requires a separate Go-owned review and confirmation before save.
5. **Product claim boundary**: provenance findings support collector review;
   they are not forensic authentication, fraud determinations, accusations, or
   guarantees of authenticity, attribution, availability, or value.
6. **Wishlist review edits — `immutable_revision`**: every permitted edit
   creates a new immutable stage revision linked to the stage root, prior
   revision, and unchanged source evidence. No revision or source evidence is
   updated in place.
7. **Wishlist cancel — `revoke_stage`**: explicit **Cancel** durably revokes
   the stage root in Go. Closing or dismissing the modal is abandonment only;
   it does not revoke the stage, which remains available until expiry.
8. **Wishlist disable — `read_revoke_no_confirm`**: when Wishlist Action is
   disabled, existing stages remain owner-readable and revocable and cleanup
   continues, but new staging, revision, and confirmation are blocked until
   re-enabled.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Configure a private collector profile (Priority: P1)

As a collector, I want one private place to record my acquisition budget,
favorite periods, disliked categories, preferred dealers, and collecting goals
so recommendations reflect my intent without becoming global admin settings.

**Why this priority**: Owner-controlled preferences are the foundation for
every personalized workflow and can ship independently without enabling any
agent mutation.

**Independent Test**: Create, read, update, clear, and restore one owner's
collector profile while another owner and an administrator are present; verify
safe defaults, validation, owner isolation, and no effect on application-wide
settings.

**Acceptance Scenarios**:

1. **Given** an owner with no stored collector profile, **When** they open
   **Settings > Collector Profile**, **Then** they receive safe empty defaults
   and may save a profile without changing admin settings.
2. **Given** valid budgets, preferences, and goals, **When** the owner saves,
   **Then** the values are returned by the owner-scoped profile contract and
   are available to subsequent recommendation runs.
3. **Given** invalid currency, negative or reversed budgets, oversized text,
   too many entries, duplicate normalized entries, or an unsupported goal
   state, **When** save is attempted, **Then** no partial update occurs and
   field-specific validation explains the correction.
4. **Given** a foreign profile or goal identifier, **When** another owner tries
   to read or change it, **Then** it is indistinguishable from an unknown
   identifier and no preference data is disclosed.
5. **Given** the feature is rolled back or disabled after profiles exist,
   **When** the owner uses the rest of Aurearia, **Then** existing data remains
   inert and recoverable and recommendations fall back to neutral defaults.

---

### User Story 2 - Receive read-only curator recommendations (Priority: P1)

As a collector, I want a curator view of collection strengths, themes, gaps,
and possible next acquisitions so I can make informed decisions without the
application changing anything.

**Why this priority**: This is the simplest useful agent slice and reuses
shipped collection, portfolio, and gap-analysis capabilities.

**Independent Test**: Run curator review for a controlled owner collection and
profile; verify recommendations are read-only, grounded in returned collection
facts and preferences, and contain evidence, confidence, rationale, and limits.

**Acceptance Scenarios**:

1. **Given** an owner with a collection and profile, **When** curator review
   runs, **Then** it identifies strengths, themes, gaps, and ranked acquisition
   ideas using only owner-scoped collection facts and active profile values.
2. **Given** a recommendation conflicts with the owner's budget, disliked
   categories, or stated goal, **When** results are ranked, **Then** the conflict
   is visible and the item is excluded or explicitly deprioritized rather than
   silently recommended.
3. **Given** sparse or contradictory collection data, **When** review
   completes, **Then** the response distinguishes observed facts from
   recommendations and states missing evidence and lower confidence.
4. **Given** the profile is empty, **When** review runs, **Then** neutral
   defaults are used, no preferences are invented, and the owner is invited to
   improve personalization.
5. **Given** the owner only views, reruns, cancels, replays, or dismisses a
   recommendation, **Then** no collection, wishlist, auction, profile, or
   settings data changes.

---

### User Story 3 - Evaluate watchlist candidates against goals (Priority: P2)

As a collector, I want existing wishlist items and eligible Coin Copilot market
results evaluated against my goals, budget, preferences, duplicates, and
collection coverage so I can focus on the best candidates.

**Why this priority**: Evaluation turns recommendations into practical review
while remaining read-only and reusing existing wishlist, availability, auction,
and Coin Copilot evidence.

**Independent Test**: Evaluate a mix of wishlist coins, active goals, and
eligible dealer/auction results containing duplicates, unavailable listings,
missing prices, and conflicting evidence; verify a bounded, explainable ranking
without creating or changing a wishlist item.

**Acceptance Scenarios**:

1. **Given** eligible wishlist items and market results, **When** evaluation
   runs, **Then** each candidate is assessed against active goals, budget,
   preferences, owned duplicates, related collection coverage, and known
   listing state.
2. **Given** the same listing is already on the owner's wishlist or appears
   more than once, **When** evaluated, **Then** the relationship is shown and
   it is not presented as an independent new opportunity.
3. **Given** price, currency, listing state, or source evidence is absent or
   stale, **When** the candidate is displayed, **Then** the missing or stale
   fact remains unknown and its effect on confidence and ranking is explicit.
4. **Given** a direct auction URL or saved search not represented by a
   wishlist item, goal, or eligible Coin Copilot result, **When** evaluation is
   requested, **Then** ingestion is declined as deferred and no new provider
   work is created.
5. **Given** an existing wishlist availability record or auction lot, **When**
   evaluation uses it, **Then** Feature 363 links to that authoritative state
   rather than replacing or silently rewriting it.

---

### User Story 4 - Review tiered provenance and documentation risk (Priority: P3)

As a collector, I want cautious, evidence-backed review flags for provenance
and documentation gaps so I know what to investigate without receiving an
unsupported accusation or authenticity claim.

**Why this priority**: Risk review is valuable but depends on trustworthy
evidence and must follow the simpler read-only recommendation workflows.

**Independent Test**: Review fixtures for missing provenance, unsupported or
conflicting claims, defensible duplicate-image evidence, broken source links,
documentation gaps, and low-confidence observations; verify every finding uses
safe tiered language and preserves all supporting and contrary evidence.

**Acceptance Scenarios**:

1. **Given** missing provenance fields, documentation, or source links, **When**
   review runs, **Then** each gap is identified as a documentation need with
   evidence, confidence, why it matters, limitations, and “needs review.”
2. **Given** a listing makes a suspicious or unverified claim, **When** the app
   has contrary, incomplete, or non-corroborating evidence, **Then** it reports
   the claim as unverified or conflicting and never labels a seller, coin, or
   listing fraudulent or inauthentic.
3. **Given** the app has defensible existing evidence that two records reuse
   the same image, **When** a duplicate-image finding is shown, **Then** the
   matching basis and limitations are cited; absent such evidence, no duplicate
   claim is made.
4. **Given** confidence is low, **When** a finding passes schema and evidence
   validation, **Then** it remains visible in the lowest review tier with
   explicit uncertainty rather than being hidden or stated as fact.
5. **Given** Feature 362 attribution evidence is unavailable, **When** risk
   review runs, **Then** baseline documentation and listing-evidence checks may
   complete, while attribution-dependent findings are marked unavailable and
   are not inferred from missing evidence.

---

### User Story 5 - Confirm Add to Wishlist from a market result (Priority: P4)

As a collector, I want an explicit Add to Wishlist control on eligible Coin
Copilot dealer and auction results so I can review mapped fields and confirm a
safe, traceable wishlist save.

**Why this priority**: This is the only mutation in the suite and follows the
read-only slices so the evidence, duplicate, and profile foundations can be
validated first.

**Independent Test**: Select an eligible dealer result and auction result,
review the staged mapping, edit permitted values, confirm once, and replay the
same submission; verify one owner-scoped wishlist item, preserved evidence and
audit metadata, and no collection-only or unrelated manual-field changes.

**Acceptance Scenarios**:

1. **Given** an eligible source-backed Coin Copilot dealer or auction result,
   **When** the owner selects **Add to Wishlist**, **Then** a review screen shows
   the source result, mapped wishlist fields, omitted fields, evidence,
   confidence, listing state, price/currency, and any duplicate warning; no
   wishlist item exists yet.
2. **Given** a valid staged review, **When** the owner explicitly confirms,
   **Then** Go creates one owner-scoped wishlist item through the canonical
   write path and records source and audit metadata.
3. **Given** the owner explicitly selects **Cancel**, **When** Go accepts the
   request, **Then** the stage root is durably revoked and can never be revised
   or confirmed; closing or dismissing the modal merely abandons the view and
   leaves the stage confirmable until its expiry.
4. **Given** the same source identity, result, or idempotency request was
   already confirmed, **When** it is submitted again, **Then** the existing
   outcome is returned or a duplicate warning blocks creation; no duplicate
   wishlist item is created.
5. **Given** the result is foreign, malformed, missing required evidence, has
   an unsafe source URL, or is no longer eligible, **When** Add to Wishlist is
   attempted, **Then** staging or confirmation fails closed without exposing
   foreign data or creating a partial item.

### Edge Cases

- Minimum and maximum budget are equal, use different currencies, are cleared,
  or conflict with a result whose currency cannot be compared.
- A profile update and recommendation run begin concurrently; the run must use
  one explicit profile snapshot rather than a mixture of versions.
- A collecting goal is deactivated or edited while watchlist evaluation is in
  progress.
- A result matches a private owned coin, a private wishlist item, or another
  owner's identifier.
- A wishlist item has no URL; a URL redirects; the source is broken; or
  availability history conflicts with the current result observation.
- Provider text or collector-entered goal text contains prompt injection,
  instructions to call tools, token-shaped text, or an unsafe URL.
- A recommendation or evidence payload exceeds tool, event, checkpoint, or UI
  limits.
- A low-confidence risk is the only available signal.
- Duplicate-image evidence is partial, compares transformed/cropped images, or
  cannot identify which record is original.
- A Coin Copilot result is truncated, replayed, expired, cancelled, lacks field
  provenance, or was produced before the currently displayed profile version.
- Cancellation races profile snapshot, recommendation completion, risk result,
  wishlist staging, or confirmation.
- Confirmation races another tab confirming the same source result.
- The feature flag, Coin Copilot, Deep Analysis, a provider, or the Python
  service becomes unavailable during a workflow.

## Requirements *(mandatory)*

### Functional Requirements

#### Collector profile foundation

- **FR-001**: The system MUST provide exactly one dedicated collector profile
  per owner. It MUST NOT store these values in global or admin application
  settings.
- **FR-002**: The owner-scoped profile read/update contract MUST include an
  optional minimum budget, optional maximum budget, three-letter currency,
  favorite periods, disliked categories, preferred dealers, active/inactive
  collecting goals, profile version, and update timestamp.
- **FR-003**: A missing profile MUST read as neutral defaults: no budget limit,
  the owner's normal display currency when available or USD otherwise, empty
  preference lists, and no active goals. Defaults MUST NOT imply a preference.
- **FR-004**: Budget amounts MUST be finite and non-negative, minimum MUST NOT
  exceed maximum, and each amount MUST be at most 100,000,000. Currency MUST be
  a supported three-letter code; unconvertible currencies MUST not be compared
  as if equivalent.
- **FR-005**: Favorite periods and disliked categories MUST each contain at
  most 20 unique normalized values of at most 100 characters. Preferred dealers
  MUST contain at most 50 unique normalized values of at most 200 characters.
- **FR-006**: A profile MUST contain at most 50 collecting goals. Each goal
  MUST have an owner-generated identity, title of 1–200 characters, optional
  description of at most 1,000 characters, priority, active/inactive state, and
  timestamps. Goal text is data, never instructions.
- **FR-007**: Profile and goal updates MUST be validated and applied atomically
  with optimistic version checking so stale updates cannot silently overwrite
  newer owner input.
- **FR-008**: The profile UI MUST live at **Settings > Collector Profile**, work
  on desktop and installed PWA layouts, explain how each preference affects
  recommendations, and support clearing values back to safe defaults.
- **FR-009**: The profile introduction MUST be additive. Existing owners require
  no destructive backfill; profiles are created on first save. Rollback MUST
  disable profile editing and dependent personalization while leaving stored
  profiles inert and recoverable. No hardcoded deployment address may appear in
  the contract, UI, or defaults.
- **FR-010**: Profile reads, writes, goals, versions, and snapshots MUST be
  scoped to the authenticated owner derived server-side. Admin role alone MUST
  NOT grant access to another owner's collector preferences.

#### Recommendations and watchlist evaluation

- **FR-011**: Curator recommendations MUST reuse the existing collection tools,
  portfolio review, and gap-analysis paths; watchlist evaluation MUST reuse
  existing wishlist availability, auction tracking, and Coin Copilot specialist
  results. Neither workflow may implement a competing data-access or provider
  path.
- **FR-012**: The curator slice MUST be read-only and independently releasable
  before every later slice. Viewing, generating, cancelling, replaying, or
  dismissing recommendations MUST NOT mutate collection, wishlist, auction,
  profile, or admin data.
- **FR-013**: Every recommendation and candidate evaluation MUST identify its
  type as a recommendation, cite the owner-scoped facts and source evidence it
  used, state confidence, explain “why this matters,” and list material
  limitations.
- **FR-014**: Curator output MUST cover collection strengths, themes, gaps, and
  ranked next-acquisition ideas and MUST show how the snapshotted collector
  profile affected inclusion, exclusion, or ranking.
- **FR-015**: Watchlist evaluation inputs MUST be limited to existing
  owner-scoped wishlist coins, active manual collecting goals, and eligible
  Coin Copilot dealer/auction results. Direct URL and saved-search ingestion is
  out of scope unless already represented by one of those inputs.
- **FR-016**: Watchlist evaluation MUST assess, where evidence exists, goal fit,
  budget fit, favorite/disliked preference fit, preferred dealer fit, owned and
  wishlist duplicates, collection coverage, source quality, listing state, and
  price/currency comparability.
- **FR-017**: Missing, stale, contradictory, or incomparable facts MUST remain
  explicit unknowns and lower confidence where material. The system MUST NOT
  invent preferences, prices, currency conversions, availability, ownership,
  dealers, or collection gaps.
- **FR-018**: Existing wishlist availability and auction records remain
  authoritative for their lifecycle. Feature 363 may cite and link them but
  MUST NOT replace their state machines, histories, schedules, or provider
  automation.

#### Tiered provenance and documentation review

- **FR-019**: Risk review MUST be limited to missing provenance,
  suspicious/unverified claims, duplicate images where defensible evidence
  already exists, broken source links, and documentation gaps.
- **FR-020**: Every finding MUST include a review tier, evidence/citations,
  confidence, “why this matters,” limitations, and the literal meaning “needs
  review.” Low-confidence validated findings MUST remain visible in the lowest
  tier.
- **FR-021**: Findings MUST distinguish observed facts, source claims,
  conflicts, and recommendations. Copy MUST use cautious terms such as
  “unverified,” “conflicting,” “not corroborated,” or “documentation gap” and
  MUST NOT allege fraud, accuse a person or dealer, or determine authenticity.
- **FR-022**: Duplicate-image findings MUST state the existing defensible match
  basis and limitations. This feature MUST NOT add bulk external scraping, a
  new image provider, or a new dependency merely to manufacture duplicate-image
  evidence.
- **FR-023**: Broken-link checks MUST use existing URL validation, provider
  allowlists, SSRF defenses, bounded fetch behavior, and availability
  capabilities. Unsafe or unapproved URLs MUST not be fetched.
- **FR-024**: Before Feature 362 is available, profile, curator, watchlist, and
  baseline documentation/listing-risk slices MAY ship using existing evidence.
  Any finding that depends on conversational access to persisted Deep Analysis
  attribution evidence, proposal confidence, provider contradictions, or
  attribution citations MUST remain gated and unavailable.
- **FR-025**: After Feature 362 is available and enabled, Feature 363 MAY consume
  its validated, persisted attribution projection only when ADR 0017 is
  accepted, Feature 362 prerequisite gates are complete, the advertised
  projection capability is persisted and enabled, and its schema version is
  supported. The consumer MUST reject unknown versions or capabilities rather
  than guess, downgrade, or reinterpret them. It MUST preserve the canonical
  projection's owner/run/job identity, complete-result digest, schema version,
  stable ordering and canonicalization rules, confidence, source conflicts,
  provider coverage, evidence URLs, omitted-entry disclosure, limitations, and
  read-only/review-gated boundaries. A supported older version MAY be read only
  through an explicit compatibility adapter that preserves every required
  field; otherwise attribution-rich findings fail closed while baseline review
  remains available.
- **FR-025a**: Feature 362 compatibility MUST use its published canonical JSON
  form: UTF-8, normalized scalar forms, lexicographically sorted object keys,
  Feature 362-defined stable array order, no unknown-field coercion, and
  SHA-256 over the complete canonical projection before bounded omission.
  Feature 363 MUST verify the advertised schema version and complete-result
  digest before use, retain required identity/evidence/conflict/coverage/
  confidence/limitation fields byte-for-byte in meaning, and expose truncation
  or omitted-entry metadata. Downgrade to a binary without a verified adapter
  disables attribution-rich risk and preserves persisted rows unchanged.

#### Confirmed Add to Wishlist

- **FR-026**: Eligible Coin Copilot dealer and auction result cards MUST expose
  an explicit **Add to Wishlist** control. Conversational text alone MUST NOT
  start staging or mutation.
- **FR-027**: Selecting the control MUST enter a Go-owned, owner-scoped review
  flow. Every stage and revision response MUST include the immutable stage-root
  and revision ids, revision number, status, version, expiry, fingerprint,
  original Coin Copilot run/execution/tool/result identity, Feature 361 schema
  version and capability, canonical source identity and normalized HTTPS URL,
  registered provider/host, canonical source digest, retained source-result
  digest, provenance and per-field evidence, confidence, observation time,
  closed listing/provider state values, paired price and ISO currency or
  neither, proposed mapping, omitted fields, validation issues, and duplicate
  warnings. Selection itself writes no wishlist item.
- **FR-027a**: `source_kind` is the closed enum `market_search` or
  `auction_search`; provider id MUST be selected from the versioned server
  registry loaded at startup, never accepted as an open client string; and
  `listing_state` is the closed enum `AVAILABLE`, `SOLD`, `ENDED`, `WITHDRAWN`,
  or `UNKNOWN`. Only registered HTTPS hosts with an exact or approved
  subdomain match may stage; redirects MUST be revalidated at every hop.
  Canonical source identity is the Feature 361 source kind, registered provider
  id, and normalized final HTTPS URL. Its digest and the source-result digest
  use lowercase SHA-256 over UTF-8 canonical JSON with sorted object keys,
  normalized scalar forms, schema-defined stable array order, and no
  credentials or fragments. A numeric price MUST have one supported ISO
  currency and a currency MUST have one numeric price; otherwise both are
  represented as unknown and no comparison or mapping occurs.
- **FR-028**: Saving MUST require a second explicit confirmation tied to the
  exact staged version. Coin Copilot and Python MUST NOT self-approve, confirm,
  directly write, receive a generic write credential, or call arbitrary write
  tools.
- **FR-029**: Only destination-valid wishlist fields MAY be proposed:
  source-backed title to name; supported numismatic identity fields to their
  corresponding wishlist fields; source URL to reference URL; observed listing
  price to the wishlist value estimate when supported; and listing state to the
  existing listing-status field. During review, the owner MAY edit only name,
  the supported destination numismatic identity fields, and the wishlist value
  estimate paired with its supported ISO currency. Source URL/identity,
  provider, listing state, provenance/evidence, observation time, and all
  original execution/result identities are immutable. Missing fields remain
  empty. Every accepted edit creates a new immutable revision with a new id,
  monotonically increasing revision number/version, canonical fingerprint,
  fresh idempotency binding, and a fresh 30-minute expiry; it links to the root,
  prior revision, and unchanged source evidence without mutating any earlier
  revision.
- **FR-030**: Currency, provider, canonical source identity, observation time,
  field provenance, evidence, and limitations MUST be preserved in the staged
  action and durable audit/provenance metadata even when the current wishlist
  coin schema has no direct field. They MUST NOT be coerced into unrelated coin
  fields.
- **FR-031**: The mapping MUST NOT populate or overwrite collection-only or
  manually maintained fields, including purchase date, actual purchase price,
  invoice/SKU, sold state, storage, private visibility, images, notes, or
  existing manually entered catalog/provenance data. Structured references may
  be added only through an already-authorized validated, additive,
  confirm-gated path.
- **FR-032**: The confirmation service MUST derive owner identity server-side,
  revalidate result eligibility and evidence, enforce the exact mapping
  allowlist, and perform wishlist creation plus audit linkage atomically through
  the canonical Go service/repository path. Python MUST never write.
- **FR-033**: Stage and confirmation requests MUST use idempotency keys and an
  immutable request fingerprint. Same-owner same-payload replay MUST return the
  original stage/revision or durable confirmation outcome; key reuse with a
  different canonical payload MUST return `409 idempotency_conflict`.
  Fingerprints MUST bind owner, operation, stage root/revision/version, original
  execution/result identity, source and source-result digests, exact mapping,
  expiry, and relevant feature/schema versions. Revision, revocation, and
  confirmation MUST revalidate owner binding, current active revision,
  fingerprint, expiry, source-result retention/digest, URL/provider
  registration, listing eligibility, duplicate state, and feature/capability
  state. Stale or tampered requests fail closed and never mutate evidence.
- **FR-034**: Duplicate prevention MUST consider the owner, eligible result
  identity, canonical source URL, normalized source URL, and existing
  source-result linkage. Concurrent or replayed submissions MUST create at most
  one wishlist item.
- **FR-035**: A confirmed save MUST record privacy-safe audit metadata sufficient
  to identify the owner, source as Coin Copilot market result, result/run and
  provider identity, evidence observation time, mapped fields, confirmation
  time, idempotency outcome, and created wishlist item. The new item's activity
  journal MUST record the confirmed assisted creation without raw prompts,
  credentials, or full provider payloads.
- **FR-035a**: Wishlist stage roots use the closed lifecycle `ACTIVE`,
  `REVOKED`, `EXPIRED`, or `CONFIRMED`; immutable revisions use `ACTIVE`,
  `SUPERSEDED`, `REVOKED`, `EXPIRED`, or `CONFIRMED`. Exactly one revision may
  be `ACTIVE` per active root. Creating a revision atomically supersedes the
  prior active revision. Explicit Cancel atomically changes the active root and
  its active revision to `REVOKED`; modal close/dismiss performs no durable
  transition. Expiry is determined from the Go database UTC clock. Revoked,
  expired, superseded, or confirmed revisions cannot be revised or confirmed.
  Revoked stages remain owner-readable only for the bounded audit period.
- **FR-035b**: The complete public mutation surface is:
  `PUT /api/collector-profile`, `POST /api/wishlist-actions/stages`,
  `POST /api/wishlist-actions/stages/{stageID}/revisions`,
  `POST /api/wishlist-actions/stages/{stageID}/revoke`, and
  `POST /api/wishlist-actions/stages/{stageID}/confirm`. The stage route returns
  `201` for created and `200` for an idempotently existing stage; revision
  returns `201` or `200`; revoke returns `200`; confirm returns `201` with
  `outcome=created` or `200` with `outcome=existing`. Validation is `400`,
  foreign/unknown identifiers are indistinguishable `404`, stale version,
  duplicate conflict, key mismatch, or concurrent transition is `409`, expired
  or revoked stage is `410`, oversized canonical input is `413`, and disabled,
  unavailable, unsupported schema/capability, or unretained source result is
  `503`. Closed error codes are: `400 invalid_request`,
  `400 invalid_mapping`, `400 invalid_source`; `404 not_found`;
  `409 stale_version`, `409 idempotency_conflict`, `409 duplicate_conflict`,
  `409 transition_conflict`; `410 stage_expired`, `410 stage_revoked`;
  `413 payload_too_large`; and `503 feature_disabled`,
  `503 dependency_unavailable`, `503 unsupported_contract`, or
  `503 source_not_retained`. Responses disclose no foreign state.
  Model/agent callback routes remain read-only and MUST NOT include any of
  these mutations or a generic write route.
- **FR-035c**: Durable stage-created, stage-existing, revision-created,
  revision-existing, revoked, confirmation-created, and confirmation-existing
  outcomes MUST be distinct from failures. A successful confirmation has one
  durable outcome row; an idempotent replay reads it. A failed attempt MUST NOT
  create a success/outcome row or wishlist journal entry. Go MUST persist
  privacy-safe stage-created, revision-created, stage-revoked,
  confirmation-created/existing, expiry, and rejected-transition audit events
  with owner, operation, ids, versions, digests, reason code, and timestamps,
  but no raw prompt, credential, provider payload, query, full URL, or private
  content. Server logs contain only privacy-safe ids/digests, status/reason,
  bounded counts/bytes, and duration; durable audit events, not logs, are the
  audit authority.

#### Safety, lifecycle, rollout, and tests

- **FR-036**: Every workflow MUST preserve owner scoping and private-coin
  visibility. A result may use an owner's private coin to advise that same
  owner, but no private fact, identifier, image, price, or inference may appear
  to another user, follower, public surface, or privacy-unsafe log.
- **FR-037**: Owner-entered and provider text MUST be treated as untrusted data.
  Prompt injection, tool instructions, hidden-prompt requests, and token-shaped
  content MUST not alter scope, tools, ranking rules, confirmation, or output
  safety.
- **FR-038**: New capability calls MUST inherit Coin Copilot's snapshotted
  iteration, tool-call, concurrency, duration, cancellation, credential,
  payload, checkpoint, replay, and retention bounds. Composition MUST NOT create
  a second unbounded budget.
- **FR-039**: Cancellation MUST be authoritative before each capability call,
  after awaited work, before staging, and before confirmation commit. Late
  recommendation or risk output MUST be discarded after cancellation wins; a
  confirmation transaction that already committed remains idempotently
  readable. For a staged Wishlist Action, only the explicit revoke route is a
  durable cancellation; closing/dismissing UI is not a cancellation signal.
  Revocation and confirmation serialize on the active stage version: revocation
  wins if committed first and confirmation returns `410 stage_revoked`; if the
  confirmation transaction commits first, revoke returns the existing
  confirmed outcome without changing it.
- **FR-040**: Each slice MUST have its own default-off feature control and safe
  fallback. The five backend settings are `CollectorProfileEnabled`,
  `CollectorCuratorEnabled`, `CollectorWatchlistEnabled`,
  `CollectorProvenanceRiskEnabled`, and `CollectorWishlistActionEnabled`; all
  default to `false`, are registered and validated at startup, are enforced by
  server-side preflight as well as hidden/disabled UI, and do not rely on a
  client flag. After disable, no new run is admitted for that slice. Read-only
  curator/watchlist/risk runs accepted before disable may finish and remain
  owner-readable/cancellable. Profile data remains stored and participates in
  account export/deletion, but profile UI and API mutation are blocked while
  its flag is off; owner-scoped reads may remain available for recovery.
  Wishlist Action follows `read_revoke_no_confirm`: existing stages remain
  owner-readable and revocable and cleanup continues, while new stage,
  revision, and confirmation requests return `503 feature_disabled` until
  re-enabled. Disabling or unavailability MUST preserve existing collection,
  wishlist, auction, Deep Analysis, and legacy chat workflows.
- **FR-041**: The feature MUST add no provider automation, external dependency,
  generic browsing, bulk scraping, direct browser-to-Python access, Python
  persistence, or hardcoded deployment network address.
- **FR-042**: Automated tests MUST cover wishlist-versus-collection field
  mapping, preservation of manual data, duplicate/idempotent submissions,
  foreign identifiers, malformed provider evidence, prompt injection,
  low-confidence risk language, private-coin leakage, cancellation/replay,
  feature-unavailable fallback, and audit/provenance metadata.
- **FR-043**: Regression tests MUST prove existing portfolio/gap analysis,
  wishlist availability, alert candidate conversion, auction tracking, Deep
  Analysis, Coin Copilot specialist results, and legacy fallback remain
  unchanged outside the explicitly composed flows.
- **FR-044**: Watchlist evaluation and Wishlist Action MUST accept only retained,
  owner-bound, validated Feature 361 `market_search` or `auction_search`
  results using specialist contract schema version `1`, with bounded
  checkpoint/result metadata and the completed T039–T053 safety baseline.
  Unknown schema versions, result kinds, capabilities, missing/truncated source
  payloads, invalid digests, or unretained results MUST fail closed. No
  unvalidated event text or reconstructed model output may substitute for the
  retained result.
- **FR-045**: Attribution-rich risk MUST require accepted ADR 0017, completed
  Feature 362 prerequisite gates, a supported projection schema version, the
  persisted projection capability, and all applicable Feature 362 and Feature
  363 flags enabled. Failure of any dependency makes only attribution-rich
  findings unavailable; it MUST NOT silently fall back to model memory or
  transient output.
- **FR-046**: Database constraints MUST enforce owner binding, not merely
  handlers: goal `(owner_id, profile_id)` references the matching profile;
  every stage revision/root/source binding shares one owner; each revision
  references its root and prior revision within that owner; confirmation
  outcome `(owner_id, stage_id)` references the matching stage and
  `(owner_id, coin_id)` references the created owner wishlist coin. Composite
  unique keys/foreign keys, checks, partial unique indexes, or an equivalently
  enforceable database mechanism MUST prevent cross-owner, multiple-active-
  revision, and multiple-outcome corruption. Migrations and tests MUST inject
  corrupt owner/id combinations and prove the database rejects them.
- **FR-047**: Retention uses safe defaults unless a higher legal or product
  authority sets a stricter policy: profiles and goals persist until owner
  clearing or account deletion; terminal recommendation, evaluation, risk, and
  profile-snapshot rows are hard deleted after 30 days; active unconfirmed
  stages expire after 30 minutes; stage roots/revisions and rejected-attempt
  audit events are hard deleted 30 days after expiry or revocation;
  failed-attempt operational audit events are hard deleted after 30 days; and
  confirmed stage snapshots and confirmation audit records are hard deleted
  after 90 days after their bounded fields are projected into the created
  coin's durable activity journal. Final wishlist-coin provenance and journal
  entries follow the canonical coin retention/deletion policy. A Go-owned UTC
  cleanup job runs at least hourly in batches of at most 500, retries failures
  with bounded exponential backoff capped at 24 hours, resumes safely after
  restart from durable state, is idempotent under overlap, and records
  privacy-safe counts/errors. Hard deletion leaves no owner-linked payload,
  audit tombstone, or recoverable soft-delete row; only aggregate
  non-identifying operational metrics may remain.
- **FR-048**: Production account export and deletion MUST include every new
  private profile, goal, snapshot, recommendation/evaluation/risk record,
  Wishlist Action root/revision/outcome, and durable audit record. Export MUST
  preserve user-understandable relationships, timestamps, statuses, and
  provenance without credentials or internal security tokens. Account deletion
  MUST remove or legally anonymize all owner-linked rows and audit records
  through the production path, including retained confirmed provenance as
  governed by the canonical coin policy; test-only cleanup is insufficient.
- **FR-049**: A Feature 363 ADR in `docs/adr/NNNN-*.md` is mandatory and MUST be
  accepted before any dependent schema or implementation is merged. It MUST
  decide profile ownership; all five slice flags; immutable Wishlist Action
  staging, revision, revocation, confirmation, and idempotency; database owner
  constraints; retention/export/deletion; mixed-version, partial-migration, and
  rollback behavior; and the invariant that models/Python never write.
- **FR-050**: Test-first evidence is a release and sequencing requirement.
  Architecture/route guards, five-flag admission and post-start behavior,
  privacy/owner isolation, migration and corruption injection, lifecycle and
  tamper cases, and concurrency/cancellation races MUST be written and shown
  failing for the intended missing behavior before constrained implementation,
  then pass afterward. This ordering is acceptance evidence, not merely a plan
  preference.
- **FR-051**: Upgrade, mixed-version, partial-migration, and rollback tests MUST
  cover old/new API binaries, workers, web clients, all five flag states,
  schema-before-code and code-before-schema deployment, interrupted and retried
  migrations/cleanup, Feature 361 schema `1`, unknown specialist/projection
  versions, and rollback to the documented compatibility guard release.
  Unsupported writers/readers MUST fail closed without deleting or rewriting
  rows. For every matrix cell, pre/post row counts, owner ids, stage/revision
  states, digests, source links, outcomes, coin links, journal entries, and
  audit content MUST be equal except for the single explicitly expected
  transition; no orphan, duplicate, cross-owner, lost, or partially populated
  row is permitted.
- **FR-052**: Merge and release require recorded evidence for `go build ./...`,
  `go vet ./...`, `go test ./...`, `go test -run TestArchitecture ./...`,
  targeted `go test -race` lifecycle/concurrency packages, clean web dependency
  installation, `vue-tsc --build`, web lint/unit/browser tests, `npm run build`,
  explicit `pip install -e ".[dev]"`, locked `uv sync` as an additional
  reproducibility check, Python syntax/type checks where configured,
  `ruff check app/ tests/`, `pytest tests/`, migration/rollback and contract
  suites, owner-isolation/security/tamper suites, CodeQL, container image build,
  Trivy with zero High/Critical findings, secret scanning, SBOM generation,
  threat-model review, and image provenance/scan evidence. The PR MUST include
  the constitution self-check, Principle IV self-check, affected-workflow
  regression evidence, and mirrored Definition of Done checklist. A local
  environment limitation is never a waiver; an equivalent hosted pass is
  required before merge or release.

### Key Entities

- **Collector Profile**: One private owner-scoped preference root containing
  budget, currency, favorite periods, disliked categories, preferred dealers,
  goals, version, and timestamps.
- **Collecting Goal**: Owner-authored active or inactive acquisition objective
  with bounded title, description, priority, and lifecycle timestamps.
- **Profile Snapshot**: Immutable bounded view of the profile version used by a
  recommendation or evaluation so results remain explainable after edits.
- **Collector Recommendation**: Read-only curator output linking observed
  collection facts and profile constraints to a recommendation, confidence,
  rationale, citations, and limitations.
- **Watchlist Evaluation**: Read-only assessment of a wishlist item or eligible
  Coin Copilot market result against goals, preferences, duplicates, collection
  coverage, listing state, and evidence quality.
- **Risk Finding**: Tiered review item for one allowed provenance or
  documentation concern, separating observations from claims and containing
  evidence, confidence, why it matters, limitations, and needs-review language.
- **Wishlist Action Stage**: Owner-scoped immutable review snapshot for one
  eligible market result. A root owns an append-only chain of immutable
  revisions containing the proposed field mapping, original source/result
  bindings and digests, evidence, duplicate status, fingerprint, version,
  expiry, and closed lifecycle state; it is not a wishlist item.
- **Wishlist Action Audit**: Durable privacy-safe record linking confirmed
  explicit intent, source result and evidence identity, mapping, idempotency
  outcome, and created wishlist item.

## Dependencies and Delivery Gates

| Slice | Can ship before Feature 362? | Required foundations |
|-------|-------------------------------|----------------------|
| Collector profile | Yes | Existing auth and owner-scoped persistence patterns |
| Curator read-only recommendations | Yes | Feature 012 collection tools; existing portfolio/gap logic; collector profile |
| Watchlist evaluation | Yes | Existing wishlist/availability/auction services; retained validated Feature 361 `market_search`/`auction_search` schema v1 results; bounded checkpoint/result metadata; completed T039–T053 baseline |
| Baseline provenance/documentation risk | Yes | Existing coin/listing data, URL safety, availability evidence, and any already-defensible image evidence |
| Attribution-evidence-rich risk | No | Accepted ADR 0017; completed Feature 362 prerequisite gates; supported persisted projection schema/capability; applicable flags |
| Confirmed Add to Wishlist | Yes, if Feature 361 gates are satisfied | Retained validated Feature 361 `market_search`/`auction_search` schema v1 result; bounded checkpoint/result metadata; completed T039–T053 baseline; canonical Go wishlist write and audit paths |

The Feature 363 ADR required by FR-049 is a blocking predecessor for dependent
schema and implementation. Dependency preflight MUST reject unknown or absent
versions/capabilities rather than optimistically admit a workflow.

## Non-Goals

- Autonomous buying, bidding, offer placement, or payment.
- Automatic, conversational, scheduled, or bulk wishlist writes.
- Forensic authentication, legal provenance certification, fraud detection, or
  accusations against collectors, dealers, auction houses, or listings.
- A marketplace, seller scoring, cross-user comparison, social recommendation,
  or exposing private collection preferences.
- Direct auction-URL ingestion, saved-search ingestion, new alert ingestion, or
  replacement of wishlist availability and auction tracking systems.
- Bulk external scraping, arbitrary browsing, new provider automation, or a new
  third-party dependency.
- Replacing collection tools, portfolio review, gap analysis, wishlist
  availability, auction specialists, Deep Analysis, Coin Copilot, or their
  existing durable state.
- Python-side database access, durable profile/memory storage, mutation, or
  approval authority.
- Plans, tasks, implementation, migrations, deployment, commits, or rollout
  execution in this specification stage.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In owner-isolation tests, 100% of profile, goal, recommendation,
  evaluation, risk, stage, confirmation, and audit operations disclose zero
  foreign or private data.
- **SC-002**: An automated desktop and installed-PWA browser workflow can open
  **Settings > Collector Profile**, set a budget, one value in each of the
  three preference lists, and one active goal, save, reload, edit, clear, and
  restore the profile with no more than one explicit save action per change;
  100% of valid fixtures round-trip exactly and 100% of invalid-boundary
  fixtures are rejected without partial updates.
- **SC-003**: In controlled curator fixtures, 100% of displayed recommendations
  identify supporting collection facts, profile effects, confidence, why the
  recommendation matters, and limitations, and create zero writes.
- **SC-004**: In watchlist fixtures, 100% of evaluated candidates identify their
  input surface and known duplicate, budget, goal, coverage, listing-state, and
  evidence-quality outcomes without inventing missing price, currency,
  availability, or preferences.
- **SC-005**: In risk-language fixtures, 100% of findings contain evidence,
  confidence, review tier, why it matters, limitations, and needs-review
  language; zero findings assert fraud, accusation, or forensic authenticity.
- **SC-006**: Low-confidence fixtures remain visible in the lowest review tier
  in 100% of tests and are labeled uncertain rather than factual.
- **SC-007**: Before Feature 362 is available, all four pre-362 slices in the
  delivery table remain usable while 100% of attribution-dependent findings
  report a clear gate instead of inferred evidence.
- **SC-008**: In Add to Wishlist acceptance tests, 100% of saves require the
  explicit control plus separate confirmation; cancelled or unconfirmed stages
  create zero wishlist items.
- **SC-009**: Mapping and preservation tests show that 100% of supported
  source-backed wishlist fields map correctly, 100% of provider/source/listing/
  price/currency/evidence metadata is preserved where supported or in linked
  audit metadata, and zero collection-only or manual fields are overwritten.
- **SC-010**: Duplicate, concurrent, and replay tests create at most one
  wishlist item for the same owner and source result; same-payload replays
  return one stable outcome and mismatched idempotency replays conflict.
- **SC-011**: Prompt-injection, malformed-evidence, unsafe-URL, foreign-ID,
  private-coin, cancellation, and feature-unavailable fixtures cause zero
  unauthorized tool calls, writes, data disclosures, accusations, or invented
  evidence.
- **SC-012**: Existing collection analysis, wishlist availability, auction
  tracking, Deep Analysis, Coin Copilot, and legacy fallback acceptance suites
  show no behavior change outside Feature 363 entry points.

## Assumptions

- The target user is an authenticated collection owner; invited friends and
  public visitors do not receive these workflows.
- “Eligible Coin Copilot market result” means a retained, owner-bound Feature
  361 dealer or auction evidence item that passes its typed schema, provider
  boundary, URL, provenance, freshness, and run-state checks.
- Profile preference matching is advisory. A disliked category or budget
  conflict affects ranking and explanation but does not enforce purchase policy.
- No currency conversion is assumed. Values in different currencies remain
  incomparable unless an existing source-backed conversion capability is
  separately authorized.
- Duplicate-image review uses only evidence already available to the
  application at execution time; absence of a detected match proves nothing.
- Existing retention rules for source Coin Copilot and Deep Analysis records
  remain authoritative for those source records. Feature 363 stores only the
  bounded snapshots and audit metadata required by FR-047 to explain its own
  durable actions.
- No higher retention authority has yet replaced the bounded defaults in
  FR-047; a future approved legal/product policy may shorten them or require a
  documented hold without weakening account deletion obligations.
