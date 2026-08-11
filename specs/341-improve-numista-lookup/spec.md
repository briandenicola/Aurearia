# Feature Specification: Improved Numista Lookup

**Feature Branch**: `341-improve-numista-lookup`  
**Created**: 2026-08-11  
**Status**: Draft  
**Input**: Improve Numista lookup for direct coin-detail and photo-assisted workflows through a shared contract, richer editable queries, explainable relevance ranking, selected-reference persistence, distinct service states, caching and telemetry, and staged result enrichment.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Find relevant matches from coin details (Priority: P1)

As a collector reviewing a coin, I want Numista search to use the coin's useful attribution evidence while letting me edit the search, so I can find the right catalog entry without repeatedly rewriting a basic query.

**Why this priority**: Direct lookup is the established catalog workflow and must produce useful, understandable results before downstream persistence adds value.

**Independent Test**: Open a coin containing ruler, denomination, mint, date, material, and inscriptions; review and edit the generated query; search; and verify that ranked results explain which evidence matched.

**Acceptance Scenarios**:

1. **Given** a coin has attribution fields, **When** the collector opens Numista lookup, **Then** the proposed editable query includes available name, ruler or issuer, denomination, mint, date, material, and inscription evidence without inserting empty values.
2. **Given** the proposed query is too narrow or contains uncertain text, **When** the collector edits it and searches, **Then** the submitted query is preserved visibly and drives the displayed results.
3. **Given** Numista returns several candidates, **When** results are displayed, **Then** candidates are ordered by application-defined relevance and each shows a concise explanation of supporting matches and material conflicts.
4. **Given** a collector chooses a candidate, **When** they add it to the coin, **Then** that Numista identifier and canonical link are saved as one structured reference without attaching unselected candidates.

---

### User Story 2 - Identify a coin from photos (Priority: P1)

As a collector using photo lookup or Quick Capture, I want visible evidence to produce editable Numista search terms and ranked candidates, so I can review the match instead of trusting a provider's default ordering.

**Why this priority**: Photo lookup is less certain than direct lookup and therefore needs stronger review, explanation, and explicit selection.

**Independent Test**: Upload coin photos without an NGC certification number, review a query built from extracted evidence, run lookup, select a result, save a Quick Capture draft, and promote it.

**Acceptance Scenarios**:

1. **Given** photos have no usable NGC certification number, **When** analysis finds mint, date, material, inscriptions, label text, ruler, denomination, or title evidence, **Then** available evidence contributes to an editable Numista query.
2. **Given** extracted text is incorrect or noisy, **When** the collector corrects or removes it before retrying, **Then** the revised query is used without requiring new photos.
3. **Given** ranked Numista candidates are shown, **When** the collector selects one and saves the lookup as a Quick Capture draft, **Then** the selected reference remains associated with the draft through later edits.
4. **Given** a draft has a selected Numista reference, **When** it is promoted to a collection or wishlist coin, **Then** exactly that reference is persisted on the promoted coin.
5. **Given** no candidate is selected, **When** the draft is saved or promoted, **Then** no Numista reference is inferred or attached automatically.

---

### User Story 3 - Understand lookup availability (Priority: P1)

As a collector, I want lookup failures and empty results to be clearly distinguished, so I know whether to revise the query, configure Numista, wait for quota recovery, or retry.

**Why this priority**: Treating all non-results alike causes incorrect user action and hides operational failures.

**Independent Test**: Exercise successful-empty, unconfigured, unavailable, quota-limited, and timed-out lookups from both direct and photo paths and verify distinct guidance without losing the query or selected draft reference.

**Acceptance Scenarios**:

