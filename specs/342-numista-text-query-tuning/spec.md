# Feature Specification: Measured Numista Text-Query Tuning

**Feature Branch**: `342-numista-text-query-tuning`  
**Created**: 2026-08-11  
**Status**: Implemented
**Input**: Improve generated Numista text queries without image search or changes to explicit collector queries.

## Governance

PR #602 is merged and Feature 341 is landed public history. Under Constitution
§0 and §18.2, this follow-up MUST NOT retroactively amend Feature 341's locked
specification, plan, tasks, ADRs, or release record. This small, independently
measurable behavior change is therefore Feature 342 rather than a Feature 341
amendment or backlog-only card.

## User Scenarios & Testing

### User Story 1 - Search with concise generated terms (Priority: P1)

As a collector using saved-coin, photo, or draft lookup, I want generated
Numista queries to contain only strong catalog terms, while retaining all
other evidence for ranking.

**Independent Test**: For curated direct, non-NGC photo, and draft evidence,
verify the generated query contains subject/ruler, reverse legend or type, and
a reliable normalized mint; excludes references, material, date, prose, and
slab/label text; and still sends the excluded fields to the scorer.

**Acceptance Scenarios**:

1. **Given** ruler, reverse evidence, mint, date, material, references, and
   prose are available, **when** a proposal is generated, **then** only the
   ruler/subject, reverse legend/type, and reliable normalized mint appear in
   `q`.
2. **Given** `SMN` or `SMNT` is an exact approved mint alias, **when** a
   proposal is generated, **then** the mint component is `Nicomedia`.
3. **Given** evidence is omitted from `q`, **when** candidates are ranked,
   **then** the original bounded evidence remains available to the existing
   application scorer.

### User Story 2 - Preserve collector query control (Priority: P1)

As a collector, I want my edited or manually entered query submitted exactly
as written, without automatic rewriting or relaxation.

**Independent Test**: Generate a proposal, edit it, and submit it; then start
with an empty/manual panel and submit a query. Verify both effective queries
match the submitted values and neither receives an automatic relaxed retry.

**Acceptance Scenarios**:

1. **Given** an untouched server proposal, **when** it is submitted, **then**
   it is marked `generated`.
2. **Given** a server proposal was changed through collector input, **when**
   it is submitted, **then** it is marked `user-edited`; that marker remains
   sticky for the panel session even if the text later equals the proposal.
3. **Given** no proposal initialized the input, **when** the collector enters
   a query, **then** it is marked `manual`.
4. **Given** a `user-edited` or `manual` request, **when** lookup runs, **then**
   the provider receives the exact submitted `q`, subject only to the existing
   1–500 character validation, and no query-relaxation retry occurs.

### User Story 3 - Relax one empty generated search (Priority: P1)

As a collector, I want one conservative retry when an untouched generated
query is too narrow, without multiplying requests or hiding the query used.

**Independent Test**: Return empty for a generated primary query and a result
for its distinct relaxed query. Verify exactly two application search attempts,
the effective relaxed query is reported, and manual/edited/error cases perform
one attempt.

**Acceptance Scenarios**:

1. **Given** an authenticated `generated` primary lookup returns `empty`,
   **when** the canonical builder can produce a distinct non-empty relaxed
   query, **then** the service performs exactly one relaxed search.
2. **Given** the primary search succeeds or returns any non-empty terminal
   state other than `empty`, **then** no relaxed query is attempted.
3. **Given** the relaxed search is also empty, **then** the final outcome is
   `empty`, identifies the relaxed attempt, and reports its effective query.
4. **Given** the client labels a query `generated` but it does not exactly
   match the server's versioned proposal for the submitted evidence, **then**
   the server treats it as `user-edited` and does not relax it.

### User Story 4 - Measure improvement safely (Priority: P2)

As a maintainer, I want deterministic fixtures and sanitized live-query
evidence to prove the change helps without adding provider or privacy risk.

**Independent Test**: Compare the landed Feature 341 verbose builder with the
new builder over frozen evidence cases and replay captured provider responses;
then verify the existing 24-known-coin scoring benchmark remains above its
binding threshold.

**Acceptance Scenarios**:

1. **Given** the curated query fixture set, **when** old and new query outcomes
   are compared, **then** the report shows per-case primary query, optional
   relaxed query, expected candidate, and rank.
2. **Given** sanitized live-query observations, **when** they are recorded,
   **then** they contain no images, credentials, owner data, raw slab text, or
   full user prose.
3. **Given** Feature 342 is implemented, **when** release gates run, **then**
   no image-search endpoint, provider capability, upload, or telemetry field
   has been added.

### Edge Cases

- No usable subject/ruler: a proposal may use the existing short structured
  title as fallback; it MUST NOT mine subject text from prose or slab text.
