# Feature Specification: OCRE Automated Deep Analysis Provider

**Feature Branch**: `345-ocre-deep-analysis-provider`
**Created**: 2026-08-15
**Status**: Implemented
**Input**: User description: "Enable OCRE (Online Coins of the Roman Empire, American Numismatic Society) as an automated Roman Imperial coin-type authority provider inside the existing Deep Analysis pipeline. Automated access uses fixed-template, parameterized Nomisma SPARQL against `https://nomisma.org/query` (no OCRE frontend scraping, no arbitrary user-supplied SPARQL). Persist bounded OCRE-derived metadata/evidence with canonical OCRE links and explicit ODbL 1.0 / American Numismatic Society attribution and share-alike posture. Exclude OCRE images and any bulk corpus ingestion. RPC remains out of scope. Opens gate G-OCRE / deferred task T155."

## Context & Background

Feature 344 shipped the Deep Analysis pipeline with OCRE staged as a
`not_automated` typed stub behind validation gate **G-OCRE** (deferred task
**T155**). The provider node, citation host allowlist (`numismatics.org`),
provider catalog entry (`{Provider: "ocre", Automatable: false, Reason:
"pending_license_validation"}`), the admin setting
`SettingDeepIdentificationOCREEnabled` (default `false`), and the internal-tool
boundary all already exist and were designed to be switched on with **no
contract respecification** — only a provider adapter plus a license/attribution
review recorded in an ADR (research §2.4). This feature opens G-OCRE and
implements OCRE as a first-class automated provider within those existing
bounds. It is deliberately narrow: it does not re-open Deep Analysis job
orchestration, SSE, cancellation, confirm-gated writes, or the other providers.

**License posture (authorized product decision)**: OCRE type data is licensed
**ODC-ODbL 1.0** with share-alike obligations on derivative *databases* and
attribution to the **American Numismatic Society**. This is a distinct license
from Nomisma (CC BY 4.0), Numista (proprietary attribution terms), and RPC
(CC BY-NC-SA). OCRE attribution and license links MUST be kept separate from
those providers everywhere they render.

**API-stability caveat (do not overclaim)**: OCRE's own APIs
(`numismatics.org/ocre/apis`) were observed returning HTTP 500 with an
intermittently unavailable front end during Feature 344 research. The
**supported, validated implementation route is the Nomisma triplestore SPARQL
endpoint** (`https://nomisma.org/query`, OCRE URI prefix
`http://numismatics.org/ocre/id/`), which is the stable path and the route the
G-OCRE gate validates. This feature makes no claim about the stability of any
OCRE-hosted API.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - OCRE contributes Roman Imperial coin-type candidates (Priority: P1)

A collector runs Deep Analysis on an unidentified Roman Imperial coin. From the
already-normalized image/quick/provider evidence (ruler/issuer, denomination,
mint, date/period, material, inscriptions/legends, and any known OCRE
identifier), the router recognizes the evidence as Roman Imperial and selects
OCRE as one of the automated providers. OCRE returns one or more typed candidate
coin-type claims, each with a canonical `numismatics.org/ocre/id/...` citation,
a human-readable label / RIC-style type ID, the fields it matched on, and a
confidence value, which appear in the synthesized report alongside other
providers' evidence.

**Why this priority**: This is the core value — turning existing normalized
evidence into authoritative Roman Imperial coin-type candidates. Without it the
feature delivers nothing.

**Independent Test**: With the OCRE flag enabled, start Deep Analysis on a coin
whose normalized evidence is clearly Roman Imperial (e.g., a known emperor +
denomination + mint). Verify OCRE is selected, one or more bounded, ranked,
deterministic candidate type claims appear with canonical
`numismatics.org/ocre/id/...` citations and matched-field explanations, and that
no OCRE candidate silently overrides image or other-provider evidence.

**Acceptance Scenarios**:

1. **Given** the OCRE flag is enabled and normalized evidence identifies a Roman
   Imperial ruler + denomination + mint, **When** the router runs, **Then** OCRE
   is selected and queried via bound Nomisma SPARQL.
2. **Given** OCRE returns matching coin types, **When** results are surfaced,
   **Then** each candidate carries a canonical `numismatics.org/ocre/id/...`
   citation, a label/type ID, the matched fields, a confidence value, and an
   explanation, as an application-owned typed claim.