1. **Given** a configured and reachable service returns no candidates, **When** lookup finishes, **Then** the state is `empty` and guidance recommends revising the query.
2. **Given** Numista is not configured, **When** lookup is requested, **Then** the state is `unconfigured` and authorized users are directed to configuration while other users receive non-sensitive guidance.
3. **Given** Numista rejects lookup because its allowance is exhausted or limited, **When** lookup finishes, **Then** the state is `quota-limited`, any known retry timing is shown, and the condition is not presented as an empty result.
4. **Given** Numista exceeds the allowed wait, **When** lookup stops, **Then** the state is `timeout`, the query remains editable, and retry is offered.
5. **Given** Numista or the application's lookup capability is otherwise unhealthy, **When** lookup fails, **Then** the state is `unavailable` with safe retry guidance.

---

### User Story 4 - Conserve quota without hiding freshness (Priority: P2)

As an instance administrator, I want repeated lookups to reuse recent results and expose health and quota signals, so collectors receive dependable results without wasting the shared Numista allowance.

**Why this priority**: The documented allowance is shared and finite; caching and visibility protect the primary workflows.

**Independent Test**: Repeat equivalent direct and photo searches, inspect freshness indicators and operational metrics, expire cached entries, and simulate health and quota failures.

**Acceptance Scenarios**:

1. **Given** an equivalent recent query has a fresh cached result, **When** either lookup path searches, **Then** it receives the reusable result without consuming another provider lookup and can see that cached data was used.
2. **Given** a cached result has exceeded its time-to-live, **When** the query is repeated, **Then** the system attempts a fresh lookup rather than silently treating stale data as current.
3. **Given** lookups occur, **When** an administrator reviews system health, **Then** they can distinguish successful, empty, unconfigured, quota-limited, timeout, unavailable, cached, and refreshed outcomes and see recent latency and quota-related signals.

---

### User Story 5 - Review useful details without excessive requests (Priority: P2)

As a collector, I want broad search results quickly and richer details only for the strongest candidates, so comparison is useful without spending quota on every result.

**Why this priority**: Candidate details improve ranking and selection, but broad discovery must remain responsive and quota-conscious.

**Independent Test**: Run a search with more candidates than the enrichment limit and verify broad results appear first, only the leading candidates are enriched, ranking is updated predictably, and partial enrichment failures do not discard usable search results.

**Acceptance Scenarios**:

1. **Given** a valid query, **When** lookup begins, **Then** a broad candidate search completes before detail enrichment.
2. **Given** broad search returns candidates, **When** initial relevance is assessed, **Then** only the configured leading subset is enriched with available catalog details.
3. **Given** enrichment adds useful mint, date, material, inscription, issuer, or image evidence, **When** final ranking is shown, **Then** explanations identify the evidence that affected relevance.
4. **Given** detail enrichment fails for one or more candidates, **When** broad search data remains usable, **Then** those results remain available with their unenriched state identified.

### Edge Cases