- No reverse legend/type or reliable mint: omit the missing component; do not
  invent it.
- Alias-like text embedded in a sentence or longer token is not a mint alias.
- A generated primary query with fewer than two distinct components has no
  relaxed retry unless relaxation produces a different non-empty query.
- Existing transient HTTP retry rules remain transport behavior and are not
  expanded by this feature.
- NGC-first behavior remains unchanged. A usable NGC result triggers no
  Numista proposal request or provider search automatically.

## Requirements

### Functional Requirements

- **FR-001**: One Go-owned, versioned query builder MUST generate proposals for
  direct, non-NGC photo, and Quick Capture draft paths.
- **FR-002**: Vue MUST collect evidence, display proposals, and track edits,
  but MUST NOT independently assemble provider query text.
- **FR-003**: Generated primary queries MUST use, in order: issuer/ruler or
  short structured title fallback; reverse inscription and/or short reverse
  type; reliable normalized mint.
- **FR-004**: Generated `q` MUST omit denomination, date/range, material,
  exact/catalog references, obverse text, visible/slab/label text, notes, and
  descriptive prose.
- **FR-005**: Omitted fields MUST remain in bounded `NumistaEvidence` for
  scoring. `reverseType` MAY be added additively to that existing contract.
- **FR-006**: Mint aliases MUST use a versioned exact allowlist. The initial
  required aliases are `SMN` and `SMNT` to `Nicomedia`; further variants
  require an explicit sanitized fixture and exact table entry.
- **FR-007**: Alias matching MUST use Unicode normalization, case folding, and
  separator removal only. Prefix, substring, fuzzy, regex-family, and
  free-prose inference are prohibited.
- **FR-008**: Proposal responses MUST include the proposal text and generation
  version `numista-query-v2`.
- **FR-009**: Lookup requests MUST identify `generated`, `user-edited`, or
  `manual` query source. `generated` is authoritative only when the query
  exactly matches the server-rebuilt proposal and version.
- **FR-010**: `user-edited` is sticky after collector input changes a proposal.
  `manual` applies when no generated proposal initialized the input.
- **FR-011**: Manual and user-edited provider queries MUST be submitted
  unchanged; server normalization remains internal to cache/scoring identity.
- **FR-012**: Only an empty, verified generated primary lookup MAY cause one
  relaxed retry. No other status or query source may do so.
- **FR-013**: The relaxed query MUST deterministically retain subject and
  reliable normalized mint, omit reverse legend/type, and run only when
  distinct and non-empty.
- **FR-014**: `effectiveQuery` MUST identify the actual query used for the
  returned/final outcome. The response MUST also identify query source,
  `primary` versus `relaxed`, and total application search attempts.
- **FR-015**: Primary and relaxed searches MUST use the existing cache
  independently by actual query. A cached primary empty MAY proceed to one
  cached or fresh relaxed attempt.
- **FR-016**: Telemetry MUST attribute safe enums for query source and attempt
  (`primary`/`relaxed`) without recording query/evidence text, mint aliases,
  images, or raw provider data.
- **FR-017**: Existing authentication, role-safe errors, cancellation,
  enrichment, selection, persistence, and transactional promotion contracts
  MUST remain unchanged.
- **FR-018**: NGC-first and no-eager-provider behavior MUST remain unchanged.
  Image search is explicitly out of scope.
- **FR-019**: The deprecated exact manual GET `/api/numista/search?q=...`
  adapter MUST preserve its current one-query behavior and MUST NOT relax.

## Success Criteria

- **SC-001**: Across at least 12 sanitized query-generation fixtures spanning
  direct, photo, draft, alias, exclusion, and missing-field cases, 100% of
  generated primary and relaxed strings match the contract.
- **SC-002**: On the frozen live-query comparison set, correct-candidate
  top-three inclusion improves by at least 10 percentage points over the
  landed verbose builder, with no case losing a previously top-three correct
  candidate unless explicitly reviewed and accepted.
- **SC-003**: The existing 24-known-coin scoring benchmark remains at or above
  85% top-three accuracy without exact-ID leakage.
- **SC-004**: 100% of manual and user-edited contract tests make one
  application search attempt and preserve the exact submitted query.
- **SC-005**: 100% of generated-empty tests make no more than two application
  search attempts, and every other status makes one.
- **SC-006**: Telemetry/privacy tests find zero query text, evidence text,
  image data, slab text, credentials, or raw provider payloads.

## Out of Scope

- Numista image search, image upload, perceptual matching, or paid-plan APIs.
- New AI/LLM parsing, speculative mintmark grammars, or broad mint ontology.
- Scoring-weight changes, candidate persistence changes, or eager NGC lookup.