3. **Given** normalized evidence already contains a known OCRE identifier,
   **When** OCRE runs, **Then** that identifier is used to resolve/confirm the
   canonical type rather than being ignored.
4. **Given** OCRE returns several plausible types, **When** candidates are
   ranked, **Then** ranking is deterministic and bounded by the result cap, and
   remaining ambiguity is preserved and visible rather than collapsed to a
   single silent answer.
5. **Given** the coin's normalized evidence is clearly NOT Roman Imperial (e.g.,
   a Greek or Islamic coin), **When** the router runs, **Then** OCRE is not
   selected absent an explicit override.

---

### User Story 2 - Transparent OCRE attribution, license, and evidence (Priority: P1)

Wherever OCRE-derived evidence appears — the Deep Analysis report, the draft
proposal, the provider status list, and any export/share of that result — the
collector sees explicit OCRE attribution and license text with working links to
the canonical coin-type record and to the ODbL 1.0 license, kept visually and
textually distinct from Nomisma/Numista attribution.

**Why this priority**: ODbL 1.0 share-alike and ANS attribution are legal
obligations of using OCRE data; omitting or conflating them is a compliance
failure, so attribution must ship with the first automated result.

**Independent Test**: Produce a Deep Analysis result that includes at least one
OCRE claim, then inspect the report, the proposal, the provider status panel,
and an export of that result. Verify each surface renders the exact attribution
string with a link to the canonical type and a link to the ODbL 1.0 license, and
that OCRE attribution is not merged with any other provider's attribution.

**Acceptance Scenarios**:

1. **Given** a completed/partial result containing an OCRE claim, **When** the
   report renders, **Then** it displays: "Coin type data: Online Coins of the
   Roman Empire (OCRE), American Numismatic Society — ODbL 1.0." with a link to
   the canonical OCRE type and a link to the ODbL 1.0 license.
2. **Given** the same result, **When** the draft proposal and provider-status
   surfaces render, **Then** the same OCRE attribution and license appear there.
3. **Given** the result is exported or shared, **When** OCRE evidence is present
   in that output, **Then** the OCRE attribution and license travel with it.
4. **Given** a result that contains both OCRE and Nomisma/Numista evidence,
   **When** attribution renders, **Then** the OCRE ODbL 1.0 / ANS attribution is
   distinct from the Nomisma/Numista attribution strings and links.
5. **Given** a result that contains no OCRE evidence, **When** surfaces render,
   **Then** no OCRE attribution is shown.

---

### User Story 3 - OCRE failures and flag-off never break Deep Analysis (Priority: P1)

Deep Analysis must remain robust: when OCRE times out, errors, returns nothing,
or returns malformed data, the overall job still completes as a typed
partial-provider outcome; and when the OCRE flag is disabled, OCRE is reported
as `not_automated` and makes zero Nomisma/SPARQL calls.

**Why this priority**: OCRE relies on community infrastructure (no SLA) and is
disabled by default; a single provider's failure or a disabled flag must never
degrade the overall Deep Analysis product.

**Independent Test**: (a) With the flag enabled, force OCRE to time out / return
HTTP 500 / return malformed bindings and verify the job still reaches a terminal
completed/partial state with OCRE listed as `timed_out`/`failed`/`no_match` and
its status explained. (b) With the flag disabled, run a Roman Imperial coin and
verify OCRE status is `not_automated`, no SPARQL call is made, and the job
completes normally.

**Acceptance Scenarios**:

1. **Given** OCRE times out or returns HTTP 500, **When** the job finishes,
   **Then** OCRE is a typed `timed_out`/`failed` partial outcome and the overall
   job still reaches completed/partial using the remaining evidence.
2. **Given** OCRE returns no matching type, **When** the job finishes, **Then**
   OCRE status is `no_match` and does not fail the job.
3. **Given** OCRE returns malformed or out-of-allowlist bindings/citations,
   **When** results are validated, **Then** invalid claims are dropped, OCRE is
   marked `failed`/`invalid_response`, and no non-canonical citation is surfaced.
4. **Given** the OCRE flag is disabled, **When** any Deep Analysis job runs,
   **Then** OCRE status is `not_automated`, zero OCRE/SPARQL calls occur, and no
   OCRE evidence or attribution appears.
5. **Given** OCRE fails, **When** telemetry records the outcome, **Then** it
   captures status/timing/counts but never notes or full legend text.

---

