# Feature Specification: Coin Copilot Specialist Market Tools

**Feature Branch**: `beta` (existing working branch; no feature branch created)
**Created**: 2026-09-18
**Status**: Draft — Ready for Planning
**Input**: GitHub issue [#723](https://github.com/briandenicola/Aurearia/issues/723), a child phase of [#721](https://github.com/briandenicola/Aurearia/issues/721), with product-owner-approved scope

## Scope Authority

This specification is the authoritative scope for issue #723 and extends the
durable, read-only Coin Copilot harness defined by feature 359. It adds exactly
four read-only specialist capabilities:

1. market and dealer search;
2. auction search;
3. price trends; and
4. similar lots.

The capabilities MUST reuse the existing Python specialist teams and their
provider boundaries. They MUST NOT duplicate search, scraping, comparison, or
analysis logic inside the Copilot harness.

Feature 359 remains authoritative for durable orchestration behavior. Go
continues to own durable thread, run, checkpoint, event, cancellation,
authorization, retention, and replay state. Python remains stateless and
database-free. This phase preserves the existing feature-flagged rollout,
legacy fallback, owner scoping, credential binding, cancellation behavior,
replay behavior, and all existing step, tool-call, time, token-observation,
concurrency, and payload bounds.

Product-owner validation on 2026-09-18 superseded Feature 359's original
single-tool execution default after a combined dealer-and-auction request
failed instead of using both approved read-only tools. New runs permit at most
three concurrent tool executions; the internal contract accepts snapshots from
one through five so existing runs remain replayable and future tuning remains
bounded.

Deep Identification handoff is explicitly deferred because starting it creates
a durable side effect. All collection, wishlist, auction, and settings writes;
proposal or approval cards; durable memory; and replacement of the legacy
router are also outside this feature.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Find current market and auction evidence (Priority: P1)

As a collector, I want Coin Copilot to search dealer listings and auction lots
so that I can discover relevant external evidence without leaving the
conversation or receiving invented listings.

**Why this priority**: Search is the core user value of this phase and supplies
the evidence used by the other specialist capabilities.

**Independent Test**: With the feature enabled and a supported model selected,
ask for currently offered Domitian denarii from dealers and auctions. Verify
that Coin Copilot invokes only the applicable specialist capabilities and
returns bounded, typed results whose listing facts are traceable to validated
source URLs and observation times.

**Acceptance Scenarios**:

1. **Given** providers return valid matching dealer listings, **When** the
   collector requests a market search, **Then** the answer presents only
   source-backed matches and preserves each result's URL, observation time,
   confidence, and availability state.
2. **Given** an auction provider returns valid matching lots, **When** the
   collector requests an auction search, **Then** the answer presents only
   source-backed lots and preserves each result's URL, observation time,
   confidence, and lot status.
3. **Given** no provider returns a match, **When** either search completes,
   **Then** the capability returns an explicit `no_match` outcome and does not
   invent a dealer, auction house, sale, lot, price, estimate, or URL.
4. **Given** some sources succeed while others time out, fail, or return
   malformed data, **When** the search completes, **Then** valid results are
   returned with an explicit `partial` outcome and source-specific warnings.

---

### User Story 2 - Understand price trends from cited observations (Priority: P1)

As a collector, I want a price-trend summary based on observable auction
results so that I can understand the available market evidence and its limits.

**Why this priority**: Price trends are valuable only when the evidence,
sample limitations, and uncertainty remain visible.

**Independent Test**: Request a price trend for a defined coin type using a
provider fixture containing dated sale observations. Verify that every cited
observation remains linked to its source and that the summary reports sample
size, date coverage, currency, confidence, and an explicit trend state.

**Acceptance Scenarios**:

1. **Given** enough valid observations are available, **When** a price trend is
   requested, **Then** the result identifies the observed direction, range,
   sample size, covered period, currency context, confidence, and supporting
   source URLs.
2. **Given** valid evidence is too sparse or incomparable for a defensible
   direction, **When** analysis completes, **Then** the trend is `unavailable`
   or `unknown`, the limitations are explained, and no prices or direction are
   fabricated.
3. **Given** observations use different currencies or price bases, **When** no
   source-backed normalization is available, **Then** values remain explicitly
   separated and are not silently treated as directly comparable.
4. **Given** provider content contains instructions aimed at the model, **When**
   the trend is analyzed, **Then** that content is treated only as untrusted
   evidence and cannot alter the plan, allowed tools, security rules, or answer
   requirements.

---

### User Story 3 - Compare a coin with similar active lots (Priority: P2)

As a collector, I want Coin Copilot to find and rank similar auction lots so
that I can compare market examples while seeing why each result is considered
similar.

**Why this priority**: Similar-lot discovery builds on search evidence and
adds value through transparent comparison rather than unsupported matching.

**Independent Test**: Ask for lots similar to a coin described by ruler,
denomination, era, and type. Verify that each returned lot has a validated
source URL, observed timestamp, confidence, explicit match reasons, and only
source-observed dealer or auction details.

**Acceptance Scenarios**:

1. **Given** valid candidate lots are available, **When** similar lots are
   requested, **Then** results are ranked using stated matching attributes and
   each result explains its similarities and material differences.
2. **Given** a candidate lacks a valid source URL or minimum identifying
   evidence, **When** results are assembled, **Then** that candidate is omitted
   or marked unavailable and is never presented as a verified lot.
3. **Given** no credible similar lot is available, **When** the capability
   completes, **Then** it returns `no_match` without padding the response with
   weak, irrelevant, or invented lots.

---

### User Story 4 - Preserve durable, bounded, safe harness behavior (Priority: P1)

As a collector, I want specialist searches to behave like existing Coin
Copilot runs so that interruption, retries, unsupported models, and external
failures do not leak data or produce duplicate or misleading work.

**Why this priority**: External providers add untrusted data and latency but
must not weaken the harness guarantees established by feature 359.

**Independent Test**: Start a specialist run, disconnect after a completed
tool event, reconnect and replay it, then start another run and cancel during a
provider wait. Verify ordered replay, no duplicate provider result, no
post-cancellation commit, and no cross-user visibility.

**Acceptance Scenarios**:

1. **Given** a provider call is in flight, **When** the owner cancels the run,
   **Then** no new specialist call starts and no late result is committed after
   cancellation wins.
2. **Given** a completed specialist result was checkpointed before disconnect,
   **When** the owner reconnects or resumes, **Then** the result is replayed
   once from Go-owned state and the provider call is not repeated solely to
   reconstruct history.
3. **Given** one user requests another user's specialist run, events, or
   checkpoint, **When** authorization is evaluated, **Then** the resource is
   indistinguishable from an unknown identifier and no evidence is disclosed.
4. **Given** the selected model lacks verified tool-calling support, **When**
   the collector uses chat, **Then** the existing unsupported-model fallback is
   used and no Coin Copilot run or specialist provider call is created.

### Edge Cases

- A provider succeeds but returns zero matches.
- One provider succeeds while another times out, fails, or returns malformed
  data.
- Every provider is unavailable.
- A result has a malformed, unsupported-scheme, credential-bearing, local, or
  otherwise unsafe URL.
- A source URL is valid but the associated dealer, auction house, sale, lot,
  price, or availability field is absent.
- The same source result appears through multiple providers or repeated calls.
- Provider text contains prompt injection, tool instructions, hidden-prompt
  requests, token-shaped text, or claims that contradict the source envelope.
- A provider response or combined result exceeds the existing per-tool,
  checkpoint, or event payload limit.
- Cancellation races provider completion or final answer generation.
- A reconnect begins after a specialist result was persisted but before the
  next checkpoint or terminal event.
- A resumed run contains a completed specialist tool call whose result was
  deterministically truncated.
- A user attempts to read, replay, resume, or cancel another user's run.
- The capability flag changes while a run is active.
- The configured model's tool support is missing, malformed, timed out, or
  ambiguous.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The Coin Copilot allowlist added by this feature MUST contain
  exactly four new read-only capabilities: market/dealer search, auction
  search, price trends, and similar lots.
- **FR-002**: Each new capability MUST delegate to the corresponding existing
  Python specialist team and provider boundary rather than reimplementing
  provider search, page retrieval, lot retrieval, trend analysis, or
  similarity analysis in the Copilot harness.
- **FR-003**: Python MUST remain stateless and database-free. Go MUST remain
  the sole durable owner of Coin Copilot threads, runs, checkpoints, events,
  authorization state, cancellation state, and replay.
- **FR-004**: The feature MUST preserve the existing default-off
  `CoinCopilotEnabled` rollout and MUST use the existing legacy-chat fallback
  when the flag is off, provider configuration is unavailable, tool support is
  unsupported or ambiguous, or harness startup fails before run acceptance.
- **FR-005**: The feature MUST preserve the existing snapshotted execution
  bounds: 8 reasoning iterations by default, 12 total tool calls by default,
  at most 3 concurrent tools by default with an accepted maximum of 5, 120
  seconds by default with a 150-second maximum, one
  active run per owner by default, provider-reported input/output token usage,
  and 32 KiB persisted result data per tool call by default.
- **FR-006**: Specialist calls MUST count against the existing run-wide
  iteration, tool-call, wall-clock, concurrency, and payload budgets; invoking
  a specialist team MUST NOT create a second unbounded budget.
- **FR-007**: Every specialist request and result MUST use a strict typed
  contract. Unknown fields, invalid state values, malformed result shapes, and
  invalid required provenance MUST fail closed or be omitted with an explicit
  warning rather than being passed through as trusted facts.
- **FR-008**: Every returned dealer listing, auction lot, sale observation, and
  similar-lot item MUST include a source URL, an observed timestamp, a
  confidence level, and a verification state. Aggregate results MUST include
  an overall outcome of `complete`, `partial`, `no_match`, or `unavailable`.
- **FR-009**: Price-trend results MUST additionally identify sample size,
  observed date coverage, currency context, trend state, and evidence
  limitations. Similar-lot results MUST additionally state the attributes that
  support the similarity ranking.
- **FR-010**: Source URLs MUST be validated before a result is accepted or
  presented. URLs MUST use an approved web scheme, contain no embedded
  credentials, resolve to an allowed provider or source boundary for that
  capability, and be preserved without invention or material rewriting.
- **FR-011**: A result without a valid source URL MUST NOT be represented as a
  verified dealer listing, auction lot, sale observation, or similar lot.
- **FR-012**: Dealer names, auction houses, sale names, lot numbers, prices,
  estimates, bids, dates, availability, and coin attributes MUST be populated
  only from validated provider evidence. Missing details MUST remain absent or
  explicitly unknown.
- **FR-013**: Provider-supplied text and fetched page content MUST be treated as
  untrusted data, never as instructions. It MUST NOT change the tool allowlist,
  trigger another capability, reveal hidden prompts or credentials, override
  cancellation, or alter owner scope.
- **FR-014**: Provider success, no-match, timeout, transport failure,
  unavailable provider, and malformed-data outcomes MUST map to typed,
  client-safe states. Internal errors and credentials MUST NOT appear in
  persisted or user-visible payloads.
- **FR-015**: When at least one source yields valid evidence and another source
  fails, the capability MUST preserve the valid evidence, mark the aggregate
  `partial`, and identify limitations without inventing replacement data.
- **FR-016**: When no valid evidence is available, the capability MUST return
  `no_match` if providers completed successfully with no matches, or
  `unavailable` if failures prevent a reliable no-match conclusion.
- **FR-017**: Tool results MUST be deterministically bounded using the existing
  persisted-result rules. Truncated results MUST retain truncation status,
  original size, persisted size, and digest, and the final answer MUST not
  imply omitted evidence was reviewed.
- **FR-018**: Duplicate results with the same canonical source identity MUST
  not be presented as independent evidence. Deduplication MUST preserve the
  strongest available provenance and disclose conflicting observations.
- **FR-019**: Go MUST continue to mint a fresh owner/run/execution/tool-bound
  credential for each execution or resume. Specialist capabilities MUST not
  receive authority for writes, arbitrary HTTP, filesystem, shell, code
  execution, direct database access, or any unlisted tool.
- **FR-020**: Cancellation MUST remain cooperative in Python and authoritative
  in Go. The system MUST check cancellation before each specialist call and
  after each awaited provider operation, and MUST discard late frames or
  results after cancellation wins.
- **FR-021**: Replay and resume MUST use persisted Go-owned events and
  checkpoints. A completed specialist call MUST not be repeated merely because
  the client disconnected or the Python process restarted.
- **FR-022**: All public specialist resources and operations MUST remain scoped
  to the authenticated owner derived server-side. Unknown and foreign
  identifiers MUST both return the same not-found behavior.
- **FR-023**: The four specialist capabilities MAY be composed with the
  existing read-only collection capabilities within one bounded run, but they
  MUST NOT invoke or imply any mutation.
- **FR-024**: Deep Identification handoff MUST NOT be exposed in this phase,
  including automatic, user-confirmed, or tool-mediated job creation, because
  starting that workflow creates durable state outside the current run.
- **FR-025**: Collection, wishlist, auction, and settings writes; proposal or
  approval cards; durable user memory; legacy-router replacement; and any
  additional specialist capability MUST remain unavailable to this feature.
- **FR-026**: Existing public event names, replay ordering, idempotency,
  retention, redaction, hidden-reasoning exclusions, and terminal-state rules
  from feature 359 MUST remain compatible.
- **FR-027**: Automated tests MUST cover, for each specialist capability,
  provider success, no-match, timeout, failure, and malformed data, and MUST
  verify the required typed outcome and provenance behavior.
- **FR-028**: Automated tests MUST reject invalid or fabricated source URLs,
  missing required provenance, unsafe URL schemes, embedded credentials, and
  dealer or auction details unsupported by provider evidence.
- **FR-029**: Automated tests MUST verify that prompt-injection content in
  provider data remains inert and cannot cause tool escalation, data leakage,
  hidden-prompt disclosure, or unsupported claims.
- **FR-030**: Automated tests MUST verify cancellation races, checkpoint/event
  replay without duplicate provider execution, cross-user isolation, and
  unsupported-model fallback with no specialist call or new durable run.

### Key Entities

- **Specialist Capability**: One of the four allowlisted read-only operations,
  including its typed input, eligible provider boundary, and run-wide budget
  consumption.
- **Specialist Result Envelope**: Bounded outcome for one capability call,
  including capability, outcome state, result items, warnings, truncation
  metadata, and provider coverage.
- **Evidence Item**: A source-backed dealer listing, auction lot, sale
  observation, or similar-lot candidate with URL, observation time,
  confidence, verification state, and capability-specific fields.
- **Price Trend Summary**: An evidence-derived aggregate with trend state,
  sample size, date coverage, currency context, confidence, limitations, and
  links to supporting evidence.
- **Provider Attempt**: A bounded, non-durable description of one provider
  outcome—success, no-match, timeout, failure, or malformed data—used to
  determine aggregate completeness without exposing internal errors.
- **Coin Copilot Run**: The existing owner-scoped durable execution that
  snapshots limits, records typed specialist results in checkpoints/events,
  and remains the authority for cancellation, replay, and terminal state.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In acceptance tests, users can complete each of the four
  specialist requests in one Coin Copilot run and receive either source-backed
  results or an explicit `no_match`/`unavailable` outcome, with no fabricated
  listing or lot details in 100% of cases.
- **SC-002**: 100% of presented dealer listings, auction lots, price
  observations, and similar lots include a valid source URL, observed
  timestamp, confidence, and verification state.
- **SC-003**: 100% of provider timeout, failure, and malformed-data test cases
  settle within the run's existing time budget and produce a typed partial or
  unavailable outcome without leaking internal error details.
- **SC-004**: Mixed-provider tests retain all valid bounded evidence while
  identifying every failed or omitted source category, with zero invented
  substitutes.
- **SC-005**: Cancellation tests commit zero specialist results or final-answer
  claims after cancellation wins, and replay tests repeat zero completed
  provider calls solely because of reconnect or resume.
- **SC-006**: Cross-user tests disclose zero run, checkpoint, event, or
  specialist-result data and make foreign identifiers indistinguishable from
  unknown identifiers.
- **SC-007**: Prompt-injection test fixtures cause zero unauthorized tool
  calls, scope changes, credential disclosures, hidden-prompt disclosures, or
  unsupported factual claims.
- **SC-008**: No accepted run exceeds its snapshotted iteration, tool-call,
  concurrency, wall-clock, or persisted-result bounds; reliable
  provider-reported token usage remains observable for every tested provider
  that supplies it.
- **SC-009**: With Coin Copilot disabled or the selected model unsupported,
  100% of tested chat requests use the existing fallback and create zero new
  Coin Copilot runs or specialist provider calls.
- **SC-010**: At least 95% of evaluators in an acceptance review can identify
  the source and confidence of every displayed market claim without consulting
  internal logs or implementation details.

## Non-Goals

- Starting, recommending as an action, or otherwise handing off to a Deep
  Identification job.
- Any collection, wishlist, auction, bid, watch, notification, or settings
  mutation.
- Proposal generation, approval cards, or applying suggested changes.
- Durable user memory or a Python-side checkpoint, cache, queue, or database.
- Replacing, changing, or removing the legacy chat router and supervisor.
- Adding arbitrary browsing, arbitrary HTTP, filesystem, shell, code
  execution, or direct database tools.
- Adding specialist capabilities other than the four listed in this
  specification.
- Changing existing Coin Copilot retention, replay, cancellation, model
  eligibility, or run-bound defaults.

## Assumptions

- Feature 359 and issue #721 provide the durable Coin Copilot harness on which
  this child phase depends.
- The existing `coin_search`, `auction_search`, `price_trends`, and
  `similar_lots` specialist teams remain the canonical domain workflows and
  may be adapted behind strict typed boundaries without duplicating their
  logic.
- Existing provider boundaries remain responsible for network access,
  provider-specific behavior, and source allowlisting.
- External market evidence is time-sensitive; `observed_at` describes when the
  application observed a source and is not a guarantee that a listing remains
  available later.
- Confidence communicates evidence quality, not authenticity, grade, value, or
  purchase advice.
- Existing feature-359 event retention, checkpoint retention, resume window,
  owner deletion, and operational rollback behavior remain unchanged.

## Dependencies

- Parent issue [#721](https://github.com/briandenicola/Aurearia/issues/721) and
  feature 359's durable Coin Copilot contracts.
- Existing Python specialist teams for market/dealer search, auction search,
  price trends, and similar lots.
- Existing provider tools and source boundaries, including their configured
  availability and permitted source domains.
- Existing Coin Copilot feature flag, model-capability preflight, execution
  credentials, typed frames, cancellation, checkpoints, event replay, and
  legacy fallback.

## Amendment 2026-09-19: one validation boundary

Shipping this spec as written produced three hand-written copies of the same
schema (Pydantic, Go structs, TypeScript validators) plus a byte-exact digest
contract between Python and Go. Every seam between those copies produced a
user-visible outage: a title containing "Æ" was rejected as a corrupt
checkpoint because Python counted ASCII-escaped bytes and Go counted UTF-8,
and every dealer result was silently discarded by the browser because its
key allow-list omitted fields the Go projection sent.

The following requirements are therefore withdrawn:

- **FR-007**, as far as it required each layer to re-validate the schema and
  fail closed. The agent service owns the schema. Go persists and streams the
  result as data, converting keys for the browser without knowing the fields.
  The browser checks only what it needs to render safely.
- **FR-008**, as far as it required per-field provenance records. Evidence
  keeps its source URL, observed time, confidence and verification state;
  the per-field provenance list is gone.
- **FR-017**, as far as it required a digest and byte counts. Tool results are
  bounded by a size cap with a truncation flag. No digest is recomputed
  across languages.

Retained without change: source URLs must be https, credential-free and on a
configured host; provider text is untrusted data screened for injection and
token shapes; results are bounded in size and item count; outcomes stay typed
so a failed source can never read as "no matches".

Evidence now also carries `image_url` and `candidate_references`, which the
original contract dropped and which the legacy pipeline had always shown.
