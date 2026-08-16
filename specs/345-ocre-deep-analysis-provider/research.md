# Phase 0 Research: OCRE Automated Deep Analysis Provider

All NEEDS CLARIFICATION items resolved. Live, low-volume research calls were made
against the Nomisma triplestore during planning; every finding below is either
directly verified against the live endpoint (dated 2026-08-15) or grounded in the
existing beta code paths cited inline.

---

## R1. Nomisma SPARQL transport contract (BLOCKING question — RESOLVED)

**Decision:** OCRE automated access uses a single **`GET`** to
`https://nomisma.org/query?query=<url-encoded SPARQL>` with headers
`Accept: application/sparql-results+json` and a **non-default `User-Agent`**.
Response is standard **SPARQL 1.1 Query Results JSON** (`head.vars`,
`results.bindings[]`).

**Live evidence (2026-08-15):**

- `GET .../query?query=...` + custom `User-Agent` + SPARQL-results `Accept` →
  **HTTP 200**, `Content-Type: application/sparql-results+json; charset=utf-8`.
- `POST .../query` (form-encoded body) → **HTTP 403** (Cloudflare edge block),
  both with and without the SPARQL-results `Accept` header.
- Default PowerShell/no-`User-Agent` GET → **HTTP 403**. A browser-like
  `User-Agent` is required to pass the edge.

**Rationale:** The spec text said "POST/GET per validated endpoint." Validation
shows **only GET is reachable**; POST is blocked at the CDN. GET fully satisfies
the fixed-template requirement (the query is a fixed string with bound URI slots)
and matches the transport already used by `services/nomisma_client.go` (which
also uses GET with a URL-encoded `queries` param). **This does not block
implementation.**

**Verified sample response** (authority=Hadrian, denomination=denarius):

```json
{ "head": { "vars": ["type","label"] },
  "results": { "bindings": [
    { "type":  {"type":"uri","value":"http://numismatics.org/ocre/id/ric.2.hdn.39b"},
      "label": {"type":"literal","xml:lang":"en","value":"RIC II Hadrian 39b"} },
    { "type":  {"type":"uri","value":"http://numismatics.org/ocre/id/ric.2.hdn.143"},
      "label": {"type":"literal","xml:lang":"en","value":"RIC II Hadrian 143"} }
  ] } }
```

- OCRE type URIs are `http://numismatics.org/ocre/id/<ric-style-slug>` — host
  `numismatics.org`, which is exactly the OCRE entry in the existing citation
  allowlist (`merge.py::CITATION_HOST_ALLOWLIST["ocre"] = {"numismatics.org"}`
  and Go `nomisma`/`numismatics.org` allowlist). No allowlist change required.
- `label` is the human-readable RIC-style type ID ("RIC II Hadrian 39b") — used
  directly as the candidate label / type ID (FR-010).

**Alternatives considered:**
- *OCRE-hosted APIs (`numismatics.org/ocre/apis`)* — rejected; observed HTTP 500
  during F344 research and re-confirmed unstable; spec forbids them as the route.
- *OCRE front-end scraping* — rejected by spec (Non-Goals) and unnecessary.
- *POST SPARQL* — technically the "correct" verb but **blocked (403)** at the
  edge; GET is the validated, working path.

**Client bounds (mirror `nomisma_client.go`):** request timeout **8 s**,
response-size cap **1 MiB** (`io.LimitReader`), `Accept` fixed, context-aware
cancellation → typed `OCRECancelled`. Never returns a raw transport/parse error.

---

## R2. Fixed SPARQL templates + injection-proof binding (RESOLVED)

**Decision:** Two **fixed** query templates, both parameter-bound **only by
application-controlled Nomisma id URIs and integer/enumerated tokens** — never by
raw user/legend text. Free legend/inscription text is used **only** in
application-owned post-query scoring (R3), never inside SPARQL.

### Template E — evidence search (no known OCRE id)

```sparql
PREFIX nmo:  <http://nomisma.org/ontology#>
PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
SELECT ?type ?label ?auth ?den ?mint WHERE {
  ?type a nmo:TypeSeriesItem ;
        skos:prefLabel ?label .
  {bindings}                       # zero or more of the lines below, all URI-bound:
  # ?type nmo:hasAuthority    <http://nomisma.org/id/{ruler_slug}> .
  # ?type nmo:hasDenomination <http://nomisma.org/id/{denom_slug}> .
  # ?type nmo:hasMint         <http://nomisma.org/id/{mint_slug}> .
  # ?type nmo:hasMaterial     <http://nomisma.org/id/{material_slug}> .
  OPTIONAL { ?type nmo:hasAuthority    ?auth }
  OPTIONAL { ?type nmo:hasDenomination ?den  }
  OPTIONAL { ?type nmo:hasMint         ?mint }
  FILTER(LANG(?label)="en")
} LIMIT {cap_plus_margin}
```