### User Story 4 - Admin enablement and provider health visibility (Priority: P2)

An administrator can enable or disable OCRE automation through the existing
admin setting, and can see OCRE's health/status (enabled/disabled, last outcome
class, whether the gate is validated) without exposing user content.

**Why this priority**: Enablement is admin-controlled and default-off; operators
need a safe switch and basic health visibility, but the collector-facing value
(US1–US3) must land first.

**Independent Test**: As an admin, toggle `SettingDeepIdentificationOCREEnabled`
and confirm the provider catalog reflects the change on the next job (automated
when true, `not_automated` when false) and that a health/status view shows the
current OCRE enablement and last outcome class.

**Acceptance Scenarios**:

1. **Given** an admin, **When** they set the OCRE flag to enabled, **Then**
   subsequent Deep Analysis jobs treat OCRE as an automatable provider subject to
   provider bounds.
2. **Given** an admin, **When** they set the OCRE flag to disabled, **Then**
   subsequent jobs report OCRE as `not_automated` with zero calls.
3. **Given** a non-admin user, **When** they attempt to change the OCRE flag,
   **Then** the change is refused (admin-only), consistent with existing settings
   authorization.
4. **Given** OCRE is enabled, **When** an operator views provider health/status,
   **Then** OCRE's enablement and most-recent outcome class are visible without
   any user notes or legend content.

---

### User Story 5 - Explicit provider override for OCRE (Priority: P2)

A collector can explicitly override provider selection to force OCRE to run (even
when the router would not have chosen it) or to exclude it, within the existing
override mechanism and provider bounds.

**Why this priority**: Power users sometimes know a coin is Roman Imperial when
the automated evidence is weak; honoring an explicit override improves control
without weakening the closed candidate list.

**Independent Test**: With the flag enabled, submit a job with an explicit
override selecting OCRE and confirm OCRE runs even if the router would have
skipped it; submit another excluding OCRE and confirm it does not run.

**Acceptance Scenarios**:

1. **Given** the OCRE flag is enabled, **When** the user override explicitly
   includes OCRE, **Then** OCRE runs regardless of router reasoning, still
   bounded by provider limits and the closed catalog.
2. **Given** the OCRE flag is enabled, **When** the user override omits OCRE,
   **Then** OCRE does not run for that job.
3. **Given** the OCRE flag is disabled, **When** a user override names OCRE,
   **Then** OCRE remains `not_automated` and makes zero calls (flag wins over
   override).

---

### Edge Cases

- **SPARQL injection strings**: normalized inputs containing SPARQL syntax,
  quotes, backslashes, angle brackets, or newlines MUST be safely escaped/bound
  in the fixed template; they can never alter query structure or inject clauses.
- **BCE/CE dates**: date/period evidence spanning BCE↔CE (negative years, era
  boundaries, regnal ranges) MUST map correctly to OCRE date semantics without
  sign/era errors.
- **Ambiguous rulers/mints**: a ruler or mint name matching multiple OCRE
  authorities yields multiple ranked candidates with ambiguity preserved, never
  a silent single pick.
- **Unsupported evidence**: evidence lacking any Roman Imperial signal (or with
  only fields OCRE cannot bind) results in OCRE being skipped or returning
  `no_match`, not an error.
- **Malformed bindings**: SPARQL responses with missing/extra/wrongly-typed
  bindings are treated as `invalid_response`; partial-but-valid rows are kept up
  to the cap, invalid rows dropped.
- **Duplicate types**: the same OCRE type URI returned more than once is
  de-duplicated before ranking and the result cap is applied to distinct types.
- **Timeout / HTTP 500**: upstream timeout or 5xx becomes a typed
  `timed_out`/`failed` partial outcome within the per-job budget; never a job
  failure and never an unbounded retry storm.
- **Result cap**: when OCRE would return more matches than the cap, results are
  deterministically truncated to the cap with ambiguity noted.
- **Cache keys**: identical bound-parameter sets reuse a cached result within the
  TTL; a change in any bound parameter (ruler, denomination, mint, date,
  material, legend, or OCRE id) yields a distinct cache key and no stale reuse
  across different inputs or across a flag change.
- **Flag changes**: toggling the flag mid-lifecycle affects only subsequent jobs;
  in-flight jobs keep their decided catalog; a disabled flag always yields zero
  calls regardless of override.
