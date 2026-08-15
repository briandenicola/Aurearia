# Phase 0 Research: Optional Nomisma.org Authority Linking

No `NEEDS CLARIFICATION` markers remain in the Technical Context — the spec's
recorded clarification (2026-08-14) already resolves whether a pre-validation
gate is required (no) and what "reconciliation works" means for planning
purposes. This document records the supporting research, confirms the PRD
non-goal reconciliation, and settles the remaining implementation-shape
decisions the spec deliberately left to planning.

## 1. Nomisma.org reconciliation service

- **Decision**: Use Nomisma's public OpenRefine-compatible reconciliation
  endpoint, `https://nomisma.org/apis/reconcile`, as the sole on-demand
  lookup mechanism. It implements the community Reconciliation Service API
  (the same protocol family OpenRefine uses). **Verified live against the
  real host on 2026-08-14** (see `.squad/decisions.md` / the corresponding
  inbox note for the full defect writeup): the request's `queries` URL
  parameter value is the query-identifier map directly — `{ "<queryId>":
  { "query": "<text>", "limit": <n> } }` — with **no outer `"queries"`
  key**; the response is the query-identifier map at the **top level** —
  `{ "<queryId>": { "result": [ { "id", "name", "type", "score", "match" },
  ... ] } }` — with **no `"results"` wrapper**. Additionally, each result's
  `"id"` is Nomisma's short local id (e.g. `"roma"`), not a full URI (the
  durable concept URI is built as `http://nomisma.org/id/<id>`, matching
  the convention documented for Nomisma's own `getLabel`/`getRdf` REST
  helpers), and `"match"` is encoded as a JSON **string** (`"true"`/
  `"false"`), not a JSON boolean. An earlier draft of this document
  described a `{ "queries": { ... } }`/`{ "results": { ... } }` double
  envelope; that description was unverified and incorrect, and has been
  corrected here and in `src/api/services/nomisma_client.go` /
  `nomisma_client_test.go`.
- **Rationale**: This is Nomisma's own documented, stable, unauthenticated
  service (`nomisma.org/documentation/apis/`), matches the spec's assumption
  that "Nomisma's reconciliation service is the intended on-demand lookup
  mechanism," and requires no API key/credential to manage or protect —
  unlike the Numista boundary (ADR 0007), there is no shared quota or secret
  to guard, which materially simplifies the client.
- **Alternatives considered**:
  - *SPARQL endpoint browsing* — rejected outright by FR-010 and the PRD
    non-goal; would let an admin browse the full dataset rather than perform
    a scoped match.
  - *`getLabel`/`getRdf`/`getMints` REST helpers* — useful only after a URI is
    already known (e.g. to re-verify a stale link later); not a search
    mechanism, so not part of the MVP search/confirm flow. Documented here
    as a plausible non-blocking future affordance for User Story 2's
    "Nomisma has since split/merged" edge case, but out of scope now.
  - *Scraping/browsing nomisma.org HTML pages* — rejected: undocumented,
    fragile, and outside the CC BY-licensed structured API surface.
- **Exact query parameter framing (GET query-string vs. POST body, and the
  precise field name for candidate type/context) is an implementation-time
  detail to pin down against the live service and captured in
  `nomisma_client_test.go` httptest fixtures** — this does not change any
  product behavior described in the spec (search → candidates → explicit
  confirm), so it is not a blocking unknown for planning. If the live
  behavior meaningfully differs from the documented JSON shape above (e.g. a
  different envelope), implementation MUST stop and flag it per the
  contradiction-handling instruction in this plan's constraints, rather than
  silently reinterpreting FR-001/FR-002.

## 2. PRD non-goal reconciliation

- **PRD §4 states**: "Index or compete with numismatic reference catalogs
  (e.g., Numista, RIC, ACSearch). The app links to them but does not
  replicate their data."