### Template K — known-OCRE-id confirmation

```sparql
PREFIX nmo:  <http://nomisma.org/ontology#>
PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
SELECT ?label ?auth ?den ?mint WHERE {
  <http://numismatics.org/ocre/id/{ocre_id_slug}> skos:prefLabel ?label .
  OPTIONAL { <...{ocre_id_slug}> nmo:hasAuthority    ?auth }
  OPTIONAL { <...{ocre_id_slug}> nmo:hasDenomination ?den  }
  OPTIONAL { <...{ocre_id_slug}> nmo:hasMint         ?mint }
  FILTER(LANG(?label)="en")
} LIMIT 1
```

**Live-verified** (2026-08-15): Template E with `hasAuthority` + `hasDenomination`
(+ optional `hasMint <.../rome>`) returns bounded RIC types; Template K on
`ric.2.hdn.39b` returns `label="RIC II Hadrian 39b"`, `auth=.../hadrian`,
`den=.../denarius`, `mint=.../rome`. Predicates `nmo:TypeSeriesItem`,
`nmo:hasAuthority`, `nmo:hasDenomination`, `nmo:hasMint`, `skos:prefLabel` all
confirmed valid.

**Injection strategy (SC-010, FR-006):**
1. Every dynamic value is a **slug**, not free text. Slugs are produced by an
   application-owned normalization map (ruler/denomination/mint/material →
   Nomisma id). Any slug is validated against `^[a-z0-9]([a-z0-9_.-]*[a-z0-9])?$`
   and length-bounded before templating; a value that fails validation is
   **dropped as a bound parameter**, never interpolated.
2. Slugs are only ever placed **inside `<...>` URI brackets** in a fixed
   position; the query skeleton (SELECT/WHERE/predicates/LIMIT) is constant.
3. No `FILTER`/`VALUES` ever receives raw text; the only literal filter is the
   constant `LANG(?label)="en"`.
4. The OCRE id slug (Template K) is validated the same way and additionally must
   match the OCRE id shape (e.g. `^ric\.[0-9a-z_.()]+$`), else → `unresolved`
   (FR: "known-identifier confirmation" edge case) rather than a query.

Because adversarial input can only ever become a **rejected slug** (dropped) or a
**bracketed URI in a fixed slot**, the emitted query is **structurally identical**
regardless of input values — the exact property SC-010 asserts via offline
fixtures.

**BCE/CE dates:** date/period is **not** bound into SPARQL in the MVP (OCRE date
modeling via `nmo:hasStartDate/hasEndDate` is heterogeneous). Instead, decoded
authority (ruler) already scopes the date range; residual date evidence feeds
scoring (R3) with correct sign/era handling in Go time math, avoiding the BCE↔CE
edge entirely at the query layer. (Documented as a deliberate MVP bound.)

**Alternatives considered:** parameterized SPARQL via a driver (Nomisma's HTTP
endpoint has no bind-parameter protocol — fixed templating is the standard);
`regex()` on `prefLabel` for free-text (rejected — invites injection and is
non-deterministic across the corpus).

---

## R3. Deterministic application-owned scoring & explanations (RESOLVED)