- **Attribution/citation host**: any candidate whose citation host is not
  `numismatics.org` (the OCRE canonical host) is rejected by the allowlist and
  never surfaced.
- **Known-identifier confirmation**: a supplied OCRE identifier that does not
  resolve to a valid canonical type is reported as unresolved rather than
  fabricated.

## Requirements *(mandatory)*

### Functional Requirements

**Provider role, routing, and inputs**

- **FR-001**: System MUST treat OCRE as a Roman Imperial coin-type authority
  provider within the existing Deep Analysis provider pipeline, as an
  application-owned typed provider node — not a new pipeline.
- **FR-002**: System MUST derive OCRE query inputs solely from already-normalized
  image/quick/provider evidence: ruler/issuer, denomination, mint, date/period,
  material, inscriptions/legends, and an optional known OCRE identifier. It MUST
  NOT accept arbitrary user-supplied query text or SPARQL.
- **FR-003**: System MUST select OCRE when the normalized evidence is relevant to
  Roman Imperial coinage, or when the user explicitly overrides provider
  selection to include OCRE, in both cases subject to the enabled flag and the
  existing provider bounds (max providers, per-job budget).
- **FR-004**: System MUST NOT select OCRE when the OCRE flag is disabled, even
  under an explicit user override (flag takes precedence over override).

**Automated access route and safety**

- **FR-005**: System MUST perform automated OCRE access exclusively via a
  fixed-template, parameterized/bound Nomisma SPARQL query against
  `https://nomisma.org/query`. It MUST NOT scrape any OCRE front end and MUST NOT
  execute arbitrary or user-supplied SPARQL.
- **FR-006**: System MUST safely escape/bind every input value into the SPARQL
  template so that injection strings cannot alter query structure.
- **FR-007**: System MUST route the OCRE upstream call through the existing
  Go-owned provider HTTP / internal-tool / job-token / budget boundary. The
  Python agent MUST remain stateless and call only the internal OCRE tool; it
  MUST NOT introduce a duplicate direct Python HTTP client to Nomisma/OCRE.
- **FR-008**: System MUST enforce, for OCRE: a per-job call budget, a request
  timeout, a response-size limit, and a deterministic result cap on the number
  of distinct candidate types surfaced.
- **FR-009**: System SHOULD add bounded TTL caching for OCRE results, keyed on the
  full set of bound query parameters, consistent with existing provider cache
  patterns (e.g., Nomisma cache) where those patterns justify it. Negative
  (`no_match`) results MAY be cached; transient failures MUST NOT be cached.

**Results, ranking, and provenance**

- **FR-010**: System MUST return OCRE results as application-owned typed
  candidate claims, each containing: a canonical `numismatics.org/ocre/id/...`
  citation, a human-readable label / type ID, the fields it matched on, a
  confidence value, an explanation, a source status, and OCRE license/attribution
  metadata.
- **FR-011**: System MUST reject and drop any OCRE candidate whose citation host
  is not the OCRE canonical host (`numismatics.org`), consistent with the
  existing per-provider citation allowlist.
- **FR-012**: System MUST rank OCRE candidates deterministically and bound them to
  the result cap; identical inputs MUST produce identical ordering.
- **FR-013**: System MUST preserve and surface residual ambiguity (multiple
  plausible types) rather than silently collapsing to one, and no OCRE candidate
  MAY silently override image evidence or another provider's claim.
- **FR-014**: System MUST de-duplicate repeated OCRE type URIs before ranking and
  cap on distinct types.

**Failure isolation and flag-off behavior**

- **FR-015**: System MUST treat OCRE provider failures, no-match, timeouts, and
  invalid/malformed responses as typed partial-provider outcomes that never fail
  the overall Deep Analysis job.
- **FR-016**: System MUST keep the existing `SettingDeepIdentificationOCREEnabled`
  setting defaulting to `false`; when disabled, OCRE status MUST remain
  `not_automated` and zero OCRE/SPARQL calls MUST occur.
- **FR-017**: System MUST restrict changing the OCRE enablement setting to
  administrators, consistent with existing settings authorization.

**Persistence and attribution**

- **FR-018**: System MUST persist only the bounded OCRE-derived
  metadata/evidence needed for the job, report, and proposal (canonical type
  link/id, label, matched fields, confidence, status, license/attribution). It
  MUST NOT build or store a local OCRE reference database or corpus.