- **Reconciliation**: This feature stores, per confirmed link, only: a
  durable concept URI, the matched label captured at confirmation time, and
  a confirmation timestamp — never a copy of Nomisma's RDF graph, partner
  corpora (types/hoards/specimens), or a locally browsable mirror of Nomisma
  concepts. Every subsequent search is a live on-demand call; there is no
  scheduled sync, cache warm, or dataset export. This is the same
  "link to it, don't replicate it" shape already applied to Numista
  (`CoinReference` stores catalog/number/URI, not Numista's dataset) and to
  `214-structured-numismatic-catalog-references` (structured pointers, not
  copies). **No contradiction found**; the feature is additive metadata
  consistent with Constitution Principle IV (proportional persistence) and
  the PRD non-goal.
- **Constitution citations**: Principle IV (Simple Complete Changes — persist
  only what FR-003 requires, not a broader mirror); §17 Quality Gate
  (workflow-contract check requires proving existing mint/coin workflows are
  unaffected); §21 Definition of Done (ADR required if a material design
  choice is made — the client/cache boundary decision below is the
  candidate).

## 3. Client/service boundary shape

- **Decision**: A single `NomismaClient` interface (`Search(ctx, query,
  limit) ([]NomismaCandidate, error)`), implemented over `net/http` with a
  bounded timeout (mirrors `geocodeRequestTimeout = 8 * time.Second`), a
  fixed `Accept: application/json` request, and a bounded response body read
  (mirrors `numistaResponseLimit` in `numista_client.go`). Errors are typed
  (`NomismaErrorKind`: `unavailable`, `no_match`, `invalid_response`,
  `invalid_request`, `cancelled`) rather than raw Go errors bubbling to the
  handler, so FR-007/FR-008/FR-009's distinct outcomes (outage vs.
  zero-result vs. ambiguous-but-present candidates) are representable without
  string-matching errors.
- **Rationale**: Matches Constitution Principle I/II (layered, single HTTP
  boundary owned by a service) and mirrors two existing, reviewed patterns in
  this codebase (`GeocodeService` for simplicity, `NumistaClient`/
  `NumistaError` for the typed-outcome shape) rather than inventing a third
  convention.
- **Alternatives considered**:
  - *Reuse `NumistaClient` directly or add Nomisma as a mode on it* —
    rejected: different wire protocol (reconciliation JSON vs. Numista's
    catalog API), different auth model (none vs. API key), and mixing them
    would violate Principle II's per-provider boundary ownership established
    by ADR 0007.
  - *Call Nomisma from the Vue client directly* — rejected by Constitution
    Principle V and this plan's constraint that API exposure stay
    same-origin; also would leak search intent/timing directly to a
    third-party host from the browser.
  - *Full Numista-style scorer/telemetry stack* — rejected as disproportionate
    (Principle IV): Nomisma reconciliation already returns a `score`/`match`
    ranking; there is no need for a second deterministic scorer, and the
    §17-required telemetry need is satisfied by reusing the existing
    generic request logging, not a bespoke 500-event ring buffer sized for a
    much higher-volume Numista workflow.

## 4. Caching

- **Decision**: One bounded in-memory TTL cache for **search responses
  only**, keyed by a normalized, lower-cased, whitespace-collapsed query
  string (SHA-256 digest, mirroring `numista_cache.go`'s key-identity
  convention so raw queries never appear in logs/metrics). Short fixed TTL
  (proposed: 10 minutes, well under Numista's 24h search TTL, since Nomisma
  concepts change far less frequently but admin re-typing during a single
  curation session is the actual scenario being optimized). Cache size bound:
  e.g. 200 entries (LRU-by-expiry eviction, same shape as
  `numistaSearchCacheEntry`). A **zero-result search is cached as a negative
  entry** for the same short TTL (prevents hammering Nomisma while an admin
  iterates on a query), but a **provider failure (`unavailable`,
  `invalid_response`) is never cached** — matches ADR 0007's "provider
  failures are not [cacheable]" rule, so a transient outage doesn't get
  "stuck" as unavailable for the TTL window.
- **Rationale**: Directly satisfies FR-011 ("bound and time-limit any
  caching... cached results MUST NOT replace the durably persisted confirmed
  link") and the spec's Assumptions section definition of "bounded
  short-lived caching." Singleflight/request-coalescing (present in
  `numista_cache.go`) is explicitly **not** ported: Nomisma search is a
  single admin, one interactive click at a time, with no concurrent-request
  fan-in scenario analogous to Numista's broad-lookup + enrichment
  concurrency — adding coalescing here would be complexity without a
  matching problem (Principle IV).