**Decision:** Ranking/scoring is a **pure Go function** (`ocre_scoring.go`), no
LLM freedom (mirrors `merge.py`'s deterministic ordering ethos). For each distinct
returned `?type`:

- **Match score** = weighted sum of bound fields that the type actually matched:
  authority > denomination > mint > material (specificity order), plus a bounded
  bonus if decoded legend tokens (application-side, case-folded) appear in the
  `prefLabel`. Weights are fixed constants.
- **Confidence** = normalized score in `[0,1]`, clamped, deterministic.
- **Explanation / matched fields** = the explicit list of fields that bound
  (e.g. `["ruler:hadrian","denomination:denarius","mint:rome"]`) — no prose.
- **Ordering** (deterministic, ties fully broken): `(-score, -matchedFieldCount,
  canonicalURI ascending)`. Identical inputs ⇒ identical ordering (SC-005).
- **De-dup** by `?type` URI **before** ranking; **result cap** applied to distinct
  types after ranking (FR-012/FR-014).
- **Ambiguity:** when ≥2 candidates survive the cap, all are surfaced as separate
  candidate claims; nothing is collapsed to a single silent pick (FR-013).
- **Tie-break:** the canonical-URI ascending final key guarantees a stable order
  even for equal scores.
- **No-match:** zero rows after binding/validation ⇒ status `no_match` (not an
  error); Template K non-resolving id ⇒ `unresolved` explanation.

**Rationale:** Determinism is a hard success criterion (SC-005) and a
constitution expectation (no LLM invention/reordering of citations). Keeping
scoring in Go (the client owner) means the Python node is a thin typed adapter.

**Alternatives considered:** letting the synthesizer LLM rank OCRE candidates
(rejected — non-deterministic, violates SC-005 and the "no LLM freedom to invent
or reorder citations" spec assumption); embedding-based similarity (rejected —
overkill, non-deterministic, out of scope).

---

## R4. Mapping candidates into existing Deep Analysis contracts (RESOLVED)

**Decision:** Emit OCRE results through the **unchanged** `ProviderEvidence` /
`ProviderClaim` contract (`agent-internal-contract.md §4`). Per surviving
candidate, one `ProviderClaim`:

- `field`: `"coin_type"` (a coin-field-allowlist key; Go drops unknown keys, so
  `coin_type` must be present in the Go allowlist — verify/extend the allowlist,
  additive).
- `value`: the RIC-style label (e.g. `"RIC II Hadrian 39b"`).
- `citation`: canonical `https://numismatics.org/ocre/id/<slug>` (host on the
  OCRE allowlist — re-validated by `merge.validate_citations` **and** Go before
  persistence; SC-001/FR-011).
- `confidence`: deterministic score from R3.
- `excerpt`: the matched-fields explanation string (bounded ≤ 500).

Row-level `ProviderEvidence.attribution` carries the fixed OCRE string (R7).
Residual ambiguity = multiple claims on `coin_type`; the evaluator/synthesizer
already surface multi-claim fields as disagreements/alternatives, so **no
override** of image or other-provider evidence occurs (FR-013). **No generic
contract field is added.**

**Rationale:** FR-001 mandates OCRE be a provider node "not a new pipeline"; the
existing typed contract already models exactly what OCRE needs.

---

## R5. Go internal-tool / token / budget boundary (RESOLVED)

**Decision:** Add one internal endpoint `POST /api/internal/tools/ocre_search`
in the existing job-token group (`main.go:845` `internalDeepProviderTools`,
`middleware.InternalJobTokenRequired`). Handler `DeepProviderToolsHandler.OCRESearch`
mirrors `NomismaSearch`:

- Read `jobID` from the minted job token (`deepProviderJobID`).
- Consume an **OCRE per-job budget** via `deepProviderBudgets.TryConsume(jobID,
  "ocre", budget)` (new provider key, same tracker). Over budget →
  `200 {"status":"quota_limited", ...}` (never 5xx).
- Call `OCREClient.Search(ctx, params, limit)`; map outcome to a status string
  (`ok|empty|invalid_response|unavailable|timeout|cancelled`).
- Return `{"status", "candidates":[OCRECandidate], "attribution":"<fixed ODbL>"}`.
- Re-validate every candidate citation host is `numismatics.org` **Go-side**
  before returning (defense in depth with Python `merge.validate_citations`).

Wiring: construct `OCREClient` + `OCRECache` in `main.go` and pass into
`NewDeepProviderToolsHandler` (extend its constructor — same shape as
`numistaClient, deepNomismaClient` today). Budget default from a new
`DeepIdentificationOCRECallBudget` setting (additive, safe default 3).

**Rationale:** Reuses the exact F344 boundary (contract §7); Python never gets a
DB handle or upstream URL. Never-5xx keeps the job robust (FR-015).

**Alternatives considered:** reuse `nomisma_search` (rejected — different query,
license, allowlist semantics, and telemetry key); a separate token group
(rejected — the existing job-token group is exactly right).

---

## R6. Python provider node + router (RESOLVED)

**Decision:**
- `providers/ocre.py` `run(entry, tools, quick_evidence, notes)` becomes an
  **automated node**: build bound params from `quick_evidence` (ruler/denom/mint/
  material/legend + optional OCRE id from `coin_fields`), call
  `tools.ocre_search(...)`, map `{status,candidates}` → typed `ProviderEvidence`
  (`contributed|no_match|failed|timed_out|not_automated`), attach fixed
  attribution. It **never** calls Nomisma directly and catches
  `ProviderToolError` → typed `failed`/`timed_out` row (mirrors numista/nomisma
  nodes).
- `graph.py`: move `"ocre"` from `_TRIVIAL_PROVIDER_NODES` into the automated
  set so it runs through `_run_one_provider`'s `tools`/timeout path **when its
  catalog entry is `automatable`**; when `automatable=false` (flag off) it still
  short-circuits to a trivial `not_automated` row (the runner already emits
  non-automatable catalog entries in the fan-out). The node signature handles
  both by inspecting `entry.automatable`.
- `router.py`: **unchanged**. Because OCRE becomes an *automatable* catalog entry
  only when enabled, it flows through the normal router selection; the router is
  instructed (existing prompt) to include plausibly-useful providers. Roman-
  Imperial relevance is expressed by the node returning `no_match` cheaply when
  no ruler slug decodes — but to honor FR-003/US1-AC5 (don't select OCRE for
  clearly non-Roman coins) the **provider-tool call is skipped inside the node**
  when zero Roman-Imperial signals decode (returns `no_match`/`skipped` without a
  call). Explicit override already flows through `provider_override` (router
  honors it verbatim).

**Rationale:** Minimal, contract-preserving; keeps the "closed catalog / Go owns
the candidate list" invariant (router.py docstring) intact.

**Alternatives considered:** a bespoke Roman-Imperial classifier LLM node
(rejected — over-engineered; ruler-slug decode is a sufficient, deterministic
gate); keeping OCRE trivial and routing in Go only (rejected — the fan-out node
is the right seam and matches numista/nomisma).

---

## R7. Attribution, license & citation host (RESOLVED)

**Decision:** Fixed attribution string (FR-019), rendered distinct from all other
providers:

> `Coin type data: Online Coins of the Roman Empire (OCRE), American Numismatic Society — ODbL 1.0.`

- Canonical **type link**: the per-candidate `https://numismatics.org/ocre/id/<slug>`
  citation (already on the claim).
- **License link**: `https://opendatacommons.org/licenses/odbl/1-0/` — a fixed
  constant rendered by a **new** `OCREAttribution.vue` (separate component from
  `NomismaAttribution.vue`, per ADR 0009's explicit warning that Phase-2 OCRE
  must not reuse CC BY wording). Uses the existing `SafeExternalLink.vue`.
- Canonical citation **host** is `numismatics.org` (verified in R1) — already the
  OCRE allowlist entry; non-`numismatics.org` candidates are rejected (FR-011).
- Attribution renders on report, proposal, provider-status, and export whenever
  ≥1 OCRE claim is present; absent otherwise (FR-020, US2).

**Rationale:** ODbL 1.0 + ANS is a legal obligation (US2) and is a **distinct**
license from Nomisma CC BY 4.0 / Numista / RPC; conflation is a compliance
failure. The attribution travels as the row `attribution` string + a
provider-keyed UI license link, so **no generic contract change** is needed.

---

## R8. Per-job budget, TTL cache & telemetry (RESOLVED)

**Decision:**
- **Budget:** new setting `DeepIdentificationOCRECallBudget` (default 3, range
  1–20), enforced by the shared `deepProviderBudgets` tracker under key `"ocre"`.
- **Cache:** new `OCRECache` mirroring `nomisma_cache.go` — bounded (200 entries),
  TTL ~10 min, keyed on a **SHA-256 of the fully-bound parameter set** (ruler,
  denom, mint, material, legend-tokens, OCRE id, limit, flag-generation). A flag
  toggle changes the key generation input so no stale reuse crosses a flag change
  (spec "Cache keys" / "Flag changes" edge cases). **Negative (`no_match`) results
  cached; transient failures (`unavailable`/`timeout`/`cancelled`) never cached.**
- **Telemetry:** reuse the provider-run row fields (`Status`, `CallCount`,
  `LatencyMS`, `ErrorKind`) + cache hit/miss counter. **Never** persist notes or
  full legend text (FR-022) — only decoded slugs/counts.

**Rationale:** Directly mirrors ADR 0009's cache posture (proven pattern) and
satisfies FR-008/FR-009/FR-022.

**Alternatives considered:** singleflight/coalescing (rejected — same reasoning
as ADR 0009: single-owner, low frequency); cross-replica cache (rejected —
in-memory is acceptable and safe to lose on restart).

---

## R9. Persistence & no-migration analysis (RESOLVED)

**Decision:** **No DB migration.** Bounded OCRE evidence persists in the existing
`DeepIdentificationProviderRun` row (one row per provider per job:
`Provider="ocre"`, `Status`, `ClaimsJSON` holding the candidate claims,
`Confidence`, `CallCount`, `LatencyMS`, `ErrorKind`) and in the job proposal
document JSON built by `buildDeepProposalDocumentJSON`. The `ocre` provider name
and all needed statuses already exist (`models/deep_identification_provider_run.go`).
`invalid_response` maps to `Status=failed` + `ErrorKind="invalid_response"`
(the enum already has that error kind).

**Rationale:** FR-018 forbids a corpus/DB; the existing bounded structures hold
exactly the per-job evidence. Additive, reversible.

**Only additive change:** the coin-field allowlist may need `coin_type` added
(additive constant), and one new **setting key** default (`OCRECallBudget`) — both
non-migrating.

---

## R10. Testing, CI & smoke test (RESOLVED)

**Decision:**
- **Offline only in CI:** Go `httptest.Server` returns canned SPARQL-results JSON
  (success, empty, malformed bindings, oversize, 500, slow/timeout); Python tests
  use a fake `ProviderToolsClient`. Fixtures assert: SC-010 injection invariance
  (bound query byte-identical across adversarial inputs), timeout→typed row,
  malformed→`invalid_response`+dropped claims, duplicate URI de-dup, result-cap
  truncation, cache hit + negative-cache + flag-change key change, router OCRE
  selection/override, partial-job survival, attribution presence/distinctness,
  no-image assertion, flag-off zero-call.
- **Manual live smoke test (excluded from CI):** a `//go:build smoke` (or skipped-
  by-default) test hitting `https://nomisma.org/query` for the Hadrian/denarius
  fixture, documented in quickstart.

**Rationale:** Matches the spec's Testing & CI Constraints and F344 precedent.

---

## R11. Backward compatibility & rollback (RESOLVED)

**Decision:** Flag `SettingDeepIdentificationOCREEnabled` stays default `false`.
Flag off ⇒ catalog entry stays `Automatable:false` ⇒ node returns
`not_automated`, **zero** SPARQL calls, no attribution (FR-004/FR-016/SC-004).
Rollback = flip flag off (instant) or revert the additive code; no data migration
to undo. In-flight jobs keep their decided catalog (spec "Flag changes" edge).

---

## R12. ADR requirement (RESOLVED)

**Decision:** Author **ADR 0010 — OCRE ODbL 1.0 Automated Provider** (status
`Proposed`, → `Accepted` on merge), recording: opening G-OCRE/T155; Nomisma
SPARQL-over-GET as the validated route (with the POST-403 finding); ODbL 1.0 /
ANS attribution + share-alike posture and why bounded per-job evidence is not a
"derivative database"; distinctness from ADR 0009 (CC BY reconcile). Update
`docs/adr/README.md` index. Per Constitution §22 this is required for a new
third-party service/license posture.

---

## Open, non-blocking legal note (for the ADR / owner)

ODbL 1.0 share-alike attaches to **derivative databases**. This feature stores
only **bounded per-job evidence** (a handful of type links/labels/matched fields
per analysis) and **never** builds or persists an OCRE corpus/mirror (FR-018/
FR-021, SC-009), so the share-alike database obligation is not triggered under
the authorized product interpretation; attribution + license link are always
rendered. This interpretation is recorded in ADR 0010 for the single owner to
ratify; it is **not** a blocker for implementation.

---

## Sources

- Live Nomisma SPARQL calls, 2026-08-15 (see R1/R2 verified samples).
- Nomisma SPARQL endpoint & ontology: `https://nomisma.org/query`,
  `http://nomisma.org/ontology#` (`nmo:TypeSeriesItem`, `nmo:hasAuthority`,
  `nmo:hasDenomination`, `nmo:hasMint`).
- OCRE (Online Coins of the Roman Empire), American Numismatic Society —
  `http://numismatics.org/ocre/`; type URI prefix `http://numismatics.org/ocre/id/`.
- ODbL 1.0 — `https://opendatacommons.org/licenses/odbl/1-0/`.
- Beta code paths: `src/api/services/nomisma_client.go`, `nomisma_cache.go` (ADR
  0009), `handlers/internal_tools.go`, `services/deep_identification_pipeline_runner.go`,
  `src/agent/app/teams/deep_identification/{router,graph,merge}.py`,
  `providers/ocre.py`, `tools/provider_tools.py`, `models/responses.py`,
  `models/requests.py`, `models/deep_identification_provider_run.go`,
  `settings_service.go`.
- Feature 344 contract: `specs/344-deep-agentic-coin-identification/contracts/agent-internal-contract.md`.