- The proposed query is empty because neither stored fields nor photo analysis produced useful evidence; search is disabled with guidance to enter at least one term.
- Very long, duplicated, conflicting, or low-confidence extracted terms are presented safely and can be removed; they do not silently outweigh stronger exact evidence.
- Date ranges span BCE/CE, use uncertain notation, or conflict with a candidate's range; scoring handles the comparison deterministically and explains the conflict.
- Inscriptions contain punctuation, damaged characters, mixed scripts, or partial legends; normalization preserves the collector's editable source text.
- Two candidates have equal relevance; a stable tie-break keeps ordering repeatable.
- A selected candidate disappears from a later search; the existing selection is retained until the collector replaces or removes it and is clearly marked as outside the latest result set.
- A draft is saved, retried, edited, or promoted more than once; the selected reference is neither duplicated nor replaced by the first search result.
- Search succeeds but all detail requests fail; broad candidates remain selectable and the overall state is not misreported as empty.
- A fresh cached empty result exists; it is returned as `empty` with freshness context rather than as an outage.
- Configuration changes or credentials are removed while cached data exists; cached results never misrepresent current configuration or health.
- Provider responses contain malformed, missing, or unexpected fields; unusable candidates are excluded or shown without optional details, and the lookup remains safe.
- Concurrent equivalent requests share reusable work where practical and do not multiply provider usage unnecessarily.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST provide one shared, explicit, typed Numista lookup contract—referred to as the shared `NumistaClient` capability—for direct coin-detail and photo-assisted paths so query, candidate, status, caching, and error meanings remain consistent.
- **FR-002**: The shared lookup contract MUST represent provider results as application-owned candidate data rather than exposing unvalidated provider payloads to user workflows.
- **FR-003**: Direct lookup MUST propose an editable query from all available relevant fields: coin name, ruler or issuer, denomination, mint, date or date range, material, obverse inscription, and reverse inscription.
- **FR-004**: Photo-assisted lookup MUST propose an editable query from all available relevant evidence: inferred title, ruler or issuer, denomination, mint, date or date range, material, inscriptions, and other visible coin or label text.
- **FR-005**: Both lookup paths MUST let the collector edit the proposed query before the first search and before any retry without requiring changes to the coin or a new photo analysis.
- **FR-006**: The system MUST preserve the exact effective query with its result set so collectors can understand and repeat what produced the candidates.
- **FR-007**: The system MUST calculate an application-owned relevance score for every usable candidate using available query, coin, extracted, and enriched candidate evidence rather than relying solely on provider ordering.
- **FR-008**: Relevance assessment MUST consider, when available, title, issuer or ruler, denomination, mint, date compatibility, material, inscriptions or visible text, and exact Numista identifier evidence.
- **FR-009**: Every displayed score or relevance band MUST include a concise explanation of meaningful matches, mismatches, and unavailable evidence; it MUST NOT imply definitive attribution.
- **FR-010**: Ranking MUST be deterministic for the same normalized evidence and candidate details, including a stable tie-break.
- **FR-011**: A collector MUST be able to explicitly select, replace, or remove a Numista candidate in direct and photo-assisted workflows.
- **FR-012**: Direct lookup MUST persist only the collector-selected candidate as a structured Numista reference containing the stable Numista identifier and canonical catalog link.
- **FR-013**: Photo-assisted lookup MUST persist the selected Numista reference with the Quick Capture draft so it survives save, resume, edit, retry, and validation failure.
- **FR-014**: Quick Capture promotion MUST transactionally copy exactly the selected Numista reference to the promoted collection or wishlist coin and MUST remain idempotent across repeated promotion attempts.
- **FR-015**: The system MUST NOT attach all returned candidates, infer selection from result order, or create a Numista reference when the collector has made no selection.
- **FR-016**: Lookup outcomes MUST distinguish at least `success`, `empty`, `unconfigured`, `quota-limited`, `timeout`, and `unavailable` in both direct and photo-assisted paths.
- **FR-017**: Each non-success outcome MUST provide user guidance appropriate to that state while withholding credentials, internal errors, and administrator-only details.
- **FR-018**: Lookup MUST use a two-stage process: broad catalog search first, followed by detail enrichment for only the leading candidate subset.
- **FR-019**: The leading enrichment subset MUST be bounded and configurable by an administrator, with a default of five candidates.
- **FR-020**: Detail enrichment MUST retrieve only information needed for comparison, explanation, display, and reference selection; a failed detail lookup MUST NOT discard a usable broad-search candidate.
- **FR-021**: Equivalent normalized searches and candidate detail lookups MUST reuse fresh cached outcomes across direct and photo-assisted paths.
- **FR-022**: Search and detail cache entries MUST have independently configurable time-to-live values, defaulting to 24 hours for searches and seven days for catalog details.
- **FR-023**: Cached data MUST expose whether it was reused and its freshness, and expired data MUST NOT be silently represented as current.
- **FR-024**: Configuration changes MUST take effect without requiring expiration of unrelated cached catalog content, and cached content MUST NOT conceal an unconfigured state.
- **FR-025**: The system MUST record operational signals for lookup path, outcome status, cache use, broad search, detail enrichment, elapsed time, candidate count, and quota-limited responses without recording credentials or full user-supplied inscriptions.
- **FR-026**: Administrators MUST be able to assess current Numista configuration and recent health, timeout, quota, cache, and enrichment outcomes from application telemetry.
- **FR-027**: Existing NGC-first photo behavior MUST remain unchanged: a usable NGC certification result does not automatically trigger Numista lookup unless the collector explicitly requests it.
- **FR-028**: Existing authenticated ownership and authorization rules MUST apply to selecting references, saving drafts, adding references to coins, and promotion.
- **FR-029**: Existing structured-reference validation and deduplication rules MUST govern persisted Numista references.