- **FR-019**: System MUST render OCRE attribution wherever OCRE evidence appears —
  Deep Analysis report, draft proposal, provider status, and exports — using the
  exact text: "Coin type data: Online Coins of the Roman Empire (OCRE), American
  Numismatic Society — ODbL 1.0.", with links to the canonical OCRE type and to
  the ODbL 1.0 license.
- **FR-020**: System MUST keep the OCRE license/attribution distinct from
  Nomisma, Numista, and RPC attribution wherever multiple providers' evidence
  co-occurs, reflecting OCRE's ODbL 1.0 share-alike posture.

**Exclusions and telemetry**

- **FR-021**: System MUST NOT use, download, cache, or display any OCRE images,
  and MUST NOT perform bulk/background ingestion of the OCRE corpus.
- **FR-022**: System MUST record OCRE telemetry (status class, timing, call/result
  counts, cache hit/miss, budget usage) without persisting user notes or full
  legend text, consistent with existing SSE/telemetry privacy rules.
- **FR-023**: System MUST expose OCRE health/status visibility (enablement,
  gate-validated state, most-recent outcome class) without exposing user content.

**Non-regression**

- **FR-024**: System MUST preserve existing behavior of the quick Identify
  lookup, the other Deep Analysis providers (Numista, Nomisma, NGC, RPC),
  confirm-gated writes, citation allowlists, SSE privacy, and feature flags. No
  OCRE change may cause automatic coin-record writes.

### Key Entities *(include if feature involves data)*

- **OCRE candidate claim**: an application-owned typed evidence row for one Roman
  Imperial coin type — canonical `numismatics.org/ocre/id/...` citation,
  label/type ID, matched fields, confidence, explanation, source status, and
  OCRE license/attribution metadata.
- **OCRE provider evidence (run summary)**: per-job OCRE outcome — status
  (`contributed`/`no_match`/`failed`/`timed_out`/`not_automated`/`invalid_response`),
  automatable flag, call count, attribution, and its list of candidate claims.
- **OCRE attribution/license descriptor**: the fixed OCRE ODbL 1.0 / ANS
  attribution string plus canonical-type and license link targets, kept distinct
  from other providers' attribution.
- **Bound SPARQL query parameters**: the normalized, escaped/bound input set
  (ruler/issuer, denomination, mint, date/period, material, legend, optional OCRE
  id) that both parameterizes the fixed template and forms the cache key.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: With OCRE enabled, 100% of surfaced OCRE candidates carry a
  canonical `numismatics.org/ocre/id/...` citation; zero surfaced candidates use
  a non-`numismatics.org` citation host.
- **SC-002**: 100% of Deep Analysis surfaces that include OCRE evidence (report,
  proposal, provider status, export) display the exact OCRE ODbL 1.0 / ANS
  attribution with working links to the canonical type and the license, distinct
  from any other provider's attribution.
- **SC-003**: 100% of OCRE provider failures, timeouts, no-match, and malformed
  responses result in a completed/partial job (zero jobs fail because of OCRE).
- **SC-004**: With the OCRE flag disabled, OCRE makes zero outbound SPARQL/OCRE
  calls and reports `not_automated` in 100% of jobs.
- **SC-005**: Repeated Deep Analysis runs on identical normalized evidence
  produce identical OCRE candidate ordering and identical candidate set (bounded
  by the result cap) — deterministic ranking verified across ≥2 runs.
- **SC-006**: No single OCRE run exceeds the configured per-job call budget,
  request timeout, response-size limit, or result cap.
- **SC-007**: Zero coin records are created or modified by an OCRE-enabled Deep
  Analysis run absent an explicit confirm-gated write.
- **SC-008**: The quick Identify lookup and the other providers show no
  behavioral change attributable to this feature (existing regression gates for
  those paths continue to pass unchanged).
- **SC-009**: No OCRE image is fetched, stored, or displayed, and no local OCRE
  reference database/corpus is created (verified by inspection of persisted
  data and outbound calls).
- **SC-010**: SPARQL-injection input strings never alter query structure —
  verified by offline fixtures asserting the bound query is structurally
  identical regardless of adversarial input values.

## Assumptions

- The existing Feature 344 Deep Analysis pipeline, provider fan-out, internal
  tool boundary, job-token/budget mechanics, SSE privacy, and the OCRE
  `not_automated` stub, catalog entry, citation allowlist (`numismatics.org`),
  and `SettingDeepIdentificationOCREEnabled` setting are all present on `beta`
  and are reused rather than re-designed.