- **Alternatives considered**:
  - *No cache at all* — rejected: an admin iterating on a query (typo fixes,
    trying aliases) would otherwise re-hit Nomisma on every keystroke-driven
    search submission; a short TTL cache is a proportionate, low-risk
    mitigation.
  - *Persist full search result sets to the database* — rejected by FR-010/
    FR-011 and the PRD non-goal; this would start to look like local dataset
    storage rather than an ephemeral lookup aid.
  - *Cache negative results indefinitely* — rejected; would contradict
    FR-008's "leave the mint location unlinked without error" being retryable
    at any time (e.g. after Nomisma adds the mint later).

## 5. Persistence shape

- **Decision**: Extend the existing `MintLocation` GORM model with three
  nullable/optional columns rather than a new table:
  - `NomismaURI *string` — durable concept URI (e.g.
    `http://nomisma.org/id/roma`)
  - `NomismaLabel string` — matched label captured at confirmation time
    (empty means unlinked, mirrors the nullable-URI-is-source-of-truth
    pattern already used for `Coin.MintLocationID`)
  - `NomismaLinkedAt *time.Time` — confirmation timestamp
  Added via the existing additive `AutoMigrate` call in `database.go`
  (`MintLocation{}` is already migrated there); SQLite adds new nullable
  columns without a destructive rewrite, consistent with the existing
  comment in `database.go` about additive `AutoMigrate` safety.
- **Rationale**: FR-003 requires only a URI, matched label, and
  confirmation timestamp — a fourth field/table would be unjustified scope
  per Principle IV. This mirrors how `338-mint-location-integration` added
  `UserID *uint` to the same struct rather than creating a companion
  ownership table.
- **Alternatives considered**:
  - *New `mint_location_nomisma_links` child table* — rejected: a
    MintLocation has at most one active confirmed link at a time (User Story
    2 replaces, not appends); a 1:1 optional relationship is exactly what
    nullable columns model, and a child table would add a join for every read
    path (Mint Map, coin detail) that shows attribution.
  - *Store the full Nomisma JSON response* — rejected by FR-010's "no
    dataset ingestion" framing; only display-necessary provenance is stored.

## 6. Collision/duplicate URI rule

- **Decision**: **Allowed** — no uniqueness constraint across
  `NomismaURI` values. Multiple global `MintLocation` rows may reference the
  same Nomisma concept URI.
- **Rationale**: Directly specified by the spec's Edge Cases section ("What
  happens if two different global mint locations are both linked to the same
  Nomisma concept? → Allowed."). The current domain model already tolerates
  one Nomisma authority mint mapping to more than one local `MintLocation`
  row (e.g. a corrected/duplicate entry both legitimately citing "Rome");
  forcing a unique constraint would block a scenario the spec explicitly
  keeps open and would require new admin-facing merge tooling that is out of
  scope for this MVP.
- **Note**: This differs from the existing `idx_mint_location_owner_name`
  unique index on `(user_id, normalized_name)` — that index protects against
  *duplicate mint locations*, a separate concern from *shared authority
  references*, and is left untouched.

## 7. Attribution rendering surfaces

- **Decision**: A single shared `NomismaAttribution.vue` component
  (props: `uri`, `label`), rendered wherever a linked global mint's
  name/details already surface without a new page:
  - `MintCoinDrawer.vue` (Mint Map popup/drawer) — the group's `mint` object
    already carries the full `MintReference` shape end-to-end
    (`src/web/src/utils/mintMap.ts`), so the new optional fields flow through
    unchanged.
  - `AdminCoinPropertiesSection.vue`'s global mint list (admin's own
    curation view, so the admin also sees the confirmed state after linking).
- **Rationale**: Satisfies FR-004 without introducing new dedicated pages,
  per the spec's Assumptions section. Coin Detail does not currently render
  the linked `MintLocation` object (only the coin's free-text `mint` and
  `mintLocationId`), so no change is needed there for MVP; if a future coin
  detail redesign surfaces the full mint object, it should render the same
  shared component rather than duplicating the attribution string.
- **Alternatives considered**: Inline duplicate markup in each surface —
  rejected by Principle IV/VI (a single shared component keeps the exact
  attribution string and link targets consistent everywhere it appears).

## 8. Authorization boundary for private mints

- **Decision**: Reuse the existing global/private split already enforced by
  `MintLocationRepository`/`MintLocationService` (`UserID == nil` means
  global). All new Nomisma routes operate only through `FindByID` +
  an explicit `existing.UserID != nil → ErrMintLocationNotFound` check,
  identical to the existing `UpdateGlobal`/`DeleteGlobal` guard pattern in
  `mint_location_service.go`. No new authorization primitive is introduced.