### Nonfunctional Expectations

- **NFR-001**: At least 95% of uncached broad searches under normal provider availability MUST present initial candidates or a distinct terminal status within 5 seconds.
- **NFR-002**: At least 95% of fresh-cache lookups MUST present results or an empty state within 1 second.
- **NFR-003**: Detail enrichment MUST be progressive or bounded so it does not delay display of usable broad results beyond the broad-search target.
- **NFR-004**: Relevance explanations MUST be understandable to collectors and must label uncertain or conflicting evidence rather than presenting a score as expert authentication.
- **NFR-005**: Provider credentials, raw internal errors, and sensitive operational details MUST never appear in collector-visible responses, logs, scoring explanations, or cached keys.
- **NFR-006**: Full inscriptions and visible label text MUST not be retained in telemetry; operational correlation MUST use non-reversible or redacted identifiers where needed.
- **NFR-007**: Lookup, caching, scoring, and selection behavior MUST be independently testable without live provider access.
- **NFR-008**: The change MUST preserve responsive behavior in the mobile/PWA photo workflow and accessible keyboard operation in both lookup paths.

### Key Entities

- **Numista Lookup Request**: The effective editable query, lookup path, normalized evidence, and request context needed to perform and explain a search.
- **Numista Candidate**: An application-owned representation of a Numista type with stable identifier, canonical link, broad result fields, optional enriched details, relevance assessment, and enrichment state.
- **Relevance Assessment**: A deterministic score or band plus matched, conflicting, and unavailable evidence used to explain ordering.
- **Lookup Outcome**: A result set and one explicit status (`success`, `empty`, `unconfigured`, `quota-limited`, `timeout`, or `unavailable`) with safe guidance and freshness context.
- **Selected Numista Reference**: The collector's explicit choice of candidate, represented by stable catalog name, Numista identifier, and canonical link.
- **Cached Lookup**: A normalized search or candidate-detail outcome with creation time, expiry time, freshness, and safe cache identity.
- **Numista Health Signal**: Aggregated operational evidence about availability, quota limitation, timeout, latency, cache use, and enrichment outcomes.
- **Quick Capture Draft**: The existing user-owned draft, extended to retain at most one selected Numista reference until removal, replacement, discard, or promotion.
- **Coin Reference**: The existing structured reference associated with a promoted or directly edited coin.

## Migration & Backward Compatibility

- Existing coins, Quick Capture drafts, and structured references remain valid without migration by users.
- Existing Numista references retain their identifiers and links and are not rescored, removed, or duplicated.
- Existing drafts without a selected Numista reference continue to save, resume, discard, and promote as before.
- Any persisted selected-reference field is optional during rollout; absent data means no selection.
- Promotion remains compatible with existing collection and wishlist targets, lifecycle states, validation, ownership, and idempotency.
- Existing direct lookup and photo lookup entry points remain available; their result and error presentations transition to the shared outcome model.
- Existing configuration remains authoritative. No credential re-entry is required.
- Cached data is disposable and does not become an authoritative collection record.
- Rollback MUST leave existing coins and references readable and MUST not strand drafts or block promotion solely because improved lookup data is absent.

## Out of Scope