- The **Nomisma triplestore SPARQL endpoint** (`https://nomisma.org/query`, OCRE
  URI prefix `http://numismatics.org/ocre/id/`) is the supported, gate-validated
  implementation route. No claim is made about the stability of OCRE-hosted APIs
  (`numismatics.org/ocre/apis`), which were observed failing during F344
  research; if the SPARQL route is unavailable, OCRE degrades to a typed
  partial-provider outcome per FR-015.
- OCRE data is licensed **ODC-ODbL 1.0** with share-alike on derivative databases
  and attribution to the **American Numismatic Society**; this feature persists
  only bounded per-job evidence (not a database) and always renders attribution,
  which is the posture authorized for this feature.
- The app is self-hosted and single-owner; OCRE enablement is an admin-controlled
  operational decision and remains default-off.
- Nomisma has no documented rate limit and is community infrastructure with no
  SLA, so the OCRE node self-throttles within the per-job budget.
- Opening gate **G-OCRE** / deferred task **T155** is authorized; the
  accompanying license/attribution review is expected to be recorded in an ADR
  (referenced as ADR-0010 in F344 research) as part of implementation planning.
- Deterministic ranking uses stable, explainable ordering (e.g., match
  specificity then confidence then canonical URI) with no LLM freedom to invent
  or reorder citations.

## Non-Goals (Out of Scope)

- **RPC Online** automation or any RPC provider work (remains blocked / manual
  reference only; separate license and access constraints).
- Any use, download, caching, or display of **OCRE images**.
- **Bulk or background ingestion** of the OCRE corpus, or building/storing a
  local OCRE reference database/mirror.
- **OCRE front-end scraping** or use of OCRE-hosted APIs as the automated route;
  Nomisma SPARQL is the sole automated path.
- **Arbitrary user-supplied SPARQL** or free-text OCRE queries.
- A **second, direct Python HTTP client** to Nomisma/OCRE; the Go-owned
  internal-tool boundary is reused.
- Changes to quick Identify, the other providers, Deep Analysis job
  orchestration/SSE/cancellation, or confirm-gated write flows beyond adding the
  OCRE node and its attribution surfaces.
- Enabling OCRE by default; it stays default-off and admin-gated.

## Migration & Configuration Impact

- **Setting**: `SettingDeepIdentificationOCREEnabled` already exists and remains
  default `false`. No new setting is required to disable OCRE; enabling is an
  admin action. New bounded configuration (per-job OCRE call budget, timeout,
  response-size limit, result cap, cache TTL) SHOULD follow existing
  Deep Analysis settings conventions and safe defaults.
- **Provider catalog**: the existing catalog entry `{Provider: "ocre",
  Automatable: false, Reason: "pending_license_validation"}` becomes conditional
  on the enabled flag — automatable when enabled, `not_automated` when disabled —
  replacing the current unconditional `not_automated`.
- **Persistence**: only bounded per-job OCRE evidence is stored via existing
  Deep Analysis persistence; no new corpus table/store. Any schema change MUST be
  additive and preserve unrelated work.
- **Attribution surfaces**: report, proposal, provider status, and export
  surfaces gain OCRE ODbL 1.0 / ANS attribution rendering, distinct from existing
  provider attribution components.
- **ADR**: a license/attribution ADR (referenced as ADR-0010) is expected to
  record the ODbL 1.0 / ANS posture and G-OCRE gate outcome during planning.

## Testing & CI Constraints

- CI MUST rely on **offline fixtures / httptest / fake internal tools** only for
  OCRE — no live network calls in CI.
- An optional **live smoke test** against `https://nomisma.org/query` MUST be
  **manual and excluded from CI**.
- Injection, BCE/CE date mapping, ambiguity, malformed-binding, duplicate,
  timeout/500, result-cap, and cache-key behaviors MUST be covered by
  deterministic offline fixtures.

## Clarifications

No blocking clarifications. Every product/authorization decision (open
G-OCRE/T155; Nomisma SPARQL as the supported route; ODbL 1.0 / ANS attribution
and share-alike; exclude images and bulk ingestion; RPC out of scope; default-off
admin-gated flag; reuse of the Go-owned internal-tool boundary) was provided as
an explicit authorized decision and is reflected directly in the Requirements,
Assumptions, and Non-Goals above.