- **Rationale**: FR-006/User Story 4 require zero code path from a private
  mint to Nomisma. Reusing the proven guard (already tested in
  `mint_location_service_test.go` for update/delete) is simpler and more
  auditable than adding a new capability check.

## 9. Phase 2 seam: OCRE/RPC catalog authorities (deferred, not designed here)

**This section is informational only.** Per additional user direction, this
plan's Phase 1 scope (Nomisma global-mint linking) is not expanded to cover
OCRE (`numismatics.org/ocre/apis`) or RPC Online
(`rpc.ashmus.ox.ac.uk/introduction`), and the two must not be designed as
interchangeable with Nomisma.

- **What was checked**: `numismatics.org/ocre/apis` is live and documents its
  own OpenRefine-compatible reconciliation service at
  `http://numismatics.org/ocre/apis/reconcile` — same *protocol family*
  (Numishare/OpenRefine reconciliation) as Nomisma's, but explicitly a
  **different corpus and entity type**: OCRE's own page states it
  reconciles "coin type corpora" (specific coin types/subtypes — the
  catalog-entry level `214-structured-numismatic-catalog-references`
  already models as `CoinReference`), not the controlled-vocabulary mint/
  person/material *concepts* Nomisma resolves. OCRE additionally supports
  extra query properties (ruler, mint, denomination) to refine a coin-type
  match — a materially richer, and differently-shaped, query contract than
  Nomisma's plain-text concept search used in Phase 1.
  `rpc.ashmus.ox.ac.uk/introduction` returned HTTP 403 to an automated fetch
  during this planning pass — its API/license terms are **unverified** and
  MUST be independently confirmed by a human or a future research task
  before any Phase 2 implementation, not assumed from OCRE's or Nomisma's
  terms.
- **Why this is not "just add another provider" to Phase 1**:
  - **Different target entity**: Nomisma links attach to `MintLocation`
    (a place); OCRE/RPC links would attach to `CoinReference` (a specific
    coin's catalog citation) — a different table, different ownership
    model (`CoinReference` is per-coin, not global/admin-curated), and a
    different UI surface (Coin Detail's reference editor, not the admin
    mint panel).
  - **Different licensing**: Confirmed Phase 1 attribution is CC BY 4.0
    (Nomisma). OCRE is an American Numismatic Society (Numishare) property;
    RPC Online is University of Oxford/Ashmolean. Neither institution's
    license has been confirmed for this repo — Phase 2 must not reuse the
    `NomismaAttribution.vue` copy or assume CC BY 4.0 applies.
  - **Different API depth**: OCRE's reconciliation supports multi-property
    refinement (ruler/mint/denomination) that a Phase 2 client would need to
    model explicitly; RPC's contract shape is entirely unverified here.
- **Existing Aurearia behavior already present** (unaffected by this plan):
  `src/agent/app/tools/numismatic_authority.py`'s `lookup_authority_uri()`
  performs **string-templated URL guessing only** — given a
  catalog/volume/number the Python agent (`src/agent/`) has already parsed
  from a coin description, it builds a plausible
  `https://numismatics.org/ocre/id/{ric-number}` or
  `https://rpc.ashmus.ox.ac.uk/id/{rpc-number}` URL (or a `results?q=`/
  `search?q=` fallback search link when the number doesn't already look like
  a canonical ID). It performs **no live API call, no reconciliation query,
  no candidate list, no admin confirmation step, and no attribution
  display** — it is a best-effort convenience for populating
  `CoinReference.URI` during AI-assisted coin intake
  (`214-structured-numismatic-catalog-references` FR-level: "uri is
  populated" when a known-authority catalog match succeeds), not an
  authority-linking feature in the sense Phase 1 builds for Nomisma. A
  future Phase 2 would decide whether to (a) upgrade this existing
  heuristic in place, or (b) introduce a new typed Go-side client — that
  decision is explicitly deferred, not made here.
- **Required before any Phase 2 implementation** (a future spec's Phase 0,
  not this plan's): confirm OCRE's and RPC's licenses independently; read
  and cite each service's actual reconciliation contract (OCRE's
  multi-property query shape; RPC's — currently unknown — contract);
  reconcile against PRD §4's non-goal the same way this plan does for
  Nomisma in §2 above; run through `/speckit.specify` →`/speckit.clarify` →
  `/speckit.plan` as its own feature rather than an amendment to 343.