- Replicating or indexing the Numista catalog locally.
- Bulk importing Numista catalog data or automatically attaching multiple references.
- Automatic attribution without explicit collector selection.
- Replacing NGC certification lookup, vision analysis, or existing structured-reference validation.
- Price estimation, valuation, grading, authenticity, or provenance conclusions.
- Editing Numista content or submitting corrections to Numista.
- Offline Numista search or offline Quick Capture promotion.
- Guaranteeing provider quota availability or exposing provider credentials.
- Reworking unrelated coin catalogs or general-purpose search ranking.

## Assumptions

- Numista remains an external catalog with a shared instance-level allowance and a configured credential managed through existing admin settings.
- A broad search result supplies stable candidate identifiers; richer catalog details can be requested separately for a bounded subset.
- Five enriched candidates balances comparison value and quota use unless an administrator chooses another positive bounded value.
- Catalog search data changes less often than user-entered coin data, making a 24-hour search TTL and seven-day detail TTL reasonable defaults.
- A single selected Numista type is sufficient for this workflow; collectors may still manage other structured references through existing reference controls.
- App-side relevance is decision support, not expert attribution, and the collector remains responsible for selection.
- Existing authentication, authorization, safe-link, settings, Quick Capture, and structured-reference rules remain authoritative.

## Dependencies and Authority

- `.specify/memory/constitution.md` is the highest authority; this feature must preserve layered boundaries, explicit typed contracts, safe errors, simple complete scope, shared-workflow regression coverage, and transactional multi-step writes.
- `docs/prd.md` identifies coin-detail Numista lookup and admin configuration as shipped capabilities and prohibits replicating third-party catalogs; this feature improves linking and comparison without creating a local catalog.
- `specs/214-structured-numismatic-catalog-references/spec.md` is the originating feature specification for structured references. Although its header still says Draft, its task list is completed and implementation is landed; it is treated as landed authority and is not amended by this feature.
- `specs/336-quick-capture/spec.md` governs draft ownership, lifecycle, and idempotent promotion. Its implementation is landed despite the Draft header and residual task-list gaps; it is treated as landed and is not modified.
- `docs/features/numista-integration.md` documents the shipped user-facing direct and photo lookup behavior. It is a lower-authority feature document and must be updated during implementation if behavior changes.
- The existing Numista credential, Quick Capture promotion workflow, structured Coin Reference capability, photo analysis, and external provider availability are required dependencies.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: In representative direct-lookup tests, at least 90% of collectors can review or edit a generated query, identify why the top candidate is ranked, and select or reject it without assistance.
- **SC-002**: In a curated set of known coins where the correct Numista type appears in broad results, the correct candidate appears in the top three after enrichment at least 85% of the time.
- **SC-003**: 100% of selected Numista references saved through photo lookup remain unchanged through draft resume and are copied exactly once during successful collection or wishlist promotion.
- **SC-004**: 0 unselected Numista candidates are persisted across direct add, draft save, or promotion test suites.
- **SC-005**: 100% of tested empty, unconfigured, quota-limited, timeout, and unavailable conditions display the correct distinct state in both lookup paths.
- **SC-006**: Repeating equivalent lookups within the default freshness period reduces external broad-search usage by at least 80% in a representative repeated-query workload.
- **SC-007**: No more than five candidate detail enrichments occur per search under default configuration, and partial enrichment failure preserves 100% of usable broad candidates.
- **SC-008**: At least 95% of normal-availability uncached searches show initial candidates or a terminal status within 5 seconds, and at least 95% of fresh-cache searches do so within 1 second.
- **SC-009**: Administrators can distinguish all required outcome states, cache use, and enrichment outcomes for 100% of sampled lookups without access to credentials or full inscription text.
- **SC-010**: Existing NGC-first lookup, direct structured-reference management, draft ownership, collection/wishlist promotion, and promotion idempotency regression scenarios continue to pass with no behavior change outside this feature's stated scope.
